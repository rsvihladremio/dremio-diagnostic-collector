//  Copyright 2023 Dremio Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// helpers module creates a strategy to determine, where to put the files we copy from the cluster.
package helpers

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func NewHCCopyStrategy(ddcfs Filesystem) *CopyStrategyHC {
	dir := time.Now().Format("20060102-150405-DDC")
	tmpDir, _ := ddcfs.MkdirTemp("", "*")
	return &CopyStrategyHC{
		StrategyName: "healthcheck",
		BaseDir:      dir,
		TmpDir:       tmpDir,
		Fs:           ddcfs,
	}
}

/*
This struct holds the details we need to copy files. The strategy is used to determine where and in what format we copy the files
*/
type CopyStrategyHC struct {
	StrategyName string     // the name of the output strategy (defasult, healthcheck etc)
	TmpDir       string     // tmp dir used for staging files
	BaseDir      string     // the base dir of where the output is routed
	Fs           Filesystem // filesystem interface (so we can pass in realof fake filesystem, assists testing)
}

/*
The healthcheck format example

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

func (s *CopyStrategyHC) CreatePath(fileType, source, nodeType string) (path string, err error) {
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
	err = s.Fs.MkdirAll(path, DirPerms)
	if err != nil {
		return path, err
	}

	return path, nil
}

func (s *CopyStrategyHC) ClusterPath() (path string, err error) {
	baseDir := s.BaseDir
	tmpDir := s.TmpDir

	path = filepath.Join(tmpDir, baseDir)
	err = s.Fs.MkdirAll(path, DirPerms)
	if err != nil {
		return path, err
	}

	return path, nil
}

// Archive calls out to the main archive function
func (s *CopyStrategyHC) ArchiveDiag(o string, outputLoc string, files []CollectedFile) error {
	// creates the summary file
	summaryFile := filepath.Join(s.TmpDir, "summary.json")
	err := s.Fs.WriteFile(summaryFile, []byte(o), 0600)
	if err != nil {
		return fmt.Errorf("failed writing summary file '%v' due to error %v", summaryFile, err)
	}
	// add the summary file to the list
	files = append(files, CollectedFile{
		Path: summaryFile,
		Size: int64(len([]byte(o))),
	})
	// cleanup when done
	defer func() {
		log.Printf("cleaning up temp directory %v", s.TmpDir)
		//temp folders stay around forever unless we tell them to go away
		if err := s.Fs.RemoveAll(s.TmpDir); err != nil {
			log.Printf("WARN: unable to remove %v due to error %v. It will need to be removed manually", s.TmpDir, err)
		}
	}()
	// create completed file (its not gzipped)
	file, err := s.createHCFiles()
	if err != nil {
		return err
	}
	// Add file info
	fi, _ := os.Stat(file)
	files = append(files, CollectedFile{
		Path: file,
		Size: fi.Size(),
	})
	// call general archive routine
	err = ArchiveDiagDirectory(outputLoc, s.TmpDir, files)
	if err != nil {
		return err
	}
	return nil
}

// This function creates a couple of supplemental files required for the HC data to be uploaded
func (s *CopyStrategyHC) createHCFiles() (file string, err error) {
	baseDir := s.BaseDir
	tmpDir := s.TmpDir

	path := filepath.Join(tmpDir, baseDir, "completed")
	compFile := filepath.Join(path, baseDir)
	err = s.Fs.MkdirAll(path, DirPerms)
	if err != nil {
		return compFile, fmt.Errorf("ERROR: failed to create HC completed dir %v due to error: %v", path, err)
	}

	txt := []byte(baseDir)
	err = s.Fs.WriteFile(compFile, txt, 0600)
	if err != nil {
		return compFile, fmt.Errorf("ERROR: failed to create HC completed file %v due to error: %v", compFile, err)

	}

	return compFile, nil

}
