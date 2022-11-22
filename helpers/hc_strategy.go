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
