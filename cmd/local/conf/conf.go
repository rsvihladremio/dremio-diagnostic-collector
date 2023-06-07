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

package conf

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/dremio/dremio-diagnostic-collector/cmd/local/conf/autodetect"
	"github.com/dremio/dremio-diagnostic-collector/cmd/simplelog"
	"github.com/google/uuid"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type CollectConf struct {
	// flags that are configurable by env or configuration
	numberThreads              int
	gcLogsDir                  string
	dremioLogDir               string
	dremioConfDir              string
	dremioEndpoint             string
	dremioUsername             string
	dremioPATToken             string
	dremioRocksDBDir           string
	numberJobProfilesToCollect int
	collectAccelerationLogs    bool
	collectAccessLogs          bool
	captureHeapDump            bool
	acceptCollectionConsent    bool

	// advanced variables setable by configuration or environement variable
	outputDir                         string
	dremioJFRTimeSeconds              int
	dremioJStackFreqSeconds           int
	dremioJStackTimeSeconds           int
	dremioLogsNumDays                 int
	dremioGCFilePattern               string
	dremioQueriesJSONNumDays          int
	jobProfilesNumSlowExec            int
	jobProfilesNumHighQueryCost       int
	jobProfilesNumSlowPlanning        int
	jobProfilesNumRecentErrors        int
	nodeMetricsCollectDurationSeconds int
	collectNodeMetrics                bool
	collectJFR                        bool
	collectJStack                     bool
	collectKVStoreReport              bool
	collectServerLogs                 bool
	collectMetaRefreshLogs            bool
	collectQueriesJSON                bool
	collectDremioConfiguration        bool
	collectReflectionLogs             bool
	collectSystemTablesExport         bool
	collectDiskUsage                  bool
	collectGCLogs                     bool
	collectWLM                        bool
	nodeName                          string

	// variables
	systemtables            []string
	unableToReadConfigError error
	dremioPID               int
}

func ReadConfFromExecLocation(overrides map[string]*pflag.Flag) (*CollectConf, error) {
	//now read in viper configuration values. This will get defaults if no values are available in the configuration files or no environment variable is set

	// find the location of the ddc executable
	execPath, err := os.Executable()
	if err != nil {
		simplelog.Errorf("Error getting executable path: '%v'. Falling back to working directory for search location", err)
		execPath = "."
	}
	// use that as the default location of the configuration
	configDir := filepath.Dir(execPath)
	return ReadConf(overrides, configDir)
}

