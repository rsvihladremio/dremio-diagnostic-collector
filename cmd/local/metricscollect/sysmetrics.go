package metricscollect

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/rsvihladremio/dremio-diagnostic-collector/cmd/simplelog"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

type SystemMetricsRow struct {
	UserCPUPercent      float64 `json:"userCPUPercent"`
	SystemCPUPercent    float64 `json:"systmeCPUPercent"`
	IdleCPUPercent      float64 `json:"idleCPUPercent"`
	NiceCPUPercent      float64 `json:"niceCPUPercent"`
	IOWaitCPUPercent    float64 `json:"ioWaitCPUPercent"`
	IRQCPUPercent       float64 `json:"irqCPUPercent"`
	SoftIRQCPUPercent   float64 `json:"softIRQCPUPercent"`
	StealCPUPercent     float64 `json:"stealCPUPercent"`
	GuestCPUPercent     float64 `json:"guestCPUPercent"`
	GuestNiceCPUPercent float64 `json:"guestCPUNicePercent"`
	QueueDepth          float64 `json:"queueDepth"`
	DiskLatency         float64 `json:"diskLatency"`
	ReadBytes           int64   `json:"readBytes"`
	WriteBytes          int64   `json:"writeBytes"`
	FreeRAMMB           float64 `json:"freeRAMMB"`
	CachedRAMMB         float64 `json:"cachedRAMMB"`
}

func writeSystemMetrics(nodeMetricsFile string, header string, addRow func(SystemMetricsRow) (string, error)) error {
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
		total := getTotalTime(cpuTimes[0])
		queueDepth := float64(weightedIOTime) / 1000
		diskLatency := float64(weightedIOTime) / float64(totalIOs)
		memoreFreeMB := float64(memoryInfo.Free) / (1024 * 1024)
		memoryCachedMB := float64(memoryInfo.Cached) / (1024 * 1024)

		row := SystemMetricsRow{
			UserCPUPercent:      (cpuTimes[0].User / total) * 100,
			SystemCPUPercent:    (cpuTimes[0].System / total) * 100,
			IdleCPUPercent:      (cpuTimes[0].Idle / total) * 100,
			NiceCPUPercent:      (cpuTimes[0].Nice / total) * 100,
			IOWaitCPUPercent:    (cpuTimes[0].Iowait / total) * 100,
			IRQCPUPercent:       (cpuTimes[0].Irq / total) * 100,
			SoftIRQCPUPercent:   (cpuTimes[0].Softirq / total) * 100,
			StealCPUPercent:     (cpuTimes[0].Steal / total) * 100,
			GuestCPUPercent:     (cpuTimes[0].Guest / total) * 100,
			GuestNiceCPUPercent: (cpuTimes[0].GuestNice / total) * 100,

			DiskLatency: diskLatency,
			QueueDepth:  queueDepth,
			FreeRAMMB:   memoreFreeMB,
			CachedRAMMB: memoryCachedMB,
		}
		rowString, err := addRow(row)
		if err != nil {
			return fmt.Errorf("unable to convert row %#v into string due to error %v", row, err)
		}
		// Output
		_, err = w.Write([]byte(rowString + "\n"))
		if err != nil {
			return fmt.Errorf("unable to write output string %v due to %v", row, err)
		}
	}
	return nil
}

func addRow(row SystemMetricsRow) (string, error) {
	otherCPU := row.NiceCPUPercent + row.IRQCPUPercent + row.SoftIRQCPUPercent + row.StealCPUPercent + row.GuestCPUPercent + row.GuestNiceCPUPercent
	var readBytesMB, writeBytesMB, freeRAMGB float64
	if row.ReadBytes > 0 {
		readBytesMB = float64(row.ReadBytes) / (1024 * 1024)
	}
	if row.WriteBytes > 0 {
		writeBytesMB = float64(row.WriteBytes) / (1024 * 1024)
	}
	if row.FreeRAMMB > 0 {
		freeRAMGB = float64(row.FreeRAMMB) / 1024.0
	}
	return fmt.Sprintf("%s\t%.2f%%\t\t%.2f%%\t\t%.2f%%\t\t%.2f%%\t\t%.2f\t\t%.2f\t\t%.2f\\t\t%.2f\t\t%.2f\n",
		time.Now().Format("15:04:05"), row.UserCPUPercent, row.SystemCPUPercent, row.IOWaitCPUPercent, otherCPU, row.QueueDepth, row.DiskLatency, readBytesMB, writeBytesMB, freeRAMGB), nil
}
func SystemMetricsToLog(nodeMetricsFileLoc string) error {
	header := fmt.Sprintf("Time\t\tUser %%\t\tSystem %%\t\tIdle %%\t\tIO Wait%%\t\tOther %%\t\tQueue Depth\tDisk Latency (ms)\tDisk Read (KB/s)\tDisk Write (KB/s)\tFree Mem (MB)\n")
	return writeSystemMetrics(nodeMetricsFileLoc, header, addRow)
}

func SystemMetricsToJSON(nodeMetricsFile string) error {
	header := ""
	addRow := func(row SystemMetricsRow) (string, error) {
		str, err := json.Marshal(&row)
		if err != nil {
			return "", fmt.Errorf("unable to marshal row %#v due to error %v", row, err)
		}
		return string(str), nil
	}
	return writeSystemMetrics(nodeMetricsFile, header, addRow)
}

func getTotalTime(c cpu.TimesStat) float64 {
	return c.User + c.System + c.Idle + c.Nice + c.Iowait + c.Irq +
		c.Softirq + c.Steal + c.Guest + c.GuestNice
}
