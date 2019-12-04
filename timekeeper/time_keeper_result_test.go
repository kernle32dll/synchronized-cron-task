package timekeeper_test

import (
	"github.com/kernle32dll/synchronized-cron-task/timekeeper"

	"errors"
	"reflect"
	"testing"
	"time"
)

const exampleJson = "{" +
	"\"Name\":\"some-task\"," +
	"\"LastExecution\":\"1991-05-23T01:02:03.000000004Z\"," +
	"\"NextExecution\":" +
	"\"1991-05-23T01:02:03.000000004Z\"," +
	"\"LastDuration\":3600000000000," +
	"\"Error\":\"some-error\"" +
	"}"

// Tests that the RedisPrefix option correctly applies.
func Test_ExecutionResult_MarshalBinary(t *testing.T) {
	// given
	option := &timekeeper.ExecutionResult{
		Name:          "some-task",
		LastExecution: time.Date(1991, 5, 23, 1, 2, 3, 4, time.UTC),
		NextExecution: time.Date(1991, 5, 23, 1, 2, 3, 4, time.UTC),
		LastDuration:  time.Hour,
		Error:         errors.New("some-error"),
	}

	// when
	result, err := option.MarshalBinary()

	// then
	if err != nil {
		t.Errorf("unexpected error, got %s", err)
	}

	expected := exampleJson
	if res := string(result); res != expected {
		t.Errorf("unexpected marshalling result, got %q, wanted %q", res, expected)
	}
}

// Tests that the RedisPrefix option correctly applies.
func Test_ExecutionResult_UnmarshalBinary(t *testing.T) {
	// given
	option := &timekeeper.ExecutionResult{}

	// when
	err := option.UnmarshalBinary([]byte(exampleJson))

	// then
	if err != nil {
		t.Errorf("unexpected error, got %s", err)
	}

	expected := &timekeeper.ExecutionResult{
		Name:          "some-task",
		LastExecution: time.Date(1991, 5, 23, 1, 2, 3, 4, time.UTC),
		NextExecution: time.Date(1991, 5, 23, 1, 2, 3, 4, time.UTC),
		LastDuration:  time.Hour,
		Error:         errors.New("some-error"),
	}
	if !reflect.DeepEqual(option, expected) {
		t.Errorf("unexpected marshalling result, got %q, wanted %q", option, expected)
	}
}

// Tests that the RedisPrefix option correctly applies.
func Test_ExecutionResult_UnmarshalBinary_error(t *testing.T) {
	// given
	option := &timekeeper.ExecutionResult{}

	// when
	err := option.UnmarshalBinary([]byte("invalid"))

	// then
	if err == nil {
		t.Error("expected error, got nil")
	}

	expected := &timekeeper.ExecutionResult{}
	if !reflect.DeepEqual(option, expected) {
		t.Errorf("unexpected marshalling result, got %q, wanted %q", option, expected)
	}
}

// Tests that slice operations on ExecutionResult behave as expected.
func Test_ExecutionResult_Slice(t *testing.T) {
	t.Run("Len", func(t *testing.T) {
		// given
		slice := timekeeper.ExecutionResultSlice{
			timekeeper.ExecutionResult{Name: "example3"},
			timekeeper.ExecutionResult{Name: "example1"},
			timekeeper.ExecutionResult{Name: "example2"},
		}

		// when
		result := slice.Len()

		// then
		if expected := len(slice); result != expected {
			t.Fatalf("unexpected execution result slice len, expected %d but got %d", expected, result)
		}
	})

	t.Run("Swap", func(t *testing.T) {
		// given
		slice := timekeeper.ExecutionResultSlice{
			timekeeper.ExecutionResult{Name: "example3"},
			timekeeper.ExecutionResult{Name: "example1"},
			timekeeper.ExecutionResult{Name: "example2"},
		}

		// when
		slice.Swap(0, 2)

		// then
		expected := timekeeper.ExecutionResultSlice{
			timekeeper.ExecutionResult{Name: "example2"},
			timekeeper.ExecutionResult{Name: "example1"},
			timekeeper.ExecutionResult{Name: "example3"},
		}
		if !reflect.DeepEqual(slice, expected) {
			t.Fatalf("unexpected execution result slice state after swap, expected %s but got %s", expected, slice)
		}
	})

	t.Run("Less", func(t *testing.T) {
		// given
		slice := timekeeper.ExecutionResultSlice{
			timekeeper.ExecutionResult{Name: "example1"},
			timekeeper.ExecutionResult{Name: "example2"},
		}

		// when
		result1 := slice.Less(0, 1)
		result2 := slice.Less(1, 0)

		// then
		if !result1 {
			t.Fatalf("unexpected execution result slice less, expected %t but got %t", true, result1)
		}
		if result2 {
			t.Fatalf("unexpected execution result slice len, expected %t but got %t", false, result2)
		}
	})
}
