package crontask

import (
	"github.com/bsm/redislock"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"

	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"time"
)

const (
	electing    = int32(1)
	notElecting = int32(0)
)

const (
	// DefaultName is the default name of a synchronized cron task.
	DefaultName = "Default Synchronized Task"

	// DefaultCronExpression is the default cron expression of a
	// synchronized cron task. 0 * * * * * means every minute.
	DefaultCronExpression = "0 * * * * *"

	// DefaultLeadershipTimeout is the default timeout of a single
	// execution of a synchronized cron task.
	DefaultLeadershipTimeout = 30 * time.Second

	// DefaultLockTimeout is the default timeout for the lock of a
	// single execution of a synchronized cron task.
	DefaultLockTimeout = 5 * time.Second

	// DefaultLockHeartbeat is the default interval, in which an
	// acquired lock should be renewed (to the total of the leadership
	// timeout).
	DefaultLockHeartbeat = 1 * time.Second
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

func (synchronizedCronTask *SynchronizedCronTask) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Name     string    `json:"name"`
		NextTime time.Time `json:"nextTime"`
	}{
		Name:     synchronizedCronTask.Name(),
		NextTime: synchronizedCronTask.NextTime(),
	})
}

// Name returns the name of the task.
func (synchronizedCronTask *SynchronizedCronTask) Name() string {
	return synchronizedCronTask.name
}

// Stop gracefully stops the task, while also freeing most of its underlying resources.
func (synchronizedCronTask *SynchronizedCronTask) Stop(ctx context.Context) {
	select {
	case <-ctx.Done():
	case <-synchronizedCronTask.cron.Stop().Done():
	}

	synchronizedCronTask.shutdownFunc()

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

// logrusCronLoggerBridge bridges logrus to work with robfig/cron.
type logrusCronLoggerBridge struct {
	logger *logrus.Logger
}

func (l logrusCronLoggerBridge) Info(msg string, keysAndValues ...interface{}) {
	l.logger.
		WithFields(l.translateKeysAndValues(keysAndValues)).
		// intentionally trace, since cron output is hardly of interest, if not low level debugging
		Trace(fmt.Sprintf("cron: %s", msg))
}

func (l logrusCronLoggerBridge) Error(err error, msg string, keysAndValues ...interface{}) {
	l.logger.
		WithFields(l.translateKeysAndValues(keysAndValues)).
		WithError(err).
		Error(fmt.Sprintf("cron: %s", msg))
}

func (l logrusCronLoggerBridge) translateKeysAndValues(keysAndValues []interface{}) logrus.Fields {
	fields := make(logrus.Fields, len(keysAndValues)/2)
	for i := 0; i < len(keysAndValues); i += 2 {
		fields[keysAndValues[i].(string)] = keysAndValues[i+1]
	}

	return fields
}

// NewSynchronizedCronTaskWithOptions creates a new SynchronizedCronTask instance, or errors out
// if the provided cron expression was invalid.
func NewSynchronizedCronTaskWithOptions(client redislock.RedisClient, taskFunc TaskFunc, options *TaskOptions) (*SynchronizedCronTask, error) {
	if options.Logger == nil {
		// Create a "noop" logger, so we don't have to check for
		// the logger being nil
		logger := logrus.New()
		logger.Out = io.Discard

		options.Logger = logger
	}

	shutdownCtx, leadershipCancel := context.WithCancel(context.Background())

	cronOptions := []cron.Option{
		cron.WithLocation(time.UTC),
		cron.WithLogger(logrusCronLoggerBridge{options.Logger}),
		cron.WithParser(cron.NewParser(
			cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
		)),
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
		); err != nil {
			if errors.Is(err, redislock.ErrNotObtained) {
				synchronizedTask.logger.Debugf("Could not gain temporary leadership for synchronized task %q - ignoring", synchronizedTask.name)
			} else if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				synchronizedTask.logger.Errorf("Forcefully giving up leadership for synchronized task %q - timeout of %s reached", synchronizedTask.name, options.LeadershipTimeout)
			} else {
				synchronizedTask.logger.Errorf("Error while trying to temporarily gain leadership for synchronized task %q: %s", synchronizedTask.name, err)
			}
		} else {
			synchronizedTask.logger.Infof("Successfully executed synchronized task %q in %s", synchronizedTask.name, time.Since(start))
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
		Name: DefaultName,

		Logger: logrus.StandardLogger(),

		CronExpression:    DefaultCronExpression,
		LeadershipTimeout: DefaultLeadershipTimeout,
		LockTimeout:       DefaultLockTimeout,
		LockHeartbeat:     DefaultLockHeartbeat,
	}

	for _, setter := range setters {
		setter(args)
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
	logger := synchronizedCronTask.logger.WithContext(ctx).WithField("task_name", synchronizedCronTask.name)

	// Try to lock
	logger.Tracef("Trying to temporarily gain leadership for synchronized task %q", synchronizedCronTask.name)

	lock, err := synchronizedCronTask.locker.Obtain(
		ctx,
		fmt.Sprintf("%s.lock", synchronizedCronTask.name),
		lockTimeout,
		nil,
	)
	if err != nil {
		return err
	}

	defer func() {
		logger.Tracef("Resigning temporary leadership for synchronized task %q", synchronizedCronTask.name)
		if err := lock.Release(ctx); err != nil {
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
	logger := synchronizedCronTask.logger.WithContext(ctx).WithField("task_name", synchronizedCronTask.name)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-doneChannel:
			if err != nil {
				return fmt.Errorf("error while executing synchronized task function %q: %w", synchronizedCronTask.name, err)
			}

			return nil
		case <-ticker.C:
			// Renew the lock
			if err := lock.Refresh(ctx, lockTimeout, nil); err != nil {
				return fmt.Errorf(
					"failed to renew leadership for synchronized task %q lock while executing: %w - crudely canceling",
					synchronizedCronTask.name, err,
				)
			}

			logger.Debugf("Renewed leadership lock for long running synchronized task %q fill", synchronizedCronTask.name)
		}
	}
}
