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

// cmd package contains all the command line flag and initialization logic for commands
package cmd

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/rsvihladremio/dremio-diagnostic-collector/cmd/local/queriesjson"
	"github.com/rsvihladremio/dremio-diagnostic-collector/cmd/threading"
)

var (
	unableToReadConfigError        error
	supportedExtensions            = []string{"yaml", "json", "toml", "hcl", "env", "props"}
	confFiles                      []string
	configIsFound                  bool
	foundConfig                    string
	outputDir                      string
	logDir                         string
	verbose                        int
	numberThreads                  int
	nodeName                       string
	gcLogsDir                      string
	dremioLogsDir                  string
	dremioEndpoint                 string
	dremioUsername                 string
	dremioPATToken                 string
	dremioStorageType              string
	awsAccessKeyID                 string
	awsSecretAccessKey             string
	awsS3Path                      string
	awsDefaultRegion               string
	azureSASURL                    string
	dremioMasterLogsNumDays        int
	dremioExecutorLogsNumDays      int
	dremioGCFilePattern            string
	dremioRocksDBDir               string
	isKubernetes                   bool
	kubernetesNamespace            string
	skipExportSystemTables         bool
	skipCollectDiskUsage           bool
	skipDownloadJobProfiles        bool
	skipCollectQueriesJSON         bool
	skipCollectKubernetesInfo      bool
	skipCollectDremioConfiguration bool
	skipCollectKVStoreReport       bool
	skipCollectServerLogs          bool
	skipCollectMetaRefreshLog      bool
	skipCollectReflectionLog       bool
	skipCollectAccelerationLog     bool
	skipCollectAccessLog           bool
	skipCollectGCLogs              bool
	skipCollectWLM                 bool
	skipHeapDump                   bool
	skipJFR                        bool
	dremioJFRTimeSeconds           int
	jobProfilesNumDays             int
	jobProfilesNumSlowExec         int
	jobProfilesNumHighQueryCost    int
	jobProfilesNumSlowPlanning     int
	jobProfilesNumRecentErrors     int
	acceptCollectionConsent        bool
)

func configurationOutDir() string {
	return path.Join(outputDir, "configuration", nodeName)
}
func jfrOutDir() string          { return path.Join(outputDir, "jfr") }
func threadDumpsOutDir() string  { return path.Join(outputDir, "jfr", "thread-dumps") }
func heapDumpsOutDir() string    { return path.Join(outputDir, "heap-dumps") }
func jobProfilesOutDir() string  { return path.Join(outputDir, "job-profiles", nodeName) }
func kubernetesOutDir() string   { return path.Join(outputDir, "kubernetes") }
func kvstoreOutDir() string      { return path.Join(outputDir, "kvstore") }
func logsOutDir() string         { return path.Join(outputDir, "logs", nodeName) }
func nodeInfoOutDir() string     { return path.Join(outputDir, "node-info", nodeName) }
func queriesOutDir() string      { return path.Join(outputDir, "queries", nodeName) }
func systemTablesOutDir() string { return path.Join(outputDir, "system-tables") }
func wlmOutDir() string          { return path.Join(outputDir, "wlm") }

type ErrorlessStringBuilder struct {
	builder strings.Builder
}

func (e *ErrorlessStringBuilder) WriteString(s string) {
	if _, err := e.builder.WriteString(s); err != nil {
		log.Fatalf("this should never return an error so this is truly critical: %v", err)
	}
}
func (e *ErrorlessStringBuilder) String() string {
	return e.builder.String()
}

func outputConsent() string {
	builder := ErrorlessStringBuilder{}
	builder.WriteString(`
	Dremio Data Collection Consent Form

	Introduction

	Dremio ("we", "us", "our") requests your consent to collect and use certain data files from your device for the purposes of diagnostics. We take your privacy seriously and will only use these files to improve our services and troubleshoot any issues you may be experiencing. 

	Data Collection and Use

	We would like to collect the following files from your device:
	`)
	if !skipExportSystemTables {
		glog.Info("skipping collection of system tables")
	}
	if !skipCollectDiskUsage {
		glog.Info("skipping collection of disk usage")
	}

	if !skipDownloadJobProfiles {
		glog.Info("skipping collection of job profiles")
	}

	if !skipCollectQueriesJSON {
		glog.Info("skipping collection of queries.json")
	}
	if !skipCollectKubernetesInfo {
		glog.Info("skipping collection of kubernetes configuration")
	}
	if !skipCollectDremioConfiguration {
		glog.Info("skipping collection of dremio configuration")
	}
	if !skipCollectKVStoreReport {
		glog.Info("skipping collection of kv store report")
	}
	if !skipCollectServerLogs {
		glog.Info("skipping collection of metadata server logs")
	}
	if !skipCollectMetaRefreshLog {
		glog.Info("skipping collection of metadata refresh logs")
	}
	if !skipCollectReflectionLog {
		glog.Info("skipping collection of reflection logs")
	}
	if !skipCollectAccelerationLog {
		glog.Info("skipping collection of accerlation logs")
	}
	if !skipCollectAccessLog {
		glog.Info("skipping collection of access logs")
	}
	if !skipCollectGCLogs {
		glog.Info("skipping collection of gc logs")
	}
	if !skipCollectWLM {
		glog.Info("skipping collection of Workload Manager")
	}
	if !skipHeapDump {
		glog.Info("skipping collection of Java Heap Dump")
	}
	if skipJFR {
		glog.Info("skipping collection of JFR")
	}
	builder.WriteString(`
	Please note that the files we collect may contain personal data. We will minimize the collection of personal data wherever possible and will anonymize the data where feasible. 

We will use these files to:

1. Identify and diagnose problems with our products or services that you are using.
2. Improve our products and services.
3. Carry out other purposes that we will disclose to you at the time we collect the files.

Consent

By clicking "I Agree", you grant us permission to access, collect, store, and use the files listed above from your device for the purposes outlined.

Withdrawal of Consent

You have the right to withdraw your consent at any time. If you wish to do so, please contact us at support@dremio.com. Upon receipt of your withdrawal request, we will stop collecting new files and will delete any files we have already collected, unless we are required by law to retain them.

Changes to this Consent Form

We reserve the right to update this consent form from time to time. Any changes will be communicated to you in advance.

By running ddc with the --accept-collection-consent flag, you acknowledge that you have read, understood, and agree to the data collection practices described in this consent form.
	`)
	return builder.String()
}

