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
	"path/filepath"
	"strconv"
	"strings"

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
func Capture(conf HostCaptureConfiguration) (files []helpers.CollectedFile, failedFiles []FailedFiles, skippedFiles []string) {
	host := conf.Host
	dremioConfDir := conf.DremioConfDir
	dremioLogDir := conf.DremioLogDir
	logger := conf.Logger
	logAge := conf.LogAge

	// Capture config files
	confFiles := []string{}

	foundConfigFiles, err := findFiles(conf, dremioConfDir+"/", false)
	if err != nil {
		logger.Printf("ERROR: host %v unable to find files in directory %v with error %v", host, dremioConfDir, err)
	} else {
		confFiles = append(confFiles, foundConfigFiles...)

	}

	// Append ongoing list of collected, failed and skipped files
	collected, failed, skipped := copyFiles(conf, "configuration", dremioConfDir, confFiles)
	files = append(files, collected...)
	failedFiles = append(failedFiles, failed...)
	skippedFiles = append(skippedFiles, skipped...)

	// Capture log files and GC log files
	logFiles := []string{}
	var filterLogs bool

	// set flag to filter or not based on default value
	if logAge == 0 {
		filterLogs = false
	} else {
		filterLogs = true
	}
	foundLogFiles, err := findFiles(conf, dremioLogDir+"/", filterLogs)
	if err != nil {
		logger.Printf("ERROR: host %v unable to find files in directory %v with error %v", host, dremioLogDir, err)
	} else {
		logger.Printf("INFO: host %v finished finding files to copy out of the log directory", host)
		logFiles = append(logFiles, foundLogFiles...)
		collected, failed, skipped := copyFiles(conf, "logs", dremioLogDir, logFiles)
		files = append(files, collected...)
		failedFiles = append(failedFiles, failed...)
		skippedFiles = append(skippedFiles, skipped...)
	}

	// Capture GC log files
	gcLogSearchString, err := findGCLogLocation(conf)
	if err != nil {
		logger.Printf("ERROR: host %v unable to find gc log location with error %v", host, err)
	} else {
		var gcLogsToCollect []string
		gcLogs, err := findFiles(conf, gcLogSearchString, filterLogs)
		if err != nil {
			logger.Printf("ERROR: host %v unable to find gc log files at %v with error %v", host, gcLogSearchString, err)
		}
		for _, gclog := range gcLogs {
			alreadyFound := false
			for _, f := range logFiles {
				//skip files already added
				if f == gclog {
					alreadyFound = true
					break
				}
			}
			if !alreadyFound {
				gcLogsToCollect = append(gcLogsToCollect, gclog)
			}
		}
		gcLogDir := filepath.Dir(gcLogSearchString)
		collected, failed, skipped := copyFiles(conf, "log", gcLogDir, gcLogsToCollect)
		files = append(files, collected...)
		failedFiles = append(failedFiles, failed...)
		skippedFiles = append(skippedFiles, skipped...)
	}

	return files, failedFiles, skippedFiles
}

// AWSE deployments archive the logs under and EFS drive mounted on
func adjustForAWSE(file, baseDir string) (nodeType, nodeName string) {
	var pathParts []string
	// If the deployment type is AWSE then we might need to rename the file to avoid clobbering files, the file tree typically looks like this
	/*
			$ tree -d /var/dremio_efs/
		/var/dremio_efs/
		├── log
		│   ├── coordinator
		│   │   ├── archive
		│   │   ├── json
		│   │   │   └── archive
		│   │   └── preview
		│   │       ├── archive
		│   │       └── json
		│   │           └── archive
		│   └── executor
		│       └── ip-10-10-10-176.eu-west-1.compute.internal
		│           ├── archive
		│           └── json
		│               └── archive
		└── thirdparty
	*/
	// So if the file path has "executor" or "coordinator" in it we assume this is AWSE
	// we pass back the node type since an executor's logs can be found on a coordinator
	// so this becomes like an override
	if strings.Contains(file, "executor") {
		nodeType = "executor"
		pathParts = strings.Split(file, string(filepath.Separator))
	} else if strings.Contains(file, "coordinator") {
		nodeType = "coordinator"
	} else {
		nodeType = ""
	}

	// find the node name
	for _, part := range pathParts {
		if strings.Contains(part, "ip-") {
			nodeName = part
		}
	}
	// return the nodetype & nodename (blank if it didnt find anything AWSE)
	return nodeType, nodeName
}

