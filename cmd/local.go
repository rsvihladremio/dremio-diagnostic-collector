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
	"math"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/spf13/cobra"
)

var (
	outputDir     string
	logDir        string
	verbose       int
	numberThreads int
	nodeName      string
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

func collect(numberThreads int) {
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

	header := fmt.Sprintf("Time\t\tiowait %%\t\tSteal %%\t\tIdle %%\t\tUser %%\t\tSystem %%\t\tQueue Depth\tDisk Latency (ms)\tDisk Read (KB/s)\tDisk Write (KB/s)\tFree Mem (MB)\tCached Mem (MB)\n")
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
		total := cpuTimes[0].Total()
		iowaitPercent := (cpuTimes[0].Iowait / total) * 100
		stealPercent := (cpuTimes[0].Steal / total) * 100
		idlePercent := (cpuTimes[0].Idle / total) * 100
		userPercent := (cpuTimes[0].User / total) * 100
		systemPercent := (cpuTimes[0].System / total) * 100

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
		row := fmt.Sprintf("%s\t%.2f%%\t\t%.2f%%\t\t%.2f%%\t\t%.2f%%\t\t%.2f%%\t\t%.2f\t\t%.2f\t\t%.2f\t\t%.2f\t\t%.2f\t\t%.2f\n",
			time.Now().Format("15:04:05"), iowaitPercent, stealPercent, idlePercent, userPercent, systemPercent, queueDepth, diskLatency, readBytes, writeBytes, memoryInfo.Free/(1024*1024), memoryInfo.Cached/(1024*1024))
		_, err := w.Write([]byte(row))
		if err != nil {
			return fmt.Errorf("unable to write output string %v due to %v", row, err)
		}
	}
	return nil
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

	// subcommand
	rootCmd.AddCommand(localCollectCmd) // Add localCollectCmd as a subcommand of rootCmd
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
	localCollectCmd.Flags().StringVar(&logDir, "log_dir", "", "Log directory")
	localCollectCmd.Flags().CountVarP(&verbose, "verbose", "v", "Logging verbosity")
	defaultThreads := getThreads(runtime.NumCPU())
	localCollectCmd.Flags().IntVarP(&numberThreads, "number-threads", "threads", defaultThreads, "control concurrency in the system")

	// Set glog flags
	flag.Set("log_dir", logDir)
	flag.Set("v", strconv.Itoa(verbose))
}
