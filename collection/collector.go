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
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"sync"

	"github.com/rsvihladremio/dremio-diagnostic-collector/diagnostics"
	"github.com/rsvihladremio/dremio-diagnostic-collector/summary"
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
	start := time.Now().UTC()
	coordinatorStr := collectionArgs.CoordinatorStr
	executorsStr := collectionArgs.ExecutorsStr
	outputLoc := collectionArgs.OutputLoc
	dremioConfDir := collectionArgs.DremioConfDir
	dremioLogDir := collectionArgs.DremioLogDir
	outputDir, err := os.MkdirTemp("", "*")
	if err != nil {
		return err
	}
	executorDir := filepath.Join(outputDir, "executors")
	err = os.Mkdir(executorDir, 0755)
	if err != nil {
		return err
	}
	coordinatorDir := filepath.Join(outputDir, "coordinators")
	err = os.Mkdir(coordinatorDir, 0755)
	if err != nil {
		return err
	}
	defer func() {
		//temp folders stay around forever unless we tell them to go away
		if err := os.RemoveAll(outputDir); err != nil {
			log.Printf("WARN: unable to remove %v due to error %v. It will need to removed manually", outputDir, err)
		}
	}()
	coordinators, err := c.FindHosts(coordinatorStr)
	if err != nil {
		return err
	}
	var files []summary.CollectedFile
	var totalFailedFiles []summary.FailedFiles
	var nodesConnectedTo int
	var m sync.Mutex
	var wg sync.WaitGroup
	coordinatorCapture := GenericHostCapture{
		c:             c,
		isCoordinator: true,
		logOutput:     logOutput,
	}
	executorCapture := GenericHostCapture{
		c:             c,
		isCoordinator: false,
		logOutput:     logOutput,
	}
	for _, coordinator := range coordinators {
		nodesConnectedTo++
		wg.Add(1)
		go func(host string) {
			defer wg.Done()
			writtenFiles, failedFiles := coordinatorCapture.Capture(host, coordinatorDir, dremioConfDir, dremioLogDir)
			m.Lock()
			totalFailedFiles = append(totalFailedFiles, failedFiles...)
			files = append(files, writtenFiles...)
			m.Unlock()
		}(coordinator)
	}
	executors, err := c.FindHosts(executorsStr)
	if err != nil {
		return err
	}
	for _, executor := range executors {
		nodesConnectedTo++
		wg.Add(1)
		go func(host string) {
			defer wg.Done()
			writtenFiles, failedFiles := executorCapture.Capture(host, executorDir, dremioConfDir, dremioLogDir)
			m.Lock()
			totalFailedFiles = append(totalFailedFiles, failedFiles...)
			files = append(files, writtenFiles...)
			m.Unlock()
		}(executor)
	}
	wg.Wait()
	end := time.Now().UTC()
	var collectionInfo summary.CollectionInfo
	collectionInfo.EndTimeUTC = end
	collectionInfo.StartTimeUTC = start
	seconds := end.Unix() - start.Unix()
	collectionInfo.TotalRuntimeStr = fmt.Sprintf("%v seconds", seconds)
	collectionInfo.ClusterInfo.TotalNodesAttempted = len(coordinators) + len(executors)
	collectionInfo.ClusterInfo.NumberNodesContacted = nodesConnectedTo
	collectionInfo.CollectedFiles = files
	totalBytes := int64(0)
	for _, f := range files {
		totalBytes += f.Size
	}
	collectionInfo.TotalBytesCollected = totalBytes
	collectionInfo.Coordinators = coordinators
	collectionInfo.Executors = executors
	collectionInfo.FailedFiles = totalFailedFiles

	o, err := collectionInfo.String()
	if err != nil {
		return err
	}
	summaryFile := filepath.Join(outputDir, "summary.json")
	err = os.WriteFile(summaryFile, []byte(o), 0600)
	if err != nil {
		return fmt.Errorf("failed writing summary file '%v' due to error %v", summaryFile, err)
	}
	files = append(files, summary.CollectedFile{
		Path: summaryFile,
		Size: int64(len([]byte(o))),
	})
	ext := filepath.Ext(outputLoc)
	if ext == ".zip" {
		if err := ZipDiag(outputLoc, outputDir, files); err != nil {
			return fmt.Errorf("unable to write zip file %v due to error %v", outputLoc, err)
		}
	} else if strings.HasSuffix(outputLoc, "tar.gz") || ext == ".tgz" {
		tempFile := strings.Join([]string{strings.TrimSuffix(outputLoc, ext), "tar"}, ".")
		if err := TarDiag(tempFile, outputDir, files); err != nil {
			return fmt.Errorf("unable to write tar file %v due to error %v", outputLoc, err)
		}
		defer func() {
			if err := os.Remove(tempFile); err != nil {
				log.Printf("WARN unable to delete file '%v' due to '%v'", tempFile, err)
			}
		}()
		if err := GZipDiag(outputLoc, outputDir, tempFile); err != nil {
			return fmt.Errorf("unable to write gz file %v due to error %v", outputLoc, err)
		}
	} else if ext == ".tar" {
		if err := TarDiag(outputLoc, outputDir, files); err != nil {
			return fmt.Errorf("unable to write tar file %v due to error %v", outputLoc, err)
		}
	}
	return nil
}

