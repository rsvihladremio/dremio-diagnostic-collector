/*
   Copyright 2022 Ryan SVIHLA

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

//collection package provides the interface for collection implementation and the actual collection execution
package collection

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/rsvihladremio/dremio-diagnostic-collector/diagnostics"
)

type Collector interface {
	CopyFromHost(hostString, source, destination string) (out string, err error)
	FindHosts(searchTerm string) (podName []string, err error)
	HostExecute(hostString string, args ...string) (out string, err error)
}

func Execute(c Collector, coordinatorStr, executorsStr, outputLoc string, logOutput io.Writer) error {
	if err := os.Mkdir("diag", 0755); err != nil {
		return err
	}
	coordinators, err := c.FindHosts(coordinatorStr)
	if err != nil {
		return err
	}
	var wg sync.WaitGroup
	for _, coordinator := range coordinators {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()
			GenericHostCapture(c, logOutput, host)
		}(coordinator)
	}
	executors, err := c.FindHosts(executorsStr)
	if err != nil {
		return err
	}
	for _, executor := range executors {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()
			GenericHostCapture(c, logOutput, host)
		}(executor)
	}
	wg.Wait()
	return nil
}

func GenericHostCapture(c Collector, logOutput io.Writer, host string) {
	logger := log.New(logOutput, fmt.Sprintf("HOST: %v", host), log.Ldate|log.Ltime|log.Lshortfile)
	if err := os.Mkdir(filepath.Join("diag", host), 0755); err != nil {
		logger.Printf("ERROR: host %v had error %v trying to make it's host dir", host, err)
		return
	}
	if err := os.Mkdir(filepath.Join("diag", host, "conf"), 0755); err != nil {
		logger.Printf("ERROR: host %v had error %v trying to make it's host conf dir", host, err)
		return
	}
	if err := os.Mkdir(filepath.Join("diag", host, "logs"), 0755); err != nil {
		logger.Printf("ERROR: host %v had error %v trying to make it's host log dir", host, err)
		return
	}
	o, err := c.HostExecute(host, diagnostics.IOStatArgs()...)
	if err != nil {
		logger.Printf("ERROR: host %v failed iostat with error %v", host, err)
	} else {
		logger.Printf("INFO: host %v finished iostat", host)
		if err := os.WriteFile(fmt.Sprintf("diag/%v/iostat.txt", host), []byte(o), 0600); err != nil {
			logger.Printf("ERROR: unable to save iostat.txt for %v due to error %v output was %v", host, err, o)
		}
	}

	confFiles := []string{"dremio.conf", "dremio-env", "logback-access.xml", "logback-admin.xml", "logback.xml"}
	for i := range confFiles {
		conf := confFiles[i]
		if out, err := c.CopyFromHost(host, filepath.Join("/", "etc", "dremio", conf), filepath.Join("diag", host, "conf", conf)); err != nil {
			logger.Printf("ERROR: unable to copy %v from host %v due to error %v output was %v", conf, host, err, o)
		} else {
			logger.Printf("INFO: host %v copied %v, and logged the following %v", conf, host, out)
		}
	}

	logFiles := []string{"audit.json", "hive.deprecated.function.warning.log", "metadata_refresh.log", "queries.json", "server.gc", "server.log", "server.out", "tracker.json"}
	for i := range logFiles {
		log := logFiles[i]
		if out, err := c.CopyFromHost(host, filepath.Join("/", "var", "log", "dremio", log), filepath.Join("diag", host, "logs", log)); err != nil {
			logger.Printf("ERROR: unable to copy %v from host %v due to error %v and output was %v", log, host, err, out)
		} else {
			logger.Printf("INFO: host %v copied %v, and logged the following %v", log, host, out)
		}
	}
}
