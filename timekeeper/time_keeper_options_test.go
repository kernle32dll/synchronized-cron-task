package timekeeper_test

import (
	"github.com/kernle32dll/synchronized-cron-task/timekeeper"

	"github.com/go-redis/redis/v8"

	"testing"
	"time"
)

// Tests that the RedisExecListName option correctly applies.
func Test_TimeKeeperOption_RedisExecListName(t *testing.T) {
	// given
	option := timekeeper.RedisExecListName("bar")
	options := &timekeeper.Options{RedisExecListName: "foo"}

	// when
	option(options)

	// then
	if options.RedisExecListName != "bar" {
		t.Errorf("redis exec list name not correctly applied, got %s", options.RedisExecListName)
	}
}

// Tests that the RedisLastExecName option correctly applies.
func Test_TimeKeeperOption_RedisLastExecName(t *testing.T) {
	// given
	option := timekeeper.RedisLastExecName("bar")
	options := &timekeeper.Options{RedisLastExecName: "foo"}

	// when
	option(options)

	// then
	if options.RedisLastExecName != "bar" {
		t.Errorf("redis last exec name not correctly applied, got %s", options.RedisLastExecName)
	}
}

// Tests that the KeepTaskList option correctly applies.
func Test_TimeKeeperOption_KeepTaskList(t *testing.T) {
	// given
	option := timekeeper.KeepTaskList(true)
	options := &timekeeper.Options{KeepTaskList: false}

	// when
	option(options)

	// then
	if options.KeepTaskList != true {
		t.Errorf("keep task list not correctly applied, got %t", options.KeepTaskList)
	}
}

// Tests that the KeepLastTask option correctly applies.
func Test_TimeKeeperOption_KeepLastTask(t *testing.T) {
	// given
	option := timekeeper.KeepLastTask(true)
	options := &timekeeper.Options{KeepLastTask: false}

	// when
	option(options)

	// then
	if options.KeepLastTask != true {
		t.Errorf("keep last task not correctly applied, got %t", options.KeepLastTask)
	}
}

// Tests that the TasksTimeOut option correctly applies.
func Test_TimeKeeperOption_CleanUpTask(t *testing.T) {
	t.Run("Enable", func(t *testing.T) {
		// given
		someClient := &redis.Client{}
		option := timekeeper.CleanUpTask(someClient, timekeeper.CleanUpTaskName("test"), timekeeper.CleanUpTasksTimeOut(time.Hour))
		options := &timekeeper.Options{CleanUpTask: nil}

		// when
		option(options)

		// then
		if options.CleanUpTask == nil {
			t.Fatal("clean up task not correctly applied, got nil")
		}

		if options.CleanUpTask.Client != someClient {
			t.Error("clean up task client not correctly applied")
		}
		if options.CleanUpTask.TaskName != "test" {
			t.Errorf("clean up task name not correctly applied, got %q", options.CleanUpTask.TaskName)
		}
		if options.CleanUpTask.TasksTimeOut != time.Hour {
			t.Errorf("clean up task time out not correctly applied, got %q", options.CleanUpTask.TasksTimeOut)
		}
	})

	t.Run("Disable", func(t *testing.T) {
		// given
		option := timekeeper.CleanUpTask(nil)
		options := &timekeeper.Options{CleanUpTask: &timekeeper.CleanUpOptions{Client: &redis.Client{}}}

		// when
		option(options)

		// then
		if options.CleanUpTask != nil {
			t.Error("clean up task not correctly applied, got non-nil")
		}
	})
}
