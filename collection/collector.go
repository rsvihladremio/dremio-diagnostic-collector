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
	"path/filepath"
	"sync"
	"time"

	"github.com/rsvihladremio/dremio-diagnostic-collector/helpers"
)

var DirPerms fs.FileMode = 0750

type CopyStrategy interface {
	GetBaseDir() string
	GetTmpDir() string
	CreatePath(ddcfs helpers.Filesystem, fileType, source, nodeType string) (path string, err error)
	GzipAllFiles(ddcfs helpers.Filesystem, path string) error
	ArchiveDiag(ddcfs helpers.Filesystem, outputLoc, outputDir string, files []helpers.CollectedFile) error
}

type Collector interface {
	CopyFromHost(hostString string, isCoordinator bool, source, destination string) (out string, err error)
	FindHosts(searchTerm string) (podName []string, err error)
	HostExecute(hostString string, isCoordinator bool, args ...string) (stdOut string, err error)
}

type Args struct {
	DDCfs                     helpers.Filesystem
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
	CopyStrategy              CopyStrategy
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
	CopyStrategy              CopyStrategy
	DDCfs                     helpers.Filesystem
}

func Execute(c Collector, s CopyStrategy, logOutput io.Writer, collectionArgs Args) error {
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
	ddcfs := collectionArgs.DDCfs
	limit := collectionArgs.SizeLimit
	excludefiles := collectionArgs.ExcludeFiles

	// Obtain a base dir depending on the copy strategy
	tmpDir := s.GetTmpDir()
	outputDir := tmpDir

	// Cleanup temp dir once complete
	defer func() {
		log.Printf("cleaning up temp directory %v", outputDir)
		//temp folders stay around forever unless we tell them to go away
		if err := ddcfs.RemoveAll(outputDir); err != nil {
			log.Printf("WARN: unable to remove %v due to error %v. It will need to be removed manually", outputDir, err)
		}
	}()

	coordinators, err := c.FindHosts(coordinatorStr)
	if err != nil {
		return err
	}
	var files []helpers.CollectedFile
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
				OutputLocation:            outputDir,
				DremioConfDir:             dremioConfDir,
				DremioLogDir:              dremioLogDir,
				GCLogOverride:             dremioGcDir,
				DurationDiagnosticTooling: collectionArgs.DurationDiagnosticTooling,
				LogAge:                    logAge,
				jfrduration:               jfrduration,
				SudoUser:                  sudoUser,
				SizeLimit:                 limit,
				ExcludeFiles:              excludefiles,
				CopyStrategy:              s,
				DDCfs:                     ddcfs,
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
				OutputLocation:            outputDir,
				DremioConfDir:             dremioConfDir,
				DremioLogDir:              dremioLogDir,
				GCLogOverride:             dremioGcDir,
				DurationDiagnosticTooling: collectionArgs.DurationDiagnosticTooling,
				LogAge:                    logAge,
				jfrduration:               jfrduration,
				SudoUser:                  sudoUser,
				ExcludeFiles:              excludefiles,
				CopyStrategy:              s,
				DDCfs:                     ddcfs,
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
	err = ddcfs.WriteFile(summaryFile, []byte(o), 0600)
	if err != nil {
		return fmt.Errorf("failed writing summary file '%v' due to error %v", summaryFile, err)
	}
	files = append(files, helpers.CollectedFile{
		Path: summaryFile,
		Size: int64(len([]byte(o))),
	})

	return s.ArchiveDiag(ddcfs, outputLoc, s.GetTmpDir(), files)
	//return archiveDiagDirectory(outputLoc, s.GetTmpDir(), files)

}

/*
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
*/
