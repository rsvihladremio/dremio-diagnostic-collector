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
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rsvihladremio/dremio-diagnostic-collector/diagnostics"
)

type FindErr struct {
	Cmd string
}

func (fe FindErr) Error() string {
	return fmt.Sprintf("find failed due to error %v:", fe.Cmd)
}

// Capture collects diagnostics, conf files and log files from the target hosts. Failures are permissive and
// are first logged and then returned at the end with the reason for the failure.
func Capture(conf HostCaptureConfiguration) (files []CollectedFile, failedFiles []FailedFiles, skippedFiles []string) {
	host := conf.Host
	dremioConfDir := conf.DremioConfDir
	dremioLogDir := conf.DremioLogDir
	logger := conf.Logger
	logAge := conf.LogAge
	jfrduration := conf.jfrduration

	// Capture any diags like iostat etc
	capturedDiagnosticFiles, failedDiagnosticFiles := captureDiagnostics(conf, "diags")
	files = append(files, capturedDiagnosticFiles...)
	failedFiles = append(failedFiles, failedDiagnosticFiles...)

	// Trigger a JFR if it is required
	if jfrduration > 0 {
		err := captureJFR(conf)
		if err != nil {
			logger.Printf("ERROR: JFR failed on host %v with error %v", host, err)
		}
	}

	// Capture config files
	confFiles := []string{}

	foundConfigFiles, err := findFiles(conf, dremioConfDir+"/", false)
	if err != nil {
		logger.Printf("ERROR: host %v unable to find files in directory %v with error %v", host, dremioConfDir, err)
	} else {
		confFiles = append(confFiles, foundConfigFiles...)

	}

	// Append ongoing list of collected, failed and skipped files
	collected, failed, skipped := copyFiles(conf, "config", dremioConfDir, confFiles)
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
		collected, failed, skipped := copyFiles(conf, "log", dremioLogDir, logFiles)
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

// captureDiagnostics runs iostat on the host, in the future it will run several diagnostics and capture them the same
// time to provide in depth analysis
// iostat must be installed on the host to be captured for this to work
func captureDiagnostics(conf HostCaptureConfiguration, fileType string) (files []CollectedFile, failedFiles []FailedFiles) {
	host := conf.Host
	c := conf.Collector
	isCoordinator := conf.IsCoordinator
	logger := conf.Logger
	var nodeType string
	s := conf.CopyStrategy
	var fileName string

	// run iostat against the host
	o, err := c.HostExecute(host, isCoordinator, diagnostics.IOStatArgs(conf.DurationDiagnosticTooling)...)
	if err != nil {
		logger.Printf("ERROR: host %v failed iostat with error %v", host, err)
	} else {
		if isCoordinator {
			nodeType = "coordinator"
		} else {
			nodeType = "executor"
		}
		cPath, err := s.CreatePath(fileType, host, nodeType)
		if err != nil {
			logger.Printf("ERROR: unable to create path for %v: %v", host, err)
		}

		//take the captured text and write it out to iostat.txt
		logger.Printf("INFO: host %v finished iostat", host)
		fileName = filepath.Join(cPath, filepath.Base("iostat.txt"))
		//fileName := filepath.Join(outputLoc, host, "iostat.txt")
		if err := os.WriteFile(fileName, []byte(o), 0600); err != nil {
			failedFiles = append(failedFiles, FailedFiles{
				Path: fileName,
				Err:  err,
			})
			logger.Printf("ERROR: unable to save iostat.txt for %v due to error %v output was %v", host, err, o)
		} else {
			//get the file size for reporting of how much we captured and transferred across the network
			fileInfo, err := os.Stat(fileName)
			// we assume zero size if we are unable to retrieve the file size
			size := int64(0)
			if err != nil {
				logger.Printf("WARN cannot get file size for file %v due to error %v. Storing size as 0", fileName, err)
			} else {
				size = fileInfo.Size()
			}
			files = append(files, CollectedFile{
				Path: fileName,
				Size: size,
			})
		}
	}

	return files, failedFiles
}

// Since a JFR typically takes longer to run, we want to trigger it but then come back later to pickup the resulting files
// We dont check to see if there is already a JFR running.
func captureJFR(conf HostCaptureConfiguration) (err error) {
	host := conf.Host
	c := conf.Collector
	//outputLoc := conf.OutputLocation
	isCoordinator := conf.IsCoordinator
	logger := conf.Logger
	jfrDuration := conf.jfrduration
	sudoUser := conf.SudoUser
	logdir := conf.DremioLogDir
	start := time.Now().Format("2006-01-02T15-04-05")
	jfrUniqID := logdir + "/" + start

	// run jfr against the host:
	// get the ps output and then deal with the filtering here
	// instead of via the shell
	var pid string

	// Existing methods here could be used, but no capability for sudo user for jcmd
	// we could add this in the future
	/*
		var p int
		pidList, err := ListJavaProcessPids(conf)
		if err != nil {
			return fmt.Errorf("unable to find gc logs due to error '%v'", err)
		}
		p, err = GetDremioPID(pidList)
		if err != nil {
			return fmt.Errorf("unable to find gc logs due to error '%v'", err)
		}
		pid = fmt.Sprint(p)
	*/

	// Get the process id using the ps command instead of jcmd (above).
	o, err := c.HostExecute(host, isCoordinator, diagnostics.JfrPid()...)
	if err != nil {
		logger.Printf("ERROR: host %v failed to get PS output for JFR %v", host, err)
	} else {
		po := strings.Split(o, "\n")
		for _, line := range po {
			if strings.Contains(line, "DremioDaemon") {
				l := strings.Fields(line)
				pid = l[0]
			}
		}
		// Check for a running JFR
		err := checkJfr(conf, pid)
		if err != nil {
			logger.Printf(err.Error())
			return err
		}
		// non sudo user (typically with k8s) will have jcmd access
		// sudo access is more typically needed with on-prem installs (ssh)
		// TODO add logging levels and log all this output with a -vv or -v level
		logger.Printf("INFO: starting JFR on host %v for %v seconds for pid %v", host, jfrDuration, pid)
		if sudoUser == "" {
			_, err := c.HostExecute(host, isCoordinator, diagnostics.JfrEnable(pid)...)
			if err != nil {
				logger.Printf("ERROR: host %v failed to enable JFR with error %v", host, err)
				return err
			}
			_, err = c.HostExecute(host, isCoordinator, diagnostics.JfrRun(pid, jfrDuration, "dremio", jfrUniqID+".jfr")...)
			if err != nil {
				logger.Printf("ERROR: host %v failed to run JFR with error %v", host, err)
				return err
			}

		} else {
			_, err := c.HostExecute(host, isCoordinator, diagnostics.JfrEnableSudo(sudoUser, pid)...)
			if err != nil {
				logger.Printf("ERROR: host %v failed to enable JFR with error %v", host, err)
				return err
			}
			_, err = c.HostExecute(host, isCoordinator, diagnostics.JfrRunSudo(sudoUser, pid, jfrDuration, "dremio", jfrUniqID+".jfr")...)
			if err != nil {
				logger.Printf("ERROR: host %v failed to run JFR with error %v", host, err)
				return err
			}
		}
	}
	return err
}

// Checks there are no existing JFRs running under the given PID
// Although it is possible to run multiple JFRs, it isnt a good idea
// from this tool, in case a customer unintentionally started several
// and potentially ran into problems.
func checkJfr(conf HostCaptureConfiguration, pid string) error {
	host := conf.Host
	c := conf.Collector
	isCoordinator := conf.IsCoordinator
	logger := conf.Logger
	sudoUser := conf.SudoUser

	logger.Printf("INFO: checking host %v for existing JFRs", host)
	// non sudo user (typically with k8s) will have jcmd access
	// sudo access is more typically needed with on-prem installs (ssh)
	if sudoUser == "" {
		o, err := c.HostExecute(host, isCoordinator, diagnostics.JfrCheck(pid)...)
		if err != nil {
			return fmt.Errorf("ERROR: host %v failed to run JFR check error %v", host, err)
		}
		resp := strings.Split(o, "\n")
		for _, line := range resp {
			if strings.Contains(line, "Recording") {
				return fmt.Errorf("WARN: host %v is already running one or more JFRs for pid %v", host, pid)
			}
		}

	} else {
		o, err := c.HostExecute(host, isCoordinator, diagnostics.JfrCheckSudo(sudoUser, pid)...)
		if err != nil {
			return fmt.Errorf("ERROR: host %v failed to run JFR check error %v", host, err)
		}
		resp := strings.Split(o, "\n")
		for _, line := range resp {
			if strings.Contains(line, "Recording") {
				return fmt.Errorf("WARN: host %v is already running one or more JFRs for pid %v", host, pid)
			}
		}

	}
	return nil
}

// copyFiles copys all files it is asked to copy to a local destination directory
// the directory must be available or it will error out
func copyFiles(conf HostCaptureConfiguration, fileType string, baseDir string, filesToCopy []string) (collectedFiles []CollectedFile, failedFiles []FailedFiles, skippedFiles []string) {
	//outputLoc := conf.OutputLocation
	host := conf.Host
	logger := conf.Logger
	c := conf.Collector
	isCoordinator := conf.IsCoordinator
	excludeFiles := conf.ExcludeFiles
	var skip bool
	var nodeType string
	s := conf.CopyStrategy
	var fileName string

	for i := range filesToCopy {
		file := filesToCopy[i]
		skip = false
		if isCoordinator {
			nodeType = "coordinator"
		} else {
			nodeType = "executor"
		}
		cPath, err := s.CreatePath(fileType, host, nodeType)
		if err != nil {
			logger.Printf("ERROR: unable to create path for %v: %v", host, err)
		}
		fileName = filepath.Join(cPath, filepath.Base(file))

		// Check each file to see if its excluded
		// if it is then add it to the skipped list
		// and set a flag to skip collection
		for _, exf := range excludeFiles {
			if exf == filepath.Base(fileName) {
				skippedFiles = append(skippedFiles, fileName)
				skip = true
			}
		}

		// The skip flag is only reset on each new file in the file of files to copy
		// TODO - at some future point we may want to support regex and / or exclude lists from a config file
		if !skip {
			if out, err := c.CopyFromHost(host, isCoordinator, file, fileName); err != nil {
				failedFiles = append(failedFiles, FailedFiles{
					Path: fileName,
					Err:  err,
				})
				logger.Printf("ERROR: unable to copy %v from host %v due to error %v and output was %v", file, host, err, out)
			} else {
				fileInfo, err := os.Stat(fileName)
				//we assume a file size of zero if we are not able to retrieve the file size for some reason
				size := int64(0)
				if err != nil {
					logger.Printf("WARN cannot get file size for file %v due to error %v. Storing size as 0", fileName, err)
				} else {
					size = fileInfo.Size()
				}
				collectedFiles = append(collectedFiles, CollectedFile{
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
	host := conf.Host
	c := conf.Collector
	isCoordinator := conf.IsCoordinator
	logAge := conf.LogAge
	var out string
	var err error

	// Protect against wildcard search base
	if searchDir == "*" {
		return []string{}, FindErr{Cmd: "wildcard search bases rejected"}
	}

	// Only use mtime for logs
	if filter {
		out, err = c.HostExecute(host, isCoordinator, "find", searchDir, "-maxdepth", "3", "-type", "f", "-mtime", fmt.Sprintf("-%v", logAge))
	} else {
		out, err = c.HostExecute(host, isCoordinator, "find", searchDir, "-maxdepth", "3", "-type", "f")
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
