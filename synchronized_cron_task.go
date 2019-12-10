package crontask

import (
	"github.com/netology-group/redislock/v7"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"

	"context"
	"fmt"
	"io/ioutil"
	"strings"
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
	name string

	cron   *cron.Cron
	locker *redislock.Client

	logger *logrus.Logger

	electionInProgress *int32
	shutdownFunc       func()
}

// Name returns the name of the task.
func (synchronizedCronTask *SynchronizedCronTask) Name() string {
	return synchronizedCronTask.name
}

// Stop gracefully stops the task, while also freeing most of its underlying resources.
func (synchronizedCronTask *SynchronizedCronTask) Stop(ctx context.Context) {
	synchronizedCronTask.shutdownFunc()

	select {
	case <-ctx.Done():
	case <-synchronizedCronTask.cron.Stop().Done():
	}

	// Allow everything to be properly gc'd
	synchronizedCronTask.electionInProgress = nil
	synchronizedCronTask.cron = nil
}

// TaskFunc is a function, that is called upon the cron firing.
type TaskFunc func(ctx context.Context, task Task) error

// Task is an abstraction of a running task.
type Task interface {
	Name() string
	NextTime() time.Time
}

// NewSynchronizedCronTaskWithOptions creates a new SynchronizedCronTask instance, or errors out
// if the provided cron expression was invalid.
func NewSynchronizedCronTaskWithOptions(client redislock.RedisClient, taskFunc TaskFunc, options *TaskOptions) (*SynchronizedCronTask, error) {
	shutdownCtx, leadershipCancel := context.WithCancel(context.Background())

	cronOptions := []cron.Option{
		cron.WithLocation(time.UTC),
	}

	if fields := strings.Fields(options.CronExpression); len(fields) > 5 {
		cronOptions = append(cronOptions, cron.WithSeconds())
	}

	synchronizedTask := &SynchronizedCronTask{
		name: options.Name,

		cron:   cron.New(cronOptions...),
		locker: redislock.New(client),

		logger: options.Logger,

		electionInProgress: new(int32),
		shutdownFunc:       leadershipCancel,
	}

	_, err := synchronizedTask.cron.AddFunc(options.CronExpression, func() {
		if atomic.LoadInt32(synchronizedTask.electionInProgress) == electing {
			synchronizedTask.logger.Tracef("Skipping election for synchronized task %q, as leadership is already owned", synchronizedTask.name)
			return
		}

		atomic.StoreInt32(synchronizedTask.electionInProgress, electing)
		defer func() {
			atomic.StoreInt32(synchronizedTask.electionInProgress, notElecting)
		}()

		// --------------

		leadershipContext, cancel := context.WithDeadline(shutdownCtx, time.Now().Add(options.LeadershipTimeout))
		defer cancel()

		start := time.Now()
		if err := synchronizedTask.handleElectionAttempt(
			leadershipContext,
			options.LockTimeout,
			options.LockHeartbeat,
			taskFunc,
		); err != nil && err != context.Canceled && err != context.DeadlineExceeded {
			synchronizedTask.logger.Errorf("Error while trying to temporarily gain leadership for synchronized task %q: %s", synchronizedTask.name, err)
		} else {
			synchronizedTask.logger.Infof("Successfully filled executed task %q in %s", synchronizedTask.name, time.Since(start))
		}
	})
	if err != nil {
		leadershipCancel()
		return nil, err
	}

	synchronizedTask.cron.Start()

	return synchronizedTask, nil
}

// NewSynchronizedCronTask creates a new SynchronizedCronTask instance, or errors out
// if the provided cron expression was invalid.
func NewSynchronizedCronTask(client redislock.RedisClient, taskFunc TaskFunc, setters ...TaskOption) (*SynchronizedCronTask, error) {
	// Default Options
	args := &TaskOptions{
		Name: "Default Synchronized Task",

		Logger: logrus.StandardLogger(),

		CronExpression:    "0 * * * * *",
		LeadershipTimeout: 30 * time.Second,
		LockTimeout:       5 * time.Second,
		LockHeartbeat:     1 * time.Second,
	}

	for _, setter := range setters {
		setter(args)
	}

	if args.Logger == nil {
		// Create a "noop" logger, so we don't have to check for
		// the logger being nil
		logger := logrus.New()
		logger.Out = ioutil.Discard

		args.Logger = logger
	}

	return NewSynchronizedCronTaskWithOptions(client, taskFunc, args)
}

// ExecuteNow forces the cron to fire immediately. Locking is still
// honored, so no concurrent task execution can be forced this way.
func (synchronizedCronTask *SynchronizedCronTask) ExecuteNow() {
	if synchronizedCronTask.cron == nil {
		synchronizedCronTask.logger.Warnf("Tried to force execution of synchronized cron task %s, which was already stopped.", synchronizedCronTask.name)
		return
	}

	synchronizedCronTask.cron.Entries()[0].Job.Run()
}

// NextTime returns the next time the cron task will fire.
func (synchronizedCronTask *SynchronizedCronTask) NextTime() time.Time {
	if synchronizedCronTask.cron == nil {
		synchronizedCronTask.logger.Warnf("Tried to retrieve next execution of synchronized cron task %s, which was already stopped.", synchronizedCronTask.name)
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
	logger := synchronizedCronTask.logger.WithField("task_name", synchronizedCronTask.name)

	// Try to lock
	logger.Tracef("Trying to temporarily gain leadership for synchronized task %q", synchronizedCronTask.name)

	lock, err := synchronizedCronTask.locker.Obtain(
		fmt.Sprintf("%s.lock", synchronizedCronTask.name),
		lockTimeout,
		&redislock.Options{
			Context: ctx,
		},
	)
	if err != nil {
		return err
	}

	defer func() {
		logger.Tracef("Resigning temporary leadership for synchronized task %q", synchronizedCronTask.name)
		if err := lock.Release(); err != nil {
			logger.Warnf("Failed to resign leadership for synchronized task %q: %s - the service should be able to recover from this", synchronizedCronTask.name, err)
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

	return synchronizedCronTask.blockForFinish(wrappedContext, doneChannel, ticker, lock, lockTimeout)
}

func (synchronizedCronTask *SynchronizedCronTask) blockForFinish(ctx context.Context,
	doneChannel chan error, ticker *time.Ticker,
	lock *redislock.Lock, lockTimeout time.Duration,
) error {
	logger := synchronizedCronTask.logger.WithField("task_name", synchronizedCronTask.name)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-doneChannel:
			if err != nil {
				return fmt.Errorf(cronErrorFormat, synchronizedCronTask.name, err)
			}

			return nil
		case <-ticker.C:
			// Renew the lock
			if err := lock.Refresh(lockTimeout, &redislock.Options{
				Context: ctx,
			}); err != nil {
				return fmt.Errorf(
					renewalErrorFormat,
					synchronizedCronTask.name, err,
				)
			}

			logger.Debugf("Renewed leadership lock for long running synchronized task %q fill", synchronizedCronTask.name)
		}
	}
}
