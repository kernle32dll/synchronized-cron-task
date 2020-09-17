package timekeeper

import (
	"github.com/kernle32dll/synchronized-cron-task"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"

	"context"
	"encoding/json"
	"sort"
	"time"
)

// TimeKeeper is a wrapper creator, used to track data about executed
// SynchronizedCronTasks in Redis.
//
// It supports graceful shutdowns via its Stop() function.
type TimeKeeper struct {
	client *redis.Client

	redisExecListName string
	redisLastExecName string

	keepTaskList bool
	keepLastTask bool

	cleanupTask *crontask.SynchronizedCronTask
}

// Stop gracefully stops the time keeper, while also freeing some of its underlying resources.
// This has no practical implications, other than inevitably stopping the internal cleanup task,
// which periodically removes timed out task executions.
func (timeKeeper *TimeKeeper) Stop(ctx context.Context) {
	if timeKeeper.cleanupTask != nil {
		timeKeeper.cleanupTask.Stop(ctx)
		timeKeeper.cleanupTask = nil
	}
}

// NewTimeKeeperWithOptions creates a new TimeKeeper instance.
func NewTimeKeeperWithOptions(client *redis.Client, options *Options) (*TimeKeeper, error) {
	if !options.KeepTaskList && !options.KeepLastTask {
		logrus.Warn(
			"Time keeper is configured to neither keep the last task nor a task list. This means, this time keeper is a no-op!",
		)
	}

	timeKeeper := &TimeKeeper{
		client: client,

		redisExecListName: options.RedisExecListName,
		redisLastExecName: options.RedisLastExecName,

		keepTaskList: options.KeepTaskList,
		keepLastTask: options.KeepLastTask,

		// init below
		cleanupTask: nil,
	}

	if options.CleanUpTask != nil {
		if options.CleanUpTask.Client != nil {
			cleanupTask, err := crontask.NewSynchronizedCronTask(
				options.CleanUpTask.Client,

				timeKeeper.WrapCronTask(func(ctx context.Context, task crontask.Task) error {
					return timeKeeper.cleanUpOldTaskRuns(ctx, options.CleanUpTask.Client, options.CleanUpTask.TasksTimeOut)
				}),

				crontask.TaskName(options.CleanUpTask.TaskName),
				crontask.CronExpression("0 * * * * *"),
			)
			if err != nil {
				return nil, err
			}

			timeKeeper.cleanupTask = cleanupTask
		} else {
			logrus.Error("Specified clean up task for time keeper, but provided no redis client - disabling task.")
		}
	}

	return timeKeeper, nil
}

// NewTimeKeeper creates a new TimeKeeper instance.
func NewTimeKeeper(client *redis.Client, setters ...Option) (*TimeKeeper, error) {
	// Default Options
	args := &Options{
		RedisExecListName: "timekeeper.executions.list",
		RedisLastExecName: "timekeeper.executions.aggregation",

		KeepTaskList: true,
		KeepLastTask: true,
	}

	// Enable default cleanup task
	CleanUpTask(client)(args)

	for _, setter := range setters {
		setter(args)
	}

	return NewTimeKeeperWithOptions(client, args)
}

// WrapCronTask registers a TaskFunc to be recorded via this time keeper.
// Actual tracking is done via the task, which is provided as part of the
// wrapped function.
func (timeKeeper *TimeKeeper) WrapCronTask(taskFunc crontask.TaskFunc) crontask.TaskFunc {
	return func(ctx context.Context, task crontask.Task) error {
		lastExec := time.Now()
		taskErr := taskFunc(ctx, task)
		lastDuration := time.Now().Sub(lastExec)

		if timeKeeper.keepTaskList || timeKeeper.keepLastTask {
			if _, err := timeKeeper.client.TxPipelined(ctx, func(pipeliner redis.Pipeliner) error {
				execution := &ExecutionResult{
					Name:          task.Name(),
					LastExecution: lastExec,
					NextExecution: task.NextTime(),
					LastDuration:  lastDuration,
					Error:         taskErr,
				}

				if timeKeeper.keepLastTask {
					pipeliner.HSet(ctx, timeKeeper.redisLastExecName, task.Name(), execution)
				}

				if timeKeeper.keepTaskList {
					pipeliner.LPush(ctx, timeKeeper.redisExecListName, execution)
				}

				return nil
			}); err != nil {
				// If there was an task error, give that precedence over the redis error
				if taskErr != nil {
					return taskErr
				}

				return err
			}
		}

		return taskErr
	}
}