func createAllDirs() error {
	if err := os.MkdirAll(configurationOutDir(), 0755); err != nil {
		return fmt.Errorf("unable to create configuration directory due to error %v", err)
	}
	if err := os.MkdirAll(jfrOutDir(), 0755); err != nil {
		return fmt.Errorf("unable to create jfr directory due to error %v", err)
	}
	if err := os.MkdirAll(threadDumpsOutDir(), 0755); err != nil {
		return fmt.Errorf("unable to create thread-dumps directory due to error %v", err)
	}
	if err := os.MkdirAll(heapDumpsOutDir(), 0755); err != nil {
		return fmt.Errorf("unable to create heap-dumps directory due to error %v", err)
	}
	if err := os.MkdirAll(jobProfilesOutDir(), 0755); err != nil {
		return fmt.Errorf("unable to create job-profiles directory due to error %v", err)
	}
	if err := os.MkdirAll(kubernetesOutDir(), 0755); err != nil {
		return fmt.Errorf("unable to create kubernetes directory due to error %v", err)
	}
	if err := os.MkdirAll(kvstoreOutDir(), 0755); err != nil {
		return fmt.Errorf("unable to create kvstore directory due to error %v", err)
	}
	if err := os.MkdirAll(logsOutDir(), 0755); err != nil {
		return fmt.Errorf("unable to create logs directory due to error %v", err)
	}
	if err := os.MkdirAll(nodeInfoOutDir(), 0755); err != nil {
		return fmt.Errorf("unable to create node-info directory due to error %v", err)
	}
	if err := os.MkdirAll(queriesOutDir(), 0755); err != nil {
		return fmt.Errorf("unable to create queries directory due to error %v", err)
	}
	if err := os.MkdirAll(systemTablesOutDir(), 0755); err != nil {
		return fmt.Errorf("unable to create system-tables directory due to error %v", err)
	}
	if err := os.MkdirAll(wlmOutDir(), 0755); err != nil {
		return fmt.Errorf("unable to create wlm directory due to error %v", err)
	}
	return nil
}

func collect(numberThreads int) {
	if err := createAllDirs(); err != nil {
		fmt.Printf("unable to create directories due to error %v\n", err)
		os.Exit(1)
	}
	t := threading.NewThreadPool(numberThreads)
	t.FireJob(collectSystemConfig)
	t.FireJob(collectJvmConfig)
	t.FireJob(collectDremioConfig)
	t.FireJob(collectDiskUsage)
	t.FireJob(collectNodeMetrics)
	t.FireJob(collectJfr)
	t.FireJob(collectJstacks)
	t.FireJob(collectKvReport)
	t.FireJob(collectWlm)
	t.FireJob(collectK8sConfig)
	t.FireJob(collectHeapDump)
	t.FireJob(collectDremioSystemTables)
	t.FireJob(collectQueriesJSON)
	t.FireJob(collectJobProfiles)
	t.FireJob(collectDremioServerLog)
	t.FireJob(collectGcLogs)
	t.FireJob(collectMetadataRefreshLog)
	t.FireJob(collectReflectionLog)
	t.FireJob(collectAccelerationLog)
	t.FireJob(collectDremioAccessLog)
	t.Wait()
}

func collectDremioConfig() error {
	glog.Info("The following alias was defined for running shell commands - $(type shell)")

	glog.Info("Collecting OS Information from $BASENAME ...")
	osInfoFile := path.Join(outputDir, "node-info", nodeName, "os_info.txt")
	w, err := os.Create(osInfoFile)
	if err != nil {
		return fmt.Errorf("unable to create file %v due to error %v", osInfoFile, err)
	}
	defer func() {
		if err := w.Sync(); err != nil {
			glog.Warningf("unable to sync the os_info.txt file due to error: %v", err)
		}
		if err := w.Close(); err != nil {
			glog.Warningf("unable to close the os_info.txt file due to error: %v", err)
		}
	}()

	glog.V(2).Info("/etc/*-release")

	_, err = w.Write([]byte("___\n>>> cat /etc/*-release\n"))
	if err != nil {
		return fmt.Errorf("unable to write release file header for os_info.txt due to error %v", err)
	}

	err = Shell(w, "cat /etc/*-release")
	if err != nil {
		return fmt.Errorf("unable to write release files for os_info.txt due to error %v", err)
	}

	_, err = w.Write([]byte("___\n>>> uname -r\n"))
	if err != nil {
		return fmt.Errorf("unable to write uname header for os_info.txt due to error %v", err)
	}

	err = Shell(w, "uname -r")
	if err != nil {
		return fmt.Errorf("unable to write uname -r for os_info.txt due to error %v", err)
	}
	_, err = w.Write([]byte("___\n>>> lsb_release -a\n"))
	if err != nil {
		return fmt.Errorf("unable to write lsb_release -r header for os_info.txt due to error %v", err)
	}
	err = Shell(w, "lsb_release -a")
	if err != nil {
		return fmt.Errorf("unable to write lsb_release -a for os_info.txt due to error %v", err)
	}
	_, err = w.Write([]byte("___\n>>> hostnamectl\n"))
	if err != nil {
		return fmt.Errorf("unable to write hostnamectl for os_info.txt due to error %v", err)
	}
	err = Shell(w, "hostnamectl")
	if err != nil {
		return fmt.Errorf("unable to write hostnamectl for os_info.txt due to error %v", err)
	}
	_, err = w.Write([]byte("___\n>>> cat /proc/meminfo\n"))
	if err != nil {
		return fmt.Errorf("unable to write /proc/meminfo header for os_info.txt due to error %v", err)
	}
	err = Shell(w, "cat /proc/meminfo")
	if err != nil {
		return fmt.Errorf("unable to write /proc/meminfo for os_info.txt due to error %v", err)
	}
	_, err = w.Write([]byte("___\n>>> lscpu\n"))
	if err != nil {
		return fmt.Errorf("unable to write lscpu header for os_info.txt due to error %v", err)
	}
	err = Shell(w, "lscpu")
	if err != nil {
		return fmt.Errorf("unable to write lscpu for os_info.txt due to error %v", err)
	}

	glog.Infof("... Collecting OS Information from %v COMPLETED", nodeName)

	glog.Infof("Collecting Configuration Information from %v ...", nodeName)
	//mkdir -p $DREMIO_HEALTHCHECK_EXPORT_DIR/configuration/$BASENAME

	glog.Warning("You may have to run the following command 'jcmd 1 VM.flags' as 'sudo' and specify '-u dremio' when running on Dremio AWSE or VM deployments")
	jvmSettingsFile := path.Join(outputDir, "node-info", nodeName, "jvm_settings.txt")
	jvmSettingsFileWriter, err := os.Create(jvmSettingsFile)
	if err != nil {
		return fmt.Errorf("unable to create file %v due to error %v", jvmSettingsFile, err)
	}
	defer func() {
		if err := jvmSettingsFileWriter.Sync(); err != nil {
			glog.Warningf("unable to sync the os_info.txt file due to error: %v", err)
		}
		if err := jvmSettingsFileWriter.Close(); err != nil {
			glog.Warningf("unable to close the os_info.txt file due to error: %v", err)
		}
	}()
	var dremioPIDOutput bytes.Buffer
	if err := Shell(&dremioPIDOutput, "bash -c \"ps ax | grep dremio | grep -v grep | awk '{print $1}'"); err != nil {
		glog.Warningf("Error trying to unlock commercial features %v. Note: newer versions of OpenJDK do not support the call VM.unlock_commercial_features. This is usually safe to ignore", err)
	}
	dremioPID, err := strconv.Atoi(dremioPIDOutput.String())
	if err != nil {
		return fmt.Errorf("unable to parse dremio PID due to error %v", err)
	}
	err = Shell(jvmSettingsFileWriter, fmt.Sprintf("jcmd %v VM.flags", dremioPID))
	if err != nil {
		return fmt.Errorf("unable to write jvm_settings.txt file due to error %v", err)
	}
	err = copyFile("/opt/dremio/conf/dremio.conf", filepath.Join(outputDir, "configuration", nodeName, "dremio.conf"))
	if err != nil {
		return fmt.Errorf("unable to copy dremio.conf due to error %v", err)
	}
	err = copyFile("/opt/dremio/conf/dremio-env", filepath.Join(outputDir, "configuration", nodeName, "dremio.env"))
	if err != nil {
		return fmt.Errorf("unable to copy dremio.env due to error %v", err)
	}
	err = copyFile("/opt/dremio/conf/logback.xml", filepath.Join(outputDir, "configuration", nodeName, "logback.xml"))
	if err != nil {
		return fmt.Errorf("unable to copy logback.xml due to error %v", err)
	}
	err = copyFile("/opt/dremio/conf/logback-access.xml", filepath.Join(outputDir, "configuration", nodeName, "logback-access.xml"))
	if err != nil {
		return fmt.Errorf("unable to copy logback-access.xml due to error %v", err)
	}
	//# shell "cat /opt/dremio/conf/core-site.xml" > $DREMIO_HEALTHCHECK_EXPORT_DIR/configuration/$BASENAME/core-site.xml

	//python3 $DREMIO_HEALTHCHECK_SCRIPT_DIR/helper/secrets_cleanser_config.py $DREMIO_HEALTHCHECK_EXPORT_DIR/configuration/$BASENAME/dremio.conf

	glog.Infof("... Collecting Configuration Information from %v COMPLETED", nodeName)

	if skipCollectDiskUsage {
		glog.Infof("Skipping Collect Disk Usage from %v ...", nodeName)
	} else {
		glog.Infof("Collecting Disk Usage from %v ...", nodeName)
		diskWriter, err := os.Create(filepath.Join(outputDir, "node-info", nodeName, "diskusage.txt"))
		if err != nil {
			return fmt.Errorf("unable to create diskusage.txt due to error %v", err)
		}
		defer func() {
			if err := diskWriter.Sync(); err != nil {
				glog.Warningf("unable to sync the os_info.txt file due to error: %v", err)
			}
			if err := diskWriter.Close(); err != nil {
				glog.Warningf("unable to close the os_info.txt file due to error: %v", err)
			}
		}()
		err = Shell(diskWriter, "df -h")
		if err != nil {
			return fmt.Errorf("unable to read df -h due to error %v", err)
		}

		if strings.Contains(nodeName, "dremio-master") {
			rocksDbDiskUsageWriter, err := os.Create(filepath.Join(outputDir, "node-info", nodeName, "rocksdb_disk_allocation.txt"))
			if err != nil {
				return fmt.Errorf("unable to create rocksdb_disk_allocation.txt due to error %v", err)
			}
			defer rocksDbDiskUsageWriter.Close()
			err = Shell(rocksDbDiskUsageWriter, "du -sh /opt/dremio/data/db/*")
			if err != nil {
				return fmt.Errorf("unable to write du -sh to rocksdb_disk_allocation.txt due to error %v", err)
			}

		}
		glog.Infof("... Collecting Disk Usage from %v COMPLETED", nodeName)
	}

	return nil
}

