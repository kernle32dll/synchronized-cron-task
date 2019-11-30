package crontask

import (
	"time"
)

// TaskOptions bundles all available configuration
// properties for a synchronized cron task.
type TaskOptions struct {
	Name           string
	CronExpression string

	LeadershipTimeout time.Duration
	LockTimeout       time.Duration
	LockHeartbeat     time.Duration
}

// TaskOption represents an option for a synchronized cron task.
type TaskOption func(*TaskOptions)

// TaskName sets the name of the synchronized cron task.
// The default is "Default Synchronized Task".
func TaskName(name string) TaskOption {
	return func(c *TaskOptions) {
		c.Name = name
	}
}

// CronExpression sets the cron expression of the synchronized cron task.
// The default is "0 * * * * *", which means every minute.
func CronExpression(cronExpression string) TaskOption {
	return func(c *TaskOptions) {
		c.CronExpression = cronExpression
	}
}

// LeadershipTimeout sets the timeout of a single execution of the
// synchronized cron task.
// The default is 30 seconds.
func LeadershipTimeout(leadershipTimeout time.Duration) TaskOption {
	return func(c *TaskOptions) {
		c.LeadershipTimeout = leadershipTimeout
	}
}

// LockTimeout sets the timeout for the lock of a single execution of the
// synchronized cron task. It is good practice to keep the timeout small,
// for fast failure detection.
// The default is 5 seconds.
func LockTimeout(lockTimeout time.Duration) TaskOption {
	return func(c *TaskOptions) {
		c.LockTimeout = lockTimeout
	}
}

// LockHeartbeat sets the interval, in which an acquired lock should be
// renewed (to the total of LeadershipTimeout). This should be smaller
// than the LockTimeout, otherwise the task will have no chance to ever
// renew the lock.
// The default is 1 seconds.
func LockHeartbeat(lockHeartbeat time.Duration) TaskOption {
	return func(c *TaskOptions) {
		c.LockHeartbeat = lockHeartbeat
	}
}
