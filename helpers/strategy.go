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
This module uses the strategy name to determine, where to put the files we copy from the cluster.
When we copy files we know where they are from and what their pupose is (e.g. logs, config etc).
With this info we can construct a path thats formed of these elements to send back to the calling
function that does the actual file copying.
*/

package helpers

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

/*
This struct holds the details we need to copy files. The strategy is used to determine where and in what format we copy the files
*/
type CopyStrategy struct {
	StrategyName string // the name of the output strategy (defasult, healthcheck etc)
	BaseDir      string // the base dir of where the output is routed
	FileType     string // what the file(s) are; configs, logs etc
	Source       string // where the files are from (usually the node or pod)
	NodeType     string // Usually "coordinator" or "executor" (ssh nodes only identify with a IP)
}

func SetupDirectories() (err error) {
	return err
}

// The healthceck uses a base directory based on current timestamp
func hcBaseDir(cs CopyStrategy) string {
	dir := time.Now().Format("20060102_150405-DDC")
	return dir
}

/*
Returns a path where files can be copied to based on the copy strategy
*/
func CreatePath(cs CopyStrategy) (path string, err error) {
	switch cs.StrategyName {
	case "default":
		path := strategyDefault(cs)
		return path, nil
	case "healthcheck":
		path := strategyHealthcheck(cs)
		return path, nil
	}
	return "", nil
}

/*
Creates a base directory for the collected files to be cxopied into
this is different depending on the collection strategy
*/
func CreateBaseDir(cs CopyStrategy) (dir string, err error) {
	switch cs.StrategyName {
	case "default":
		dir := "."
		return dir, nil
	case "healthcheck":
		dir := hcBaseDir(cs)
		return dir, nil
	}
	err = fmt.Errorf("ERROR: unable to create base directory for strategy %v", cs.StrategyName)
	return "", err
}

/*
The default strategy follows out "traditional" format

output.zip (named)
|── coordinators
|   └──node
│       ├── config
│       ├── logs
│       └── ...
|── executors
|   └──node
│       ├── config
│       ├── logs
│       └── ...
├── kubernetes
├── kvstore
├── ...
*/
func strategyDefault(cs CopyStrategy) string {
	baseDir := cs.BaseDir
	source := cs.Source
	fileType := cs.FileType
	nodeType := cs.NodeType
	var path string
	if nodeType == "coordinator" {
		path = filepath.Join(baseDir, "coordinators", source, fileType)

	} else {
		path = filepath.Join(baseDir, "executors", source, fileType)
	}
	return path
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

func strategyHealthcheck(cs CopyStrategy) string {
	baseDir := cs.BaseDir
	source := cs.Source
	fileType := cs.FileType
	nodeType := cs.NodeType
	var path string
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
	return path
}
