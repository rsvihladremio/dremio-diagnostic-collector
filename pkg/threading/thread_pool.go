//	Copyright 2023 Dremio Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// threading package provides support for simple concurrency and threading
package threading

import (
	"fmt"
	"sync"

	"github.com/rsvihladremio/dremio-diagnostic-collector/cmd/simplelog"
)

type ThreadPool struct {
	wg            sync.WaitGroup
	numberThreads int
	jobs          chan func() error
	pendingJobs   int
	started       bool
	mut           sync.Mutex
}

func NewThreadPool(numberThreads int) *ThreadPool {
	//by default support 4 million jobs
	jobs := make(chan func() error, 4000000)

	return &ThreadPool{
		numberThreads: numberThreads,
		jobs:          jobs,
	}
}

func NewThreadPoolWithJobQueue(numberThreads, jobQueueSize int) *ThreadPool {
	jobs := make(chan func() error, jobQueueSize)

	return &ThreadPool{
		numberThreads: numberThreads,
		jobs:          jobs,
	}
}

// AddJob adds a job to the thread pool. It increases the wait group counter and sends the job to the jobs channel.
func (t *ThreadPool) AddJob(job func() error) {
	t.mut.Lock()
	t.pendingJobs++
	t.mut.Unlock()
	t.wg.Add(1)
	t.jobs <- job
}

// Start initializes the ThreadPool to listen for jobs and start the worker goroutines.
func (t *ThreadPool) Start() {
	t.mut.Lock()
	t.started = true
	t.mut.Unlock()
	for i := 0; i < t.numberThreads; i++ {
		go t.worker()
	}
}

// worker listens for jobs on the jobs channel and executes them. Each job runs on its own goroutine.
func (t *ThreadPool) worker() {
	for job := range t.jobs {
		go func(j func() error) {
			err := j()
			if err != nil {
				simplelog.Infof("Failed to execute job: %v", err)
			}
			t.mut.Lock()
			t.pendingJobs--
			t.mut.Unlock()
			t.wg.Done()
		}(job)
	}
}

// Wait blocks until all jobs have finished. If no jobs were added, it returns an error.
func (t *ThreadPool) Wait() error {
	t.mut.Lock()
	if !t.started {
		t.mut.Unlock()
		return fmt.Errorf("thread pool was never started before calling wait")
	}
	if t.pendingJobs == 0 {
		t.mut.Unlock()
		return fmt.Errorf("thread pool wait called with no pending jobs this is unexpected")
	}
	t.mut.Unlock()
	t.wg.Wait()
	close(t.jobs)
	return nil
}

// PendingJobs returns the number of jobs that are pending.
func (t *ThreadPool) PendingJobs() int {
	t.mut.Lock()
	defer t.mut.Unlock()
	return t.pendingJobs
}
