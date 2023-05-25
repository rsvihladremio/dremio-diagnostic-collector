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
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"math"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/rsvihladremio/dremio-diagnostic-collector/cmd/local/queriesjson"
	"github.com/rsvihladremio/dremio-diagnostic-collector/cmd/simplelog"

	"github.com/rsvihladremio/dremio-diagnostic-collector/cmd/ddcio"
	"github.com/rsvihladremio/dremio-diagnostic-collector/cmd/threading"
)

//constants

// package variables
var (
	//kubernetesConfTypes = []string{"nodes", "sc", "pvc", "pv", "service", "endpoints", "pods", "deployments", "statefulsets", "daemonset", "replicaset", "cronjob", "job", "events", "ingress", "limitrange", "resourcequota", "hpa", "pdb", "pc"}
	supportedExtensions = []string{"yaml", "json", "toml", "hcl", "env", "props"}
	systemtables        = [...]string{
		//	"\\\"tables\\\"",
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

	unableToReadConfigError error
	confFiles               []string
	configIsFound           bool
	foundConfig             string
)

func configurationOutDir() string {
	return path.Join(outputDir, "configuration", nodeName)
}
func jfrOutDir() string          { return path.Join(outputDir, "jfr") }
func threadDumpsOutDir() string  { return path.Join(outputDir, "jfr", "thread-dumps", nodeName) }
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
		simplelog.Errorf("this should never return an error so this is truly critical: %v", err)
		os.Exit(1)
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

	if collectNodeMetrics {
		simplelog.Info("collecting node metrics")
		builder.WriteString("\t* cpu, io and memory metrics\n")
	}

	if collectDiskUsage {
		simplelog.Info("collecting disk usage")
		builder.WriteString("\t* df -h output\n")
	}

	if collectDremioConfiguration {
		simplelog.Info("collecting dremio configuration")
		builder.WriteString("\t* dremio-env, dremio.conf, logback.xml, and logback-access.xml\n")
	}

	if collectSystemTablesExport {
		simplelog.Info("collecting system tables")
		builder.WriteString(fmt.Sprintf("\t* the following system tables: %v\n", strings.Join(systemtables[:], ",")))
	}

	if collectKVStoreReport {
		simplelog.Info("collecting kv store report")
		builder.WriteString("\t* usage statistics on the internal Key Value Store (KVStore)\n")
		builder.WriteString("\t* list of all sources, their type and name\n")
	}

	if collectServerLogs {
		simplelog.Info("collecting metadata server logs")
		builder.WriteString("\t* server.log including any archived versions, and server.out\n")
	}

	if collectQueriesJSON {
		simplelog.Info("collecting queries.json")
		builder.WriteString("\t* queries.json including archived versions\n")
	}

	if collectMetaRefreshLogs {
		simplelog.Info("collecting metadata refresh logs")
		builder.WriteString("\t* metadata_refresh.log including any archived versions\n")
	}

	if collectReflectionLogs {
		simplelog.Info("\tcollecting reflection logs")
		builder.WriteString("\t* reflection.log including archived versions\n")
	}

	if collectAccelerationLogs {
		simplelog.Info("collecting acceleration logs")
		builder.WriteString("\t* acceleration.log including archived versions\n")
	}

	if numberJobProfilesToCollect > 0 {
		simplelog.Infof("collecting %v job profiles", numberJobProfilesToCollect)
		builder.WriteString(fmt.Sprintf("\t* %v job profiles randomly selected\n", numberJobProfilesToCollect))
	}

	if collectAccessLogs {
		simplelog.Info("collecting access logs")
		builder.WriteString("\t* access.log including archived versions\n")
	}

	if collectGCLogs {
		simplelog.Info("collecting gc logs")
		builder.WriteString("\t* all gc.log files produced by dremio\n")
	}

	if collectWLM {
		simplelog.Info("collecting Workload Manager information")
		builder.WriteString("\t* Work Load Manager queue names and rule names\n")
	}

	if captureHeapDump {
		simplelog.Info("collecting Java Heap Dump")
		builder.WriteString("\t*A Java heap dump which contains a copy of all data in the JVM heap\n")
	}

	if collectJStack {
		simplelog.Info("collecting JStacks")
		builder.WriteString("\t* Java thread dumps collected via jstack\n")
	}

	if collectJFR {
		simplelog.Info("collecting JFR")
		builder.WriteString("\t* Java Flight Recorder diagnostic information\n")
	}
	builder.WriteString(`

	Please note that the files we collect may contain confidential data. We will minimize the collection of confidential data wherever possible and will anonymize the data where feasible. 

We will use these files to:

1. Identify and diagnose problems with our products or services that you are using.
2. Improve our products and services.
3. Carry out other purposes that we will disclose to you at the time we collect the files.

Withdrawal of Consent

You have the right to withdraw your consent at any time. If you wish to do so, please contact us at support@dremio.com. Upon receipt of your withdrawal request, we will stop collecting new files and will delete any files we have already collected, unless we are required by law to retain them.

Changes to this Consent Form

We reserve the right to update this consent form from time to time.

Consent

By running ddc with the --accept-collection-consent flag, you acknowledge that you have read, understood, and agree to the data collection practices described in this consent form.

`)
	return builder.String()
}