// copyFiles copys all files it is asked to copy to a local destination directory
// the directory must be available or it will error out
func copyFiles(conf HostCaptureConfiguration, fileType string, baseDir string, filesToCopy []string) (collectedFiles []helpers.CollectedFile, failedFiles []FailedFiles, skippedFiles []string) {
	host := conf.Host
	logger := conf.Logger
	isCoordinator := conf.IsCoordinator
	excludeFiles := conf.ExcludeFiles
	var skip bool
	var nodeType string
	s := conf.CopyStrategy
	var fileName string
	var cPath string
	var err error

	// iterate over all files and copy
	for i := range filesToCopy {
		file := filesToCopy[i]
		skip = false
		if isCoordinator {
			nodeType = "coordinator"
		} else {
			nodeType = "executor"
		}
		// Check file to see if it's an AWSE type deployment
		// if it is we adjust the type and name as needed
		awseNodeType, awseNodeName := adjustForAWSE(file, baseDir)
		if awseNodeType == "coordinator" {
			nodeType = awseNodeType
			// AWSE coordinator, we still use the IP
			cPath, err = s.CreatePath(fileType, host, nodeType)
		} else if awseNodeType == "executor" {
			nodeType = awseNodeType
			// AWSE coordinator, but executor logs, we use the AWS node name from the path
			cPath, err = s.CreatePath(fileType, awseNodeName, nodeType)
		} else {
			// Default, we use the node type from the command line and the IP
			cPath, err = s.CreatePath(fileType, host, nodeType)
		}
		if err != nil {
			logger.Printf("ERROR: unable to create path for %v: %v", host, err)
		}
		// Create the file name and path finally
		fileName = filepath.Join(cPath, filepath.Base(file))

		// Check each file to see if its excluded
		// if it is then add it to the skipped list
		// and set a flag to skip collection
		for _, exf := range excludeFiles {
			f := filepath.Base(fileName)
			matched, err := filepath.Match(exf, f)
			if err != nil {
				logger.Printf("ERROR: trying to find a match for %v with file %v", exf, f)
			}
			if matched {
				skippedFiles = append(skippedFiles, fileName)
				skip = true
			}
		}

		// The skip flag is only reset on each new file in the file of files to copy
		// TODO - at some future point we may want to support regex and / or exclude lists from a config file
		if !skip {
			if out, err := ComposeCopy(conf, file, fileName); err != nil {
				failedFiles = append(failedFiles, FailedFiles{
					Path: fileName,
					Err:  err,
				})
				logger.Printf("ERROR: unable to copy %v from host %v due to error %v and output was %v", file, host, err, out)
			} else {
				fileInfo, err := conf.DDCfs.Stat(fileName)
				//we assume a file size of zero if we are not able to retrieve the file size for some reason
				size := int64(0)
				if err != nil {
					logger.Printf("WARN cannot get file size for file %v due to error %v. Storing size as 0", fileName, err)
				} else {
					size = fileInfo.Size()
				}
				collectedFiles = append(collectedFiles, helpers.CollectedFile{
					Path: fileName,
					Size: size,
				})
				logger.Printf("INFO: host %v copied %v to %v", host, file, fileName)
			}
		}

	}
	return collectedFiles, failedFiles, skippedFiles
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

// findGCLogLocation retrieves the gc log location with a search string to greedily retrieve everything by prefix
func findGCLogLocation(conf HostCaptureConfiguration) (gcLogLoc string, err error) {
	if conf.GCLogOverride != "" {
		return conf.GCLogOverride, nil
	}
	pidList, err := ListJavaProcessPids(conf)
	if err != nil {
		return "", fmt.Errorf("unable to find gc logs due to error '%v'", err)
	}
	pid, err := GetDremioPID(pidList)
	if err != nil {
		return "", fmt.Errorf("unable to find gc logs due to error '%v'", err)
	}
	startupFlags, err := GetStartupFlags(conf, pid)
	if err != nil {
		return "", fmt.Errorf("unable to find gc logs due to error '%v'", err)
	}

	logLocation, err := ParseGCLogFromFlags(startupFlags)
	if err != nil {
		return "", fmt.Errorf("unable to find gc logs due to error '%v'", err)
	}
	return logLocation + "*", nil
}

// ListJavaProcessPids uses jcmd to list the processes running on the jvm
func ListJavaProcessPids(conf HostCaptureConfiguration) (pidList string, err error) {
	host := conf.Host
	collector := conf.Collector
	isCoordinator := conf.IsCoordinator
	out, err := collector.HostExecute(host, isCoordinator, "jcmd", "-l")
	if err != nil {
		return "unable to retrieve pid of dremio due to error '%v'", err
	}
	return out, nil
}

// GetDremioPID loops through the output of jcmd -l and finds the dremio pid
func GetDremioPID(pidList string) (pid int, err error) {
	for _, line := range strings.Split(pidList, "\n") {
		if strings.HasSuffix(strings.TrimSpace(line), "com.dremio.dac.daemon.DremioDaemon") {
			tokens := strings.Split(line, " ")
			if len(tokens) != 2 {
				return -1, fmt.Errorf("unexpected result trying to read pid for string '%v' there are '%v' tokens but we expected 2. This is a critical error and should be reported", line, len(tokens))
			}
			return strconv.Atoi(tokens[0])
		}
	}
	return -1, fmt.Errorf("unable to find process 'com.dremio.dac.daemon.DremioDaemon' inside '%v'", pidList)
}

// GetStartupFlags uses jcmd to get the startup parameters for a given pid
func GetStartupFlags(conf HostCaptureConfiguration, pid int) (flags string, err error) {
	host := conf.Host
	collector := conf.Collector
	isCoordinator := conf.IsCoordinator
	return collector.HostExecute(host, isCoordinator, "ps", "-f", strconv.Itoa(pid))
}

// ParseGCLogFromFlags takes a given string with java startup flags and finds the gclog directive
func ParseGCLogFromFlags(startupFlagsStr string) (gcLogLocation string, err error) {
	tokens := strings.Split(startupFlagsStr, " ")
	var found []int
	for i, token := range tokens {
		if strings.HasPrefix(token, "-Xloggc:") {
			found = append(found, i)
		}
	}
	if len(found) == 0 {
		return "", nil
	}
	lastIndex := found[len(found)-1]
	last := tokens[lastIndex]
	gcLogLocationTokens := strings.Split(last, "-Xloggc:")
	if len(gcLogLocationTokens) != 2 {
		return "", fmt.Errorf("unexpected items in string '%v', expected only 2 items but found %v", last, len(gcLogLocationTokens))
	}
	return gcLogLocationTokens[1], nil
}