func collectJvmConfig() error {
	gcMatchFunc := func(filename string) bool {
		return strings.HasPrefix(filename, "gc") && strings.HasSuffix(filename, ".log")
	}
	files, err := findMatchingFiles(gcLogsDir, gcMatchFunc)
	if err != nil {
		return fmt.Errorf("unable to search for gc logs in directory %v due to error %v", gcLogsDir, err)
	}
	for _, file := range files {
		if err := copyFile(file, logsOutDir()); err != nil {
			return fmt.Errorf("unable to copy gclog %v due to error %v", file, err)
		}
	}
	return nil
}

func findMatchingFiles(dirPath string, matchFunc func(filename string) bool) ([]string, error) {
	matchingFiles := []string{}

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("Error accessing path %q: %v\n", path, err)
			return err
		}

		// Check if the current file matches the provided criteria
		if !info.IsDir() && matchFunc(info.Name()) {
			matchingFiles = append(matchingFiles, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return matchingFiles, nil
}

func copyFile(srcPath, dstPath string) error {
	// Open the source file
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Create the destination file
	dstFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// Copy the contents of the source file to the destination file
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	// Flush the written data to disk
	err = dstFile.Sync()
	if err != nil {
		return err
	}

	return nil
}

func collectNodeMetrics() error {
	nodeMetricsFile := path.Join(outputDir, "node-info", nodeName, "metrics.txt")
	w, err := os.Create(nodeMetricsFile)
	if err != nil {
		return fmt.Errorf("unable to create file %v due to error %v", nodeMetricsFile, err)
	}
	iterations := 60
	interval := time.Second

	header := fmt.Sprintf("Time\t\tUser %%\t\tSystem %%\t\tIdle %%\t\tNice %%\t\tIOwait %%\t\tIRQ %%\t\tSteal %%\t\tGuest %%\t\tGuest Nice %%\t\tQueue Depth\tDisk Latency (ms)\tDisk Read (KB/s)\tDisk Write (KB/s)\tFree Mem (MB)\tCached Mem (MB)\n")
	_, err = w.Write([]byte(header))
	if err != nil {
		return fmt.Errorf("unable to write output string %v due to %v", header, err)
	}
	prevDiskIO, _ := disk.IOCounters()
	for i := 0; i < iterations; i++ {
		// Sleep
		if i > 0 {
			time.Sleep(interval)
		}

		// CPU Times
		cpuTimes, _ := cpu.Times(false)
		total := getTotalTime(cpuTimes[0])
		userPercent := (cpuTimes[0].User / total) * 100
		systemPercent := (cpuTimes[0].System / total) * 100
		idlePercent := (cpuTimes[0].Idle / total) * 100
		nicePercent := (cpuTimes[0].Nice / total) * 100
		iowaitPercent := (cpuTimes[0].Iowait / total) * 100
		irqPercent := (cpuTimes[0].Irq / total) * 100
		softIrqPercent := (cpuTimes[0].Softirq / total) * 100
		stealPercent := (cpuTimes[0].Steal / total) * 100
		guestPercent := (cpuTimes[0].Guest / total) * 100
		guestNicePercent := (cpuTimes[0].GuestNice / total) * 100

		// Memory
		memoryInfo, _ := mem.VirtualMemory()

		// Disk I/O
		diskIO, _ := disk.IOCounters()
		var weightedIOTime, totalIOs uint64
		var readBytes, writeBytes float64
		for _, io := range diskIO {
			weightedIOTime += io.WeightedIO
			totalIOs += io.IoTime

			if prev, ok := prevDiskIO[io.Name]; ok {
				readBytes += float64(io.ReadBytes-prev.ReadBytes) / 1024
				writeBytes += float64(io.WriteBytes-prev.WriteBytes) / 1024
			}
		}
		prevDiskIO = diskIO

		queueDepth := float64(weightedIOTime) / 1000
		diskLatency := float64(weightedIOTime) / float64(totalIOs)

		// Output
		row := fmt.Sprintf("%s\t%.2f%%\t\t%.2f%%\t\t%.2f%%\t\t%.2f%%\t\t%.2f%%\t\t%.2f%%\t\t%.2f%%\t\t%.2f%%\t\t%.2f%%\t\t%.2f%%\t\t%.2f\t\t%.2f\t\t%.2f\t\t%.2f\t\t%.2f\t\t%.2f\n",
			time.Now().Format("15:04:05"), userPercent, systemPercent, idlePercent, nicePercent, iowaitPercent, irqPercent, softIrqPercent, stealPercent, guestPercent, guestNicePercent, queueDepth, diskLatency, readBytes, writeBytes, float64(memoryInfo.Free)/(1024*1024), float64(memoryInfo.Cached)/(1024*1024))
		_, err := w.Write([]byte(row))
		if err != nil {
			return fmt.Errorf("unable to write output string %v due to %v", row, err)
		}
	}
	return nil
}

func getTotalTime(c cpu.TimesStat) float64 {
	return c.User + c.System + c.Idle + c.Nice + c.Iowait + c.Irq +
		c.Softirq + c.Steal + c.Guest + c.GuestNice
}

func collectJfr() error {
	if !skipJFR {
		var dremioPIDOutput bytes.Buffer
		if err := Shell(&dremioPIDOutput, "bash -c \"ps ax | grep dremio | grep -v grep | awk '{print $1}'"); err != nil {
			glog.Warningf("Error trying to unlock commercial features %v. Note: newer versions of OpenJDK do not support the call VM.unlock_commercial_features. This is usually safe to ignore", err)
		}
		dremioPID, err := strconv.Atoi(dremioPIDOutput.String())
		if err != nil {
			return fmt.Errorf("unable to parse dremio PID due to error %v", err)
		}

		var w bytes.Buffer
		if err := Shell(&w, fmt.Sprintf("jcmd %v VM.unlock_commercial_features", dremioPID)); err != nil {
			glog.Warningf("Error trying to unlock commercial features %v. Note: newer versions of OpenJDK do not support the call VM.unlock_commercial_features. This is usually safe to ignore", err)
		}
		glog.V(2).Infof("node: %v - jfr unlock commerictial output - %v", nodeName, w.String())
		w = bytes.Buffer{}
		if err := Shell(&w, fmt.Sprintf("jcmd %v JFR.start name=\"DREMIO_JFR\" settings=profile maxage=%vs  filename=%v/%v.jfr dumponexit=true", dremioPID, dremioJFRTimeSeconds, jfrOutDir(), nodeName)); err != nil {
			return fmt.Errorf("unable to run JFR due to error %v", err)
		}
		glog.V(2).Infof("node: %v - jfr start output - %v", nodeName, w.String())
		time.Sleep(time.Duration(dremioJFRTimeSeconds) * time.Second)
		// do not "optimize". the recording first needs to be stopped for all processes before collecting the data.
		glog.Info("... stopping JFR $BASEPOD")
		w = bytes.Buffer{}
		if err := Shell(&w, fmt.Sprintf("jcmd %v JFR.dump name=\"DREMIO_JFR\"", dremioPID)); err != nil {
			return fmt.Errorf("unable to dump JFR due to error %v", err)
		}
		glog.V(2).Infof("node: %v - jfr dump output %v", nodeName, w.String())
		w = bytes.Buffer{}
		if err := Shell(&w, fmt.Sprintf("jcmd %v JFR.stop name=\"DREMIO_JFR\"", dremioPID)); err != nil {
			return fmt.Errorf("unable to dump JFR due to error %v", err)
		}
		glog.V(2).Infof("node: %v - jfr stop output %v", nodeName, w.String())
		w = bytes.Buffer{}
		if err := Shell(&w, fmt.Sprintf("rm -f %v/%v.jfr", jfrOutDir(), nodeName)); err != nil {
			return fmt.Errorf("unable to dump JFR due to error %v", err)
		}
	}
	return nil
}

func collectJstacks() error {
	return nil
}

func collectKvReport() error {
	err := validateAPICredentials()
	if err != nil {
		return err
	}
	filename := "kvstore-report.zip"
	apipath := "/apiv2/kvstore/report"
	url := dremioEndpoint + apipath
	headers := map[string]string{"Accept": "application/octet-stream"}
	body, err := apiRequest(url, dremioPATToken, "GET", headers)
	if err != nil {
		return fmt.Errorf("unable to retrieve KV store report from %s due to error %v", url, err)
	}
	sb := string(body)
	kvStoreReportFile := path.Join(kvstoreOutDir(), filename)
	file, err := os.Create(kvStoreReportFile)
	if err != nil {
		return fmt.Errorf("unable to create file %s due to error %v", filename, err)
	}
	defer errCheck(file.Close)
	_, err = fmt.Fprint(file, sb)
	if err != nil {
		return fmt.Errorf("unable to create file %s due to error %v", filename, err)
	}
	log.Println("SUCCESS - Created " + filename)
	return nil
}

func collectWlm() error {
	err := validateAPICredentials()
	if err != nil {
		return err
	}
	apiobjects := [][]string{
		{"/api/v3/wlm/queue", "queues.json"},
		{"/api/v3/wlm/rule", "rules.json"},
	}
	for _, apiobject := range apiobjects {
		apipath := apiobject[0]
		filename := apiobject[1]
		url := dremioEndpoint + apipath
		headers := map[string]string{"Content-Type": "application/json"}
		body, err := apiRequest(url, dremioPATToken, "GET", headers)
		if err != nil {
			return fmt.Errorf("unable to retrieve WLM from %s due to error %v", url, err)
		}
		sb := string(body)
		wlmFile := path.Join(wlmOutDir(), filename)
		file, err := os.Create(wlmFile)
		if err != nil {
			return fmt.Errorf("unable to create file %s due to error %v", filename, err)
		}
		defer errCheck(file.Close)
		_, err = fmt.Fprint(file, sb)
		if err != nil {
			return fmt.Errorf("unable to create file %s due to error %v", filename, err)
		}
		log.Println("SUCCESS - Created " + filename)
	}
	return nil
}

func collectHeapDump() error {
	return nil
}

func collectQueriesJSON() error {
	return nil
}

func collectJobProfiles() error {
	err := validateAPICredentials()
	if err != nil {
		return err
	}
	files, err := os.ReadDir(queriesOutDir())
	if err != nil {
		return err
	}
	queriesjsons := []string{}
	for _, file := range files {
		queriesjsons = append(queriesjsons, path.Join(queriesOutDir(), file.Name()))
	}

	queriesrows := queriesjson.CollectQueriesJSON(queriesjsons)
	profilesToCollect := map[string]string{}

	slowplanqueriesrows := queriesjson.GetSlowPlanningJobs(queriesrows, jobProfilesNumSlowPlanning)
	queriesjson.AddRowsToSet(slowplanqueriesrows, profilesToCollect)

	slowexecqueriesrows := queriesjson.GetSlowExecJobs(queriesrows, jobProfilesNumSlowExec)
	queriesjson.AddRowsToSet(slowexecqueriesrows, profilesToCollect)

	highcostqueriesrows := queriesjson.GetHighCostJobs(queriesrows, jobProfilesNumHighQueryCost)
	queriesjson.AddRowsToSet(highcostqueriesrows, profilesToCollect)

	errorqueriesrows := queriesjson.GetRecentErrorJobs(queriesrows, jobProfilesNumRecentErrors)
	queriesjson.AddRowsToSet(errorqueriesrows, profilesToCollect)

	log.Println("jobProfilesNumSlowPlanning:", jobProfilesNumSlowPlanning)
	log.Println("jobProfilesNumSlowExec:", jobProfilesNumSlowExec)
	log.Println("jobProfilesNumHighQueryCost:", jobProfilesNumHighQueryCost)
	log.Println("jobProfilesNumRecentErrors:", jobProfilesNumRecentErrors)

	log.Println("Downloading", len(profilesToCollect), "job profiles...")
	for key := range profilesToCollect {
		err := downloadJobProfile(key)
		if err != nil {
			log.Println(err) // Print instead of Error
		}
	}
	log.Println("Finished downloading", len(profilesToCollect), "job profiles")

	return nil
}

func downloadJobProfile(jobid string) error {
	apipath := "/apiv2/support/" + jobid + "/download"
	filename := jobid + ".zip"
	url := dremioEndpoint + apipath
	headers := map[string]string{"Accept": "application/octet-stream"}
	body, err := apiRequest(url, dremioPATToken, "POST", headers)
	if err != nil {
		return err
	}
	sb := string(body)
	jobProfileFile := path.Join(jobProfilesOutDir(), filename)
	file, err := os.Create(jobProfileFile)
	if err != nil {
		return fmt.Errorf("unable to create file %s due to error %v", filename, err)
	}
	defer errCheck(file.Close)
	_, err = fmt.Fprint(file, sb)
	if err != nil {
		return fmt.Errorf("unable to create file %s due to error %v", filename, err)
	}
	return nil
}

// maskPasswordsInYAML searches through all text YAML and replaces the values of all keys case-insensitively named `*password*`
func maskPasswordsInYAML(yamlText string) string {
	return yamlText
}

// maskPasswordsInJSON searches through all text JSON and replaces the values of all keys case-insensitively named `*password*`
func maskPasswordsInJSON(jsonText string) string {
	return jsonText
}

func collectK8sConfig() error {
	//foreach yaml file
	//maskPasswordKeysYAML
	return nil
}

func collectDremioSystemTables() error {
	err := validateAPICredentials()
	if err != nil {
		return err
	}
	// TODO: Row limit and sleem MS need to be configured
	rowlimit := 100000
	sleepms := 100

	systemtables := [...]string{
		"\\\"tables\\\"",
		"boot",
		"fragments",
		"jobs",
		"materializations",
		"membership",
		"memory",
		"nodes",
		"options",
		"privileges",
		"reflection_dependencies",
		"reflections",
		"refreshes",
		"roles",
		"services",
		"slicing_threads",
		"table_statistics",
		"threads",
		"version",
		"views",
		"cache.datasets",
		"cache.mount_points",
		"cache.objects",
		"cache.storage_plugins",
	}

	for _, systable := range systemtables {
		filename := "sys." + systable + ".json"
		body, err := downloadSysTable(systable, rowlimit, sleepms)
		if err != nil {
			return err
		}
		dat := make(map[string]interface{})
		err = json.Unmarshal(body, &dat)
		if err != nil {
			return fmt.Errorf("unable to unmarshall JSON response - %w", err)
		}
		if err == nil {
			rowcount := dat["returnedRowCount"].(float64)
			if int(rowcount) == rowlimit {
				log.Println("WARNING: Returned row count for sys." + systable + " has been limited to " + strconv.Itoa(rowlimit))
			}
		}
		sb := string(body)
		systemTableFile := path.Join(systemTablesOutDir(), filename)
		file, err := os.Create(systemTableFile)
		if err != nil {
			return fmt.Errorf("unable to create file %v due to error %v", filename, err)
		}
		defer errCheck(file.Close)
		_, err = fmt.Fprint(file, sb)
		if err != nil {
			return fmt.Errorf("unable to create file %s due to error %v", filename, err)
		}
		log.Println("SUCCESS - Created " + filename)
	}
	return nil
}

func downloadSysTable(systable string, rowlimit int, sleepms int) ([]byte, error) {
	// TODO: Consider using official api/v3, requires paging of job results
	headers := map[string]string{"Content-Type": "application/json"}
	sqlurl := dremioEndpoint + "/api/v3/sql"
	joburl := dremioEndpoint + "/api/v3/job/"
	jobid, err := postQuery(sqlurl, dremioPATToken, headers, systable)
	if err != nil {
		return nil, err
	}
	jobstateurl := joburl + jobid
	jobstate := "RUNNING"
	for jobstate == "RUNNING" {
		time.Sleep(time.Duration(sleepms) * time.Millisecond)
		body, err := apiRequest(jobstateurl, dremioPATToken, "GET", headers)
		if err != nil {
			return nil, fmt.Errorf("unable to retrieve job state from %s due to error %v", jobstateurl, err)
		}
		dat := make(map[string]interface{})
		err = json.Unmarshal(body, &dat)
		if err != nil {
			return nil, fmt.Errorf("unable to unmarshall JSON response - %w", err)
		}
		jobstate = dat["jobState"].(string)
	}
	if jobstate == "COMPLETED" {
		jobresultsurl := dremioEndpoint + "/apiv2/job/" + jobid + "/data?offset=0&limit=" + strconv.Itoa(rowlimit)
		log.Println("Retrieving job results ...")
		body, err := apiRequest(jobresultsurl, dremioPATToken, "GET", headers)
		if err != nil {
			return nil, fmt.Errorf("unable to retrieve job results from %s due to error %v", jobresultsurl, err)
		}
		return body, nil
	}
	return nil, fmt.Errorf("unable to retrieve job results for sys." + systable)
}

func collectDremioServerLog() error {
	return nil
}

func collectGcLogs() error {
	return nil
}

func collectMetadataRefreshLog() error {
	return nil
}

func collectReflectionLog() error {
	return nil
}

func collectDremioAccessLog() error {
	return nil
}

func collectSystemConfig() error {
	return nil
}

func collectAccelerationLog() error {
	return nil
}

func collectDiskUsage() error {
	return nil
}

var localCollectCmd = &cobra.Command{
	Use:   "local-collect",
	Short: "retrieves all the dremio logs and diagnostics for the local node and saves the results in a compatible format for Dremio support",
	Long:  `Retrieves all the dremio logs and diagnostics for the local node and saves the results in a compatible format for Dremio support. This subcommand needs to be run with enough permissions to read the /proc filesystem, the dremio logs and configuration files`,
	Run: func(cmd *cobra.Command, args []string) {
		glog.Info("searching for the following optional configuration files in the current directory %v", strings.Join(confFiles, ", "))
		if !configIsFound {
			glog.Warningf("unable to read any of the valid config file formats (%v) due to error '%v' - falling back to defaults, command line flags and environment variables", strings.Join(supportedExtensions, ","), unableToReadConfigError)
		} else {
			glog.Infof("INFO: found config file %v", foundConfig)
		}
		if acceptCollectionConsent {
			fmt.Println(outputConsent())
			os.Exit(1)
		}
		//check if required flags are set
		requiredFlags := []string{"dremio-endpoint", "dremio-username", "dremio-pat-token", "dremio-storage-type"}

		failed := false
		for _, flag := range requiredFlags {
			if viper.GetString(flag) == "" {
				log.Printf("Error: required flag '--%s' not set", flag)
				failed = true
			}
		}
		if failed {
			err := cmd.Usage()
			if err != nil {
				log.Fatalf("unable to even print usage, this is critical report this bug %v", err)
			}
			os.Exit(1)
		}

		// Run application
		defer glog.Flush()
		glog.Info("Starting collection...")
		collect(numberThreads)
	},
}

func getThreads(cpus int) int {
	numCPU := math.Round(float64(cpus / 2.0))
	return int(math.Max(numCPU, 2))
}

func getOutputDir(now time.Time) string {
	nowStr := now.Format("20060102-150405")
	return filepath.Join(os.TempDir(), "ddc", nowStr)
}

// Shell executes a shell command with shell expansion and appends its output to the provided io.Writer.
func Shell(writer io.Writer, commandLine string) error {
	cmd := exec.Command("bash", "-c", commandLine)
	cmd.Stdout = writer
	cmd.Stderr = writer

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("command execution failed: %w", err)
	}

	return nil
}