func (timeKeeper *TimeKeeper) cleanUpOldTaskRuns(ctx context.Context, client *redis.Client, taskListTimeOut time.Duration) error {
	timeOutPoint := time.Now().Add(-taskListTimeOut)

	for {
		lastElemList, err := client.LRange(ctx, timeKeeper.redisExecListName, -1, -1).Result()
		if err != nil {
			return nil
		}

		// List is empty, nothing to do
		if len(lastElemList) == 0 {
			return nil
		}

		lastElem := lastElemList[0]

		execResult := &ExecutionResult{}
		if err := json.Unmarshal([]byte(lastElem), execResult); err != nil {
			return err
		}

		if execResult.LastExecution.Before(timeOutPoint) {
			// Note, its always safe to rpop, as we only lpush, and thus never
			// interfere with the end of the list
			if err := client.RPop(ctx, timeKeeper.redisExecListName).Err(); err != nil {
				return err
			}

			continue
		}

		// Last task is not outside the timeout window yet, so others can't either.
		// We are done here.
		return nil
	}
}

// GetAllRuns returns all tasks executions recorded by the time keeper, and not yet cleaned up.
// Possible errors are related to redis connection problems.
func (timeKeeper *TimeKeeper) GetAllRuns(ctx context.Context, offset, limit int64) ([]ExecutionResult, error) {
	results, err := timeKeeper.client.LRange(ctx, timeKeeper.redisExecListName, offset, offset+limit-1).Result()
	if err != nil {
		return nil, err
	}

	return unmarshalExecutionResults(results...)
}

// CountAllRuns returns the total amount of task executions recorded by the time keeper, and
// not yet cleaned up. Possible errors are related to redis connection problems.
func (timeKeeper *TimeKeeper) CountAllRuns(ctx context.Context) (int64, error) {
	count, err := timeKeeper.client.LLen(ctx, timeKeeper.redisExecListName).Result()
	if err != nil {
		return 0, err
	}

	return count, nil
}

// GetLastRunOfAllTasks returns the latest execution result of all tasks recorded by the time keeper.
// This implies that tasks which have not yet run are not returned. Possible errors are related to
//redis connection problems.
func (timeKeeper *TimeKeeper) GetLastRunOfAllTasks(ctx context.Context) ([]ExecutionResult, error) {
	resultsMap, err := timeKeeper.client.HGetAll(ctx, timeKeeper.redisLastExecName).Result()
	if err != nil {
		return nil, err
	}

	results := make([]string, len(resultsMap))
	i := 0
	for _, result := range resultsMap {
		results[i] = result
		i++
	}

	resultList, err := unmarshalExecutionResults(results...)
	if err != nil {
		return nil, err
	}

	sliceList := ExecutionResultSlice(resultList)
	sort.Sort(sliceList)

	return sliceList, nil
}

// CountTasks returns the amount of individual tasks recorded by the time keeper.
// This implies that tasks which have not yet run are not counted in. Possible errors
// are related to redis connection problems.
func (timeKeeper *TimeKeeper) CountTasks(ctx context.Context) (int64, error) {
	count, err := timeKeeper.client.HLen(ctx, timeKeeper.redisLastExecName).Result()
	if err != nil {
		return 0, err
	}

	return count, nil
}

// GetLastRunOfTask returns the latest execution of a given task name.
// If the task has not been run yet, an error is returned. Possible errors are
// related to redis connection problems.
func (timeKeeper *TimeKeeper) GetLastRunOfTask(ctx context.Context, name string) (ExecutionResult, error) {
	result, err := timeKeeper.client.HGet(ctx, timeKeeper.redisLastExecName, name).Result()
	if err != nil {
		return ExecutionResult{}, err
	}

	resultsList, err := unmarshalExecutionResults(result)
	if err != nil {
		return ExecutionResult{}, err
	}

	return resultsList[0], nil
}

func unmarshalExecutionResults(results ...string) ([]ExecutionResult, error) {
	resultList := make([]ExecutionResult, len(results))
	for i, result := range results {
		execResult := &ExecutionResult{}
		if err := execResult.UnmarshalBinary([]byte(result)); err != nil {
			return nil, err
		}

		resultList[i] = *execResult
	}
	return resultList, nil
}
