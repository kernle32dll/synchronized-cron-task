[![Build Status](https://travis-ci.com/kernle32dll/synchronized-cron-task.svg?branch=master)](https://travis-ci.com/kernle32dll/synchronized-cron-task)
[![GoDoc](https://godoc.org/github.com/kernle32dll/synchronized-cron-task?status.svg)](http://godoc.org/github.com/kernle32dll/synchronized-cron-task)
[![Go Report Card](https://goreportcard.com/badge/github.com/kernle32dll/synchronized-cron-task)](https://goreportcard.com/report/github.com/kernle32dll/synchronized-cron-task)
[![codecov](https://codecov.io/gh/kernle32dll/synchronized-cron-task/branch/master/graph/badge.svg)](https://codecov.io/gh/kernle32dll/synchronized-cron-task)

# synchronized-cron-task

synchronized-cron-task is a lightweight wrapper around [github.com/robfig/cron](https://github.com/robfig/cron), synchronizing via a redis lock.

What this means ist that synchronized-cron-task provides a guarantee that with any number of running instances of an application, the given cron task
is only ever executed on one instance at a time.

**Note: Unlike most my projects, this project is licenced under the [Apache License 2.0](./LICENSE)!**

Download:

```
go get github.com/kernle32dll/synchronized-cron-task
```

Detailed documentation can be found on [GoDoc](https://godoc.org/github.com/kernle32dll/synchronized-cron-task).

## Compatibility

synchronized-cron-task is automatically tested against the following:
 
- Go 1.13.X, 1.14.X and 1.15.X
- Redis 4.X, 5.X and 6.X.

## Getting started

This project consists of two tightly intertwined functionalities. [SynchronizedCronTask](./synchronized_cron_task.go), and [TimeKeeper](./timekeeper/time_keeper.go).

The former is the heart of the project, providing the mentioned synchronized cron execution. The latter can optionally be used to collect data
about individual `SynchronizedCronTask` executions, such as run time or errors (if any).

Examples to use [with](./examples/example-with-timekeeper/main.go) and [without](./examples/example-without-timekeeper/main.go) time keeper can be
found in the [examples](./examples) directory.

## Synchronized cron task

A synchronized cron task can be created via two methods:

```go
crontask.NewSynchronizedCronTask(redisClient, someFunc, options...)

crontask.NewSynchronizedCronTaskWithOptions(redisClient, someFunc, &crontask.TaskOptions{})
```

The former takes an array of `crontask.TaskOption` elements. Corresponding functions can be found at the [source](./synchronized_cron_task_options.go),
or on [GoDoc](https://godoc.org/github.com/kernle32dll/synchronized-cron-task#TaskOption)

The synchronized cron task will be executed asynchronously in the background. Nothing more has to be done for it to work.

Its [ExecuteNow()](https://godoc.org/github.com/kernle32dll/synchronized-cron-task#SynchronizedCronTask.ExecuteNow) and
[NextTime()](https://godoc.org/github.com/kernle32dll/synchronized-cron-task#SynchronizedCronTask.NextTime) functions can be
used at any time for some additional control.

A synchronized cron task includes an graceful shutdown method [Stop(ctx)](https://godoc.org/github.com/kernle32dll/synchronized-cron-task#SynchronizedCronTask.Stop),
which irreversibly shuts down the task. This should be done before application shutdown, to ensure that the current
execution - if running - exits gracefully.

## Time keeper

A time keeper can be - just like a synchronized cron task - created via two methods:

```go
timekeeper.NewTimeKeeper(redisClient, options...)

timekeeper.NewTimeKeeper(redisClient, &timekeeper.Options{})
```

The former takes an array of `timekeeper.Option` elements. Corresponding functions can be found at the [source](./timekeeper/time_keeper_options.go),
or on [GoDoc](https://godoc.org/github.com/kernle32dll/synchronized-cron-task/timekeeper#Option)

For the time keeper to actually do something, it must wrap task functions via its [WrapCronTask](https://godoc.org/github.com/kernle32dll/synchronized-cron-task/timekeeper#TimeKeeper.WrapCronTask)
function, which are then used in a synchronized cron task.

For example:

```go

timeKeeper, err := timekeeper.NewTimeKeeper(redisClient)
if err != nil {
    // oh no, and error...
}

// New synchronized cron task
task, err := crontask.NewSynchronizedCronTask(
    redisClient,

    timeKeeper.WrapCronTask(
        func(ctx context.Context, task crontask.Task) error {
            // ... do something
            return nil
        },
    ),
)

```

A time keeper provides some functions, that allow introspection of recorded task executions. E.g. the last execution of a given task,
or the total number of executions. These functions provide - ony way or another - [ExecutionResult](https://godoc.org/github.com/kernle32dll/synchronized-cron-task/timekeeper#ExecutionResult)
objects, which in themselves contain useful meta information, such as time of last and next execution, or errors (if any occurred).

Just like an synchronized cron task, a time keeper includes an graceful shutdown method [Stop(ctx)](https://godoc.org/github.com/kernle32dll/synchronized-cron-task/timekeeper#TimeKeeper.Stop),
which irreversibly shuts down the clean up task of an time keeper, if it exists. This should be done before application shutdown,
to ensure that the cleanup task - if running - exits gracefully.
