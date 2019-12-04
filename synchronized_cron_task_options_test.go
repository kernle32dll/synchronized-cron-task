package crontask

import (
	"github.com/sirupsen/logrus"
	"testing"
	"time"
)

// Tests that the TaskName option correctly applies.
func Test_TaskOption_TaskName(t *testing.T) {
	// given
	option := TaskName("bar")
	options := &TaskOptions{Name: "foo"}

	// when
	option(options)

	// then
	if options.Name != "bar" {
		t.Errorf("name not correctly applied, got %s", options.Name)
	}
}

// Tests that the CronExpression option correctly applies.
func Test_TaskOption_CronExpression(t *testing.T) {
	// given
	option := CronExpression("bar")
	options := &TaskOptions{CronExpression: "foo"}

	// when
	option(options)

	// then
	if options.CronExpression != "bar" {
		t.Errorf("cron expression not correctly applied, got %s", options.CronExpression)
	}
}

// Tests that the Logger option correctly applies.
func Test_TaskOption_Logger(t *testing.T) {
	// given
	option := Logger(&logrus.Logger{})
	options := &TaskOptions{Logger: nil}

	// when
	option(options)

	// then
	if options.Logger == nil {
		t.Error("logger not correctly applied, got nil")
	}
}

// Tests that the LeadershipTimeout option correctly applies.
func Test_TaskOption_LeadershipTimeout(t *testing.T) {
	// given
	option := LeadershipTimeout(time.Second)
	options := &TaskOptions{LeadershipTimeout: time.Hour}

	// when
	option(options)

	// then
	if options.LeadershipTimeout != time.Second {
		t.Errorf("leadership timeout not correctly applied, got %s", options.LeadershipTimeout)
	}
}

// Tests that the LockTimeout option correctly applies.
func Test_TaskOption_LockTimeout(t *testing.T) {
	// given
	option := LockTimeout(time.Second)
	options := &TaskOptions{LockTimeout: time.Hour}

	// when
	option(options)

	// then
	if options.LockTimeout != time.Second {
		t.Errorf("lock timeout not correctly applied, got %s", options.LockTimeout)
	}
}

// Tests that the LockHeartbeat option correctly applies.
func Test_TaskOption_LockHeartbeat(t *testing.T) {
	// given
	option := LockHeartbeat(time.Second)
	options := &TaskOptions{LockHeartbeat: time.Hour}

	// when
	option(options)

	// then
	if options.LockHeartbeat != time.Second {
		t.Errorf("lock heartbeat not correctly applied, got %s", options.LockHeartbeat)
	}
}