type GenericHostCapture struct {
	isCoordinator bool
	logOutput     io.Writer
	c             Collector
}

func (g *GenericHostCapture) Capture(host, outputLoc, dremioConfDir, dremioLogDir string) (files []summary.CollectedFile, failedFiles []summary.FailedFiles) {
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

	logger := log.New(g.logOutput, fmt.Sprintf("HOST: %v", host), log.Ldate|log.Ltime|log.Lshortfile)
	if err := os.Mkdir(filepath.Join(outputLoc, host), 0755); err != nil {
		logger.Printf("ERROR: host %v had error %v trying to make it's host dir", host, err)
		return files, failedFiles
	}

	if err := os.Mkdir(filepath.Join(outputLoc, host, "conf"), 0755); err != nil {
		logger.Printf("ERROR: host %v had error %v trying to make it's host conf dir", host, err)
		return files, failedFiles
	}
	if err := os.Mkdir(filepath.Join(outputLoc, host, "logs"), 0755); err != nil {
		logger.Printf("ERROR: host %v had error %v trying to make it's host log dir", host, err)
		return files, failedFiles
	}
	o, err := g.c.HostExecute(host, g.isCoordinator, diagnostics.IOStatArgs()...)
	if err != nil {
		logger.Printf("ERROR: host %v failed iostat with error %v", host, err)
	} else {
		logger.Printf("INFO: host %v finished iostat", host)
		fileName := filepath.Join(outputLoc, host, "iostat.txt")
		fileInfo, err := os.Stat(fileName)
		size := int64(0)
		if err != nil {
			logger.Printf("WARN cannot get file size for file %v due to error %v. Storing size as 0", fileName, err)
		} else {
			size = fileInfo.Size()
		}
		if err := os.WriteFile(fileName, []byte(o), 0600); err != nil {
			files = append(files, summary.CollectedFile{
				Path: fileName,
				Size: size,
			})
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
		fileInfo, err := os.Stat(fileName)
		size := int64(0)
		if err != nil {
			logger.Printf("WARN cannot get file size for file %v due to error %v. Storing size as 0", fileName, err)
		} else {
			size = fileInfo.Size()
		}
		if out, err := g.c.CopyFromHost(host, g.isCoordinator, conf, fileName); err != nil {
			logger.Printf("ERROR: unable to copy %v from host %v due to error %v output was %v", conf, host, err, out)
		} else {
			files = append(files, summary.CollectedFile{
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
			failedFiles = append(failedFiles, summary.FailedFiles{
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
			files = append(files, summary.CollectedFile{
				Path: fileName,
				Size: size,
			})
			logger.Printf("INFO: host %v copied %v to %v", host, log, fileName)
		}
	}
	return files, failedFiles
}

func TarDiag(tarFileName string, baseDir string, files []summary.CollectedFile) error {
	// Create a buffer to write our archive to.
	tarFile, err := os.Create(tarFileName)
	if err != nil {
		return err
	}
	// Create a new tar archive.
	tw := tar.NewWriter(tarFile)

	defer func() {
		err := tw.Close()
		if err != nil {
			log.Printf("unable to close file %v due to error %v", tarFileName, err)
		}
	}()
	for _, collectedFile := range files {
		file := collectedFile.Path
		log.Printf("taring file %v", file)
		fileInfo, err := os.Stat(file)
		if err != nil {
			return err
		}
		rf, err := os.Open(file)
		if err != nil {
			return err
		}
		hdr := &tar.Header{
			Name: file[len(baseDir):],
			Mode: 0600,
			Size: fileInfo.Size(),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		_, err = io.Copy(tw, rf)
		if err != nil {
			return err
		}
	}
	if err := tw.Close(); err != nil {
		return err
	}
	return nil
}

func GZipDiag(zipFileName string, baseDir string, file string) error {
	// Create a buffer to write our archive to.
	zipFile, err := os.Create(zipFileName)
	if err != nil {
		return err
	}
	// Create a new gzip archive.
	w := gzip.NewWriter(zipFile)
	defer func() {
		err := w.Close()
		if err != nil {
			log.Printf("unable to close file %v due to error %v", zipFileName, err)
		}
	}()
	log.Printf("gzipping file %v into %v", file, zipFileName)
	rf, err := os.Open(file)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, rf)
	if err != nil {
		return err
	}
	return nil
}

func ZipDiag(zipFileName string, baseDir string, files []summary.CollectedFile) error {
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
	for _, collectedFile := range files {
		file := collectedFile.Path
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
