package timekeeper_test

import (
	crontask "github.com/kernle32dll/synchronized-cron-task"
	"github.com/kernle32dll/synchronized-cron-task/timekeeper"

	"github.com/go-redis/redis/v7"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"context"
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
	}
	for i := range redisVersions {
		version := redisVersions[i]

		t.Run(fmt.Sprintf("redis:%s", version), func(t *testing.T) {
			t.Parallel()

			client, closer := getRedisClient(t, version)
			defer closer(context.Background())

			timeKeeper := timekeeper.NewTimeKeeper(client)

			testFunc := timeKeeper.WrapCronTask(func(ctx context.Context, task crontask.Task) error {
				return nil
			})

			nextTime := time.Now().Add(time.Hour * 24)

			task1 := &TaskMock{NameVal: "example1", NextTimeVal: nextTime}
			if err := testFunc(context.Background(), task1); err != nil {
				t.Fatal(err)
			}

			task2 := &TaskMock{NameVal: "example2", NextTimeVal: nextTime}
			if err := testFunc(context.Background(), task2); err != nil {
				t.Fatal(err)
			}

			t.Run("Retrieval", func(t *testing.T) {
				t.Run("CountAllRuns", func(t *testing.T) {
					count, err := timeKeeper.CountAllRuns(context.Background())
					if err != nil {
						t.Fatalf("unexpected error %q", err)
					}

					if expected := int64(2); count != expected {
						t.Fatalf("expected count %d, but got %d", expected, count)
					}
				})

				t.Run("CountTasks", func(t *testing.T) {
					count, err := timeKeeper.CountTasks(context.Background())
					if err != nil {
						t.Fatalf("unexpected error %q", err)
					}

					if expected := int64(2); count != expected {
						t.Fatalf("expected count %d, but got %d", expected, count)
					}
				})

				t.Run("GetAllRuns", func(t *testing.T) {
					runs, err := timeKeeper.GetAllRuns(context.Background(), 0, 1)
					if err != nil {
						t.Fatalf("unexpected error %q", err)
					}

					count := len(runs)
					if expected := 1; count != expected {
						t.Fatalf("expected count of runs %d, but got %d", expected, count)
					}

					run := runs[0]
					if expected := task2.NameVal; run.Name != expected {
						t.Errorf("expected run name %q, but got %q", expected, run.Name)
					}
				})

				t.Run("GetAllRuns_offset", func(t *testing.T) {
					runs, err := timeKeeper.GetAllRuns(context.Background(), 1, 1)
					if err != nil {
						t.Fatalf("unexpected error %q", err)
					}

					count := len(runs)
					if expected := 1; count != expected {
						t.Fatalf("expected count of runs %d, but got %d", expected, count)
					}

					run := runs[0]
					if expected := task1.NameVal; run.Name != expected {
						t.Errorf("expected run name %q, but got %q", expected, run.Name)
					}
				})
			})
		})
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
