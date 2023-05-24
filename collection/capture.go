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
	"fmt"
	"path"
	"strings"

	"github.com/rsvihladremio/dremio-diagnostic-collector/cmd/ddcio"
	"github.com/rsvihladremio/dremio-diagnostic-collector/cmd/simplelog"
	"github.com/rsvihladremio/dremio-diagnostic-collector/helpers"
)

type FindErr struct {
	Cmd string
}

func (fe FindErr) Error() string {
	return fmt.Sprintf("find failed due to error %v:", fe.Cmd)
}

// Capture collects diagnostics, conf files and log files from the target hosts. Failures are permissive and
// are first logged and then returned at the end with the reason for the failure.
func Capture(conf HostCaptureConfiguration, outDir string) (files []helpers.CollectedFile, failedFiles []FailedFiles, skippedFiles []string) {
	host := conf.Host

	tarGZ := conf.NodeCaptureOutput
	foundTarGzFiles, err := findFiles(conf, tarGZ, false)
	if err != nil {
		simplelog.Errorf("ERROR: host %v unable to find tar.gz in directory %v with error %v", host, tarGZ, err)
	} else {
		for _, sourceFile := range foundTarGzFiles {
			destFile := path.Join(outDir, path.Base(sourceFile))
			if err := ddcio.CopyFile(sourceFile, destFile); err != nil {
				failedFiles = append(failedFiles, FailedFiles{
					Path: destFile,
					Err:  err,
				})
				simplelog.Errorf("unable to copy file %v from host %v to directory %v due to error %v", sourceFile, host, outDir, err)
			} else {
				fileInfo, err := conf.DDCfs.Stat(destFile)
				//we assume a file size of zero if we are not able to retrieve the file size for some reason
				size := int64(0)
				if err != nil {
					simplelog.Warningf("cannot get file size for file %v due to error %v. Storing size as 0", destFile, err)
				} else {
					size = fileInfo.Size()
				}
				files = append(files, helpers.CollectedFile{
					Path: destFile,
					Size: size,
				})
				simplelog.Infof("host %v copied %v to %v", host, sourceFile, destFile)

			}
		}
	}
	return files, failedFiles, skippedFiles
}

// findFiles runs a simple ls -1 command to find all the top level files and nothing more
// this does mean you will have some errors.
// it will also attempt to find the gclogs based on startup flags if there is no gclog override specified
func findFiles(conf HostCaptureConfiguration, searchDir string, filter bool) ([]string, error) {
	logAge := conf.LogAge
	var out string
	var err error

	// Protect against wildcard search base
	if searchDir == "*" {
		return []string{}, FindErr{Cmd: "wildcard search bases rejected"}
	}

	// Only use mtime for logs
	if filter {
		out, err = ComposeExecute(conf, []string{"find", searchDir, "-maxdepth", "4", "-type", "f", "-mtime", fmt.Sprintf("-%v", logAge), "2>/dev/null"})
	} else {
		out, err = ComposeExecute(conf, []string{"find", searchDir, "-maxdepth", "4", "-type", "f", "2>/dev/null"})
	}

	// For find commands we simply ignore exit status 1 and continue
	// since this is usually something like a "Permission denied" which, in the
	// context of a find command can be ignored.
	if err != nil && !strings.Contains(string(err.Error()), "exit status 1") {
		return []string{}, fmt.Errorf("file search failed failed due to error %v", err)
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
