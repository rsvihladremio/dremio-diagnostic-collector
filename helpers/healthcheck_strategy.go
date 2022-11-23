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

/*
This module creates a strategy to determine, where to put the files we copy from the cluster.
*/

package helpers

import (
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/rsvihladremio/dremio-diagnostic-collector/cli"
)

func NewHCCopyStrategy(ddcfs Filesystem) *CopyStrategyHC {
	dir := time.Now().Format("20060102-150405-DDC")
	tmpDir, _ := ddcfs.MkdirTemp("", "*")
	return &CopyStrategyHC{
		StrategyName: "healthcheck",
		BaseDir:      dir,
		TmpDir:       tmpDir,
	}
}

/*
This struct holds the details we need to copy files. The strategy is used to determine where and in what format we copy the files
*/
type CopyStrategyHC struct {
	StrategyName string // the name of the output strategy (defasult, healthcheck etc)
	TmpDir       string // tmp dir used for staging files
	BaseDir      string // the base dir of where the output is routed
	ZipPath      string // the base dir of the copied file (may include additional subdirs below BaseDir)
}

// Returns the base dir
func (s *CopyStrategyHC) GetBaseDir() string {
	dir := s.BaseDir
	return dir
}

// Returns the tmp dir
func (s *CopyStrategyHC) GetTmpDir() string {
	dir := s.TmpDir
	return dir
}

/*

The healthcheck format example

// TODO use - not _
// use logs not log
// gzip all files
// change config to configuration


20221110-141414-DDC (the suffix DDC to identify a diag uploaded from the collector)
├── configuration
│   ├── dremio-executor-0 -- 1.2.3.4-C
│   ├── dremio-executor-1 -- 1.2.3.5-E
│   ├── dremio-executor-2
│   └── dremio-master-0
├── dremio-cloner
├── job-profiles
├── kubernetes
├── kvstore
├── logs
│   ├── dremio-executor-0
│   ├── dremio-executor-1
│   ├── dremio-executor-2
│   └── dremio-master-0
├── node-info
│   ├── dremio-executor-0
│   ├── dremio-executor-1
│   ├── dremio-executor-2
│   └── dremio-master-0
├── queries
├── query-analyzer
│   ├── chunks
│   ├── errorchunks
│   ├── errormessages
│   └── results
└── system-tables
*/

func (s *CopyStrategyHC) CreatePath(ddcfs Filesystem, fileType, source, nodeType string) (path string, err error) {
	baseDir := s.BaseDir
	tmpDir := s.TmpDir

	// We only tag a suffix of '-C' / '-E' for ssh nodes, the K8s pods are desriptive enough to determine the coordinator / executpr
	var isK8s bool
	if strings.Contains(source, "dremio-master") || strings.Contains(source, "dremio-executor") || strings.Contains(source, "dremio-coordinator") {
		isK8s = true
	}
	if !isK8s { // SSH node types
		if nodeType == "coordinator" {
			path = filepath.Join(tmpDir, baseDir, fileType, source+"-C")

		} else {
			path = filepath.Join(tmpDir, baseDir, fileType, source+"-E")
		}
	} else { // K8s node types
		path = filepath.Join(tmpDir, baseDir, fileType, source)
	}
	err = ddcfs.MkdirAll(path, DirPerms)
	if err != nil {
		return path, err
	}

	return path, nil
}

func (s *CopyStrategyHC) GzipAllFiles(ddcfs Filesystem, path string) (err error) {
	var foundFiles []string
	if runtime.GOOS == "windows" {
		// Currently windows gzipping isnt supported
		return nil
	}
	foundFiles, err = findAllFiles(path)
	if err != nil {
		return err
	}

	for _, file := range foundFiles {
		if file == "" {
			break
		}
		zf := file + ".gz"
		fmt.Printf("file: %v\n", zf)
		err = gZipFile(ddcfs, zf, file)
		if err != nil {
			return err
		}
	}
	return nil

}

func findAllFiles(path string) ([]string, error) {
	cmd := cli.Cli{}
	f := []string{}
	out, err := cmd.Execute("find", path, "-type", "f")
	if err != nil {
		return f, err
	}
	f = strings.Split(out, "\n")
	return f, nil
}

func gZipFile(ddcfs Filesystem, zipFileName, file string) error {
	// Create a buffer to write our archive to.
	zipFile, err := ddcfs.Create(filepath.Clean(zipFileName))
	if err != nil {
		return err
	}
	defer func() {
		err := zipFile.Close()
		if err != nil {
			log.Printf("unable to close file %v due to error %v", zipFileName, err)
		}

	}()
	// Create a new gzip archive.
	w := gzip.NewWriter(zipFile)
	defer func() {
		err := w.Close()
		if err != nil {
			log.Printf("unable to close file %v due to error %v", zipFileName, err)
		}
	}()
	log.Printf("gzipping file %v into %v", file, zipFileName)
	rf, err := os.Open(filepath.Clean(file))
	if err != nil {
		return err
	}
	defer func() {
		err := rf.Close()
		if err != nil {
			log.Printf("unable to close file %v due to error %v", zipFileName, err)
		}
		err = ddcfs.Remove(rf.Name())
		if err != nil {
			log.Printf("unable to remove file %v due to error %v", rf, err)
		}
	}()
	_, err = io.Copy(w, rf)
	if err != nil {
		return err
	}
	return nil
}

// Archive calls out to the main archive function
func (s *CopyStrategyHC) ArchiveDiag(ddcfs Filesystem, outputLoc, outputDir string, files []CollectedFile) error {
	err := s.GzipAllFiles(ddcfs, s.TmpDir)
	if err != nil {
		log.Printf("ERROR: when gzipping files for archive: %v", err)
	}
	err = ArchiveDiagDirectory(outputLoc, s.GetTmpDir(), files)
	if err != nil {
		return err
	}
	return nil
}
