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

// collection package provides the interface for collection implementation and the actual collection execution
package collection

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rsvihladremio/dremio-diagnostic-collector/helpers"
)

var DirPerms fs.FileMode = 0750

type Collector interface {
	CopyFromHost(hostString string, isCoordinator bool, source, destination string) (out string, err error)
	FindHosts(searchTerm string) (podName []string, err error)
	HostExecute(hostString string, isCoordinator bool, args ...string) (stdOut string, err error)
}

type Args struct {
	Cfs                       helpers.FileSystem
	CoordinatorStr            string
	ExecutorsStr              string
	OutputLoc                 string
	DremioConfDir             string
	DremioLogDir              string
	DremioGcDir               string
	GCLogOverride             string
	DurationDiagnosticTooling int
	LogAge                    int
	JfrDuration               int
	SudoUser                  string
	SizeLimit                 int64
	ExcludeFiles              []string
	CopyStrategy              helpers.CopyStrategy
}

type HostCaptureConfiguration struct {
	Logger                    *log.Logger
	IsCoordinator             bool
	Collector                 Collector
	Host                      string
	OutputLocation            string
	DremioConfDir             string
	DremioLogDir              string
	DurationDiagnosticTooling int
	GCLogOverride             string
	LogAge                    int
	jfrduration               int
	SudoUser                  string
	SizeLimit                 int64
	ExcludeFiles              []string
	CopyStrategy              helpers.CopyStrategy
}

func Execute(c Collector, logOutput io.Writer, collectionArgs Args) error {
	start := time.Now().UTC()
	coordinatorStr := collectionArgs.CoordinatorStr
	executorsStr := collectionArgs.ExecutorsStr
	outputLoc := collectionArgs.OutputLoc
	dremioConfDir := collectionArgs.DremioConfDir
	dremioLogDir := collectionArgs.DremioLogDir
	dremioGcDir := collectionArgs.GCLogOverride
	logAge := collectionArgs.LogAge
	jfrduration := collectionArgs.JfrDuration
	sudoUser := collectionArgs.SudoUser
	cfs := collectionArgs.Cfs
	limit := collectionArgs.SizeLimit
	excludefiles := collectionArgs.ExcludeFiles

	// Obtain a base dir depending on the copy strategy
	baseDir, err := helpers.CreateBaseDir(collectionArgs.CopyStrategy)
	if err != nil {
		return err
	}
	// Tag this base dir onto a temp dir on the local filesystem
	tmpDir, err := cfs.MkdirTemp("", "*")
	if err != nil {
		return err
	}
	outputDir := filepath.Join(tmpDir, baseDir)
	cfs.MkdirAll(outputDir, DirPerms)
	if err != nil {
		return err
	}
	// Write the temp path + base dir back to the copy strategy
	collectionArgs.CopyStrategy.BaseDir = outputDir

	executorDir := filepath.Join(outputDir, "executors")
	err = cfs.Mkdir(executorDir, DirPerms)
	if err != nil {
		return err
	}
	coordinatorDir := filepath.Join(outputDir, "coordinators")
	err = cfs.Mkdir(coordinatorDir, DirPerms)
	if err != nil {
		return err
	}

	// Cleanup - we may want to move this into archiveDiagDirectory
	defer func() {
		log.Printf("cleaning up temp directory %v", outputDir)
		//temp folders stay around forever unless we tell them to go away
		if err := cfs.RemoveAll(outputDir); err != nil {
			log.Printf("WARN: unable to remove %v due to error %v. It will need to be removed manually", outputDir, err)
		}
	}()
	coordinators, err := c.FindHosts(coordinatorStr)
	if err != nil {
		return err
	}
	var files []CollectedFile
	var totalFailedFiles []FailedFiles
	var totalSkippedFiles []string
	var nodesConnectedTo int
	var m sync.Mutex
	var wg sync.WaitGroup

	for _, coordinator := range coordinators {
		nodesConnectedTo++
		wg.Add(1)
		go func(host string) {
			defer wg.Done()
			logger := log.New(logOutput, "", log.Ldate|log.Ltime|log.Lshortfile)
			coordinatorCaptureConf := HostCaptureConfiguration{
				Collector:                 c,
				IsCoordinator:             true,
				Logger:                    logger,
				Host:                      host,
				OutputLocation:            coordinatorDir,
				DremioConfDir:             dremioConfDir,
				DremioLogDir:              dremioLogDir,
				GCLogOverride:             dremioGcDir,
				DurationDiagnosticTooling: collectionArgs.DurationDiagnosticTooling,
				LogAge:                    logAge,
				jfrduration:               jfrduration,
				SudoUser:                  sudoUser,
				SizeLimit:                 limit,
				ExcludeFiles:              excludefiles,
				CopyStrategy:              collectionArgs.CopyStrategy,
			}
			writtenFiles, failedFiles, skippedFiles := Capture(coordinatorCaptureConf)
			m.Lock()
			totalFailedFiles = append(totalFailedFiles, failedFiles...)
			totalSkippedFiles = append(totalSkippedFiles, skippedFiles...)
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
			logger := log.New(logOutput, "", log.Ldate|log.Ltime|log.Lshortfile)
			executorCaptureConf := HostCaptureConfiguration{
				Collector:                 c,
				IsCoordinator:             false,
				Logger:                    logger,
				Host:                      host,
				OutputLocation:            executorDir,
				DremioConfDir:             dremioConfDir,
				DremioLogDir:              dremioLogDir,
				GCLogOverride:             dremioGcDir,
				DurationDiagnosticTooling: collectionArgs.DurationDiagnosticTooling,
				LogAge:                    logAge,
				jfrduration:               jfrduration,
				SudoUser:                  sudoUser,
				ExcludeFiles:              excludefiles,
				CopyStrategy:              collectionArgs.CopyStrategy,
			}
			writtenFiles, failedFiles, skippedFiles := Capture(executorCaptureConf)
			m.Lock()
			totalFailedFiles = append(totalFailedFiles, failedFiles...)
			totalSkippedFiles = append(totalSkippedFiles, skippedFiles...)
			files = append(files, writtenFiles...)
			m.Unlock()
		}(executor)
	}
	wg.Wait()
	end := time.Now().UTC()
	var collectionInfo SummaryInfo
	collectionInfo.EndTimeUTC = end
	collectionInfo.StartTimeUTC = start
	seconds := end.Unix() - start.Unix()
	collectionInfo.TotalRuntimeSeconds = seconds
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
	collectionInfo.SkippedFiles = totalSkippedFiles

	o, err := collectionInfo.String()
	if err != nil {
		return err
	}
	summaryFile := filepath.Join(outputDir, "summary.json")
	err = cfs.WriteFile(summaryFile, []byte(o), 0600)
	if err != nil {
		return fmt.Errorf("failed writing summary file '%v' due to error %v", summaryFile, err)
	}
	files = append(files, CollectedFile{
		Path: summaryFile,
		Size: int64(len([]byte(o))),
	})

	// Setup healthchek format directories
	var nodes []string
	nodes = append(nodes, coordinators...)
	nodes = append(nodes, executors...)
	err = helpers.SetupDirs(filepath.Dir(outputLoc), nodes)
	if err != nil {
		log.Printf("WARN: error setting up dirs for health check data. Error was %v", err)
	}
	return archiveDiagDirectory(outputLoc, outputDir, files)
}

// archiveDiagDirectory will detect the extension asked for and use the correct archival library
// to archive the old directory. It supports: .tgz, .tar.gz and .zip extensions
func archiveDiagDirectory(outputLoc, outputDir string, files []CollectedFile) error {
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