func ReadConf(overrides map[string]*pflag.Flag, configDir string) (*CollectConf, error) {
	defaultThreads := autodetect.GetThreads()
	defaultCaptureSeconds := 60
	// set node name
	hostName, err := os.Hostname()
	if err != nil {
		hostName = fmt.Sprintf("unknown-%v", uuid.New())
	}
	SetViperDefaults(defaultThreads, hostName, defaultCaptureSeconds, getOutputDir(time.Now()))

	c := &CollectConf{}
	c.systemtables = []string{
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
	dremioPID, err := autodetect.GetDremioPID()
	if err != nil {
		simplelog.Errorf("disabling Heap Dump Capture, Jstack and JFR collection: %v", err)
		//return &CollectConf{}, fmt.Errorf("read config stopped due to error %v", err)
	}
	c.dremioPID = dremioPID
	dremioPIDIsValid := dremioPID > 0

	supportedExtensions := []string{"yaml", "json", "toml", "hcl", "env", "props"}
	foundConfig := ParseConfig(configDir, viper.SupportedExts, overrides)
	// now we can setup verbosity as we are parsing it in the ParseConfig function
	verboseString := viper.GetString("verbose")
	verbose := strings.Count(verboseString, "v")
	if verbose >= 3 {
		fmt.Println("verbosity level DEBUG")
	} else if verbose == 2 {
		fmt.Println("verbosity level INFO")
	} else if verbose == 1 {
		fmt.Println("verbosity level WARNING")
	} else {
		fmt.Println("verbosity level ERROR")
	}
	simplelog.InitLogger(verbose)

	if foundConfig == "" {
		simplelog.Warningf("was unable to read any of the valid config file formats (%v) due to error '%v' - falling back to defaults, command line flags and environment variables", strings.Join(supportedExtensions, ","), c.unableToReadConfigError)
	} else {
		simplelog.Infof("found config file %v", foundConfig)
	}
	c.acceptCollectionConsent = viper.GetBool(KeyAcceptCollectionConsent)
	c.collectAccelerationLogs = viper.GetBool(KeyCollectAccelerationLog)
	c.collectAccessLogs = viper.GetBool(KeyCollectAccessLog)
	c.gcLogsDir = viper.GetString(KeyDremioGCLogsDir)
	c.dremioLogDir = viper.GetString(KeyDremioLogDir)
	c.nodeName = viper.GetString(KeyNodeName)
	isAWSE, err := autodetect.IsAWSE()
	if err != nil {
		simplelog.Warningf("unable to determind if node is AWSE or not due to error %v", err)
	}
	if isAWSE {
		isExec, err := autodetect.IsAWSEExecutor(c.nodeName)
		if err != nil {
			simplelog.Errorf("unable to detect if this was an executor so will not apply AWSE log path fix this may mean no log collection %v", err)
		} else if isExec {
			if strings.Contains(c.dremioLogDir, c.nodeName) {
				simplelog.Warningf("node name %v already included in log directory of %v make this is intentional as you do not need to put the node name in the log path", c.nodeName, c.dremioLogDir)
			} else {
				// ok so looks like we need to adjust this since the node name is not already in the path
				c.dremioLogDir = path.Join(c.dremioLogDir, "executor", c.nodeName)
				simplelog.Infof("AWSE detected adding the node name %v to the log directory path %v", c.nodeName, c.dremioLogDir)
			}
		} else {
			if strings.Contains(c.dremioLogDir, "coordinator") {
				simplelog.Warningf("coordinator already included in log directory of %v make this is intentional as you do not need to put the coordinator in the log path", c.dremioLogDir)
			} else {
				c.dremioLogDir = path.Join(c.dremioLogDir, "coordinator")
				simplelog.Infof("AWSE coordinator node detected adding coordinator name to log dir %v", c.dremioLogDir)
			}
		}
	}
	c.dremioConfDir = viper.GetString(KeyDremioConfDir)
	c.numberThreads = viper.GetInt(KeyNumberThreads)
	c.dremioEndpoint = viper.GetString(KeyDremioEndpoint)
	c.dremioUsername = viper.GetString(KeyDremioUsername)
	c.dremioPATToken = viper.GetString(KeyDremioPatToken)
	c.dremioRocksDBDir = viper.GetString(KeyDremioRocksdbDir)
	c.collectDremioConfiguration = viper.GetBool(KeyCollectDremioConfiguration)
	c.numberJobProfilesToCollect = viper.GetInt(KeyNumberJobProfiles)
	c.captureHeapDump = viper.GetBool(KeyCaptureHeapDump) && dremioPIDIsValid

	// system diag
	c.collectNodeMetrics = viper.GetBool(KeyCollectMetrics)
	c.nodeMetricsCollectDurationSeconds = viper.GetInt(KeyNodeMetricsCollectDurationSeconds)
	c.collectDiskUsage = viper.GetBool(KeyCollectDiskUsage)

	// log collect
	c.outputDir = viper.GetString(KeyTmpOutputDir)
	c.dremioLogsNumDays = viper.GetInt(KeyDremioLogsNumDays)
	c.dremioQueriesJSONNumDays = viper.GetInt(KeyDremioQueriesJSONNumDays)
	c.dremioGCFilePattern = viper.GetString(KeyDremioGCFilePattern)
	c.collectQueriesJSON = viper.GetBool(KeyCollectQueriesJSON)
	c.collectServerLogs = viper.GetBool(KeyCollectServerLogs)
	c.collectMetaRefreshLogs = viper.GetBool(KeyCollectMetaRefreshLog)
	c.collectReflectionLogs = viper.GetBool(KeyCollectReflectionLog)
	c.collectGCLogs = viper.GetBool(KeyCollectGCLogs)
	c.gcLogsDir = viper.GetString(KeyDremioGCLogsDir)
	parsedGCLogDir, err := autodetect.FindGCLogLocation()
	if err != nil {
		if c.gcLogsDir == "" {
			simplelog.Warningf("Must set dremio-gclogs-dir manually since we are unable to retrieve gc log location from pid due to error %v", err)
		}
	}
	if parsedGCLogDir != "" {
		if c.gcLogsDir == "" {
			simplelog.Infof("setting gc logs to %v", parsedGCLogDir)
		} else {
			simplelog.Warningf("overriding gc logs location from %v to %v due to detection of gclog directory", c.gcLogsDir, parsedGCLogDir)
		}
		c.gcLogsDir = parsedGCLogDir
	}

	// jfr config
	c.collectJFR = viper.GetBool(KeyCollectJFR) && dremioPIDIsValid
	c.dremioJFRTimeSeconds = viper.GetInt(KeyDremioJFRTimeSeconds)
	// jstack config
	c.collectJStack = viper.GetBool(KeyCollectJStack) && dremioPIDIsValid
	c.dremioJStackTimeSeconds = viper.GetInt(KeyDremioJStackTimeSeconds)
	c.dremioJStackFreqSeconds = viper.GetInt(KeyDremioJStackFreqSeconds)

	// collect rest apis
	personalAccessTokenPresent := c.dremioPATToken != ""
	if !personalAccessTokenPresent {
		simplelog.Warningf("disabling all Workload Manager, System Table, KV Store, and Job Profile collection since the --dremio-pat-token is not set")
	}
	c.collectWLM = viper.GetBool(KeyCollectWLM) && personalAccessTokenPresent
	c.collectSystemTablesExport = viper.GetBool(KeyCollectSystemTablesExport) && personalAccessTokenPresent
	c.collectKVStoreReport = viper.GetBool(KeyCollectKVStoreReport) && personalAccessTokenPresent

	numberJobProfilesToCollect, jobProfilesNumHighQueryCost, jobProfilesNumSlowExec, jobProfilesNumRecentErrors, jobProfilesNumSlowPlanning := CalculateJobProfileSettings(c)
	c.numberJobProfilesToCollect = numberJobProfilesToCollect
	c.jobProfilesNumHighQueryCost = jobProfilesNumHighQueryCost
	c.jobProfilesNumSlowExec = jobProfilesNumSlowExec
	c.jobProfilesNumRecentErrors = jobProfilesNumRecentErrors
	c.jobProfilesNumSlowPlanning = jobProfilesNumSlowPlanning
	simplelog.Infof("Current configuration: %+v", viper.AllSettings())

	return c, nil
}

func getOutputDir(now time.Time) string {
	nowStr := now.Format("20060102-150405")
	return filepath.Join(os.TempDir(), "ddc", nowStr)
}

func (c *CollectConf) GcLogsDir() string {
	return c.gcLogsDir
}

func (c *CollectConf) CollectNodeMetrics() bool {
	return c.collectNodeMetrics
}

func (c *CollectConf) CollectJFR() bool {
	return c.collectJFR
}

func (c *CollectConf) CollectJStack() bool {
	return c.collectJStack
}

func (c *CollectConf) CaptureHeapDump() bool {
	return c.captureHeapDump
}

func (c *CollectConf) CollectWLM() bool {
	return c.collectWLM
}

func (c *CollectConf) CollectGCLogs() bool {
	return c.collectGCLogs
}

func (c *CollectConf) CollectDiskUsage() bool {
	return c.collectDiskUsage
}

func (c *CollectConf) CollectDremioConfiguration() bool {
	return c.collectDremioConfiguration
}

func (c *CollectConf) CollectSystemTablesExport() bool {
	return c.collectSystemTablesExport
}

func (c *CollectConf) CollectKVStoreReport() bool {
	return c.collectKVStoreReport
}

func (c *CollectConf) Systemtables() []string {
	return c.systemtables
}

func (c *CollectConf) CollectServerLogs() bool {
	return c.collectServerLogs
}

func (c *CollectConf) CollectQueriesJSON() bool {
	return c.collectQueriesJSON
}

func (c *CollectConf) CollectMetaRefreshLogs() bool {
	return c.collectMetaRefreshLogs
}

func (c *CollectConf) CollectReflectionLogs() bool {
	return c.collectReflectionLogs
}

func (c *CollectConf) CollectAccelerationLogs() bool {
	return c.collectAccelerationLogs
}

func (c *CollectConf) NumberJobProfilesToCollect() int {
	return c.numberJobProfilesToCollect
}

func (c *CollectConf) CollectAccessLogs() bool {
	return c.collectAccessLogs
}

// only works on coordinator so does not need node name
func (c *CollectConf) HeapDumpsOutDir() string    { return path.Join(c.outputDir, "heap-dumps") }
func (c *CollectConf) JobProfilesOutDir() string  { return path.Join(c.outputDir, "job-profiles") }
func (c *CollectConf) KubernetesOutDir() string   { return path.Join(c.outputDir, "kubernetes") }
func (c *CollectConf) KVstoreOutDir() string      { return path.Join(c.outputDir, "kvstore") }
func (c *CollectConf) SystemTablesOutDir() string { return path.Join(c.outputDir, "system-tables") }
func (c *CollectConf) WLMOutDir() string          { return path.Join(c.outputDir, "wlm") }

// works on all nodes but includes node name in file name
func (c *CollectConf) JFROutDir() string { return path.Join(c.outputDir, "jfr") }

// per node out directories
func (c *CollectConf) ConfigurationOutDir() string {
	return path.Join(c.outputDir, "configuration", c.nodeName)
}
func (c *CollectConf) LogsOutDir() string     { return path.Join(c.outputDir, "logs", c.nodeName) }
func (c *CollectConf) NodeInfoOutDir() string { return path.Join(c.outputDir, "node-info", c.nodeName) }
func (c *CollectConf) QueriesOutDir() string  { return path.Join(c.outputDir, "queries", c.nodeName) }
func (c *CollectConf) ThreadDumpsOutDir() string {
	return path.Join(c.outputDir, "jfr", "thread-dumps", c.nodeName)
}

func (c *CollectConf) DremioEndpoint() string {
	return c.dremioEndpoint
}

func (c *CollectConf) DremioPATToken() string {
	return c.dremioPATToken
}

func (c *CollectConf) AcceptCollectionConsent() bool {
	return c.acceptCollectionConsent
}

func (c *CollectConf) NodeName() string {
	return c.nodeName
}

func (c *CollectConf) OutputDir() string {
	return c.outputDir
}

func (c *CollectConf) NumberThreads() int {
	return c.numberThreads
}

func (c *CollectConf) JobProfilesNumSlowPlanning() int {
	return c.jobProfilesNumSlowPlanning
}

func (c *CollectConf) JobProfilesNumSlowExec() int {
	return c.jobProfilesNumSlowExec
}

func (c *CollectConf) JobProfilesNumHighQueryCost() int {
	return c.jobProfilesNumHighQueryCost
}

func (c *CollectConf) JobProfilesNumRecentErrors() int {
	return c.jobProfilesNumRecentErrors
}

func (c *CollectConf) DremioPID() int {
	return c.dremioPID
}

func (c *CollectConf) DremioConfDir() string {
	return c.dremioConfDir
}

func (c *CollectConf) DremioJFRTimeSeconds() int {
	return c.dremioJFRTimeSeconds
}

func (c *CollectConf) DremioJStackTimeSeconds() int {
	return c.dremioJStackTimeSeconds
}

func (c *CollectConf) DremioJStackFreqSeconds() int {
	return c.dremioJStackFreqSeconds
}

func (c *CollectConf) DremioLogDir() string {
	return c.dremioLogDir
}

func (c *CollectConf) DremioGCFilePattern() string {
	return c.dremioGCFilePattern
}

func (c *CollectConf) DremioQueriesJSONNumDays() int {
	return c.dremioQueriesJSONNumDays
}

func (c *CollectConf) DremioLogsNumDays() int {
	return c.dremioLogsNumDays
}

func (c *CollectConf) NodeMetricsCollectDurationSeconds() int {
	return c.nodeMetricsCollectDurationSeconds
}
