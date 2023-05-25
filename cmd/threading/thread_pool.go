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
	"sync"

	"github.com/rsvihladremio/dremio-diagnostic-collector/cmd/simplelog"
)

type ThreadPool struct {
	wg            *sync.WaitGroup
	numberThreads int
	counter       int
	jobs          []Job
}

type Job struct {
	exec func()
	id   int
}

// NewThreadPool creates a thread pool based on channels which will run no more than the parameter numberThreads
// of threads concurrently
func NewThreadPool(numberThreads int) *ThreadPool {
	wg := new(sync.WaitGroup)

	return &ThreadPool{
		wg:            wg,
		numberThreads: numberThreads,
		jobs:          []Job{},
		counter:       0,
	}
}

// FireJob launches a func() up to the number of threads allowed by the thread pool
func (t *ThreadPool) FireJob(job func() error) {
	t.counter++
	j := func() {
		//aquire a lock by sending a value to the channel (can be any value)
		defer func() {
			t.wg.Done()
		}()
		//execute the job
		err := job()
		if err != nil {
			simplelog.Infof("failed the job %v", err)
		}
	}

	t.jobs = append(t.jobs, Job{
		exec: j,
		id:   t.counter,
	})
}

// Wait waits for goroutines to finish by acquiring all slots.
func (t *ThreadPool) Wait() {
	simplelog.Infof("%v jobs to process", len(t.jobs))
	for i, job := range t.jobs {
		j := job
		simplelog.Infof("Starting thread #%v ", j.id)
		t.wg.Add(1)
		go func() {
			j.exec()
			simplelog.Infof("Thread #%v completed", j.id)
		}()

		if i > 0 && i%t.numberThreads == 0 {
			simplelog.Infof("waiting on threads")
			t.wg.Wait()
		}
	}

	//all the rest
	t.wg.Wait()
}
