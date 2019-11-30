[![Build Status](https://travis-ci.com/kernle32dll/synchronized-cron-task.svg?branch=master)](https://travis-ci.com/kernle32dll/synchronized-cron-task)
[![GoDoc](https://godoc.org/github.com/kernle32dll/synchronized-cron-task?status.svg)](http://godoc.org/github.com/kernle32dll/synchronized-cron-task)
[![Go Report Card](https://goreportcard.com/badge/github.com/kernle32dll/synchronized-cron-task)](https://goreportcard.com/report/github.com/kernle32dll/synchronized-cron-task)
[![codecov](https://codecov.io/gh/kernle32dll/synchronized-cron-task/branch/master/graph/badge.svg)](https://codecov.io/gh/kernle32dll/synchronized-cron-task)

# synchronized-cron-task

synchronized-cron-task is a lightweight wrapper around [github.com/robfig/cron](https://github.com/robfig/cron), synchronizing via a redis lock.

What this means ist that synchronized-cron-task provides a guarantee that with any number of running instances of an application, the given cron task
is only ever executed on one instance at a time.

**Note: This project builds on top of the pending version 7.X of [github.com/go-redis/redis](https://github.com/go-redis/redis). This was
chosen for the sole reason to make synchronized-cron-task work effortlessly in tandem with [github.com/kernle32dll/planlagt](https://github.com/kernle32dll/planlagt)**

**Note: Unlike most my projects, this project is licenced under the [Apache License 2.0](./LICENSE)!**

**-WARNING-: This is not production tested, and api-breaking may occur while no 1.0 release has been tagged**

Download:

```
go get github.com/kernle32dll/synchronized-cron-task
```

Detailed documentation can be found on [GoDoc](https://godoc.org/github.com/kernle32dll/synchronized-cron-task).

## Getting started

This project consists of two tightly intertwined functionalities. [SynchronizedCronTask](./synchronized_cron_task.go), and [TimeKeeper](./time_keeper.go).

The former is the hard of the project, providing the mentioned synchronized cron execution. The latter can optionally used to collect data
about `SynchronizedCronTask` executions, such as run time or errors (if any).