func init() {
	// command line flags

	//make default tmp directory
	now := time.Now()
	tmpDir := getOutputDir(now)
	localCollectCmd.Flags().StringVarP(&outputDir, "tmp-output-dir", "o", tmpDir, "temporary output directory for log collection")
	if err := viper.BindPFlag("tmp-output-dir", localCollectCmd.Flags().Lookup("tmp-output-dir")); err != nil {
		log.Fatalf("unable to bind configuration for tmp-output-dir due to error: %v", err)
	}

	// set node name
	hostName, err := os.Hostname()
	if err != nil {
		hostName = fmt.Sprintf("unknown-%v", uuid.New())
	}
	localCollectCmd.Flags().StringVarP(&nodeName, "node-name", "n", hostName, "name to give to the node")
	if err := viper.BindPFlag("node-name", localCollectCmd.Flags().Lookup("node-name")); err != nil {
		log.Fatalf("unable to bind configuration for node-name due to error: %v", err)
	}

	localCollectCmd.Flags().StringVar(&logDir, "collect-log-dir", "", "logging output directory for the collector")
	if err := viper.BindPFlag("collect-log-dir", localCollectCmd.Flags().Lookup("collect-log-dir")); err != nil {
		log.Fatalf("unable to bind configuration for collect-log-dir to error: %v", err)
	}

	localCollectCmd.Flags().StringVar(&gcLogsDir, "dremio-gclogs-dir", "/var/log/dremio", "directory with gc logs on dremio")
	if err := viper.BindPFlag("dremio-gclogs-dir", localCollectCmd.Flags().Lookup("dremio-gclogs-dir")); err != nil {
		log.Fatalf("unable to bind configuration for dremio-gclogs-dir to error: %v", err)
	}

	localCollectCmd.Flags().StringVar(&dremioLogsDir, "dremio-log-dir", "/var/log/dremio", "directory with application logs on dremio")
	if err := viper.BindPFlag("dremio-log-dir", localCollectCmd.Flags().Lookup("dremio-log-dir")); err != nil {
		log.Fatalf("unable to bind configuration for dremio-log-dir to error: %v", err)
	}

	localCollectCmd.Flags().CountVarP(&verbose, "verbose", "v", "Logging verbosity")
	if err := viper.BindPFlag("verbose", localCollectCmd.Flags().Lookup("verbose")); err != nil {
		log.Fatalf("unable to bind configuration for verbose to error: %v", err)
	}

	defaultThreads := getThreads(runtime.NumCPU())
	localCollectCmd.Flags().IntVarP(&numberThreads, "number-threads", "t", defaultThreads, "control concurrency in the system")
	if err := viper.BindPFlag("number-threads", localCollectCmd.Flags().Lookup("number-threads")); err != nil {
		log.Fatalf("unable to bind configuration for number-threads to error: %v", err)
	}

	// Add flags for Dremio connection information
	localCollectCmd.Flags().StringVar(&dremioEndpoint, "dremio-endpoint", "http://localhost:9047", "Dremio REST API endpoint")
	if err := viper.BindPFlag("dremio-endpoint", localCollectCmd.Flags().Lookup("dremio-endpoint")); err != nil {
		log.Fatalf("unable to bind configuration for dremio-endpoint to error: %v", err)
	}
	localCollectCmd.Flags().StringVar(&dremioUsername, "dremio-username", "<DREMIO_ADMIN_USER>", "Dremio username")
	if err := viper.BindPFlag("dremio-username", localCollectCmd.Flags().Lookup("dremio-username")); err != nil {
		log.Fatalf("unable to bind configuration for dremio-username to error: %v", err)
	}

	localCollectCmd.Flags().StringVar(&dremioPATToken, "dremio-pat-token", "<DREMIO_PAT>", "Dremio Personal Access Token (PAT)")
	if err := viper.BindPFlag("dremio-pat-token", localCollectCmd.Flags().Lookup("dremio-pat-token")); err != nil {
		log.Fatalf("unable to bind configuration for dremio-pat-token to error: %v", err)
	}

	localCollectCmd.Flags().StringVar(&dremioStorageType, "dremio-storage-type", "adls", "Dremio storage type (adls, s3, azure, or hdfs)")
	if err := viper.BindPFlag("dremio-storage-type", localCollectCmd.Flags().Lookup("dremio-storage-type")); err != nil {
		log.Fatalf("unable to bind configuration for dremio-storage-type to error: %v", err)
	}

	// Add flags for AWS information
	localCollectCmd.Flags().StringVar(&awsAccessKeyID, "aws-access-key-id", "NOTSET", "AWS Access Key ID")
	if err := viper.BindPFlag("aws-access-key-id", localCollectCmd.Flags().Lookup("aws-access-key-id")); err != nil {
		log.Fatalf("unable to bind configuration for aws-access-key-id to error: %v", err)
	}

	localCollectCmd.Flags().StringVar(&awsSecretAccessKey, "aws-secret-access-key", "NOTSET", "AWS Secret Access Key")
	if err := viper.BindPFlag("aws-secret-access-key", localCollectCmd.Flags().Lookup("aws-secret-access-key")); err != nil {
		log.Fatalf("unable to bind configuration for aws-access-access-key to error: %v", err)
	}

	localCollectCmd.Flags().StringVar(&awsS3Path, "aws-s3-path", "NOTSET", "S3 path for Dremio data")
	if err := viper.BindPFlag("aws-s3-path", localCollectCmd.Flags().Lookup("aws-s3-path")); err != nil {
		log.Fatalf("unable to bind configuration for aws-s3-path to error: %v", err)
	}

	localCollectCmd.Flags().StringVar(&awsDefaultRegion, "aws-default-region", "us-west-1", "Default region for AWS")
	if err := viper.BindPFlag("aws-default-region", localCollectCmd.Flags().Lookup("aws-default-region")); err != nil {
		log.Fatalf("unable to bind configuration for aws-default-region to error: %v", err)
	}

	// Add flags for Azure information
	localCollectCmd.Flags().StringVar(&azureSASURL, "azure-sas-url", "<AZURE_SAS_URL>", "Azure SAS URL for Dremio data")
	if err := viper.BindPFlag("azure-sas-url", localCollectCmd.Flags().Lookup("azure-sas-url")); err != nil {
		log.Fatalf("unable to bind configuration for azure-sas-url to error: %v", err)
	}

	// Add flags for Dremio diagnostic collection options

	localCollectCmd.Flags().IntVar(&dremioMasterLogsNumDays, "dremio-master-logs-num-days", 3, "Number of days of Dremio master logs to collect for the Master Logs collector")
	if err := viper.BindPFlag("dremio-master-logs-num-days", localCollectCmd.Flags().Lookup("dremio-master-logs-num-days")); err != nil {
		log.Fatalf("unable to bind configuration for dremio-master-logs-num-days to error: %v", err)
	}

	localCollectCmd.Flags().IntVar(&dremioExecutorLogsNumDays, "dremio-executor-logs-num-days", 3, "Number of days of Dremio executor logs to collect for the Executor Logs collector")
	if err := viper.BindPFlag("dremio-executor-logs-num-days", localCollectCmd.Flags().Lookup("dremio-executor-logs-num-days")); err != nil {
		log.Fatalf("unable to bind configuration for dremio-master-logs-num-days to error: %v", err)
	}

	//localCollectCmd.Flags().StringVar(&dremioLogDir, "dremio-log-dir", "/opt/dremio/data/log", "Path to Dremio log directory")
	localCollectCmd.Flags().StringVar(&dremioGCFilePattern, "dremio-gc-file-pattern", "gc*.log", "File pattern to match for Dremio GC logs")
	if err := viper.BindPFlag("dremio-gc-file-pattern", localCollectCmd.Flags().Lookup("dremio-gc-file-pattern")); err != nil {
		log.Fatalf("unable to bind configuration for dremio-gc-file-pattern to error: %v", err)
	}

	localCollectCmd.Flags().StringVar(&dremioRocksDBDir, "dremio-rocksdb-dir", "/opt/dremio/data/db", "Path to Dremio RocksDB directory")
	if err := viper.BindPFlag("dremio-rocksdb-dir", localCollectCmd.Flags().Lookup("dremio-rocksdb-dir")); err != nil {
		glog.Fatalf("unable to bind flag due to error %v", err)
	}

	// Add flags for Kubernetes information
	localCollectCmd.Flags().BoolVar(&isKubernetes, "is-kubernetes", false, "Set to true if running in a Kubernetes environment")
	if err := viper.BindPFlag("is-kubernetes", localCollectCmd.Flags().Lookup("is-kubernetes")); err != nil {
		glog.Fatalf("unable to bind flag due to error %v", err)
	}

	localCollectCmd.Flags().StringVar(&kubernetesNamespace, "kubernetes-namespace", "default", "Kubernetes namespace")
	if err := viper.BindPFlag("kubernetes-namespace", localCollectCmd.Flags().Lookup("kubernetes-namespace")); err != nil {
		glog.Fatalf("unable to bind flag due to error %v", err)
	}

	// Add flags for skipping collectors

	localCollectCmd.Flags().BoolVar(&skipExportSystemTables, "skip-export-system-tables", false, "Skip the Export System Tables collector")
	if err := viper.BindPFlag("skip-export-system-tables", localCollectCmd.Flags().Lookup("skip-export-system-tables")); err != nil {
		glog.Fatalf("unable to bind flag due to error %v", err)
	}

	localCollectCmd.Flags().BoolVar(&skipCollectDiskUsage, "skip-collect-disk-usage", false, "Skip the Collect Disk Usage collector")
	if err := viper.BindPFlag("skip-collect-disk-usage", localCollectCmd.Flags().Lookup("skip-collect-disk-usage")); err != nil {
		glog.Fatalf("unable to bind flag due to error %v", err)
	}

	localCollectCmd.Flags().BoolVar(&skipDownloadJobProfiles, "skip-download-job-profiles", false, "Skip the Download Job Profiles collector")
	if err := viper.BindPFlag("skip-download-job-profiles", localCollectCmd.Flags().Lookup("skip-download-job-profiles")); err != nil {
		glog.Fatalf("unable to bind flag due to error %v", err)
	}

	localCollectCmd.Flags().BoolVar(&skipCollectQueriesJSON, "skip-collect-queries-json", false, "Skip the Collect Queries JSON collector")
	if err := viper.BindPFlag("skip-collect-queries-json", localCollectCmd.Flags().Lookup("skip-collect-queries-json")); err != nil {
		glog.Fatalf("unable to bind flag due to error %v", err)
	}

	localCollectCmd.Flags().BoolVar(&skipCollectKubernetesInfo, "skip-collect-kubernetes-info", true, "Skip the Collect Kubernetes Info collector")
	if err := viper.BindPFlag("skip-collect-kubernetes-info", localCollectCmd.Flags().Lookup("skip-collect-kubernetes-info")); err != nil {
		glog.Fatalf("unable to bind flag due to error %v", err)
	}

	localCollectCmd.Flags().BoolVar(&skipCollectDremioConfiguration, "skip-collect-dremio-configuration", false, "Skip the Collect Dremio Configuration collector")
	if err := viper.BindPFlag("skip-collect-dremio-configuration", localCollectCmd.Flags().Lookup("skip-collect-dremio-configuration")); err != nil {
		glog.Fatalf("unable to bind flag due to error %v", err)
	}

	localCollectCmd.Flags().BoolVar(&skipCollectKVStoreReport, "skip-collect-kvstore-report", false, "Skip the Collect KVStore Report collector")
	if err := viper.BindPFlag("skip-collect-kvstore-report", localCollectCmd.Flags().Lookup("skip-collect-kvstore-report")); err != nil {
		glog.Fatalf("unable to bind flag due to error %v", err)
	}

	localCollectCmd.Flags().BoolVar(&skipCollectServerLogs, "skip-collect-server-logs", false, "Skip the Collect Server Logs collector")
	if err := viper.BindPFlag("skip-collect-server-logs", localCollectCmd.Flags().Lookup("skip-collect-server-logs")); err != nil {
		glog.Fatalf("unable to bind flag due to error %v", err)
	}

	localCollectCmd.Flags().BoolVar(&skipCollectMetaRefreshLog, "skip-collect-meta-refresh-log", false, "Skip the Collect Meta Refresh Log collector")
	if err := viper.BindPFlag("skip-collect-meta-refresh-log", localCollectCmd.Flags().Lookup("skip-collect-meta-refresh-log")); err != nil {
		glog.Fatalf("unable to bind flag due to error %v", err)
	}

	localCollectCmd.Flags().BoolVar(&skipCollectReflectionLog, "skip-collect-reflection-log", false, "Skip the Collect Reflection Log collector")
	if err := viper.BindPFlag("skip-collect-reflection-log", localCollectCmd.Flags().Lookup("skip-collect-reflection-log")); err != nil {
		glog.Fatalf("unable to bind flag due to error %v", err)
	}

	localCollectCmd.Flags().BoolVar(&skipCollectAccelerationLog, "skip-collect-acceleration-log", true, "Skip the Collect Acceleration Log collector")
	if err := viper.BindPFlag("skip-collect-acceleration-log", localCollectCmd.Flags().Lookup("skip-collect-acceleration-log")); err != nil {
		glog.Fatalf("unable to bind flag due to error %v", err)
	}

	localCollectCmd.Flags().BoolVar(&skipCollectAccessLog, "skip-collect-access-log", false, "Skip the Collect Access Log collector")
	if err := viper.BindPFlag("skip-collect-access-log", localCollectCmd.Flags().Lookup("skip-collect-access-log")); err != nil {
		glog.Fatalf("unable to bind flag due to error %v", err)
	}

	localCollectCmd.Flags().BoolVar(&skipCollectGCLogs, "skip-collect-gc-logs", false, "Skip the Collect GC Logs collector")
	if err := viper.BindPFlag("skip-collect-gc-logs", localCollectCmd.Flags().Lookup("skip-collect-gc-logs")); err != nil {
		glog.Fatalf("unable to bind flag due to error %v", err)
	}

	localCollectCmd.Flags().BoolVar(&skipCollectWLM, "skip-collect-wlm", false, "Skip the Collect WLM collector")
	if err := viper.BindPFlag("skip-collect-wlm", localCollectCmd.Flags().Lookup("skip-collect-wlm")); err != nil {
		glog.Fatalf("unable to bind flag due to error %v", err)
	}

	localCollectCmd.Flags().BoolVar(&skipHeapDump, "skip-heap-dump", true, "Skip the Heap Dump collector")
	if err := viper.BindPFlag("skip-heap-dump", localCollectCmd.Flags().Lookup("skip-heap-dump")); err != nil {
		glog.Fatalf("unable to bind flag due to error %v", err)
	}

	localCollectCmd.Flags().BoolVar(&skipJFR, "skip-jfr", true, "Skip the JFR (Java Flight Recorder) collection")
	if err := viper.BindPFlag("skip-jfr", localCollectCmd.Flags().Lookup("skip-jfr")); err != nil {
		glog.Fatalf("unable to bind flag due to error %v", err)
	}

	// Add flags for other options
	localCollectCmd.Flags().IntVar(&dremioJFRTimeSeconds, "dremio-jfr-time-seconds", 300, "Duration in seconds to run the JFR collector")
	if err := viper.BindPFlag("dremio-jfr-time-seconds", localCollectCmd.Flags().Lookup("dremio-jfr-time-seconds")); err != nil {
		glog.Fatalf("unable to bind flag due to error %v", err)
	}

	localCollectCmd.Flags().IntVar(&jobProfilesNumDays, "job-profiles-num-days", 28, "Number of days of job profile history to collect")
	if err := viper.BindPFlag("job-profiles-num-days", localCollectCmd.Flags().Lookup("job-profiles-num-days")); err != nil {
		glog.Fatalf("unable to bind flag due to error %v", err)
	}

	localCollectCmd.Flags().IntVar(&jobProfilesNumSlowExec, "job-profiles-num-slow-exec", 10000, "Number of slowest job profiles to collect by execution time")
	if err := viper.BindPFlag("job-profiles-num-slow-exec", localCollectCmd.Flags().Lookup("job-profiles-num-slow-exec")); err != nil {
		glog.Fatalf("unable to bind flag due to error %v", err)
	}

	localCollectCmd.Flags().IntVar(&jobProfilesNumHighQueryCost, "job-profiles-num-high-query-cost", 5000, "Number of job profiles to collect with the highest query cost")
	if err := viper.BindPFlag("job-profiles-num-high-query-cost", localCollectCmd.Flags().Lookup("job-profiles-num-high-query-cost")); err != nil {
		glog.Fatalf("unable to bind flag due to error %v", err)
	}

	localCollectCmd.Flags().IntVar(&jobProfilesNumSlowPlanning, "job-profiles-num-slow-planning", 5000, "Number of slowest job profiles to collect by planning time")
	if err := viper.BindPFlag("job-profiles-num-slow-planning", localCollectCmd.Flags().Lookup("job-profiles-num-slow-planning")); err != nil {
		glog.Fatalf("unable to bind flag due to error %v", err)
	}

	localCollectCmd.Flags().IntVar(&jobProfilesNumRecentErrors, "job-profiles-num-recent-errors", 5000, "Number of most recent job profiles to collect with errors")
	if err := viper.BindPFlag("job-profiles-num-recent-errors", localCollectCmd.Flags().Lookup("job-profiles-num-recent-errors")); err != nil {
		glog.Fatalf("unable to bind flag due to error %v", err)
	}

	// consent form
	localCollectCmd.Flags().BoolVar(&acceptCollectionConsent, "accept-collection-consent", false, "consent for collection of files, if not true, then collection will stop and a log message will be generated")
	if err := viper.BindPFlag("accept-collection-consent", localCollectCmd.Flags().Lookup("accept-collection-consent")); err != nil {
		glog.Fatalf("unable to bind flag due to error %v", err)
	}

	// Set glog flags
	if err := flag.Set("log_dir", logDir); err != nil {
		log.Printf("WARN: unable to set flag 'log_dir' due to error '%v', this is unexpected and should be reported as a bug", err)
	}
	if err := flag.Set("v", strconv.Itoa(verbose)); err != nil {
		log.Printf("WARN: unable to set flag 'v' due to error '%v', this is unexpected and should be reported as a bug", err)
	}
	//Viper will use the values from the configuration file, environment variables,
	//and command line flags in the following order of precedence (highest to lowest):
	//command line flags, environment variables, and then the configuration file.
	//This means that the command line flags will override the environment variables and configuration file values if they are set.
	baseConfig := "ddc-config"
	viper.SetConfigName(baseConfig) // Name of config file (without extension)

	//find the location of the ddc executable
	execPath, err := os.Executable()
	if err != nil {
		log.Printf("Error getting executable path: '%v'. Falling back to working directory for search location", err)
		execPath = "."
	}
	// use that as the default location of the configuration
	configDir := filepath.Dir(execPath)
	viper.AddConfigPath(configDir)

	for _, e := range supportedExtensions {
		confFiles = append(confFiles, fmt.Sprintf("%v.%v", baseConfig, e))
	}

	//searching for all known
	for _, ext := range supportedExtensions {
		viper.SetConfigType(ext)
		unableToReadConfigError := viper.ReadInConfig()
		if unableToReadConfigError == nil {
			configIsFound = true
			foundConfig = fmt.Sprintf("%v.%v", baseConfig, ext)
			break
		}
	}

	viper.AutomaticEnv() // Automatically read environment variables

}

