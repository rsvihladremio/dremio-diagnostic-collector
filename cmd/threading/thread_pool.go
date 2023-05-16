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

import "github.com/golang/glog"

type ThreadPool struct {
	semaphore chan bool
}

// NewThreadPool creates a thread pool based on channels which will run no more than the parameter numberThreads
// of threads concurrently
func NewThreadPool(numberThreads int) *ThreadPool {
	semaphore := make(chan bool, numberThreads)
	return &ThreadPool{
		semaphore: semaphore,
	}
}

// FireJob launches a func() up to the number of threads allowed by the thread pool
func (t *ThreadPool) FireJob(job func() error) {
	go func() {
		//aquire a lock by sending a value to the channel (can be any value)
		t.semaphore <- true
		defer func() {
			<-t.semaphore // Release semaphore slot.
		}()
		//execute the job
		err := job()
		if err != nil {
			glog.Error(err)
		}
	}()
}

// Wait waits for goroutines to finish by acquiring all slots.
func (t *ThreadPool) Wait() {
	glog.Flush()
	for i := 0; i < cap(t.semaphore); i++ {
		t.semaphore <- true
	}
}
