package crontask

import (
	"testing"
	"time"
)

// Tests that the TimeKeeperRedisPrefix option correctly applies.
func Test_TimeKeeperOption_TimeKeeperRedisPrefix(t *testing.T) {
	// given
	option := TimeKeeperRedisPrefix("bar")
	options := &TimeKeeperOptions{RedisPrefix: "foo"}

	// when
	option(options)

	// then
	if options.RedisPrefix != "bar" {
		t.Errorf("redis prefix not correctly applied, got %s", options.RedisPrefix)
	}
}

// Tests that the TimeKeeperRedisExecListName option correctly applies.
func Test_TimeKeeperOption_TimeKeeperRedisExecListName(t *testing.T) {
	// given
	option := TimeKeeperRedisExecListName("bar")
	options := &TimeKeeperOptions{RedisExecListName: "foo"}

	// when
	option(options)

	// then
	if options.RedisExecListName != "bar" {
		t.Errorf("redis exec list name not correctly applied, got %s", options.RedisExecListName)
	}
}

// Tests that the TimeKeeperRedisLastExecName option correctly applies.
func Test_TimeKeeperOption_TimeKeeperRedisLastExecName(t *testing.T) {
	// given
	option := TimeKeeperRedisLastExecName("bar")
	options := &TimeKeeperOptions{RedisLastExecName: "foo"}

	// when
	option(options)

	// then
	if options.RedisLastExecName != "bar" {
		t.Errorf("redis last exec name not correctly applied, got %s", options.RedisLastExecName)
	}
}

// Tests that the TimeKeeperKeepTaskList option correctly applies.
func Test_TimeKeeperOption_TimeKeeperKeepTaskList(t *testing.T) {
	// given
	option := TimeKeeperKeepTaskList(true)
	options := &TimeKeeperOptions{KeepTaskList: false}

	// when
	option(options)

	// then
	if options.KeepTaskList != true {
		t.Errorf("keep task list not correctly applied, got %t", options.KeepTaskList)
	}
}

// Tests that the TimeKeeperKeepLastTask option correctly applies.
func Test_TimeKeeperOption_TimeKeeperKeepLastTask(t *testing.T) {
	// given
	option := TimeKeeperKeepLastTask(true)
	options := &TimeKeeperOptions{KeepLastTask: false}

	// when
	option(options)

	// then
	if options.KeepLastTask != true {
		t.Errorf("keep last task not correctly applied, got %t", options.KeepLastTask)
	}
}

// Tests that the TimeKeeperTaskListTimeOut option correctly applies.
func Test_TimeKeeperOption_TimeKeeperTaskListTimeOut(t *testing.T) {
	// given
	option := TimeKeeperTaskListTimeOut(time.Second)
	options := &TimeKeeperOptions{TaskListTimeOut: time.Hour}

	// when
	option(options)

	// then
	if options.TaskListTimeOut != time.Second {
		t.Errorf("task list timeout not correctly applied, got %s", options.TaskListTimeOut)
	}
}
