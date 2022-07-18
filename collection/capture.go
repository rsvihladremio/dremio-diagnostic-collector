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
	"strings"

	"github.com/rsvihladremio/dremio-diagnostic-collector/diagnostics"
)

type GenericHostCapture struct {
	isCoordinator bool
	logOutput     io.Writer
	c             Collector
}

func (g *GenericHostCapture) Capture(host, outputLoc, dremioConfDir, dremioLogDir string) (files []CollectedFile, failedFiles []FailedFiles) {
	findFiles := func(host string, searchDir string) ([]string, error) {
		out, err := g.c.HostExecute(host, g.isCoordinator, "ls", "-1", searchDir)
		if err != nil {
			return []string{}, fmt.Errorf("ls -l failed due to error %v", err)
		}

		rawFoundFiles := strings.Split(out, "\n")
		var foundFiles []string
		for _, f := range rawFoundFiles {
			if f != "" {
				foundFiles = append(foundFiles, f)
			}
		}
		return foundFiles, nil
	}

	logger := log.New(g.logOutput, fmt.Sprintf("HOST: %v - ", host), log.Ldate|log.Ltime|log.Lshortfile)
	if err := os.Mkdir(filepath.Join(outputLoc, host), DirPerms); err != nil {
		logger.Printf("ERROR: host %v had error %v trying to make it's host dir", host, err)
		return files, failedFiles
	}

	if err := os.Mkdir(filepath.Join(outputLoc, host, "conf"), DirPerms); err != nil {
		logger.Printf("ERROR: host %v had error %v trying to make it's host conf dir", host, err)
		return files, failedFiles
	}
	if err := os.Mkdir(filepath.Join(outputLoc, host, "logs"), DirPerms); err != nil {
		logger.Printf("ERROR: host %v had error %v trying to make it's host log dir", host, err)
		return files, failedFiles
	}
	o, err := g.c.HostExecute(host, g.isCoordinator, diagnostics.IOStatArgs()...)
	if err != nil {
		logger.Printf("ERROR: host %v failed iostat with error %v", host, err)
	} else {
		logger.Printf("INFO: host %v finished iostat", host)
		fileName := filepath.Join(outputLoc, host, "iostat.txt")
		if err := os.WriteFile(fileName, []byte(o), 0600); err != nil {
			failedFiles = append(failedFiles, FailedFiles{
				Path: fileName,
				Err:  err,
			})
			logger.Printf("ERROR: unable to save iostat.txt for %v due to error %v output was %v", host, err, o)
		} else {
			fileInfo, err := os.Stat(fileName)
			size := int64(0)
			if err != nil {
				logger.Printf("WARN cannot get file size for file %v due to error %v. Storing size as 0", fileName, err)
			} else {
				size = fileInfo.Size()
			}
			files = append(files, CollectedFile{
				Path: fileName,
				Size: size,
			})
		}
	}

	confFiles := []string{}
	foundConfigFiles, err := findFiles(host, dremioConfDir)
	if err != nil {
		logger.Printf("ERROR: host %v unable to find files in directory %v with error %v", host, dremioConfDir, err)
	} else {
		for _, c := range foundConfigFiles {
			confFiles = append(confFiles, filepath.Join(dremioConfDir, c))
		}
	}
	for i := range confFiles {
		conf := confFiles[i]
		fileName := filepath.Join(outputLoc, host, "conf", filepath.Base(conf))
		if out, err := g.c.CopyFromHost(host, g.isCoordinator, conf, fileName); err != nil {
			failedFiles = append(failedFiles, FailedFiles{
				Path: fileName,
				Err:  err,
			})
			logger.Printf("ERROR: unable to copy %v from host %v due to error %v output was %v", conf, host, err, out)
		} else {
			fileInfo, err := os.Stat(fileName)
			size := int64(0)
			if err != nil {
				logger.Printf("WARN cannot get file size for file %v due to error %v. Storing size as 0", fileName, err)
			} else {
				size = fileInfo.Size()
			}
			files = append(files, CollectedFile{
				Path: fileName,
				Size: size,
			})
			logger.Printf("INFO: host %v copied %v to %v", host, conf, fileName)
		}
	}

	logFiles := []string{}
	foundLogFiles, err := findFiles(host, dremioLogDir)
	if err != nil {
		logger.Printf("ERROR: host %v unable to find files in directory %v with error %v", host, dremioLogDir, err)
	} else {
		logger.Printf("INFO: host %v finished finding files to copy out of the log directory", host)
		for _, c := range foundLogFiles {
			logFiles = append(logFiles, filepath.Join(dremioLogDir, c))
		}
	}
	for i := range logFiles {
		log := logFiles[i]
		fileName := filepath.Join(outputLoc, host, "logs", filepath.Base(log))
		if out, err := g.c.CopyFromHost(host, g.isCoordinator, log, fileName); err != nil {
			failedFiles = append(failedFiles, FailedFiles{
				Path: fileName,
				Err:  err,
			})
			logger.Printf("ERROR: unable to copy %v from host %v due to error %v and output was %v", log, host, err, out)
		} else {
			fileInfo, err := os.Stat(fileName)
			size := int64(0)
			if err != nil {
				logger.Printf("WARN cannot get file size for file %v due to error %v. Storing size as 0", fileName, err)
			} else {
				size = fileInfo.Size()
			}
			files = append(files, CollectedFile{
				Path: fileName,
				Size: size,
			})
			logger.Printf("INFO: host %v copied %v to %v", host, log, fileName)
		}
	}
	return files, failedFiles
}
