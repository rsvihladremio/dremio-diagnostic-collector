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
	"fmt"
	"log"
	"path/filepath"
	"time"
)

func NewBACopyStrategy(ddcfs Filesystem) *CopyStrategyBasic {
	dir := time.Now().Format("20060102-150405-DDC")
	tmpDir, _ := ddcfs.MkdirTemp("", "*")
	return &CopyStrategyBasic{
		StrategyName: "basic",
		BaseDir:      dir,
		TmpDir:       tmpDir,
		Fs:           ddcfs,
	}
}

/*
This struct holds the details we need to copy files. The strategy is used to determine where and in what format we copy the files
*/
type CopyStrategyBasic struct {
	StrategyName string     // the name of the output strategy (defasult, healthcheck etc)
	TmpDir       string     // tmp dir used for staging files
	BaseDir      string     // the base dir of where the output is routed
	Fs           Filesystem // filesystem interface (so we can pass in realof fake filesystem, assists testing)
}

type CollectedFile struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

/*

The default format example

./ (the suffix DDC to identify a diag uploadedf from the collector)
├── coordinators
│   	├──── logs
│		│		└─ dremio-master-0 / 1.2.3.4-C
│		│
│   	└──── config
│				└─ dremio-executor-0 / 10.2.3.4-E

...

*/

func (s *CopyStrategyBasic) CreatePath(fileType, source, nodeType string) (path string, err error) {
	baseDir := s.BaseDir
	tmpDir := s.TmpDir

	// With this strategy nodes arealreayd grouped under a parent directory for type
	// so there is no need for an identifier suffix for SSH nodes
	if nodeType == "coordinator" {
		path = filepath.Join(tmpDir, baseDir, "coordinators", source, fileType)
	} else {
		path = filepath.Join(tmpDir, baseDir, "executors", source, fileType)
	}

	err = s.Fs.MkdirAll(path, DirPerms)
	if err != nil {
		return path, err
	}

	return path, nil
}

// This method is a noop for this strategy
func (s *CopyStrategyBasic) GzipAllFiles(path string) ([]CollectedFile, error) {
	return nil, nil
}

// Archive calls out to the main archive function
func (s *CopyStrategyBasic) ArchiveDiag(o string, outputFile string, files []CollectedFile) error {
	// creates the summary file
	summaryFile := filepath.Join(s.TmpDir, "summary.json")
	err := s.Fs.WriteFile(summaryFile, []byte(o), 0600)
	if err != nil {
		return fmt.Errorf("failed writing summary file '%v' due to error %v", summaryFile, err)
	}
	// cleanup when done
	defer func() {
		log.Printf("cleaning up temp directory %v", s.TmpDir)
		//temp folders stay around forever unless we tell them to go away
		if err := s.Fs.RemoveAll(s.TmpDir); err != nil {
			log.Printf("WARN: unable to remove %v due to error %v. It will need to be removed manually", s.TmpDir, err)
		}
	}()

	// call general archive routine
	err = ArchiveDiagDirectory(outputFile, s.TmpDir, files)
	if err != nil {
		return err
	}
	return nil
}
