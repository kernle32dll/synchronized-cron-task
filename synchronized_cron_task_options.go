package crontask

import (
	"github.com/sirupsen/logrus"
	"time"
)

// TaskOptions bundles all available configuration
// properties for a synchronized cron task.
type TaskOptions struct {
	Name           string
	CronExpression string

	Logger *logrus.Logger

	LeadershipTimeout time.Duration
	LockTimeout       time.Duration
	LockHeartbeat     time.Duration
}

// TaskOption represents an option for a synchronized cron task.
type TaskOption func(*TaskOptions)

// TaskName sets the name of the synchronized cron task.
// The default is crontask.DefaultName.
func TaskName(name string) TaskOption {
	return func(c *TaskOptions) {
		c.Name = name
	}
}

// CronExpression sets the cron expression of the synchronized cron task.
// The default is crontask.DefaultCronExpression.
func CronExpression(cronExpression string) TaskOption {
	return func(c *TaskOptions) {
		c.CronExpression = cronExpression
	}
}

// Logger sets the logger of the synchronized cron task.
// The default is the logrus global default logger.
func Logger(logger *logrus.Logger) TaskOption {
	return func(c *TaskOptions) {
		c.Logger = logger
	}
}

// LeadershipTimeout sets the timeout of a single execution of the
// synchronized cron task.
// The default is crontask.DefaultLeadershipTimeout.
func LeadershipTimeout(leadershipTimeout time.Duration) TaskOption {
	return func(c *TaskOptions) {
		c.LeadershipTimeout = leadershipTimeout
	}
}

// LockTimeout sets the timeout for the lock of a single execution of the
// synchronized cron task. It is good practice to keep the timeout small,
// for fast failure detection.
// The default is crontask.DefaultLockTimeout.
func LockTimeout(lockTimeout time.Duration) TaskOption {
	return func(c *TaskOptions) {
		c.LockTimeout = lockTimeout
	}
}

// LockHeartbeat sets the interval, in which an acquired lock should be
// renewed (to the total of LeadershipTimeout). This should be smaller
// than the LockTimeout, otherwise the task will have no chance to ever
// renew the lock.
// The default is crontask.DefaultLockHeartbeat.
func LockHeartbeat(lockHeartbeat time.Duration) TaskOption {
	return func(c *TaskOptions) {
		c.LockHeartbeat = lockHeartbeat
	}
}
