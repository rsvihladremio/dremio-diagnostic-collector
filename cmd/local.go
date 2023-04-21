/*
Copyright 2023 Dremio

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
package cmd

import (
	"flag"
	"fmt"
	"io"
	"math"
	"os"
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
)

var (
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
	dremioQueryAnalyzerNumDays     int
	dremioMasterLogsNumDays        int
	dremioExecutorLogsNumDays      int
	dremioLogDir                   string
	dremioGCFilePattern            string
	dremioRocksDBDir               string
	isKubernetes                   bool
	kubernetesNamespace            string
	skipDremioCloner               bool
	skipQueryAnalyzer              bool
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
	skipHeapDumpCoordinator        bool
	skipHeapDumpExecutor           bool
	skipJFRCoordinator             bool
	skipJFRExecutor                bool
	dremioJFRTimeSeconds           int
	jobProfilesNumDays             int
	jobProfilesNumSlowExec         int
	jobProfilesNumHighQueryCost    int
	jobProfilesNumSlowPlanning     int
	jobProfilesNumRecentErrors     int
	skipPrometheusExport           bool
	prometheusEndpoint             string
	prometheusNumDays              int
	prometheusDremioCoordFilter    string
	prometheusDremioExecFilter     string
	prometheusChunkSizeHours       int
	configurationOutDir            = path.Join(outputDir, "configuration", nodeName)
	jfrOutDir                      = path.Join(outputDir, "jfr")
	threadDumpsOutDir              = path.Join(outputDir, "jfr", "thread-dumps")
	heapDumpsOutDir                = path.Join(outputDir, "heap-dumps")
	promOutDir                     = path.Join(outputDir, "prometheus")
	jobProfilesOutDir              = path.Join(outputDir, "job-profiles", nodeName)
	kubernetesOutDir               = path.Join(outputDir, "kubernetes")
	kvstoreOutDir                  = path.Join(outputDir, "kvstore")
	logsOutDir                     = path.Join(outputDir, "logs", nodeName)
	nodeInfoOutDir                 = path.Join(outputDir, "node-info", nodeName)
	queriesOutDir                  = path.Join(outputDir, "queries", nodeName)
	systemTablesOutDir             = path.Join(outputDir, "system-tables")
	wlmOutDir                      = path.Join(outputDir, "wlm")
)

type ThreadPool struct {
	semaphore chan bool
}

func NewThreadPool(numberThreads int) *ThreadPool {
	semaphore := make(chan bool, numberThreads)
	return &ThreadPool{
		semaphore: semaphore,
	}
}

// FireJob launches a func() up to the number of threads allowed by the thread pool
func (t *ThreadPool) FireJob(job func() error) {
	go func() {
		//aquire a lock by sending a value to the channel (can be any value)
		t.semaphore <- true
		defer func() {
			<-t.semaphore // Release semaphore slot.
		}()
		//execute the job
		err := job()
		if err != nil {
			glog.Error(err)
		}
	}()
}

// Wait waits for goroutines to finish by acquiring all slots.
func (t *ThreadPool) Wait() {
	for i := 0; i < cap(t.semaphore); i++ {
		t.semaphore <- true
	}
}

func createAllDirs() error {
	if err := os.MkdirAll(configurationOutDir, 0755); err != nil {
		return fmt.Errorf("unable to create configuration directory due to error %v", err)
	}
	if err := os.MkdirAll(jfrOutDir, 0755); err != nil {
		return fmt.Errorf("unable to create jfr directory due to error %v", err)
	}
	if err := os.MkdirAll(threadDumpsOutDir, 0755); err != nil {
		return fmt.Errorf("unable to create thread-dumps directory due to error %v", err)
	}
	if err := os.MkdirAll(heapDumpsOutDir, 0755); err != nil {
		return fmt.Errorf("unable to create heap-dumps directory due to error %v", err)
	}
	if err := os.MkdirAll(promOutDir, 0755); err != nil {
		return fmt.Errorf("unable to create prometheus directory due to error %v", err)
	}
	if err := os.MkdirAll(jobProfilesOutDir, 0755); err != nil {
		return fmt.Errorf("unable to create job-profiles directory due to error %v", err)
	}
	if err := os.MkdirAll(kubernetesOutDir, 0755); err != nil {
		return fmt.Errorf("unable to create kubernetes directory due to error %v", err)
	}
	if err := os.MkdirAll(kvstoreOutDir, 0755); err != nil {
		return fmt.Errorf("unable to create kvstore directory due to error %v", err)
	}
	if err := os.MkdirAll(logsOutDir, 0755); err != nil {
		return fmt.Errorf("unable to create logs directory due to error %v", err)
	}
	if err := os.MkdirAll(nodeInfoOutDir, 0755); err != nil {
		return fmt.Errorf("unable to create node-info directory due to error %v", err)
	}
	if err := os.MkdirAll(queriesOutDir, 0755); err != nil {
		return fmt.Errorf("unable to create queries directory due to error %v", err)
	}
	if err := os.MkdirAll(systemTablesOutDir, 0755); err != nil {
		return fmt.Errorf("unable to create system-tables directory due to error %v", err)
	}
	if err := os.MkdirAll(wlmOutDir, 0755); err != nil {
		return fmt.Errorf("unable to create wlm directory due to error %v", err)
	}
	return nil
}

func collect(numberThreads int) {
	if err := createAllDirs(); err != nil {
		fmt.Printf("unable to create directories due to error %v\n", err)
		os.Exit(1)
	}
	t := NewThreadPool(numberThreads)
	t.FireJob(collectSystemConfig)
	t.FireJob(collectJvmConfig)
	t.FireJob(collectDremioConfig)
	t.FireJob(collectDiskUsage)
	t.FireJob(collectNodeMetrics)
	t.FireJob(collectPromMetrics)
	t.FireJob(collectJfr)
	t.FireJob(collectJstacks)
	t.FireJob(collectKvReport)
	t.FireJob(collectWlm)
	t.FireJob(collectK8sConfig)
	t.FireJob(collectHeapDump)
	t.FireJob(collectDremioSystemTables)
	t.FireJob(collectQueriesJson)
	t.FireJob(collectJobProfiles)
	t.FireJob(collectDremioServerLog)
	t.FireJob(collectGcLogs)
	t.FireJob(collectMetadataRefreshLog)
	t.FireJob(collectReflectionLog)
	t.FireJob(collectAccelerationLog)
	t.FireJob(collectDremioAccessLog)
	t.FireJob(collectDremioQueryAnalyzer)
	t.FireJob(collectDremioCloner)
	t.Wait()
}

func collectDremioConfig() error {
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
		if err := copyFile(file, logsOutDir); err != nil {
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

func collectPromMetrics() error {
	return nil
}

func collectJfr() error {
	return nil
}

func collectJstacks() error {
	return nil
}

func collectKvReport() error {
	return nil
}

func collectWlm() error {
	return nil
}

func collectHeapDump() error {
	return nil
}

func collectQueriesJson() error {
	return nil
}

func collectJobProfiles() error {
	return nil
}

func collectDremioServerLog() error {
	return nil
}

func collectK8sConfig() error {
	return nil
}

func collectDremioSystemTables() error {
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
func collectDremioQueryAnalyzer() error {
	return nil
}

func collectDremioCloner() error {
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

func init() {
	// command line flags

	//make default tmp directory
	now := time.Now()
	tmpDir := getOutputDir(now)
	localCollectCmd.Flags().StringVarP(&outputDir, "tmp-output-dir", "o", tmpDir, "temporary output directory for log collection")
	// set node name
	hostName, err := os.Hostname()
	if err != nil {
		hostName = fmt.Sprintf("unknown-%v", uuid.New())
	}
	localCollectCmd.Flags().StringVarP(&nodeName, "node-name", "n", hostName, "name to give to the node")
	localCollectCmd.Flags().StringVar(&logDir, "collect-log-dir", "", "logging output directory for the collector")
	localCollectCmd.Flags().StringVar(&gcLogsDir, "dremio-gclogs-dir", "/var/log/dremio", "directory with gc logs on dremio")
	localCollectCmd.Flags().StringVar(&dremioLogsDir, "dremio-log-dir", "/var/log/dremio", "directory with application logs on dremio")
	localCollectCmd.Flags().CountVarP(&verbose, "verbose", "v", "Logging verbosity")
	defaultThreads := getThreads(runtime.NumCPU())
	localCollectCmd.Flags().IntVarP(&numberThreads, "number-threads", "t", defaultThreads, "control concurrency in the system")

	// Add flags for Dremio connection information
	localCollectCmd.Flags().StringVar(&dremioEndpoint, "dremio-endpoint", "http://dremio-client:9047", "Dremio REST API endpoint")
	localCollectCmd.Flags().StringVar(&dremioUsername, "dremio-username", "<DREMIO_ADMIN_USER>", "Dremio username")
	localCollectCmd.Flags().StringVar(&dremioPATToken, "dremio-pat-token", "<DREMIO_PAT>", "Dremio Personal Access Token (PAT)")
	localCollectCmd.Flags().StringVar(&dremioStorageType, "dremio-storage-type", "adls", "Dremio storage type (adls, s3, azure, or hdfs)")

	// Add flags for AWS information
	localCollectCmd.Flags().StringVar(&awsAccessKeyID, "aws-access-key-id", "NOTSET", "AWS Access Key ID")
	localCollectCmd.Flags().StringVar(&awsSecretAccessKey, "aws-secret-access-key", "NOTSET", "AWS Secret Access Key")
	localCollectCmd.Flags().StringVar(&awsS3Path, "aws-s3-path", "NOTSET", "S3 path for Dremio data")

	// Add flags for Azure information
	localCollectCmd.Flags().StringVar(&azureSASURL, "azure-sas-url", "<AZURE_SAS_URL>", "Azure SAS URL for Dremio data")

	// Add flags for Dremio diagnostic collection options
	localCollectCmd.Flags().IntVar(&dremioQueryAnalyzerNumDays, "dremio-query-analyzer-num-days", 28, "Number of days of query history to collect for the Query Analyzer collector")
	localCollectCmd.Flags().IntVar(&dremioMasterLogsNumDays, "dremio-master-logs-num-days", 3, "Number of days of Dremio master logs to collect for the Master Logs collector")
	localCollectCmd.Flags().IntVar(&dremioExecutorLogsNumDays, "dremio-executor-logs-num-days", 3, "Number of days of Dremio executor logs to collect for the Executor Logs collector")
	//localCollectCmd.Flags().StringVar(&dremioLogDir, "dremio-log-dir", "/opt/dremio/data/log", "Path to Dremio log directory")
	localCollectCmd.Flags().StringVar(&dremioGCFilePattern, "dremio-gc-file-pattern", "gc*.log", "File pattern to match for Dremio GC logs")
	localCollectCmd.Flags().StringVar(&dremioRocksDBDir, "dremio-rocksdb-dir", "/opt/dremio/data/db", "Path to Dremio RocksDB directory")

	// Add flags for Kubernetes information
	localCollectCmd.Flags().BoolVar(&isKubernetes, "is-kubernetes", false, "Set to true if running in a Kubernetes environment")

	localCollectCmd.Flags().StringVar(&kubernetesNamespace, "kubernetes namespace", "default", "Kubernetes namespace")
	// Add flags for skipping collectors
	localCollectCmd.Flags().BoolVar(&skipDremioCloner, "skip-dremio-cloner", true, "Skip the Dremio Cloner collector")
	localCollectCmd.Flags().BoolVar(&skipQueryAnalyzer, "skip-query-analyzer", false, "Skip the Query Analyzer collector")
	localCollectCmd.Flags().BoolVar(&skipExportSystemTables, "skip-export-system-tables", false, "Skip the Export System Tables collector")
	localCollectCmd.Flags().BoolVar(&skipCollectDiskUsage, "skip-collect-disk-usage", false, "Skip the Collect Disk Usage collector")
	localCollectCmd.Flags().BoolVar(&skipDownloadJobProfiles, "skip-download-job-profiles", false, "Skip the Download Job Profiles collector")
	localCollectCmd.Flags().BoolVar(&skipCollectQueriesJSON, "skip-collect-queries-json", false, "Skip the Collect Queries JSON collector")
	localCollectCmd.Flags().BoolVar(&skipCollectKubernetesInfo, "skip-collect-kubernetes-info", true, "Skip the Collect Kubernetes Info collector")
	localCollectCmd.Flags().BoolVar(&skipCollectDremioConfiguration, "skip-collect-dremio-configuration", false, "Skip the Collect Dremio Configuration collector")
	localCollectCmd.Flags().BoolVar(&skipCollectKVStoreReport, "skip-collect-kvstore-report", false, "Skip the Collect KVStore Report collector")
	localCollectCmd.Flags().BoolVar(&skipCollectServerLogs, "skip-collect-server-logs", false, "Skip the Collect Server Logs collector")
	localCollectCmd.Flags().BoolVar(&skipCollectMetaRefreshLog, "skip-collect-meta-refresh-log", false, "Skip the Collect Meta Refresh Log collector")
	localCollectCmd.Flags().BoolVar(&skipCollectReflectionLog, "skip-collect-reflection-log", false, "Skip the Collect Reflection Log collector")
	localCollectCmd.Flags().BoolVar(&skipCollectAccelerationLog, "skip-collect-acceleration-log", true, "Skip the Collect Acceleration Log collector")
	localCollectCmd.Flags().BoolVar(&skipCollectAccessLog, "skip-collect-access-log", false, "Skip the Collect Access Log collector")
	localCollectCmd.Flags().BoolVar(&skipCollectGCLogs, "skip-collect-gc-logs", false, "Skip the Collect GC Logs collector")
	localCollectCmd.Flags().BoolVar(&skipCollectWLM, "skip-collect-wlm", false, "Skip the Collect WLM collector")
	localCollectCmd.Flags().BoolVar(&skipHeapDumpCoordinator, "skip-heap-dump-coordinator", true, "Skip the Heap Dump Coordinator collector")
	localCollectCmd.Flags().BoolVar(&skipHeapDumpExecutor, "skip-heap-dump-executor", true, "Skip the Heap Dump Executor collector")
	localCollectCmd.Flags().BoolVar(&skipJFRCoordinator, "skip-jfr-coordinator", true, "Skip the JFR Coordinator collector")
	localCollectCmd.Flags().BoolVar(&skipJFRExecutor, "skip-jfr-executor", true, "Skip the JFR Executor collector")

	// Add flags for other options
	localCollectCmd.Flags().IntVar(&dremioJFRTimeSeconds, "dremio-jfr-time-seconds", 300, "Duration in seconds to run the JFR collector")
	localCollectCmd.Flags().IntVar(&jobProfilesNumDays, "job-profiles-num-days", 28, "Number of days of job profile history to collect")
	localCollectCmd.Flags().IntVar(&jobProfilesNumSlowExec, "job-profiles-num-slow-exec", 10000, "Number of slowest job profiles to collect by execution time")
	localCollectCmd.Flags().IntVar(&jobProfilesNumHighQueryCost, "job-profiles-num-high-query-cost", 5000, "Number of job profiles to collect with the highest query cost")
	localCollectCmd.Flags().IntVar(&jobProfilesNumSlowPlanning, "job-profiles-num-slow-planning", 5000, "Number of slowest job profiles to collect by planning time")
	localCollectCmd.Flags().IntVar(&jobProfilesNumRecentErrors, "job-profiles-num-recent-errors", 5000, "Number of most recent job profiles to collect with errors")
	localCollectCmd.Flags().BoolVar(&skipPrometheusExport, "skip-prometheus-export", true, "Skip exporting results to Prometheus")
	localCollectCmd.Flags().StringVar(&prometheusEndpoint, "prometheus-endpoint", "http://localhost:9090", "Prometheus endpoint")
	localCollectCmd.Flags().IntVar(&prometheusNumDays, "prometheus-num-days", 28, "Number of days of data to export to Prometheus")
	localCollectCmd.Flags().StringVar(&prometheusDremioCoordFilter, "prometheus-dremio-coord-filter", "{container='dremio-master-coordinator'}", "Prometheus filter expression for Dremio master coordinator metrics")
	localCollectCmd.Flags().StringVar(&prometheusDremioExecFilter, "prometheus-dremio-exec-filter", "{container='dremio-executor'}", "Prometheus filter expression for Dremio executor metrics")
	localCollectCmd.Flags().IntVar(&prometheusChunkSizeHours, "prometheus-chunk-size-hours", 6, "Chunk size in hours for exporting data to Prometheus")

	// Mark required flags
	localCollectCmd.MarkFlagRequired("dremio-endpoint")
	localCollectCmd.MarkFlagRequired("dremio-username")
	localCollectCmd.MarkFlagRequired("dremio-pat-token")
	localCollectCmd.MarkFlagRequired("dremio-storage-type")

	// Set glog flags
	flag.Set("log_dir", logDir)
	flag.Set("v", strconv.Itoa(verbose))
}