// ### Helper functions
func validateAPICredentials() error {
	log.Printf("Validating REST API user credentials...")
	url := dremioEndpoint + "/apiv2/login"
	headers := map[string]string{"Content-Type": "application/json"}
	_, err := apiRequest(url, dremioPATToken, "GET", headers)
	return err
}

func apiRequest(url string, pat string, request string, headers map[string]string) ([]byte, error) {
	log.Printf("Requesting %s", url)
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(request, url, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to create request due to error %v", err)
	}
	authorization := "Bearer " + pat
	req.Header.Set("Authorization", authorization)
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf(res.Status)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func postQuery(url string, pat string, headers map[string]string, systable string) (string, error) {
	log.Printf("Collecting sys." + systable)

	sqlbody := "{\"sql\": \"SELECT * FROM sys." + systable + "\"}"
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("POST", url, strings.NewReader(sqlbody))
	if err != nil {
		return "", fmt.Errorf("unable to create request due to error %v", err)
	}
	authorization := "Bearer " + pat
	req.Header.Set("Authorization", authorization)

	for key, value := range headers {
		req.Header.Set(key, value)
	}
	res, err := client.Do(req)

	if err != nil {
		return "", err
	}
	if res.StatusCode != 200 {
		return "", fmt.Errorf(res.Status)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	var job map[string]string
	if err := json.Unmarshal(body, &job); err != nil {
		return "", err
	}
	return job["id"], nil
}

func errCheck(f func() error) {
	err := f()
	if err != nil {
		fmt.Println("Received error:", err)
	}
}
