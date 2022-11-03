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
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rsvihladremio/dremio-diagnostic-collector/diagnostics"
)

type HostCaptureConfiguration struct {
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
}

type FindErr struct {
	Cmd string
}

func (fe FindErr) Error() string {
	return fmt.Sprintf("find failed due to error %v:", fe.Cmd)
}

// Capture collects diagnostics, conf files and log files from the target hosts. Failures are permissive and
// are first logged and then returned at the end with the reason for the failure.
func Capture(conf HostCaptureConfiguration) (files []CollectedFile, failedFiles []FailedFiles) {
	host := conf.Host
	dremioConfDir := conf.DremioConfDir
	dremioLogDir := conf.DremioLogDir
	logger := conf.Logger
	logAge := conf.LogAge
	jfrduration := conf.jfrduration

	err := setupDiagDir(conf)
	if err != nil {
		logger.Printf("ERROR: failed to setup diag directory for host %v due to error %v, will not collect diags for this host", host, err)
		return []CollectedFile{}, []FailedFiles{}
	}

	capturedDiagnosticFiles, failedDiagnosticFiles := captureDiagnostics(conf)

	// Trigger a JFR if it is required
	if jfrduration > 0 {
		err := captureJFR(conf)
		if err != nil {
			logger.Printf("ERROR: JFR failed on host %v with error %v", host, err)
		}
	}

	files = append(files, capturedDiagnosticFiles...)
	failedFiles = append(failedFiles, failedDiagnosticFiles...)

	confFiles := []string{}
	foundConfigFiles, err := findFiles(conf, dremioConfDir+"/", false)
	if err != nil {
		logger.Printf("ERROR: host %v unable to find files in directory %v with error %v", host, dremioConfDir, err)
	} else {
		confFiles = append(confFiles, foundConfigFiles...)
	}

	collected, failed := copyFiles(conf, "conf", dremioConfDir, confFiles)
	files = append(files, collected...)
	failedFiles = append(failedFiles, failed...)

	logFiles := []string{}
	var filterLogs bool

	// set flag to filter or not ased on default value
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
		collected, failed := copyFiles(conf, "log", dremioLogDir, logFiles)
		files = append(files, collected...)
		failedFiles = append(failedFiles, failed...)
	}
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
		collected, failed := copyFiles(conf, "log", gcLogDir, gcLogsToCollect)
		files = append(files, collected...)
		failedFiles = append(failedFiles, failed...)
	}

	return files, failedFiles
}

// captureDiagnostics runs iostat on the host, in the future it will run several diagnostics and capture them the same
// time to provide in depth analysis
// iostat must be installed on the host to be captured for this to work
func captureDiagnostics(conf HostCaptureConfiguration) (files []CollectedFile, failedFiles []FailedFiles) {
	host := conf.Host
	c := conf.Collector
	outputLoc := conf.OutputLocation
	isCoordinator := conf.IsCoordinator
	logger := conf.Logger

	// run iostat against the host
	o, err := c.HostExecute(host, isCoordinator, diagnostics.IOStatArgs(conf.DurationDiagnosticTooling)...)
	if err != nil {
		logger.Printf("ERROR: host %v failed iostat with error %v", host, err)
	} else {
		//take the captured text and write it out to iostat.txt
		logger.Printf("INFO: host %v finished iostat", host)
		fileName := filepath.Join(outputLoc, host, "iostat.txt")
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
	jfrUniqId := logdir + "/" + start

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
		logger.Printf("INFO: starting JFR on host %v for %v seconds for pid %v", host, jfrDuration, pid)
		if sudoUser == "" {
			_, err := c.HostExecute(host, isCoordinator, diagnostics.JfrEnable(pid)...)
			if err != nil {
				logger.Printf("ERROR: host %v failed to enable JFR with error %v", host, err)
				return err
			}
			_, err = c.HostExecute(host, isCoordinator, diagnostics.JfrRun(pid, jfrDuration, "dremio", jfrUniqId+".jfr")...)
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
			_, err = c.HostExecute(host, isCoordinator, diagnostics.JfrRunSudo(sudoUser, pid, jfrDuration, "dremio", jfrUniqId+".jfr")...)
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

// setupDiagDir creates all necessary subfolders for the host subfolder in the diag tarball
func setupDiagDir(conf HostCaptureConfiguration) error {
	host := conf.Host
	outputLoc := conf.OutputLocation
	logger := conf.Logger

	if err := os.Mkdir(filepath.Join(outputLoc, host), DirPerms); err != nil {
		logger.Printf("ERROR: host %v had error %v trying to make it's host dir", host, err)
		return err
	}

	if err := os.Mkdir(filepath.Join(outputLoc, host, "conf"), DirPerms); err != nil {
		logger.Printf("ERROR: host %v had error %v trying to make it's host conf dir", host, err)
		return err
	}

	if err := os.MkdirAll(filepath.Join(outputLoc, host, "log/archive"), DirPerms); err != nil {
		logger.Printf("ERROR: host %v had error %v trying to make it's host log dir", host, err)
		return err
	}

	if err := os.MkdirAll(filepath.Join(outputLoc, host, "log/json/archive"), DirPerms); err != nil {
		logger.Printf("ERROR: host %v had error %v trying to make it's host log dir", host, err)
		return err
	}
	return nil
}

// copyFiles copys all files it is asked to copy to a local destination directory
// the directory must be available or it will error out
func copyFiles(conf HostCaptureConfiguration, destDir string, baseDir string, filesToCopy []string) (collectedFiles []CollectedFile, failedFiles []FailedFiles) {
	outputLoc := conf.OutputLocation
	host := conf.Host
	logger := conf.Logger
	c := conf.Collector
	isCoordinator := conf.IsCoordinator

	for i := range filesToCopy {
		log := filesToCopy[i]

		var fileName string
		extraPath := filepath.Dir(strings.TrimPrefix(log, baseDir))
		if extraPath == "" {
			fileName = filepath.Join(outputLoc, host, destDir, filepath.Base(log))
		} else {
			fileName = filepath.Join(outputLoc, host, destDir, extraPath, filepath.Base(log))
		}

		if out, err := c.CopyFromHost(host, isCoordinator, log, fileName); err != nil {
			failedFiles = append(failedFiles, FailedFiles{
				Path: fileName,
				Err:  err,
			})
			logger.Printf("ERROR: unable to copy %v from host %v due to error %v and output was %v", log, host, err, out)
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
			logger.Printf("INFO: host %v copied %v to %v", host, log, fileName)
		}
	}
	return collectedFiles, failedFiles
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
	if err != nil {
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
