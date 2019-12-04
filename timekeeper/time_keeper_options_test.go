package timekeeper

import (
	"testing"
	"time"
)

// Tests that the RedisPrefix option correctly applies.
func Test_TimeKeeperOption_RedisPrefix(t *testing.T) {
	// given
	option := RedisPrefix("bar")
	options := &Options{RedisPrefix: "foo"}

	// when
	option(options)

	// then
	if options.RedisPrefix != "bar" {
		t.Errorf("redis prefix not correctly applied, got %s", options.RedisPrefix)
	}
}

// Tests that the RedisExecListName option correctly applies.
func Test_TimeKeeperOption_RedisExecListName(t *testing.T) {
	// given
	option := RedisExecListName("bar")
	options := &Options{RedisExecListName: "foo"}

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
	option := RedisLastExecName("bar")
	options := &Options{RedisLastExecName: "foo"}

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
	option := KeepTaskList(true)
	options := &Options{KeepTaskList: false}

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
	option := KeepLastTask(true)
	options := &Options{KeepLastTask: false}

	// when
	option(options)

	// then
	if options.KeepLastTask != true {
		t.Errorf("keep last task not correctly applied, got %t", options.KeepLastTask)
	}
}

// Tests that the TaskListTimeOut option correctly applies.
func Test_TimeKeeperOption_ListTimeOut(t *testing.T) {
	// given
	option := TaskListTimeOut(time.Second)
	options := &Options{TaskListTimeOut: time.Hour}

	// when
	option(options)

	// then
	if options.TaskListTimeOut != time.Second {
		t.Errorf("task list timeout not correctly applied, got %s", options.TaskListTimeOut)
	}
}
