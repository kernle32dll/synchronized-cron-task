package timekeeper

import (
	"time"
)

// Options bundles all available configuration
// properties for a time keeper.
type Options struct {
	RedisPrefix       string
	RedisExecListName string
	RedisLastExecName string

	KeepTaskList bool
	KeepLastTask bool

	TaskListTimeOut time.Duration
}

// Option represents an option for a time keeper.
type Option func(*Options)

// RedisPrefix sets redis key prefix used by the time keeper.
// The default is "timekeeper".
func RedisPrefix(redisPrefix string) Option {
	return func(c *Options) {
		c.RedisPrefix = redisPrefix
	}
}

// RedisExecListName sets redis key for the list used to track
// all executions of tasks managed by the time keeper.
// The default is "executions.list".
func RedisExecListName(redisExecListName string) Option {
	return func(c *Options) {
		c.RedisExecListName = redisExecListName
	}
}

// RedisLastExecName sets redis key for the set used to track
// the latest execution of tasks managed by the time keeper.
// The default is "executions.aggregation".
func RedisLastExecName(redisLastExecName string) Option {
	return func(c *Options) {
		c.RedisLastExecName = redisLastExecName
	}
}

// KeepTaskList enables or disables the keeping of a list of all
// executions of tasks managed by the time keeper.
// The default is true.
func KeepTaskList(keepTaskList bool) Option {
	return func(c *Options) {
		c.KeepTaskList = keepTaskList
	}
}

// KeepLastTask enables or disables the keeping of a set of the
// latest execution of tasks managed by the time keeper.
// The default is true.
func KeepLastTask(keepTaskList bool) Option {
	return func(c *Options) {
		c.KeepLastTask = keepTaskList
	}
}

// TaskListTimeOut sets the timeout, after which older task executions
// are purged from the task execution list.
// The default is 30 days.
func TaskListTimeOut(taskListTimeOut time.Duration) Option {
	return func(c *Options) {
		c.TaskListTimeOut = taskListTimeOut
	}
}
