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
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/rsvihladremio/dremio-diagnostic-collector/diagnostics"
)

type HostCaptureConfiguration struct {
	Logger         *log.Logger
	IsCoordinator  bool
	Collector      Collector
	Host           string
	OutputLocation string
	DremioConfDir  string
	DremioLogDir   string
}

func Capture(conf HostCaptureConfiguration) (files []CollectedFile, failedFiles []FailedFiles) {
	host := conf.Host
	c := conf.Collector
	outputLoc := conf.OutputLocation
	dremioConfDir := conf.DremioConfDir
	dremioLogDir := conf.DremioLogDir
	isCoordinator := conf.IsCoordinator
	logger := conf.Logger

	err := setupDiagDir(conf)
	if err != nil {
		logger.Printf("ERROR: failed to setup diag directory for host %v due to error %v, will not collect diags for this host", host, err)
		return []CollectedFile{}, []FailedFiles{}
	}
	o, err := c.HostExecute(host, isCoordinator, diagnostics.IOStatArgs()...)
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
	foundConfigFiles, err := findFiles(conf, dremioConfDir)
	if err != nil {
		logger.Printf("ERROR: host %v unable to find files in directory %v with error %v", host, dremioConfDir, err)
	} else {
		for _, c := range foundConfigFiles {
			confFiles = append(confFiles, filepath.Join(dremioConfDir, c))
		}
	}

	collected, failed := copyFiles(conf, "conf", confFiles)
	files = append(files, collected...)
	failedFiles = append(failedFiles, failed...)

	logFiles := []string{}
	foundLogFiles, err := findFiles(conf, dremioLogDir)
	if err != nil {
		logger.Printf("ERROR: host %v unable to find files in directory %v with error %v", host, dremioLogDir, err)
	} else {
		logger.Printf("INFO: host %v finished finding files to copy out of the log directory", host)
		for _, c := range foundLogFiles {
			logFiles = append(logFiles, filepath.Join(dremioLogDir, c))
		}
		collected, failed := copyFiles(conf, "log", logFiles)
		files = append(files, collected...)
		failedFiles = append(failedFiles, failed...)
	}

	return files, failedFiles
}

func setupDiagDir(conf HostCaptureConfiguration) error {
	host := conf.Host
	outputLoc := conf.OutputLocation
	logger := conf.Logger

	if err := os.Mkdir(filepath.Join(outputLoc, host), DirPerms); err != nil {
		logger.Printf("ERROR: host %v had error %v trying to make it's host dir", host, err)
		return err
	}

	if err := os.Mkdir(filepath.Join(outputLoc, host, "conf"), DirPerms); err != nil {
		logger.Printf("ERROR: host %v had error %v trying to make it's host conf dir", host, err)
		return err
	}
	if err := os.Mkdir(filepath.Join(outputLoc, host, "logs"), DirPerms); err != nil {
		logger.Printf("ERROR: host %v had error %v trying to make it's host log dir", host, err)
		return err
	}
	return nil
}

func copyFiles(conf HostCaptureConfiguration, destDir string, filesToCopy []string) (collectedFiles []CollectedFile, failedFiles []FailedFiles) {
	outputLoc := conf.OutputLocation
	host := conf.Host
	logger := conf.Logger
	c := conf.Collector
	isCoordinator := conf.IsCoordinator

	for i := range filesToCopy {
		log := filesToCopy[i]
		fileName := filepath.Join(outputLoc, host, destDir, filepath.Base(log))
		if out, err := c.CopyFromHost(host, isCoordinator, log, fileName); err != nil {
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
			collectedFiles = append(collectedFiles, CollectedFile{
				Path: fileName,
				Size: size,
			})
			logger.Printf("INFO: host %v copied %v to %v", host, log, fileName)
		}
	}
	return collectedFiles, failedFiles
}

func findFiles(conf HostCaptureConfiguration, searchDir string) ([]string, error) {
	host := conf.Host
	c := conf.Collector
	isCoordinator := conf.IsCoordinator

	out, err := c.HostExecute(host, isCoordinator, "ls", "-1", searchDir)
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
