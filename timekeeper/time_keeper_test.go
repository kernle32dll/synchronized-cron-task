package timekeeper

import (
	crontask "github.com/kernle32dll/synchronized-cron-task"

	"github.com/go-redis/redis/v8"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

type TaskMock struct {
	NameVal     string
	NextTimeVal time.Time
}

func (m *TaskMock) Name() string {
	return m.NameVal
}

func (m *TaskMock) NextTime() time.Time {
	return m.NextTimeVal
}

//---------

func Test_TimeKeeper(t *testing.T) {
	redisVersions := []string{
		"4-alpine",
		"5-alpine",
		"6-alpine",
	}
	for i := range redisVersions {
		version := redisVersions[i]

		t.Run(fmt.Sprintf("redis:%s", version), func(t *testing.T) {
			t.Parallel()

			t.Run("Retrieval", testRetrievalMethods(version))

			t.Run("CleanUp", testCleanup(version))
		})
	}
}

func testRetrievalMethods(version string) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()

		client, closer := getRedisClient(t, version)
		defer closeClient(t, client, closer)

		timeKeeper, err := NewTimeKeeper(client, CleanUpTask(nil))
		if err != nil {
			t.Fatalf("unexpected error %q", err)
		}

		defer timeKeeper.Stop(context.Background())

		testFunc := timeKeeper.WrapCronTask(func(ctx context.Context, task crontask.Task) error {
			return nil
		})

		nextTime := time.Now().Add(time.Hour * 24)

		// example1 is executed once
		task1 := &TaskMock{NameVal: "example1", NextTimeVal: nextTime}
		if err := testFunc(context.Background(), task1); err != nil {
			t.Fatal(err)
		}

		// example2 is executed twice
		task2 := &TaskMock{NameVal: "example2", NextTimeVal: nextTime}
		if err := testFunc(context.Background(), task2); err != nil {
			t.Fatal(err)
		}
		minTimeOfSecondExec := time.Now()
		if err := testFunc(context.Background(), task2); err != nil {
			t.Fatal(err)
		}

		t.Run("CountAllRuns", testCountAllRuns(timeKeeper))

		t.Run("CountTasks", testCountTasks(timeKeeper))

		t.Run("GetAllRuns", testGetAllRuns(timeKeeper, task2))

		t.Run("GetAllRuns_offset", testGetAllRunsWithOffset(timeKeeper, task1))

		t.Run("GetLastRunOfAllTasks", testGetLastRunOfAllTasks(timeKeeper, task1, task2))

		t.Run("GetLastRunOfTask", testGetLastRunOfTask(timeKeeper, task2, minTimeOfSecondExec))
	}
}

func testCountAllRuns(timeKeeper *TimeKeeper) func(t *testing.T) {
	return func(t *testing.T) {
		count, err := timeKeeper.CountAllRuns(context.Background())
		if err != nil {
			t.Fatalf("unexpected error %q", err)
		}

		if expected := int64(3); count != expected {
			t.Fatalf("expected count %d, but got %d", expected, count)
		}
	}
}

func testCountTasks(timeKeeper *TimeKeeper) func(t *testing.T) {
	return func(t *testing.T) {
		count, err := timeKeeper.CountTasks(context.Background())
		if err != nil {
			t.Fatalf("unexpected error %q", err)
		}

		if expected := int64(2); count != expected {
			t.Fatalf("expected count %d, but got %d", expected, count)
		}
	}
}

func testGetAllRuns(timeKeeper *TimeKeeper, task *TaskMock) func(t *testing.T) {
	return func(t *testing.T) {
		runs, err := timeKeeper.GetAllRuns(context.Background(), 0, 1)
		if err != nil {
			t.Fatalf("unexpected error %q", err)
		}

		count := len(runs)
		if expected := 1; count != expected {
			t.Fatalf("expected count of runs %d, but got %d", expected, count)
		}

		run := runs[0]
		if expected := task.NameVal; run.Name != expected {
			t.Errorf("expected run name %q, but got %q", expected, run.Name)
		}
	}
}

func testGetAllRunsWithOffset(timeKeeper *TimeKeeper, task *TaskMock) func(t *testing.T) {
	return func(t *testing.T) {
		runs, err := timeKeeper.GetAllRuns(context.Background(), 2, 1)
		if err != nil {
			t.Fatalf("unexpected error %q", err)
		}

		count := len(runs)
		if expected := 1; count != expected {
			t.Fatalf("expected count of runs %d, but got %d", expected, count)
		}

		run := runs[0]
		if expected := task.NameVal; run.Name != expected {
			t.Errorf("expected run name %q, but got %q", expected, run.Name)
		}
	}
}

