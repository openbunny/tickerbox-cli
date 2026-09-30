// SPDX-License-Identifier: MIT

package concurrent

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestRunExecutesAllTasks(t *testing.T) {
	var count atomic.Int32
	tasks := make([]func(), 10)
	for i := range tasks {
		tasks[i] = func() { count.Add(1) }
	}
	Run(3, tasks...)
	if got := count.Load(); got != 10 {
		t.Errorf("count = %d; want 10", got)
	}
}

func TestRunNoTasks(t *testing.T) {
	Run(3)
}

func TestRunRespectsLimit(t *testing.T) {
	const limit = 3
	var inFlight, maxInFlight atomic.Int32
	tasks := make([]func(), 20)
	for i := range tasks {
		tasks[i] = func() {
			n := inFlight.Add(1)
			for {
				max := maxInFlight.Load()
				if n <= max || maxInFlight.CompareAndSwap(max, n) {
					break
				}
			}
			time.Sleep(2 * time.Millisecond)
			inFlight.Add(-1)
		}
	}
	Run(limit, tasks...)
	if got := maxInFlight.Load(); got > limit {
		t.Errorf("max concurrent tasks = %d; want <= %d", got, limit)
	}
}

func TestRunLimitAboveTaskCount(t *testing.T) {
	var count atomic.Int32
	Run(100, func() { count.Add(1) }, func() { count.Add(1) })
	if got := count.Load(); got != 2 {
		t.Errorf("count = %d; want 2", got)
	}
}

func TestRunZeroLimitRunsAllTasks(t *testing.T) {
	var count atomic.Int32
	Run(0, func() { count.Add(1) }, func() { count.Add(1) }, func() { count.Add(1) })
	if got := count.Load(); got != 3 {
		t.Errorf("count = %d; want 3", got)
	}
}
