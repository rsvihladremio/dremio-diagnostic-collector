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
	"archive/zip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"sync"

	"github.com/rsvihladremio/dremio-diagnostic-collector/diagnostics"
)

type Collector interface {
	CopyFromHost(hostString string, isCoordinator bool, source, destination string) (out string, err error)
	FindHosts(searchTerm string) (podName []string, err error)
	HostExecute(hostString string, isCoordinator bool, args ...string) (stdOut string, err error)
}

type Args struct {
	CoordinatorStr string
	ExecutorsStr   string
	OutputLoc      string
	DremioConfDir  string
	DremioLogDir   string
}

func Execute(c Collector, logOutput io.Writer, collectionArgs Args) error {
	coordinatorStr := collectionArgs.CoordinatorStr
	executorsStr := collectionArgs.ExecutorsStr
	outputLoc := collectionArgs.OutputLoc
	outputDir := outputLoc[:len(outputLoc)-len(filepath.Ext(outputLoc))]
	dremioConfDir := collectionArgs.DremioConfDir
	dremioLogDir := collectionArgs.DremioLogDir
	if err := os.Mkdir(outputDir, 0755); err != nil {
		return err
	}
	defer func() {
		//	if err := os.RemoveAll(outputDir); err != nil {
		//	log.Printf("WARN: unable to remove %v due to error %v. It will need to removed manually", outputDir, err)
		//}
	}()
	coordinators, err := c.FindHosts(coordinatorStr)
	if err != nil {
		return err
	}
	var files []string
	var m sync.Mutex
	var wg sync.WaitGroup
	for _, coordinator := range coordinators {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()
			writtenFiles := GenericHostCapture(c, true, logOutput, host, outputDir, dremioConfDir, dremioLogDir)
			m.Lock()
			files = append(files, writtenFiles...)
			m.Unlock()
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
			writtenFiles := GenericHostCapture(c, false, logOutput, host, outputDir, dremioConfDir, dremioLogDir)
			m.Lock()
			files = append(files, writtenFiles...)
			m.Unlock()
		}(executor)
	}
	wg.Wait()
	ext := filepath.Ext(outputLoc)
	if ext == ".zip" {
		if err := ZipDiag(outputLoc, outputDir, files); err != nil {
			return fmt.Errorf("unable to write zip file %v due to error %v", outputLoc, err)
		}
	}
	return nil
}

func GenericHostCapture(c Collector, isCoordinator bool, logOutput io.Writer, host, outputLoc, dremioConfDir, dremioLogDir string) (files []string) {
	findFiles := func(host string, searchDir string) ([]string, error) {
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

	logger := log.New(logOutput, fmt.Sprintf("HOST: %v", host), log.Ldate|log.Ltime|log.Lshortfile)
	if err := os.Mkdir(filepath.Join(outputLoc, host), 0755); err != nil {
		logger.Printf("ERROR: host %v had error %v trying to make it's host dir", host, err)
		return []string{}
	}

	if err := os.Mkdir(filepath.Join(outputLoc, host, "conf"), 0755); err != nil {
		logger.Printf("ERROR: host %v had error %v trying to make it's host conf dir", host, err)
		return []string{}
	}
	if err := os.Mkdir(filepath.Join(outputLoc, host, "logs"), 0755); err != nil {
		logger.Printf("ERROR: host %v had error %v trying to make it's host log dir", host, err)
		return []string{}
	}
	o, err := c.HostExecute(host, isCoordinator, diagnostics.IOStatArgs()...)
	if err != nil {
		logger.Printf("ERROR: host %v failed iostat with error %v", host, err)
	} else {
		logger.Printf("INFO: host %v finished iostat", host)
		fileName := filepath.Join(outputLoc, host, "iostat.txt")
		if err := os.WriteFile(fileName, []byte(o), 0600); err != nil {
			files = append(files, fileName)
			logger.Printf("ERROR: unable to save iostat.txt for %v due to error %v output was %v", host, err, o)
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
		if out, err := c.CopyFromHost(host, isCoordinator, conf, fileName); err != nil {
			logger.Printf("ERROR: unable to copy %v from host %v due to error %v output was %v", conf, host, err, out)
		} else {
			files = append(files, fileName)
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
		if out, err := c.CopyFromHost(host, isCoordinator, log, fileName); err != nil {
			logger.Printf("ERROR: unable to copy %v from host %v due to error %v and output was %v", log, host, err, out)
		} else {
			files = append(files, fileName)
			logger.Printf("INFO: host %v copied %v to %v", host, log, fileName)
		}
	}
	return files
}

func ZipDiag(zipFileName string, baseDir string, files []string) error {
	// Create a buffer to write our archive to.
	zipFile, err := os.Create(zipFileName)
	if err != nil {
		return err
	}
	// Create a new zip archive.
	w := zip.NewWriter(zipFile)
	defer func() {
		err := w.Close()
		if err != nil {
			log.Printf("unable to close file %v due to error %v", zipFileName, err)
		}
	}()
	// Add some files to the archive.
	for _, file := range files {

		log.Printf("zipping file %v", file)
		f, err := w.Create(file[len(baseDir):])
		if err != nil {
			return err
		}
		rf, err := os.Open(file)
		if err != nil {
			return err
		}
		_, err = io.Copy(f, rf)
		if err != nil {
			return err
		}
	}
	return nil
}
