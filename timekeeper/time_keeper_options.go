package timekeeper

import (
	"github.com/go-redis/redis/v8"
	"time"
)

// Options bundles all available configuration
// properties for a time keeper.
type Options struct {
	RedisExecListName string
	RedisLastExecName string

	KeepTaskList bool
	KeepLastTask bool

	CleanUpTask *CleanUpOptions
}

// Option represents an option for a time keeper.
type Option func(*Options)

// RedisExecListName sets redis key for the list used to track
// all executions of tasks managed by the time keeper.
// The default is "timekeeper.executions.list".
func RedisExecListName(redisExecListName string) Option {
	return func(c *Options) {
		c.RedisExecListName = redisExecListName
	}
}

// RedisLastExecName sets redis key for the set used to track
// the latest execution of tasks managed by the time keeper.
// The default is "timekeeper.executions.aggregation".
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

// CleanUpTask enables the clean up task, which discards old executions.
// The default is nil.
func CleanUpTask(client *redis.Client, setters ...CleanUpOption) Option {
	return func(c *Options) {
		if client != nil {
			c.CleanUpTask = &CleanUpOptions{
				Client:       client,
				TasksTimeOut: 30 * (24 * time.Hour),
				TaskName:     "timekeeper.cleanup",
			}

			for _, setter := range setters {
				setter(c.CleanUpTask)
			}
		} else {
			c.CleanUpTask = nil
		}
	}
}

// CleanUpOptions bundles all available configuration
// properties for a time keeper clean up task.
type CleanUpOptions struct {
	Client       *redis.Client
	TasksTimeOut time.Duration
	TaskName     string
}

// CleanUpOption represents an option for a clean up task.
type CleanUpOption func(*CleanUpOptions)

// CleanUpTaskName sets the name of the cleanup task.
// The default is "timekeeper.cleanup".
func CleanUpTaskName(taskName string) CleanUpOption {
	return func(c *CleanUpOptions) {
		c.TaskName = taskName
	}
}

// CleanUpTasksTimeOut sets the timeout after which tasks are cleaned up.
// The default is 30 days.
func CleanUpTasksTimeOut(tasksTimeOut time.Duration) CleanUpOption {
	return func(c *CleanUpOptions) {
		c.TasksTimeOut = tasksTimeOut
	}
}
