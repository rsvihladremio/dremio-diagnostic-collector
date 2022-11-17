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
When we copy files we know where they are from and what their pupose is (e.g. logs, config etc).
With this info we can construct a path thats formed of these elements to send back to the calling
function that does the actual file copying.
*/

package helpers

import (
	"path/filepath"
	"strings"
	"time"
)

func NewHCCopyStrategy(name string) *CopyStrategyHC {
	return &CopyStrategyHC{
		StrategyName: name,
	}
}

/*
This struct holds the details we need to copy files. The strategy is used to determine where and in what format we copy the files
*/
type CopyStrategyHC struct {
	StrategyName string // the name of the output strategy (defasult, healthcheck etc)
	BaseDir      string // the base dir of where the output is routed
	FileType     string // what the file(s) are; configs, logs etc
	Source       string // where the files are from (usually the node or pod)
	NodeType     string // Usually "coordinator" or "executor" (ssh nodes only identify with a IP)
}

// The healthceck uses a base directory based on current timestamp
func (s *CopyStrategyHC) SetBaseDir(path string) string {
	dir := time.Now().Format("20060102_150405-DDC")
	s.BaseDir = filepath.Join(path, dir)
	return s.BaseDir
}

// Returns the base dir
func (s *CopyStrategyHC) GetBaseDir() string {
	dir := s.BaseDir
	return dir
}

func (s *CopyStrategyHC) SetType(fileType string) {
	s.FileType = fileType
}

func (s *CopyStrategyHC) GetType() string {
	return s.FileType
}

/*

The default strategy follows the healthcheck format

20221110-141414-DDC - ?
├── configuration
│   ├── dremio-executor-0 -- 1.2.3.4-C
│   ├── dremio-executor-1 -- 12.3.45-E
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
	s.FileType = fileType
	s.Source = source
	s.NodeType = nodeType
	var isK8s bool
	if strings.Contains(source, "dremio-master") || strings.Contains(source, "dremio-executor") || strings.Contains(source, "dremio-coordinator") {
		isK8s = true
	}
	if !isK8s {
		if nodeType == "coordinator" {
			path = filepath.Join(baseDir, fileType, source+"-C")
		} else {
			path = filepath.Join(baseDir, fileType, source+"-E")
		}
	} else {
		path = filepath.Join(baseDir, fileType, source)
	}
	return path, nil
}