func createAllDirs() error {
	var perms fs.FileMode = 0750
	if err := os.MkdirAll(configurationOutDir(), perms); err != nil {
		return fmt.Errorf("unable to create configuration directory due to error %v", err)
	}
	if err := os.MkdirAll(jfrOutDir(), perms); err != nil {
		return fmt.Errorf("unable to create jfr directory due to error %v", err)
	}
	if err := os.MkdirAll(threadDumpsOutDir(), perms); err != nil {
		return fmt.Errorf("unable to create thread-dumps directory due to error %v", err)
	}
	if err := os.MkdirAll(heapDumpsOutDir(), perms); err != nil {
		return fmt.Errorf("unable to create heap-dumps directory due to error %v", err)
	}
	if err := os.MkdirAll(jobProfilesOutDir(), perms); err != nil {
		return fmt.Errorf("unable to create job-profiles directory due to error %v", err)
	}
	if err := os.MkdirAll(kubernetesOutDir(), perms); err != nil {
		return fmt.Errorf("unable to create kubernetes directory due to error %v", err)
	}
	if err := os.MkdirAll(kvstoreOutDir(), perms); err != nil {
		return fmt.Errorf("unable to create kvstore directory due to error %v", err)
	}
	if err := os.MkdirAll(logsOutDir(), perms); err != nil {
		return fmt.Errorf("unable to create logs directory due to error %v", err)
	}
	if err := os.MkdirAll(nodeInfoOutDir(), perms); err != nil {
		return fmt.Errorf("unable to create node-info directory due to error %v", err)
	}
	if err := os.MkdirAll(queriesOutDir(), perms); err != nil {
		return fmt.Errorf("unable to create queries directory due to error %v", err)
	}
	if err := os.MkdirAll(systemTablesOutDir(), perms); err != nil {
		return fmt.Errorf("unable to create system-tables directory due to error %v", err)
	}
	if err := os.MkdirAll(wlmOutDir(), perms); err != nil {
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

	//put all things that take time up front

	// os diagnostic collection
	if !collectNodeMetrics {
		simplelog.Info("Skipping Collecting Node Metrics...")
	} else {
		t.FireJob(runCollectNodeMetrics)
	}

	if !collectJFR {
		simplelog.Info("skipping Collection of Java Flight Recorder Information")
	} else {
		t.FireJob(runCollectJFR)
	}

	if !collectJStack {
		simplelog.Info("skipping Collection of java thread dumps")
	} else {
		t.FireJob(runCollectJStacks)
	}

	if !captureHeapDump {
		simplelog.Info("skipping Capture of Java Heap Dump")
	} else {
		t.FireJob(runCollectHeapDump)
	}

	if collectDiskUsage {
		simplelog.Infof("Skipping Collect Disk Usage from %v ...", nodeName)
	} else {
		t.FireJob(runCollectDiskUsage)
	}

	if collectDremioConfiguration {
		simplelog.Infof("Skipping Dremio config from %v ...", nodeName)
	} else {
		t.FireJob(runCollectDremioConfig)
	}
	t.FireJob(runCollectOSConfig)

	// log collection

	if !collectQueriesJSON && numberJobProfilesToCollect == 0 {
		simplelog.Info("Skipping Collect Queries JSON ...")
	} else {
		if !collectQueriesJSON {
			simplelog.Warning("NOT Skipping collection of Queries JSON, because --number-job-profiles is greater than 0 and job profile download requires queries.json ...")
		}
		t.FireJob(runCollectQueriesJSON)
	}

	if !collectServerLogs {
		simplelog.Info("Skipping Collect Server Logs  ...")
	} else {
		t.FireJob(runCollectDremioServerLog)
	}
	if !collectGCLogs {
		simplelog.Info("Skipping Collect Garbage Collection Logs  ...")
	} else {
		t.FireJob(runCollectGcLogs)
	}
	if !collectMetaRefreshLogs {
		simplelog.Info("Skipping Collect Metadata Refresh Logs  ...")
	} else {
		t.FireJob(runCollectMetadataRefreshLogs)
	}
	if !collectReflectionLogs {
		simplelog.Info("Skipping Collect Reflection Logs  ...")
	} else {
		t.FireJob(runCollectReflectionLogs)
	}

	if !collectAccelerationLogs {
		simplelog.Info("Skipping Collect Acceleration Logs  ...")
	} else {
		t.FireJob(runCollectAccelerationLogs)
	}

	if !collectAccessLogs {
		simplelog.Info("Skipping Collect Access Logs  ...")
	} else {
		t.FireJob(runCollectDremioAccessLogs)
	}

	t.FireJob(runCollectJvmConfig)

	// rest call collections

	if collectKVStoreReport {
		simplelog.Info("skipping Capture of KV Store Report")
	} else {
		t.FireJob(runCollectKvReport)
	}

	if collectWLM {
		simplelog.Info("skipping Capture of Workload Manager Report")
	} else {
		t.FireJob(runCollectWLM)
	}

	if !collectSystemTablesExport {
		simplelog.Info("Skipping Collect of Export System Tables...")
	} else {
		t.FireJob(runCollectDremioSystemTables)
	}

	if numberJobProfilesToCollect == 0 {
		simplelog.Info("Skipping Collect of Job Profiles...")
	} else {
		t.FireJob(runCollectJobProfiles)
	}

	t.Wait()
}

func runCollectDiskUsage() error {
	diskWriter, err := os.Create(path.Clean(filepath.Join(outputDir, "node-info", nodeName, "diskusage.txt")))
	if err != nil {
		return fmt.Errorf("unable to create diskusage.txt due to error %v", err)
	}
	defer func() {
		if err := diskWriter.Sync(); err != nil {
			simplelog.Warningf("unable to sync the os_info.txt file due to error: %v", err)
		}
		if err := diskWriter.Close(); err != nil {
			simplelog.Warningf("unable to close the os_info.txt file due to error: %v", err)
		}
	}()
	err = ddcio.Shell(diskWriter, "df -h")
	if err != nil {
		simplelog.Warningf("unable to read df -h due to error %v", err)
	}

	if strings.Contains(nodeName, "dremio-master") {
		rocksDbDiskUsageWriter, err := os.Create(path.Clean(filepath.Join(outputDir, "node-info", nodeName, "rocksdb_disk_allocation.txt")))
		if err != nil {
			return fmt.Errorf("unable to create rocksdb_disk_allocation.txt due to error %v", err)
		}
		defer func() {
			if err := rocksDbDiskUsageWriter.Close(); err != nil {
				simplelog.Warningf("unable to close rocksdb usage writer the file maybe incomplete %v", err)
			}
		}()
		err = ddcio.Shell(rocksDbDiskUsageWriter, "du -sh /opt/dremio/data/db/*")
		if err != nil {
			simplelog.Warningf("unable to write du -sh to rocksdb_disk_allocation.txt due to error %v", err)
		}

	}
	simplelog.Infof("... Collecting Disk Usage from %v COMPLETED", nodeName)

	return nil
}

func runCollectOSConfig() error {
	simplelog.Info("Collecting OS Information from $BASENAME ...")
	osInfoFile := path.Join(outputDir, "node-info", nodeName, "os_info.txt")
	w, err := os.Create(path.Clean(osInfoFile))
	if err != nil {
		return fmt.Errorf("unable to create file %v due to error %v", path.Clean(osInfoFile), err)
	}
	defer func() {
		if err := w.Sync(); err != nil {
			simplelog.Warningf("unable to sync the os_info.txt file due to error: %v", err)
		}
		if err := w.Close(); err != nil {
			simplelog.Warningf("unable to close the os_info.txt file due to error: %v", err)
		}
	}()

	simplelog.Debug("/etc/*-release")

	_, err = w.Write([]byte("___\n>>> cat /etc/*-release\n"))
	if err != nil {
		simplelog.Warningf("unable to write release file header for os_info.txt due to error %v", err)
	}

	err = ddcio.Shell(w, "cat /etc/*-release")
	if err != nil {
		simplelog.Warningf("unable to write release files for os_info.txt due to error %v", err)
	}

	_, err = w.Write([]byte("___\n>>> uname -r\n"))
	if err != nil {
		simplelog.Warningf("unable to write uname header for os_info.txt due to error %v", err)
	}

	err = ddcio.Shell(w, "uname -r")
	if err != nil {
		simplelog.Warningf("unable to write uname -r for os_info.txt due to error %v", err)
	}
	_, err = w.Write([]byte("___\n>>> lsb_release -a\n"))
	if err != nil {
		simplelog.Warningf("unable to write lsb_release -r header for os_info.txt due to error %v", err)
	}
	err = ddcio.Shell(w, "lsb_release -a")
	if err != nil {
		simplelog.Warningf("unable to write lsb_release -a for os_info.txt due to error %v", err)
	}
	_, err = w.Write([]byte("___\n>>> hostnamectl\n"))
	if err != nil {
		simplelog.Warningf("unable to write hostnamectl for os_info.txt due to error %v", err)
	}
	err = ddcio.Shell(w, "hostnamectl")
	if err != nil {
		simplelog.Warningf("unable to write hostnamectl for os_info.txt due to error %v", err)
	}
	_, err = w.Write([]byte("___\n>>> cat /proc/meminfo\n"))
	if err != nil {
		simplelog.Warningf("unable to write /proc/meminfo header for os_info.txt due to error %v", err)
	}
	err = ddcio.Shell(w, "cat /proc/meminfo")
	if err != nil {
		simplelog.Warningf("unable to write /proc/meminfo for os_info.txt due to error %v", err)
	}
	_, err = w.Write([]byte("___\n>>> lscpu\n"))
	if err != nil {
		simplelog.Warningf("unable to write lscpu header for os_info.txt due to error %v", err)
	}
	err = ddcio.Shell(w, "lscpu")
	if err != nil {
		simplelog.Warningf("unable to write lscpu for os_info.txt due to error %v", err)
	}

	simplelog.Infof("... Collecting OS Information from %v COMPLETED", nodeName)
	return nil
}

func runCollectDremioConfig() error {
	simplelog.Infof("Collecting Configuration Information from %v ...", nodeName)
	//mkdir -p $DREMIO_HEALTHCHECK_EXPORT_DIR/configuration/$BASENAME

	err := copyFile("/opt/dremio/conf/dremio.conf", filepath.Join(outputDir, "configuration", nodeName, "dremio.conf"))
	if err != nil {
		simplelog.Warningf("unable to copy dremio.conf due to error %v", err)
	}
	err = copyFile("/opt/dremio/conf/dremio-env", filepath.Join(outputDir, "configuration", nodeName, "dremio.env"))
	if err != nil {
		simplelog.Warningf("unable to copy dremio.env due to error %v", err)
	}
	err = copyFile("/opt/dremio/conf/logback.xml", filepath.Join(outputDir, "configuration", nodeName, "logback.xml"))
	if err != nil {
		simplelog.Warningf("unable to copy logback.xml due to error %v", err)
	}
	err = copyFile("/opt/dremio/conf/logback-access.xml", filepath.Join(outputDir, "configuration", nodeName, "logback-access.xml"))
	if err != nil {
		simplelog.Warningf("unable to copy logback-access.xml due to error %v", err)
	}
	//# ddcio.Shell "cat /opt/dremio/conf/core-site.xml" > $DREMIO_HEALTHCHECK_EXPORT_DIR/configuration/$BASENAME/core-site.xml

	//python3 $DREMIO_HEALTHCHECK_SCRIPT_DIR/helper/secrets_cleanser_config.py $DREMIO_HEALTHCHECK_EXPORT_DIR/configuration/$BASENAME/dremio.conf

	simplelog.Infof("... Collecting Configuration Information from %v COMPLETED", nodeName)

	return nil
}

func runCollectJvmConfig() error {
	simplelog.Warning("You may have to run the following command 'jcmd 1 VM.flags' as 'sudo' and specify '-u dremio' when running on Dremio AWSE or VM deployments")
	jvmSettingsFile := path.Join(outputDir, "node-info", nodeName, "jvm_settings.txt")
	jvmSettingsFileWriter, err := os.Create(path.Clean(jvmSettingsFile))
	if err != nil {
		return fmt.Errorf("unable to create file %v due to error %v", path.Clean(jvmSettingsFile), err)
	}
	defer func() {
		if err := jvmSettingsFileWriter.Sync(); err != nil {
			simplelog.Warningf("unable to sync the os_info.txt file due to error: %v", err)
		}
		if err := jvmSettingsFileWriter.Close(); err != nil {
			simplelog.Warningf("unable to close the os_info.txt file due to error: %v", err)
		}
	}()
	dremioPID, err := getDremioPID()
	if err != nil {
		return fmt.Errorf("unable to get dremio PID %v", err)
	}
	err = ddcio.Shell(jvmSettingsFileWriter, fmt.Sprintf("jcmd %v VM.flags", dremioPID))
	if err != nil {
		simplelog.Warningf("unable to write jvm_settings.txt file due to error %v", err)
	}
	return nil
}

func copyFile(srcPath, dstPath string) error {
	// Open the source file
	srcFile, err := os.Open(path.Clean(srcPath))
	if err != nil {
		return err
	}
	defer func() {
		if err := srcFile.Close(); err != nil {
			simplelog.Warningf("unable to close %v due to error %v", path.Clean(srcPath), err)
		}
	}()

	// Create the destination file
	dstFile, err := os.Create(path.Clean(dstPath))
	if err != nil {
		return err
	}
	defer func() {
		if err := dstFile.Close(); err != nil {
			simplelog.Errorf("unable to close file %v due to error %v", path.Clean(dstPath), err)
			os.Exit(1)
		}
	}()

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

func runCollectNodeMetrics() error {
	simplelog.Info("Collecting Node Metrics for 60 seconds ....")
	nodeMetricsFile := path.Clean(path.Join(outputDir, "node-info", nodeName, "metrics.txt"))
	w, err := os.Create(path.Clean(nodeMetricsFile))
	if err != nil {
		return fmt.Errorf("unable to create file %v due to error '%v'", nodeMetricsFile, err)
	}
	defer func() {
		if err := w.Close(); err != nil {
			simplelog.Errorf("Failure writing file %v as we are unable to close it due to error '%v'", nodeMetricsFile, err)
		}
	}()

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

func runCollectJFR() error {
	dremioPID, err := getDremioPID()
	if err != nil {
		return fmt.Errorf("unable to get dremio PID %v", err)
	}
	var w bytes.Buffer
	if err := ddcio.Shell(&w, fmt.Sprintf("jcmd %v VM.unlock_commercial_features", dremioPID)); err != nil {
		simplelog.Warningf("Error trying to unlock commercial features %v. Note: newer versions of OpenJDK do not support the call VM.unlock_commercial_features. This is usually safe to ignore", err)
	}
	simplelog.Debugf("node: %v - jfr unlock commerictial output - %v", nodeName, w.String())
	w = bytes.Buffer{}
	if err := ddcio.Shell(&w, fmt.Sprintf("jcmd %v JFR.start name=\"DREMIO_JFR\" settings=profile maxage=%vs  filename=%v/%v.jfr dumponexit=true", dremioPID, dremioJFRTimeSeconds, jfrOutDir(), nodeName)); err != nil {
		return fmt.Errorf("unable to run JFR due to error %v", err)
	}
	simplelog.Debugf("node: %v - jfr start output - %v", nodeName, w.String())
	time.Sleep(time.Duration(dremioJFRTimeSeconds) * time.Second)
	// do not "optimize". the recording first needs to be stopped for all processes before collecting the data.
	simplelog.Info("... stopping JFR $BASEPOD")
	w = bytes.Buffer{}
	if err := ddcio.Shell(&w, fmt.Sprintf("jcmd %v JFR.dump name=\"DREMIO_JFR\"", dremioPID)); err != nil {
		return fmt.Errorf("unable to dump JFR due to error %v", err)
	}
	simplelog.Debugf("node: %v - jfr dump output %v", nodeName, w.String())
	w = bytes.Buffer{}
	if err := ddcio.Shell(&w, fmt.Sprintf("jcmd %v JFR.stop name=\"DREMIO_JFR\"", dremioPID)); err != nil {
		return fmt.Errorf("unable to dump JFR due to error %v", err)
	}
	simplelog.Debugf("node: %v - jfr stop output %v", nodeName, w.String())
	w = bytes.Buffer{}
	if err := ddcio.Shell(&w, fmt.Sprintf("rm -f %v/%v.jfr", jfrOutDir(), nodeName)); err != nil {
		return fmt.Errorf("unable to dump JFR due to error %v", err)
	}

	return nil
}

func runCollectJStacks() error {
	simplelog.Info("Collecting GC logs ...")
	threadDumpFreq := dremioJStackFreqSeconds
	iterations := dremioJStackTimeSeconds / threadDumpFreq
	simplelog.Infof("Running Java thread dumps every %v second(s) for a total of %v iterations ...", threadDumpFreq, iterations)
	dremioPID, err := getDremioPID()
	if err != nil {
		return fmt.Errorf("unable to get dremio PID %v", err)
	}
	for i := 0; i < iterations; i++ {
		var w bytes.Buffer
		if err := ddcio.Shell(&w, fmt.Sprintf("jcmd %v Thread.print -l", dremioPID)); err != nil {
			simplelog.Warningf("unable to capture jstack of pid %v due to error %v", dremioPID, err)
		}
		date := time.Now().Format("2006-01-02_15_04_05")
		threadDumpFileName := path.Join(threadDumpsOutDir(), fmt.Sprintf("threadDump-%s-%s.txt", nodeName, date))
		if err := os.WriteFile(path.Clean(threadDumpFileName), w.Bytes(), 0600); err != nil {
			return fmt.Errorf("unable to write thread dump %v due to error %v", threadDumpFileName, err)
		}
		simplelog.Infof("Saved %v", threadDumpFileName)
		simplelog.Infof("Waiting %v second(s) ...", threadDumpFreq)
		time.Sleep(time.Duration(threadDumpFreq))
	}
	return nil
}

func runCollectKvReport() error {
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
	file, err := os.Create(path.Clean(kvStoreReportFile))
	if err != nil {
		return fmt.Errorf("unable to create file %s due to error %v", filename, err)
	}
	defer errCheck(file.Close)
	_, err = fmt.Fprint(file, sb)
	if err != nil {
		return fmt.Errorf("unable to create file %s due to error %v", filename, err)
	}
	simplelog.Info("SUCCESS - Created " + filename)
	return nil
}

func runCollectWLM() error {
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
		wlmFile := path.Clean(path.Join(wlmOutDir(), filename))
		file, err := os.Create(path.Clean(wlmFile))
		if err != nil {
			return fmt.Errorf("unable to create file %s due to error %v", filename, err)
		}
		defer errCheck(file.Close)
		_, err = fmt.Fprint(file, sb)
		if err != nil {
			return fmt.Errorf("unable to create file %s due to error %v", filename, err)
		}
		simplelog.Infof("SUCCESS - Created " + filename)
	}
	return nil
}

func runCollectHeapDump() error {
	simplelog.Info("Capturing Java Heap Dump")
	dremioPID, err := getDremioPID()
	if err != nil {
		return fmt.Errorf("unable to get dremio pid %v", err)
	}
	baseName := fmt.Sprintf("%v.hprof", nodeName)
	hprofFile := fmt.Sprintf("/tmp/%v.hprof", baseName)
	hprofGzFile := fmt.Sprintf("%v.gz", hprofFile)
	if err := os.Remove(path.Clean(hprofGzFile)); err != nil {
		simplelog.Warningf("unable to remove hprof.gz file with error %v", err)
	}
	if err := os.Remove(path.Clean(hprofFile)); err != nil {
		simplelog.Warningf("unable to remove hprof file with error %v", err)
	}
	var w bytes.Buffer
	if err := ddcio.Shell(&w, fmt.Sprintf("jmap -dump:format=b,file=%v %v", hprofFile, dremioPID)); err != nil {
		return fmt.Errorf("unable to capture heap dump %v", err)
	}
	simplelog.Infof("heap dump output %v", w.String())
	if err := gzipFile(hprofFile, hprofGzFile); err != nil {
		return fmt.Errorf("unable to gzip heap dump file")
	}
	if err := os.Remove(path.Clean(hprofFile)); err != nil {
		simplelog.Warningf("unable to remove old hprof file, must remove manually %v", err)
	}
	dest := path.Join(heapDumpsOutDir(), baseName+".gz")
	if err := os.Rename(path.Clean(hprofGzFile), path.Clean(dest)); err != nil {
		return fmt.Errorf("unable to move heap dump to %v due to error %v", dest, err)
	}
	return nil
}

func runCollectQueriesJSON() error {
	simplelog.Info("Collecting GC logs ...")
	err := exportArchivedLogs(dremioLogDir, "queries.json", "queries", dremioQueriesJSONNumDays)
	if err != nil {
		return fmt.Errorf("failed to export archived logs: %v", err)
	}

	simplelog.Warning("Queries.json from scale-out coordinators must be collected separately!")

	simplelog.Info("... collecting Queries JSON for Job Profiles COMPLETED")
	return nil
}

func runCollectJobProfiles() error {

	simplelog.Info("Collecting Job Profiles...")
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

	if len(queriesjsons) == 0 {
		simplelog.Warning("no queries.json files found. This is probably an executor, so we are skipping collection of Job Profiles")
		return nil
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

	simplelog.Infof("jobProfilesNumSlowPlanning: %v", jobProfilesNumSlowPlanning)
	simplelog.Infof("jobProfilesNumSlowExec: %v", jobProfilesNumSlowExec)
	simplelog.Infof("jobProfilesNumHighQueryCost: %v", jobProfilesNumHighQueryCost)
	simplelog.Infof("jobProfilesNumRecentErrors: %v", jobProfilesNumRecentErrors)

	simplelog.Infof("Downloading %v job profiles...", len(profilesToCollect))
	for key := range profilesToCollect {
		err := downloadJobProfile(key)
		if err != nil {
			simplelog.Error(err.Error()) // Print instead of Error
		}
	}
	simplelog.Infof("Finished downloading %v job profiles", len(profilesToCollect))

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
	jobProfileFile := path.Clean(path.Join(jobProfilesOutDir(), filename))
	file, err := os.Create(path.Clean(jobProfileFile))
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

func runCollectDremioSystemTables() error {

	simplelog.Info("Collecting results from Export System Tables...")
	err := validateAPICredentials()
	if err != nil {
		return err
	}
	// TODO: Row limit and sleem MS need to be configured
	rowlimit := 100000
	sleepms := 100

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
				simplelog.Warning("Returned row count for sys." + systable + " has been limited to " + strconv.Itoa(rowlimit))
			}
		}
		sb := string(body)
		systemTableFile := path.Join(systemTablesOutDir(), filename)
		file, err := os.Create(path.Clean(systemTableFile))
		if err != nil {
			return fmt.Errorf("unable to create file %v due to error %v", filename, err)
		}
		defer errCheck(file.Close)
		_, err = fmt.Fprint(file, sb)
		if err != nil {
			return fmt.Errorf("unable to create file %s due to error %v", filename, err)
		}
		simplelog.Info("SUCCESS - Created " + filename)
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
		simplelog.Info("Retrieving job results ...")
		body, err := apiRequest(jobresultsurl, dremioPATToken, "GET", headers)
		if err != nil {
			return nil, fmt.Errorf("unable to retrieve job results from %s due to error %v", jobresultsurl, err)
		}
		return body, nil
	}
	return nil, fmt.Errorf("unable to retrieve job results for sys." + systable)
}

func runCollectDremioServerLog() error {
	simplelog.Info("Collecting GC logs ...")
	if err := exportArchivedLogs(dremioLogDir, "server.log", "server", dremioLogsNumDays); err != nil {
		return fmt.Errorf("trying to archive server logs we got error: %v", err)
	}
	simplelog.Info("... collecting server.out")
	src := path.Join(dremioLogDir, "server.out")
	dest := path.Join(logsOutDir(), "server.out")
	if err := copyFile(path.Clean(src), path.Clean(dest)); err != nil {
		return fmt.Errorf("unable to copy %v to %v due to error %v", src, dest, err)
	}
	simplelog.Warning("Server logs from executors and scale-out coordinators must be collected separately!")
	simplelog.Info("... collecting server logs COMPLETED")

	return nil
}

func runCollectGcLogs() error {
	simplelog.Info("Collecting GC logs ...")
	files, err := os.ReadDir(path.Clean(gcLogsDir))
	if err != nil {
		return fmt.Errorf("error reading directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		matched, err := filepath.Match(dremioGCFilePattern, file.Name())
		if err != nil {
			simplelog.Errorf("error matching file pattern %v with error '%v'", dremioGCFilePattern, err)
		}
		if matched {
			srcPath := filepath.Join(gcLogsDir, file.Name())
			destPath := filepath.Join(logsOutDir(), file.Name())
			if err := copyFile(path.Clean(srcPath), path.Clean(destPath)); err != nil {
				return fmt.Errorf("error copying file %s: %w", file.Name(), err)
			}
			simplelog.Debugf("Copied file %s to %s", srcPath, destPath)
		}
	}
	simplelog.Warning("GC logs from executors and scale-out coordinators must be collected separately!")
	simplelog.Info("... collecting GC logs COMPLETED")

	return nil
}

func runCollectMetadataRefreshLogs() error {
	simplelog.Info("Collecting metadata refresh logs from Coordinator(s) ...")
	if err := exportArchivedLogs(dremioLogDir, "metadata_refresh.log", "metadata_refresh", dremioLogsNumDays); err != nil {
		return fmt.Errorf("unable to collect metadata refresh logs due to error %v", err)
	}
	simplelog.Warning("Metadata refresh logs from scale-out coordinators must be collected separately!")
	simplelog.Info("... collecting meta data refresh logs from Coordinator(s) COMPLETED")
	return nil
}

func runCollectReflectionLogs() error {
	simplelog.Info("Collecting reflection logs from Coordinator(s) ...")
	if err := exportArchivedLogs(dremioLogDir, "reflection.log", "reflection", dremioLogsNumDays); err != nil {
		return fmt.Errorf("unable to collect reflection logs due to error %v", err)
	}
	simplelog.Info("... collecting reflection logs from Coordinator(s) COMPLETED")

	return nil
}

func runCollectDremioAccessLogs() error {

	simplelog.Info("Collecting access logs from Coordinator(s) ...")
	simplelog.Warning("Access logs from scale-out coordinators must be collected separately!")
	if err := exportArchivedLogs(dremioLogDir, "access.log", "access", dremioLogsNumDays); err != nil {
		return fmt.Errorf("unable to archive access.logs due to error %v", err)
	}
	simplelog.Info("... collecting access logs from Coordinator(s) COMPLETED")

	return nil
}

func runCollectAccelerationLogs() error {

	simplelog.Info("Collecting acceleration logs from Coordinator(s) ...")
	simplelog.Warning("Acceleration logs from scale-out coordinators must be collected separately!")
	if err := exportArchivedLogs(dremioLogDir, "acceleration.log", "acceleration", dremioLogsNumDays); err != nil {
		return fmt.Errorf("unable to archive acceleration.logs due to error %v", err)
	}
	simplelog.Info("... collecting acceleragtion logs from Coordinator(s) COMPLETED")

	return nil
}

func gzipFile(src, dst string) error {
	sourceFile, err := os.Open(path.Clean(src))
	if err != nil {
		return err
	}
	defer func() {
		if err := sourceFile.Close(); err != nil {
			simplelog.Errorf("unable to close source file %v due to error %v", sourceFile, err)
		}
	}()

	destFile, err := os.Create(path.Clean(dst))
	if err != nil {
		return err
	}
	defer func() {
		if err := destFile.Close(); err != nil {
			simplelog.Errorf("unable to close gzip file %v due to error %v", dst, err)
		}
	}()

	gzipWriter := gzip.NewWriter(destFile)
	defer gzipWriter.Close()

	_, err = io.Copy(gzipWriter, sourceFile)
	if err != nil {
		return fmt.Errorf("unable to create gzip due to error %v", err)
	}

	return nil
}

func exportArchivedLogs(logDir string, unarchivedFile string, logPrefix string, archiveDays int) error {
	src := path.Join(logDir, unarchivedFile)
	dest := path.Join(logsOutDir(), unarchivedFile)
	//instead of copying it we just archive it to a new location
	if err := gzipFile(path.Clean(src), path.Clean(dest+".gz")); err != nil {
		return fmt.Errorf("archiving of log file %v failed due to error %v", unarchivedFile, err)
	}

	today := time.Now()

	for i := 0; i <= archiveDays; i++ {
		processingDate := today.AddDate(0, 0, -i).Format("2006-01-02")
		files, err := os.ReadDir(filepath.Join(logDir, "archive"))
		if err != nil {
			simplelog.Error(err.Error())
			os.Exit(1)
		}

		for _, f := range files {
			if strings.HasPrefix(f.Name(), logPrefix+"."+processingDate) && strings.HasSuffix(f.Name(), ".gz") {
				simplelog.Info("Copying archive file for " + processingDate + ": " + f.Name())
				src := filepath.Join(logDir, "archive", f.Name())
				dst := logsOutDir()
				err := copyFile(path.Clean(src), path.Clean(dst))
				if err != nil {
					return fmt.Errorf("unable to rename file")
				}
			}
		}
	}
	return nil
}

// findGCLogLocation retrieves the gc log location with a search string to greedily retrieve everything by prefix
func findGCLogLocation() (gcLogLoc string, err error) {
	pid, err := getDremioPID()
	if err != nil {
		return "", fmt.Errorf("unable to find gc logs due to error '%v'", err)
	}
	var startupFlags bytes.Buffer
	err = ddcio.Shell(&startupFlags, fmt.Sprintf("ps -f %v", pid))
	if err != nil {
		return "", fmt.Errorf("unable to find gc logs due to error '%v'", err)
	}
	logLocation, err := ParseGCLogFromFlags(startupFlags.String())
	if err != nil {
		return "", fmt.Errorf("unable to find gc logs due to error '%v'", err)
	}
	return logLocation + "*", nil
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
	return path.Dir(gcLogLocationTokens[1]), nil
}

var localCollectCmd = &cobra.Command{
	Use:   "local-collect",
	Short: "retrieves all the dremio logs and diagnostics for the local node and saves the results in a compatible format for Dremio support",
	Long:  `Retrieves all the dremio logs and diagnostics for the local node and saves the results in a compatible format for Dremio support. This subcommand needs to be run with enough permissions to read the /proc filesystem, the dremio logs and configuration files`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if !cmd.Flags().Changed("verbose") {
			verbose = 2 // Default INFO
		}
		simplelog.InitLogger(verbose)
		//now read in viper configuration values. This will get defaults if no values are available in the configuration files or no environment variable is set

		baseConfig := "ddc"
		viper.SetConfigName(baseConfig) // Name of config file (without extension)

		//find the location of the ddc executable
		execPath, err := os.Executable()
		if err != nil {
			simplelog.Errorf("Error getting executable path: '%v'. Falling back to working directory for search location", err)
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

		viper.SetDefault("tmp-output-dir", getOutputDir(time.Now()))
		outputDir = viper.GetString("tmp-output-dir")

		// set node name
		hostName, err := os.Hostname()
		if err != nil {
			hostName = fmt.Sprintf("unknown-%v", uuid.New())
		}
		viper.SetDefault("node-name", hostName)
		nodeName = viper.GetString("node-name")

		//system diag

		viper.SetDefault("collect-metrics", true)
		collectNodeMetrics = viper.GetBool("collect-metrics")

		viper.SetDefault("collect-disk-usage", true)
		collectDiskUsage = viper.GetBool("collect-disk-usage")

		// log collect
		viper.SetDefault("dremio-logs-num-days", 7)
		dremioLogsNumDays = viper.GetInt("dremio-logs-num-days")

		viper.SetDefault("dremio-queries-json-num-days", 28)
		dremioQueriesJSONNumDays = viper.GetInt("dremio-queries-json-num-days")

		viper.SetDefault("dremio-gc-file-pattern", "gc*.log")
		dremioGCFilePattern = viper.GetString("dremio-gc-file-pattern")

		viper.SetDefault("collect-queries-json", true)
		collectQueriesJSON = viper.GetBool("collect-queries-json")

		viper.SetDefault("collect-server-logs", true)
		collectServerLogs = viper.GetBool("collect-server-logs")

		viper.SetDefault("collect-meta-refresh-log", true)
		collectMetaRefreshLogs = viper.GetBool("collect-meta-refresh-log")

		viper.SetDefault("collect-reflection-log", true)
		collectReflectionLogs = viper.GetBool("collect-reflection-log")

		viper.SetDefault("skip-collect-gc-logs", true)
		collectReflectionLogs = viper.GetBool("skip-collect-gc-logs")

		parsedGCLogDir, err := findGCLogLocation()
		if err != nil {
			simplelog.Errorf("Must set dremio-gclogs-dir manually since we are unable to retrieve gc log location from pid due to error %v", err)
		}
		viper.SetDefault("dremio-gclogs-dir", parsedGCLogDir)
		gcLogsDir = viper.GetString("dremio-gclogs-dir")

		// jfr config
		viper.SetDefault("collect-jfr", true)
		collectJFR = viper.GetBool("collect-jfr")
		defaultCaptureSeconds := 60
		viper.SetDefault("dremio-jfr-time-seconds", defaultCaptureSeconds)
		dremioJFRTimeSeconds = viper.GetInt("dremio-jfr-time-seconds")
		// jstack config
		viper.SetDefault("collect-jstack", true)
		collectJStack = viper.GetBool("collect-jstack")
		viper.SetDefault("dremio-jstack-time-seconds", defaultCaptureSeconds)
		dremioJStackTimeSeconds = viper.GetInt("dremio-jstack-time-seconds")
		viper.SetDefault("dremio-jstack-freq-seconds", 1)
		dremioJStackFreqSeconds = viper.GetInt("dremio-jstack-freq-seconds")

		// collect rest apis
		personalAccessTokenPresent := dremioPATToken != ""
		if !personalAccessTokenPresent {
			simplelog.Warningf("disabling all Workload Manager, System Table, KV Store, and Job Profile collection since the --dremio-pat-token is not set")
		}

		viper.SetDefault("collect-wlm", personalAccessTokenPresent)
		collectWLM = viper.GetBool("collect-wlm")

		viper.SetDefault("collect-system-tables-export", personalAccessTokenPresent)
		collectSystemTablesExport = viper.GetBool("collect-system-tables-export")

		viper.SetDefault("collect-kvstore-report", personalAccessTokenPresent)
		collectKVStoreReport = viper.GetBool("collect-kvstore-report")

		// don't bother doing any of the calculation if personal access token is not present in fact zero out everything
		if !personalAccessTokenPresent {
			numberJobProfilesToCollect = 0
			jobProfilesNumHighQueryCost = 0
			jobProfilesNumSlowExec = 0
			jobProfilesNumRecentErrors = 0
			jobProfilesNumSlowPlanning = 0
		} else {
			// check if job profile is set
			var defaultJobProfilesNumSlowExec int
			var defaultJobProfilesNumRecentErrors int
			var defaultJobProfilesNumSlowPlanning int
			var defaultJobProfilesNumHighQueryCost int
			if numberJobProfilesToCollect > 0 {
				if numberJobProfilesToCollect < 4 {
					//so few that it is not worth being clever
					defaultJobProfilesNumSlowExec = numberJobProfilesToCollect
				} else {
					defaultJobProfilesNumSlowExec = int(float64(numberJobProfilesToCollect) * 0.4)
					defaultJobProfilesNumRecentErrors = int(float64(defaultJobProfilesNumRecentErrors) * 0.2)
					defaultJobProfilesNumSlowPlanning = int(float64(defaultJobProfilesNumSlowPlanning) * 0.2)
					defaultJobProfilesNumHighQueryCost = int(float64(defaultJobProfilesNumHighQueryCost) * 0.2)
					//grab the remainder and drop on top of defaultJobProfilesNumSlowExec
					totalAllocated := defaultJobProfilesNumSlowExec + defaultJobProfilesNumRecentErrors + defaultJobProfilesNumSlowPlanning + defaultJobProfilesNumHighQueryCost
					diff := numberJobProfilesToCollect - totalAllocated
					defaultJobProfilesNumSlowExec += diff
				}
				simplelog.Infof("setting default values for slow execution profiles: %v, recent error profiles %v, slow planning profiles %v, high query cost profiles %v",
					defaultJobProfilesNumSlowExec,
					defaultJobProfilesNumRecentErrors,
					defaultJobProfilesNumSlowPlanning,
					defaultJobProfilesNumHighQueryCost)
			}

			// job profile specific numbers
			viper.SetDefault("job-profiles-num-high-query-cost", defaultJobProfilesNumHighQueryCost)
			viper.SetDefault("job-profiles-num-slow-exec", defaultJobProfilesNumSlowExec)
			viper.SetDefault("job-profiles-num-recent-errors", defaultJobProfilesNumSlowPlanning)
			viper.SetDefault("job-profiles-num-slow-planning", defaultJobProfilesNumHighQueryCost)
			jobProfilesNumHighQueryCost = viper.GetInt("job-profiles-num-high-query-cost")
			if jobProfilesNumHighQueryCost != defaultJobProfilesNumHighQueryCost {
				simplelog.Warningf("job-profiles-num-high-query-cost changed to %v by configuration", jobProfilesNumHighQueryCost)
			}
			jobProfilesNumSlowExec = viper.GetInt("job-profiles-num-slow-exec")
			if jobProfilesNumSlowExec != defaultJobProfilesNumSlowExec {
				simplelog.Warningf("job-profiles-num-slow-exec changed to %v by configuration", jobProfilesNumSlowExec)
			}
			jobProfilesNumRecentErrors = viper.GetInt("job-profiles-num-recent-errors")
			if jobProfilesNumRecentErrors != defaultJobProfilesNumRecentErrors {
				simplelog.Warningf("job-profiles-num-recent-errors changed to %v by configuration", jobProfilesNumRecentErrors)
			}
			jobProfilesNumSlowPlanning = viper.GetInt("job-profiles-num-slow-planning")
			if jobProfilesNumSlowPlanning != defaultJobProfilesNumSlowPlanning {
				simplelog.Warningf("job-profiles-num-slow-planning changed to %v by configuration", jobProfilesNumSlowPlanning)
			}
			totalAllocated := defaultJobProfilesNumSlowExec + defaultJobProfilesNumRecentErrors + defaultJobProfilesNumSlowPlanning + defaultJobProfilesNumHighQueryCost
			if totalAllocated > 0 && totalAllocated != numberJobProfilesToCollect {
				numberJobProfilesToCollect = totalAllocated
				simplelog.Warningf("due to configuration parameters new total jobs profiles collected has been adjusted to %v", totalAllocated)
			}
		}
	},
	Run: func(cmd *cobra.Command, args []string) {

		simplelog.Infof("searching for the following optional configuration files in the current directory %v", strings.Join(confFiles, ", "))
		if !configIsFound {
			simplelog.Warningf("unable to read any of the valid config file formats (%v) due to error '%v' - falling back to defaults, command line flags and environment variables", strings.Join(supportedExtensions, ","), unableToReadConfigError)
		} else {
			simplelog.Infof("INFO: found config file %v", foundConfig)
		}
		if !acceptCollectionConsent {
			fmt.Println(outputConsent())
			os.Exit(1)
		}
		//check if required flags are set
		requiredFlags := []string{"dremio-endpoint", "dremio-username"}

		failed := false
		for _, flag := range requiredFlags {
			if viper.GetString(flag) == "" {
				simplelog.Errorf("required flag '--%s' not set", flag)
				failed = true
			}
		}
		if failed {
			err := cmd.Usage()
			if err != nil {
				simplelog.Errorf("unable to even print usage, this is critical report this bug %v", err)
				os.Exit(1)
			}
			os.Exit(1)
		}

		// Run application
		simplelog.Info("Starting collection...")
		collect(numberThreads)
		ddcLoc, err := os.Executable()
		if err != nil {
			simplelog.Warningf("unable to find ddc itself..so can't copy it's log due to error %v", err)
		} else {
			ddcDir := path.Dir(ddcLoc)
			if err := copyFile(filepath.Join(ddcDir, "ddc.log"), path.Join(outputDir, fmt.Sprintf("ddc-%v.log", nodeName))); err != nil {
				simplelog.Warningf("uanble to copy log to archive due to error %v", err)
			}
		}
		tarballName := outputDir + nodeName + ".tar.gz"
		simplelog.Infof("collection complete. Archiving %v to %v...", outputDir, tarballName)
		if err := TarGzDir(outputDir, tarballName); err != nil {
			simplelog.Errorf("unable to compress archive exiting due to error %v", err)
			os.Exit(1)
		}
		simplelog.Infof("Archive %v complete", tarballName)
	},
}

func TarGzDir(srcDir, dest string) error {
	tarGzFile, err := os.Create(path.Clean(dest))
	if err != nil {
		return err
	}
	defer tarGzFile.Close()

	gzWriter := gzip.NewWriter(tarGzFile)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// Make sure the srcDir is an absolute path and ends with '/'
	srcDir = filepath.Clean(srcDir) + string(filepath.Separator)

	return filepath.Walk(srcDir, func(filePath string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get the relative path of the file
		relativePath, err := filepath.Rel(srcDir, filePath)
		if err != nil {
			return err
		}

		// Make sure the relative path starts with './'
		if !strings.HasPrefix(relativePath, ".") {
			relativePath = "./" + relativePath
		}

		header, err := tar.FileInfoHeader(fileInfo, relativePath)
		if err != nil {
			return err
		}

		header.Name = relativePath

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		if !fileInfo.IsDir() {
			file, err := os.Open(path.Clean(filePath))
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(tarWriter, file)
			return err
		}

		return nil
	})
}

func getThreads(cpus int) int {
	numCPU := math.Round(float64(cpus / 2.0))
	return int(math.Max(numCPU, 2))
}

func getOutputDir(now time.Time) string {
	nowStr := now.Format("20060102-150405")
	return filepath.Join(os.TempDir(), "ddc", nowStr)
}

func getDremioPID() (int, error) {
	var dremioPIDOutput bytes.Buffer
	if err := ddcio.Shell(&dremioPIDOutput, "jps | grep DremioDaemon | awk '{print $1}'"); err != nil {
		simplelog.Warningf("Error trying to unlock commercial features %v. Note: newer versions of OpenJDK do not support the call VM.unlock_commercial_features. This is usually safe to ignore", err)
	}
	dremioIDString := strings.TrimSpace(dremioPIDOutput.String())
	dremioPID, err := strconv.Atoi(dremioIDString)
	if err != nil {
		return -1, fmt.Errorf("unable to parse pid from text '%v' due to error %v", dremioIDString, err)
	}
	return dremioPID, nil
}

func init() {
	rootCmd.AddCommand(localCollectCmd)
	// consent form
	localCollectCmd.Flags().BoolVar(&acceptCollectionConsent, "accept-collection-consent", false, "consent for collection of files, if not true, then collection will stop and a log message will be generated")
	if err := viper.BindPFlag("accept-collection-consent", localCollectCmd.Flags().Lookup("accept-collection-consent")); err != nil {
		simplelog.Errorf("unable to bind flag due to error %v", err)
	}

	// command line flags ..default is set at runtime due to the CountVarP not having this capacity
	localCollectCmd.Flags().CountVarP(&verbose, "verbose", "v", "Logging verbosity")
	if err := viper.BindPFlag("verbose", localCollectCmd.Flags().Lookup("verbose")); err != nil {
		simplelog.Errorf("unable to bind configuration for verbose to error: %v", err)
	}

	localCollectCmd.Flags().BoolVar(&collectAccelerationLogs, "collect-acceleration-log", false, "Run the Collect Acceleration Log collector")
	if err := viper.BindPFlag("collect-acceleration-log", localCollectCmd.Flags().Lookup("collect-acceleration-log")); err != nil {
		simplelog.Errorf("unable to bind flag due to error %v", err)
	}

	localCollectCmd.Flags().BoolVar(&collectAccessLogs, "collect-access-log", false, "Run the Collect Access Log collector")
	if err := viper.BindPFlag("collect-access-log", localCollectCmd.Flags().Lookup("collect-access-log")); err != nil {
		simplelog.Errorf("unable to bind flag due to error %v", err)
	}

	//TODO detect gc log dir
	localCollectCmd.Flags().StringVar(&gcLogsDir, "dremio-gclogs-dir", "", "by default will read from the Xloggc flag, otherwise you can override it here")
	if err := viper.BindPFlag("dremio-gclogs-dir", localCollectCmd.Flags().Lookup("dremio-gclogs-dir")); err != nil {
		simplelog.Errorf("unable to bind configuration for dremio-gclogs-dir to error: %v", err)
	}

	//TODO detect log dir
	localCollectCmd.Flags().StringVar(&dremioLogDir, "dremio-log-dir", "/var/log/dremio", "directory with application logs on dremio")
	if err := viper.BindPFlag("dremio-log-dir", localCollectCmd.Flags().Lookup("dremio-log-dir")); err != nil {
		simplelog.Errorf("unable to bind configuration for dremio-log-dir to error: %v", err)
	}

	defaultThreads := getThreads(runtime.NumCPU())
	localCollectCmd.Flags().IntVarP(&numberThreads, "number-threads", "t", defaultThreads, "control concurrency in the system")
	if err := viper.BindPFlag("number-threads", localCollectCmd.Flags().Lookup("number-threads")); err != nil {
		simplelog.Errorf("unable to bind configuration for number-threads to error: %v", err)
	}

	// Add flags for Dremio connection information
	localCollectCmd.Flags().StringVar(&dremioEndpoint, "dremio-endpoint", "http://localhost:9047", "Dremio REST API endpoint")
	if err := viper.BindPFlag("dremio-endpoint", localCollectCmd.Flags().Lookup("dremio-endpoint")); err != nil {
		simplelog.Errorf("unable to bind configuration for dremio-endpoint to error: %v", err)
	}

	localCollectCmd.Flags().StringVar(&dremioUsername, "dremio-username", "dremio", "Dremio username")
	if err := viper.BindPFlag("dremio-username", localCollectCmd.Flags().Lookup("dremio-username")); err != nil {
		simplelog.Errorf("unable to bind configuration for dremio-username to error: %v", err)
	}

	localCollectCmd.Flags().StringVar(&dremioPATToken, "dremio-pat-token", "", "Dremio Personal Access Token (PAT)")
	if err := viper.BindPFlag("dremio-pat-token", localCollectCmd.Flags().Lookup("dremio-pat-token")); err != nil {
		simplelog.Errorf("unable to bind configuration for dremio-pat-token to error: %v", err)
	}

	localCollectCmd.Flags().StringVar(&dremioRocksDBDir, "dremio-rocksdb-dir", "/opt/dremio/data/db", "Path to Dremio RocksDB directory")
	if err := viper.BindPFlag("dremio-rocksdb-dir", localCollectCmd.Flags().Lookup("dremio-rocksdb-dir")); err != nil {
		simplelog.Errorf("unable to bind flag due to error %v", err)
	}

	localCollectCmd.Flags().BoolVar(&collectDremioConfiguration, "collect-dremio-configuration", true, "Collect Dremio Configuration collector")
	if err := viper.BindPFlag("collect-dremio-configuration", localCollectCmd.Flags().Lookup("collect-dremio-configuration")); err != nil {
		simplelog.Errorf("unable to bind flag due to error %v", err)
	}

	localCollectCmd.Flags().IntVar(&numberJobProfilesToCollect, "number-job-profiles", 25000, "Randomly retrieve number job profiles from the server based on queries.json data but must have --dremio-pat-token set to use")
	if err := viper.BindPFlag("number-job-profiles", localCollectCmd.Flags().Lookup("number-job-profiles")); err != nil {
		simplelog.Errorf("unable to bind flag due to error %v", err)
	}

	localCollectCmd.Flags().BoolVar(&captureHeapDump, "capture-heap-dump", false, "Run the Heap Dump collector")
	if err := viper.BindPFlag("capture-heap-dump", localCollectCmd.Flags().Lookup("capture-heap-dump")); err != nil {
		simplelog.Errorf("unable to bind flag due to error %v", err)
	}

}

// ### Helper functions
func validateAPICredentials() error {
	simplelog.Info("Validating REST API user credentials...")
	url := dremioEndpoint + "/apiv2/login"
	headers := map[string]string{"Content-Type": "application/json"}
	_, err := apiRequest(url, dremioPATToken, "GET", headers)
	return err
}

func apiRequest(url string, pat string, request string, headers map[string]string) ([]byte, error) {
	simplelog.Infof("Requesting %s", url)
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
	simplelog.Info("Collecting sys." + systable)

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
