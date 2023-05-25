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

// collection package provides the interface for collection implementation and the actual collection execution
package collection

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/rsvihladremio/dremio-diagnostic-collector/cmd/simplelog"
	"github.com/rsvihladremio/dremio-diagnostic-collector/helpers"
)

var DirPerms fs.FileMode = 0750

type CopyStrategy interface {
	CreatePath(fileType, source, nodeType string) (path string, err error)
	ArchiveDiag(o string, outputLoc string, files []helpers.CollectedFile) error
	GetTmpDir() string
}

type Collector interface {
	CopyFromHost(hostString string, isCoordinator bool, source, destination string) (out string, err error)
	CopyToHost(hostString string, isCoordinator bool, source, destination string) (out string, err error)
	CopyFromHostSudo(hostString string, isCoordinator bool, sudoUser, source, destination string) (out string, err error)
	CopyToHostSudo(hostString string, isCoordinator bool, sudoUser, source, destination string) (out string, err error)
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
	NodeCaptureOutput         string
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
	operationSystem := runtime.GOOS
	arch := runtime.GOARCH
	var ddcLoc string
	var err error
	ddcLoc, err = os.Executable()
	if err != nil {
		return fmt.Errorf("unable to to find ddc cannot copy it to hosts due to error '%v'", err)
	}
	if operationSystem == "linux" && arch == "amd64" {
		simplelog.Infof("using linux ddc")
	} else {
		// we need to use the exec in the folder next to ddc should be /linux and should contain a ddc exec ddc.yaml
		ddcDir := path.Join(path.Dir(ddcLoc), "linux")
		ddcLoc = path.Join(ddcDir, "ddc")
	}
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
				OutputLocation:            outputLoc,
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
				NodeCaptureOutput:         "/tmp/ddc",
			}
			writtenFiles, failedFiles, skippedFiles := Capture(coordinatorCaptureConf, ddcLoc, s.GetTmpDir())
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
				Collector:     c,
				IsCoordinator: false,
				Logger:        logger,
				Host:          host,
				//OutputLocation:            outputDir,
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
				NodeCaptureOutput:         "/tmp/ddc",
			}
			writtenFiles, failedFiles, skippedFiles := Capture(executorCaptureConf, ddcLoc, outputLoc)
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

	if tarballs, err := FindTarGzFiles(s.GetTmpDir()); err != nil {
		return err
	} else {
		for _, t := range tarballs {
			if err := ExtractTarGz(t, s.GetTmpDir()); err != nil {
				return err
			}
			if err := os.Remove(t); err != nil {
				return err
			}
		}
	}
	// archives the collected files
	// creates the summary file too
	return s.ArchiveDiag(o, outputLoc, files)

}

func ExtractTarGz(gzFilePath, dest string) error {
	reader, err := os.Open(gzFilePath)
	if err != nil {
		return err
	}
	defer reader.Close()

	gzReader, err := gzip.NewReader(reader)
	if err != nil {
		return err
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()

		switch {
		case err == io.EOF:
			return nil
		case err != nil:
			return err
		case header == nil:
			continue
		}

		target := filepath.Join(dest, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}
		case tar.TypeReg:
			file, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			defer file.Close()
			if _, err := io.Copy(file, tarReader); err != nil {
				return err
			}
		}
	}
}

func FindTarGzFiles(rootDir string) ([]string, error) {
	var files []string
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".tar.gz") {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}
