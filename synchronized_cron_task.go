package crontask

import (
	"github.com/netology-group/redislock/v7"
	"github.com/robfig/cron"
	"github.com/sirupsen/logrus"

	"context"
	"fmt"
	"sync/atomic"
	"time"
)

const (
	electing    = int32(1)
	notElecting = int32(0)
)

type SynchronizedCronTask struct {
	Name string

	cron   *cron.Cron
	locker *redislock.Client

	electionInProgress *int32
	shutdownFunc       func()
}

type TaskConfig struct {
	LeadershipCronInterval string
	LeadershipTimeout      time.Duration
	LockTimeout            time.Duration
	LockHeartbeat          time.Duration
}

func (synchronizedCronTask *SynchronizedCronTask) Stop() {
	synchronizedCronTask.shutdownFunc()
	synchronizedCronTask.cron.Stop()

	// Allow everything to be properly gc'd
	synchronizedCronTask.electionInProgress = nil
	synchronizedCronTask.cron = nil
}

type TaskFunc func(ctx context.Context, task *SynchronizedCronTask) error

func CreateSynchronizedCronTask(client redislock.RedisClient, name string, cronTaskConfig TaskConfig, taskFunc TaskFunc) (*SynchronizedCronTask, error) {
	shutdownCtx, leadershipCancel := context.WithCancel(context.Background())

	synchronizedTask := &SynchronizedCronTask{
		Name: name,

		cron:   cron.NewWithLocation(time.UTC),
		locker: redislock.New(client),

		electionInProgress: new(int32),
		shutdownFunc:       leadershipCancel,
	}

	if err := synchronizedTask.cron.AddFunc(cronTaskConfig.LeadershipCronInterval, func() {
		if atomic.LoadInt32(synchronizedTask.electionInProgress) == electing {
			logrus.Tracef("Skipping election for synchronized task %q, as leadership is already owned", synchronizedTask.Name)
			return
		}

		atomic.StoreInt32(synchronizedTask.electionInProgress, electing)
		defer func() {
			atomic.StoreInt32(synchronizedTask.electionInProgress, notElecting)
		}()

		// --------------

		leadershipContext, cancel := context.WithDeadline(shutdownCtx, time.Now().Add(cronTaskConfig.LeadershipTimeout))
		defer cancel()

		start := time.Now()
		if err := synchronizedTask.handleElectionAttempt(
			leadershipContext,
			cronTaskConfig.LockTimeout,
			cronTaskConfig.LockHeartbeat,
			taskFunc,
		); err != nil && err != context.Canceled && err != context.DeadlineExceeded {
			logrus.Errorf("Error while trying to temporarily gain leadership for synchronized task %q: %s", synchronizedTask.Name, err)
		} else {
			logrus.Infof("Successfully filled executed task %q in %s", synchronizedTask.Name, time.Since(start))
		}
	}); err != nil {
		leadershipCancel()
		return nil, err
	}

	synchronizedTask.cron.Start()

	return synchronizedTask, nil
}

func (synchronizedCronTask *SynchronizedCronTask) ExecuteNow() {
	synchronizedCronTask.cron.Entries()[0].Job.Run()
}

func (synchronizedCronTask *SynchronizedCronTask) NextTime() time.Time {
	return synchronizedCronTask.cron.Entries()[0].Schedule.Next(time.Now())
}

func (synchronizedCronTask *SynchronizedCronTask) handleElectionAttempt(
	ctx context.Context,
	lockTimeout time.Duration,
	lockHeartbeat time.Duration,
	taskFunc TaskFunc,
) error {
	logger := logrus.WithField("task_name", synchronizedCronTask.Name)

	// Try to lock
	logger.Tracef("Trying to temporarily gain leadership for synchronized task %q", synchronizedCronTask.Name)

	lock, err := synchronizedCronTask.locker.Obtain(
		fmt.Sprintf("%s.lock", synchronizedCronTask.Name),
		lockTimeout,
		&redislock.Options{
			Context: ctx,
		},
	)
	if err != nil {
		return err
	}

	defer func() {
		logger.Tracef("Resigning temporary leadership for synchronized task %q", synchronizedCronTask.Name)
		if err := lock.Release(); err != nil {
			logger.Warnf("Failed to resign leadership for synchronized task %q - the service should be able to recover from this", synchronizedCronTask.Name)
		}
	}()

	// Heartbeat ticker to retain the lock while we execute the handler
	ticker := time.NewTicker(lockHeartbeat)
	defer ticker.Stop()

	// Wrap the context, so we can signal into the go routine if we need to abort mid-lock
	wrappedContext, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	doneChannel := make(chan error, 1)
	go func() {
		doneChannel <- taskFunc(wrappedContext, synchronizedCronTask)
	}()

	for {
		select {
		case <-wrappedContext.Done():
			return wrappedContext.Err()
		case err := <-doneChannel:
			if err != nil {
				return fmt.Errorf("error while executing synchronized task function %q: %w", synchronizedCronTask.Name, err)
			} else {
				return nil
			}
		case <-ticker.C:
			// Renew the lock
			if err := lock.Refresh(lockTimeout, &redislock.Options{
				Context: ctx,
			}); err != nil {
				return fmt.Errorf(
					"failed to renew leadership for synchronized task %q lock while executing: %w - crudely canceling",
					synchronizedCronTask.Name, err,
				)
			}

			logger.Debugf("Renewed leadership lock for long running synchronized task %q fill", synchronizedCronTask.Name)
		}
	}
}
