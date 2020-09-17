package main

import (
	crontask "github.com/kernle32dll/synchronized-cron-task"
	"github.com/kernle32dll/synchronized-cron-task/timekeeper"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"

	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	logger := logrus.New()
	logger.Level = logrus.TraceLevel

	client := redis.NewClient(&redis.Options{
		Network: "tcp",
		Addr:    "127.0.0.1:6379",
	})

	// -----

	// Just go with the default time keeper
	timeKeeper, err := timekeeper.NewTimeKeeper(client)
	if err != nil {
		panic(fmt.Errorf("unexpected error: %s", err))
	}

	// New synchronized cron task
	task, err := crontask.NewSynchronizedCronTask(
		client,

		timeKeeper.WrapCronTask(
			func(ctx context.Context, task crontask.Task) error {
				logger.Println("hello cron")
				return nil
			},
		),

		// every 10 seconds
		crontask.CronExpression("0/10 * * * * *"),

		crontask.Logger(logger),
	)
	if err != nil {
		panic(fmt.Errorf("unexpected error: %s", err))
	}

	// -----

	// Block till shutdown signal is received
	c := make(chan os.Signal, 1)
	signal.Notify(c,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	// Asynchronously print out execution stats every 15 seconds
	go func() {
		for {
			time.Sleep(15 * time.Second)

			lastRun, err := timeKeeper.GetLastRunOfTask(context.Background(), task.Name())
			if err != nil {
				logger.Errorf("unexpected error: %s", err)
				continue
			}

			if err != nil {
				logrus.Debugf("%s: lastExec=%s nextExec=%s error=%s", lastRun.Name, lastRun.LastExecution, lastRun.NextExecution, lastRun.Error)
			} else {
				logrus.Debugf("%s: lastExec=%s nextExec=%s error=nil", lastRun.Name, lastRun.LastExecution, lastRun.NextExecution)
			}
		}
	}()

	<-c

	logrus.Info("Shutting down")

	shutdownCtx, cancel := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
	defer cancel() // prevent context leak

	task.Stop(shutdownCtx)

	logrus.Info("Shutdown complete")
}
