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

package awselogs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	local "github.com/dremio/dremio-diagnostic-collector/v3/cmd/local"
	"github.com/dremio/dremio-diagnostic-collector/v3/cmd/local/conf"
	"github.com/dremio/dremio-diagnostic-collector/v3/pkg/archive"
	"github.com/dremio/dremio-diagnostic-collector/v3/pkg/simplelog"
	"github.com/spf13/cobra"
)

var EFSLogDir string
var OutDir string
var OutFile string
var AWSELogsCmd = &cobra.Command{
	Use:   "awselogs",
	Short: "Log only collect of AWSE from the coordinator node",
	Long:  `Log only collect of AWSE from the coordinator node`,
	Run: func(cmd *cobra.Command, args []string) {
		simplelog.LogStartMessage()
		defer simplelog.LogEndMessage()
		if err := Execute(EFSLogDir, OutDir, OutFile); err != nil {
			simplelog.Errorf("exiting %v", err)
			os.Exit(1)
		}
	},
}

func Execute(efsLogDir string, tarballOutDir string, outFile string) error {

	efsLogDir, err := filepath.Abs(efsLogDir)
	if err != nil {
		return fmt.Errorf("cannot get abs for dir %v due to error %w", efsLogDir, err)
	}
	fmt.Println("EFS dir: " + efsLogDir)

	entries, err := os.ReadDir(filepath.Join(efsLogDir, "executor"))
	if err != nil {
		return fmt.Errorf("failed listing EFS log dir: %w", err)
	}
	outDir, err := filepath.Abs(tarballOutDir)
	if err != nil {
		return fmt.Errorf("cannot get abs for dir %v due to error %w", tarballOutDir, err)
	}
	if err := os.MkdirAll(outDir, 0700); err != nil {
		return fmt.Errorf("unable to create dir %w", err)
	}
	outFile, err = filepath.Abs(outFile)
	if err != nil {
		return fmt.Errorf("cannot get abs for dir %v due to error %w", outFile, err)
	}
	defer func() {
		if err := os.RemoveAll(outDir); err != nil {
			simplelog.Warningf("unable to cleanup folder %v due to error: %v", outDir, err)
		}
	}()

	coordinatorNode := "coordinator"
	overrides := make(map[string]string)
	overrides[conf.KeyDisableRESTAPI] = "true"
	overrides[conf.KeyCollectDiskUsage] = "false"
	overrides[conf.KeyCollectJFR] = "false"
	overrides[conf.KeyCollectJStack] = "false"
	overrides[conf.KeyCaptureHeapDump] = "false"
	overrides[conf.KeyCollectJVMFlags] = "false"
	overrides[conf.KeyCollectKVStoreReport] = "false"
	overrides[conf.KeyCollectOSConfig] = "false"
	overrides[conf.KeyCollectSystemTablesExport] = "false"
	overrides[conf.KeyCollectGCLogs] = "true"
	overrides[conf.KeyCollectDremioConfiguration] = "false"
	overrides[conf.KeyDremioPidDetection] = "false"
	overrides[conf.KeyCollectTtop] = "false"
	overrides[conf.KeyTarballOutDir] = fmt.Sprintf("%v-%v", outDir, time.Now().UnixNano())
	overrides[conf.KeyNodeName] = coordinatorNode
	overrides[conf.KeyDremioGCLogsDir] = filepath.Join(efsLogDir, coordinatorNode)
	overrides[conf.KeyDremioLogDir] = filepath.Join(efsLogDir, coordinatorNode)

	if _, err := local.Execute([]string{}, overrides); err != nil {
		return fmt.Errorf("unable to collect entry %v due to error %w", coordinatorNode, err)
	}
	fileName := fmt.Sprintf("%v.tar.gz", filepath.Join(overrides[conf.KeyTarballOutDir], coordinatorNode))
	destFileName := fmt.Sprintf("%v.tar.gz", coordinatorNode)
	if err := os.Rename(fileName, filepath.Join(outDir, destFileName)); err != nil {
		return err
	}
	for _, entry := range entries {
		overrides := make(map[string]string)
		overrides[conf.KeyDisableRESTAPI] = "true"
		overrides[conf.KeyCollectDiskUsage] = "false"
		overrides[conf.KeyCollectJFR] = "false"
		overrides[conf.KeyCollectJStack] = "false"
		overrides[conf.KeyCaptureHeapDump] = "false"
		overrides[conf.KeyCollectJVMFlags] = "false"
		overrides[conf.KeyCollectKVStoreReport] = "false"
		overrides[conf.KeyCollectOSConfig] = "false"
		overrides[conf.KeyCollectSystemTablesExport] = "false"
		overrides[conf.KeyCollectGCLogs] = "true"
		overrides[conf.KeyCollectDremioConfiguration] = "false"
		overrides[conf.KeyDremioPidDetection] = "false"
		overrides[conf.KeyCollectTtop] = "false"
		overrides[conf.KeyTarballOutDir] = fmt.Sprintf("%v-%v", outDir, time.Now().UnixNano())
		overrides[conf.KeyNodeName] = entry.Name()
		overrides[conf.KeyDremioGCLogsDir] = filepath.Join(efsLogDir, "executor", entry.Name())
		overrides[conf.KeyDremioLogDir] = filepath.Join(efsLogDir, "executor", entry.Name())

		if _, err := local.Execute([]string{}, overrides); err != nil {
			return fmt.Errorf("unable to collect entry %v due to error %w", entry.Name(), err)
		}
		fileName := fmt.Sprintf("%v.tar.gz", filepath.Join(overrides[conf.KeyTarballOutDir], entry.Name()))
		destFileName := fmt.Sprintf("%v.tar.gz", entry.Name())
		if err := os.Rename(fileName, filepath.Join(outDir, destFileName)); err != nil {
			return err
		}
	}
	outDirEntries, err := os.ReadDir(outDir)
	if err != nil {
		return fmt.Errorf("unable to read dir %v due to %w", outDir, err)
	}
	simplelog.Infof("%v entries in %v", len(outDirEntries), outDir)
	if len(outDirEntries) == 0 {
		return fmt.Errorf("nothing captured or found in %v", outDir)
	}
	for _, e := range outDirEntries {
		if strings.HasSuffix(e.Name(), ".tar.gz") {
			tgzLoc := filepath.Join(outDir, e.Name())
			if err := archive.ExtractTarGz(tgzLoc, outDir); err != nil {
				simplelog.Errorf("unable to extract tarball %v due to error %v", tgzLoc, err)
				continue
			}
			if err := os.Remove(tgzLoc); err != nil {
				simplelog.Errorf("unable to remove tgz %v due to error: %v", tgzLoc, err)
				continue
			}
		}
	}
	simplelog.Infof("archive folder '%v' into '%v'", outDir, outFile)
	return archive.TarGzDir(outDir, outFile)
}

func init() {
	AWSELogsCmd.Flags().StringVar(&EFSLogDir, "efs-log-dir", "/var/dremio_efs/log/", "location to search for log folders in EFS")
	AWSELogsCmd.Flags().StringVar(&OutDir, "tmp-out-dir", "/tmp/ddc-awse", "output location for files")
	AWSELogsCmd.Flags().StringVar(&OutFile, "out-file", "diag.tgz", "output file")
}
