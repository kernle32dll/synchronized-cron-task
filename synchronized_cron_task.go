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

// SynchronizedCronTask describes a task, which is identified by a cron expression and a
// redis client it uses to synchronize execution across running instances.
//
// It supports graceful shutdowns via its Stop() function.
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

// Stop gracefully stops the task, while also freeing most of its underlying resources.
func (synchronizedCronTask *SynchronizedCronTask) Stop() {
	synchronizedCronTask.shutdownFunc()
	synchronizedCronTask.cron.Stop()

	// Allow everything to be properly gc'd
	synchronizedCronTask.electionInProgress = nil
	synchronizedCronTask.cron = nil
}

// TaskFunc is a function, that is called upon the cron firing.
type TaskFunc func(ctx context.Context, task *SynchronizedCronTask) error

// CreateSynchronizedCronTask creates a new SynchronizedCronTask instance, or errors out
// if the provided cron expression was invalid.
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

// ExecuteNow forces the cron to fire immediately. Locking is still
// honored, so no concurrent task execution can be forced this way.
func (synchronizedCronTask *SynchronizedCronTask) ExecuteNow() {
	if synchronizedCronTask.cron == nil {
		logrus.Warnf("Tried to force execution of synchronized cron task %s, which was already stopped.", synchronizedCronTask.Name)
		return
	}

	synchronizedCronTask.cron.Entries()[0].Job.Run()
}

// NextTime returns the next time the cron task will fire.
func (synchronizedCronTask *SynchronizedCronTask) NextTime() time.Time {
	if synchronizedCronTask.cron == nil {
		logrus.Warnf("Tried to retrieve next execution of synchronized cron task %s, which was already stopped.", synchronizedCronTask.Name)
		return time.Time{}
	}

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
			}

			return nil
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