func testGetLastRunOfAllTasks(timeKeeper *TimeKeeper, task1 *TaskMock, task2 *TaskMock) func(t *testing.T) {
	return func(t *testing.T) {
		runs, err := timeKeeper.GetLastRunOfAllTasks(context.Background())
		if err != nil {
			t.Fatalf("unexpected error %q", err)
		}

		count := len(runs)
		if expected := 2; count != expected {
			t.Fatalf("expected count of runs %d, but got %d", expected, count)
		}

		run1 := runs[0]
		if expected := task1.NameVal; run1.Name != expected {
			t.Errorf("expected task 1 name %q, but got %q", expected, run1.Name)
		}

		run2 := runs[1]
		if expected := task2.NameVal; run2.Name != expected {
			t.Errorf("expected task 2 name %q, but got %q", expected, run2.Name)
		}
	}
}

func testGetLastRunOfTask(timeKeeper *TimeKeeper, task *TaskMock, minTimeOfSecondExec time.Time) func(t *testing.T) {
	return func(t *testing.T) {
		result, err := timeKeeper.GetLastRunOfTask(context.Background(), task.Name())
		if err != nil {
			t.Fatalf("unexpected error %q", err)
		}

		if expected := task.NameVal; result.Name != expected {
			t.Errorf("expected task 1 name %q, but got %q", expected, result.Name)
		}

		if expected := minTimeOfSecondExec; result.LastExecution.Before(expected) {
			t.Errorf("expected execution time does not match - expected > %q, but got %q", expected, result.LastExecution)
		}
	}
}

func testCleanup(version string) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()

		client, closer := getRedisClient(t, version)
		defer closeClient(t, client, closer)

		timeKeeper, err := NewTimeKeeper(client, CleanUpTask(client, CleanUpTasksTimeOut(0)))
		if err != nil {
			t.Fatalf("unexpected error %q", err)
		}

		defer timeKeeper.Stop(context.Background())

		testFunc := timeKeeper.WrapCronTask(func(ctx context.Context, task crontask.Task) error {
			return nil
		})

		// add a task
		task := &TaskMock{NameVal: "example1", NextTimeVal: time.Now().Add(time.Hour * 24)}
		if err := testFunc(context.Background(), task); err != nil {
			t.Fatal(err)
		}

		count, err := timeKeeper.CountTasks(context.Background())
		if err != nil {
			t.Fatalf("unexpected error %q", err)
		}

		if expected := int64(1); count != expected {
			t.Fatalf("expected count %d as test prerequisite, but got %d", expected, count)
		}

		timeKeeper.cleanupTask.ExecuteNow()
		time.Sleep(time.Second)

		count, err = timeKeeper.CountAllRuns(context.Background())
		if err != nil {
			t.Fatalf("unexpected error %q", err)
		}

		if expected := int64(1); count != expected {
			t.Fatalf("expected count %d, but got %d", expected, count)
		}
	}
}

func Test_RedisErrors(t *testing.T) {
	// create client, but immediately close
	client := redis.NewClient(&redis.Options{
		Network: "tcp",
		Addr:    "does-not-exist:6379",
	})
	closeClient(t, client, nil)

	timeKeeper, err := NewTimeKeeper(client, CleanUpTask(nil))
	if err != nil {
		t.Fatalf("unexpected error %q", err)
	}

	defer timeKeeper.Stop(context.Background())

	t.Run("CountAllRuns", testForError(func(ctx context.Context) error {
		_, err = timeKeeper.CountAllRuns(ctx)
		return err
	}))

	t.Run("CountTasks", testForError(func(ctx context.Context) error {
		_, err = timeKeeper.CountTasks(ctx)
		return err
	}))

	t.Run("GetAllRuns", testForError(func(ctx context.Context) error {
		_, err = timeKeeper.GetAllRuns(ctx, 0, 10)
		return err
	}))

	t.Run("GetLastRunOfAllTasks", testForError(func(ctx context.Context) error {
		_, err = timeKeeper.GetLastRunOfAllTasks(ctx)
		return err
	}))

	t.Run("GetLastRunOfTask", testForError(func(ctx context.Context) error {
		_, err = timeKeeper.GetLastRunOfTask(ctx, "some-task")
		return err
	}))
}

func testForError(testFunc func(ctx context.Context) error) func(t *testing.T) {
	expectedErr := errors.New("redis: client is closed")

	return func(t *testing.T) {
		err := testFunc(context.Background())
		if err == nil || err.Error() != expectedErr.Error() {
			t.Errorf("expected %q, got %q", expectedErr, err)
		}
	}
}

func closeClient(t *testing.T, client *redis.Client, closer func(context.Context) error) {
	if err := client.Close(); err != nil {
		t.Logf("unexpected error shutting down redis client: %s", err)
	}

	if closer != nil {
		if err := closer(context.Background()); err != nil {
			t.Logf("unexpected error shutting down redis container: %s", err)
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
