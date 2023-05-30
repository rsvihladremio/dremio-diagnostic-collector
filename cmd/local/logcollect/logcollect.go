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

// package logcollect contains the logic for log collection in the local-collect sub command
package logcollect

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/rsvihladremio/dremio-diagnostic-collector/cmd/ddcio"
	"github.com/rsvihladremio/dremio-diagnostic-collector/cmd/simplelog"
)

type LogCollector struct {
	dremioGCFilePattern      string
	dremioLogDir             string
	logsOutDir               string
	gcLogsDir                string
	queriesOutDir            string
	dremioLogsNumDays        int
	dremioQueriesJSONNumDays int
}

func NewLogCollector(dremioLogDir, logsOutDir, gcLogsDir, dremioGCFilePattern, queriesOutDir string, dremioQueriesJSONNumDays, dremioLogsNumDays int) *LogCollector {
	return &LogCollector{
		dremioLogDir:             dremioLogDir,
		logsOutDir:               logsOutDir,
		dremioLogsNumDays:        dremioLogsNumDays,
		dremioQueriesJSONNumDays: dremioQueriesJSONNumDays,
		dremioGCFilePattern:      dremioGCFilePattern,
		queriesOutDir:            queriesOutDir,
		gcLogsDir:                gcLogsDir,
	}
}

func (l *LogCollector) RunCollectDremioServerLog() error {
	simplelog.Info("Collecting GC logs ...")
	if err := l.exportArchivedLogs(l.dremioLogDir, "server.log", "server", l.dremioLogsNumDays); err != nil {
		simplelog.Errorf("trying to archive server logs we got error: %v", err)
	}
	simplelog.Info("... collecting server.out")
	src := path.Join(l.dremioLogDir, "server.out")
	dest := path.Join(l.logsOutDir, "server.out")
	if err := ddcio.CopyFile(path.Clean(src), path.Clean(dest)); err != nil {
		return fmt.Errorf("unable to copy %v to %v due to error %v", src, dest, err)
	}
	simplelog.Info("... collecting server logs COMPLETED")

	return nil
}

func (l *LogCollector) RunCollectGcLogs() error {
	simplelog.Info("Collecting GC logs ...")
	files, err := os.ReadDir(path.Clean(l.gcLogsDir))
	if err != nil {
		return fmt.Errorf("error reading directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		matched, err := filepath.Match(l.dremioGCFilePattern, file.Name())
		if err != nil {
			simplelog.Errorf("error matching file pattern %v with error '%v'", l.dremioGCFilePattern, err)
		}
		if matched {
			srcPath := filepath.Join(l.gcLogsDir, file.Name())
			destPath := filepath.Join(l.logsOutDir, file.Name())
			if err := ddcio.CopyFile(path.Clean(srcPath), path.Clean(destPath)); err != nil {
				return fmt.Errorf("error copying file %s: %w", file.Name(), err)
			}
			simplelog.Debugf("Copied file %s to %s", srcPath, destPath)
		}
	}
	simplelog.Warning("GC logs from executors and scale-out coordinators must be collected separately!")
	simplelog.Info("... collecting GC logs COMPLETED")

	return nil
}

func (l *LogCollector) RunCollectMetadataRefreshLogs() error {
	simplelog.Info("Collecting metadata refresh logs from Coordinator(s) ...")
	if err := l.exportArchivedLogs(l.dremioLogDir, "metadata_refresh.log", "metadata_refresh", l.dremioLogsNumDays); err != nil {
		return fmt.Errorf("unable to collect metadata refresh logs due to error %v", err)
	}
	simplelog.Warning("Metadata refresh logs from scale-out coordinators must be collected separately!")
	simplelog.Info("... collecting meta data refresh logs from Coordinator(s) COMPLETED")
	return nil
}

func (l *LogCollector) RunCollectReflectionLogs() error {
	simplelog.Info("Collecting reflection logs from Coordinator(s) ...")
	if err := l.exportArchivedLogs(l.dremioLogDir, "reflection.log", "reflection", l.dremioLogsNumDays); err != nil {
		return fmt.Errorf("unable to collect reflection logs due to error %v", err)
	}
	simplelog.Info("... collecting reflection logs from Coordinator(s) COMPLETED")

	return nil
}

func (l *LogCollector) RunCollectDremioAccessLogs() error {
	simplelog.Info("Collecting access logs from Coordinator(s) ...")
	simplelog.Warning("Access logs from scale-out coordinators must be collected separately!")
	if err := l.exportArchivedLogs(l.dremioLogDir, "access.log", "access", l.dremioLogsNumDays); err != nil {
		return fmt.Errorf("unable to archive access.logs due to error %v", err)
	}
	simplelog.Info("... collecting access logs from Coordinator(s) COMPLETED")

	return nil
}

func (l *LogCollector) RunCollectAccelerationLogs() error {
	simplelog.Info("Collecting acceleration logs from Coordinator(s) ...")
	simplelog.Warning("Acceleration logs from scale-out coordinators must be collected separately!")
	if err := l.exportArchivedLogs(l.dremioLogDir, "acceleration.log", "acceleration", l.dremioLogsNumDays); err != nil {
		return fmt.Errorf("unable to archive acceleration.logs due to error %v", err)
	}
	simplelog.Info("... collecting acceleragtion logs from Coordinator(s) COMPLETED")

	return nil
}

func (l *LogCollector) RunCollectQueriesJSON() error {
	simplelog.Info("Collecting queries.json ...")
	err := l.exportArchivedLogs(l.dremioLogDir, "queries.json", "queries", l.dremioQueriesJSONNumDays)
	if err != nil {
		return fmt.Errorf("failed to export archived logs: %v", err)
	}

	simplelog.Warning("Queries.json from scale-out coordinators must be collected separately!")

	simplelog.Info("... collecting Queries JSON for Job Profiles COMPLETED")
	return nil
}

func (l *LogCollector) exportArchivedLogs(logDir string, unarchivedFile string, logPrefix string, archiveDays int) error {
	src := path.Join(logDir, unarchivedFile)
	var outDir string
	if logPrefix == "queries" {
		outDir = l.queriesOutDir
	} else {
		outDir = l.logsOutDir
	}
	dest := path.Join(outDir, unarchivedFile)
	//instead of copying it we just archive it to a new location
	if err := ddcio.GzipFile(path.Clean(src), path.Clean(dest+".gz")); err != nil {
		return fmt.Errorf("archiving of log file %v failed due to error %v", unarchivedFile, err)
	}

	today := time.Now()

	for i := 0; i <= archiveDays; i++ {
		processingDate := today.AddDate(0, 0, -i).Format("2006-01-02")
		files, err := os.ReadDir(filepath.Join(logDir, "archive"))
		if err != nil {
			//no archives to read so we can skip this
			simplelog.Errorf("unable to read archive folder due to error %v", err)
			break
		}

		for _, f := range files {
			if strings.HasPrefix(f.Name(), logPrefix+"."+processingDate) && strings.HasSuffix(f.Name(), ".gz") {
				simplelog.Info("Copying archive file for " + processingDate + ": " + f.Name())
				src := filepath.Join(logDir, "archive", f.Name())
				dst := filepath.Join(outDir, f.Name())
				err := ddcio.CopyFile(path.Clean(src), path.Clean(dst))
				if err != nil {
					simplelog.Errorf("unable to copy file due to error %v", err)
				}
			}
		}
	}
	return nil
}
