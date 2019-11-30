package crontask

import (
	"time"
)

// TimeKeeperOptions bundles all available configuration
// properties for a time keeper.
type TimeKeeperOptions struct {
	RedisPrefix       string
	RedisExecListName string
	RedisLastExecName string

	KeepTaskList bool
	KeepLastTask bool

	TaskListTimeOut time.Duration
}

// TimeKeeperOption represents an option for a time keeper.
type TimeKeeperOption func(*TimeKeeperOptions)

// TimeKeeperRedisPrefix sets redis key prefix used by the time keeper.
// The default is "timekeeper".
func TimeKeeperRedisPrefix(redisPrefix string) TimeKeeperOption {
	return func(c *TimeKeeperOptions) {
		c.RedisPrefix = redisPrefix
	}
}

// TimeKeeperRedisExecListName sets redis key for the list used to track
// all executions of tasks managed by the time keeper.
// The default is "executions.list".
func TimeKeeperRedisExecListName(redisExecListName string) TimeKeeperOption {
	return func(c *TimeKeeperOptions) {
		c.RedisExecListName = redisExecListName
	}
}

// TimeKeeperRedisLastExecName sets redis key for the set used to track
// the latest execution of tasks managed by the time keeper.
// The default is "executions.aggregation".
func TimeKeeperRedisLastExecName(redisLastExecName string) TimeKeeperOption {
	return func(c *TimeKeeperOptions) {
		c.RedisLastExecName = redisLastExecName
	}
}

// TimeKeeperKeepTaskList enables or disables the keeping of a list of all
// executions of tasks managed by the time keeper.
// The default is true.
func TimeKeeperKeepTaskList(keepTaskList bool) TimeKeeperOption {
	return func(c *TimeKeeperOptions) {
		c.KeepTaskList = keepTaskList
	}
}

// TimeKeeperKeepLastTask enables or disables the keeping of a set of the
// latest execution of tasks managed by the time keeper.
// The default is true.
func TimeKeeperKeepLastTask(keepTaskList bool) TimeKeeperOption {
	return func(c *TimeKeeperOptions) {
		c.KeepLastTask = keepTaskList
	}
}

// TimeKeeperTaskListTimeOut sets the timeout, after which older task executions
// are purged from the task execution list.
// The default is 30 days.
func TimeKeeperTaskListTimeOut(taskListTimeOut time.Duration) TimeKeeperOption {
	return func(c *TimeKeeperOptions) {
		c.TaskListTimeOut = taskListTimeOut
	}
}
