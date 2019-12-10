package crontask_test

import (
	crontask "github.com/kernle32dll/synchronized-cron-task"

	"github.com/go-redis/redis/v7"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

type ExecutionTracker struct {
	count  int
	retErr error
}

func (executionTracker *ExecutionTracker) getFunc() crontask.TaskFunc {
	return func(ctx context.Context, task crontask.Task) error {
		executionTracker.count++
		return executionTracker.retErr
	}
}

func Test_SynchronizedCronTask(t *testing.T) {
	redisVersions := []string{
		"4-alpine",
		"5-alpine",
	}
	for i := range redisVersions {
		version := redisVersions[i]

		t.Run(fmt.Sprintf("redis:%s", version), func(t *testing.T) {
			t.Parallel()

			t.Run("basic-execution-test", basicExecutionTest(version))

			t.Run("concurrent-execution-test", concurrentExecutionTest(version))

			t.Run("stopped-execution-test", stoppedTests(version))

			t.Run("error-in-execution-test", errorTest(version))
		})
	}
}

func basicExecutionTest(redisVersion string) func(t *testing.T) {
	return func(t *testing.T) {
		// given
		client, closer := getRedisClient(t, redisVersion)
		defer closeClient(t, client, closer)

		logger, hook := test.NewNullLogger()
		logger.Level = logrus.TraceLevel

		executionTracker := &ExecutionTracker{}
		task, err := crontask.NewSynchronizedCronTask(
			client,
			executionTracker.getFunc(),
			crontask.CronExpression("0 0 1 1 *"),
			crontask.Logger(logger),
		)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		// when
		task.ExecuteNow()

		// then
		if executionTracker.count != 1 {
			t.Fail()
		}

		if expected := "Default Synchronized Task"; expected != task.Name() {
			t.Fatalf("expected name %q as default cron task , but got %q", expected, task.Name())
		}

		logContains(
			t, hook,

			"Trying to temporarily gain leadership for synchronized task",
			"Resigning temporary leadership for synchronized task",
			"Successfully filled executed task",
		)
	}
}

func concurrentExecutionTest(redisVersion string) func(t *testing.T) {
	return func(t *testing.T) {
		// given
		client, closer := getRedisClient(t, redisVersion)
		defer closeClient(t, client, closer)

		logger, hook := test.NewNullLogger()
		logger.Level = logrus.TraceLevel

		executionTracker := &ExecutionTracker{}
		task, err := crontask.NewSynchronizedCronTask(
			client,
			func(ctx context.Context, task crontask.Task) error {
				time.Sleep(time.Second)
				return executionTracker.getFunc()(ctx, task)
			},
			crontask.CronExpression("0 0 1 1 *"),
			crontask.Logger(logger),
		)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		// when
		go task.ExecuteNow()
		task.ExecuteNow()

		// then
		if executionTracker.count != 1 {
			t.Fail()
		}

		logContains(
			t, hook,

			"leadership is already owned",
			"Successfully filled executed task",
		)
	}
}

func stoppedTests(redisVersion string) func(t *testing.T) {
	return func(t *testing.T) {
		// given
		client, closer := getRedisClient(t, redisVersion)
		defer closeClient(t, client, closer)

		logger, hook := test.NewNullLogger()
		logger.Level = logrus.TraceLevel

		executionTracker := &ExecutionTracker{}
		task, err := crontask.NewSynchronizedCronTask(
			client,
			executionTracker.getFunc(),
			crontask.CronExpression("0 0 1 1 *"),
			crontask.Logger(logger),
		)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		// Immediately stop
		task.Stop()

		t.Run("ExecuteNow", func(t *testing.T) {
			// when
			task.ExecuteNow()

			// then
			if executionTracker.count != 0 {
				t.Fail()
			}

			logContains(
				t, hook,

				"Tried to force execution of synchronized cron task",
			)
		})

		t.Run("NextTime", func(t *testing.T) {
			// when
			task.NextTime()

			// then
			logContains(
				t, hook,

				"Tried to retrieve next execution of synchronized cron task",
			)
		})
	}
}

func errorTest(redisVersion string) func(t *testing.T) {
	return func(t *testing.T) {
		// given
		client, closer := getRedisClient(t, redisVersion)
		defer closeClient(t, client, closer)

		logger, hook := test.NewNullLogger()
		logger.Level = logrus.TraceLevel

		executionTracker := &ExecutionTracker{}
		executionTracker.retErr = errors.New("some error")

		task, err := crontask.NewSynchronizedCronTask(
			client,
			executionTracker.getFunc(),
			crontask.CronExpression("0 0 1 1 *"),
			crontask.Logger(logger),
		)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		// when
		task.ExecuteNow()

		// then
		if executionTracker.count != 1 {
			t.Fail()
		}

		logContains(
			t, hook,

			"error while executing synchronized task function \"Default Synchronized Task\": some error",
		)
	}
}

func closeClient(t *testing.T, client *redis.Client, closer func(context.Context) error) {
	if err := client.Close(); err != nil {
		t.Logf("unexpected error shutting down redis client: %s", err)
	}

	if err := closer(context.Background()); err != nil {
		t.Logf("unexpected error shutting down redis container: %s", err)
	}
}

func logContains(t *testing.T, hook *test.Hook, phrases ...string) {
	for _, phrase := range phrases {
		found := false
		for _, entry := range hook.AllEntries() {
			if strings.Contains(entry.Message, phrase) {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("Log did not contain phrase %q", phrase)
		}
	}
}

func getRedisClient(t *testing.T, version string) (*redis.Client, func(context.Context) error) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        fmt.Sprintf("redis:%s", version),
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
	}

	t.Log("Starting up container")
	redisContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatal(err)
	}

	ip, err := redisContainer.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}

	port, err := redisContainer.MappedPort(ctx, "6379")
	if err != nil {
		t.Fatal(err)
	}

	address := fmt.Sprintf("%s:%s", ip, port.Port())

	t.Logf("redis client started at %q", address)

	client := redis.NewClient(&redis.Options{
		Network: "tcp",
		Addr:    address,
	})

	return client, redisContainer.Terminate
}
