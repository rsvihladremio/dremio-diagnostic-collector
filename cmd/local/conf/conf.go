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

// package conf provides configuration for the local-collect command
package conf

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dremio/dremio-diagnostic-collector/cmd/local/conf/autodetect"
	"github.com/dremio/dremio-diagnostic-collector/cmd/local/restclient"
	"github.com/dremio/dremio-diagnostic-collector/pkg/simplelog"
	"github.com/google/uuid"
	"github.com/spf13/cast"
)

func GetString(confData map[string]interface{}, key string) string {
	if v, ok := confData[key]; ok {
		return cast.ToString(v)
	}
	return ""
}

func GetInt(confData map[string]interface{}, key string) int {
	if v, ok := confData[key]; ok {
		return cast.ToInt(v)
	}
	return 0
}

func GetBool(confData map[string]interface{}, key string) bool {
	if v, ok := confData[key]; ok {
		return cast.ToBool(v)
	}
	return false
}

// We just strip suffix at the moment. More checks can be added here
func SanitiseURL(url string) string {
	return strings.TrimSuffix(url, "/")
}

type CollectConf struct {
	// flags that are configurable by env or configuration
	numberThreads              int
	disableRESTAPI             bool
	gcLogsDir                  string
	dremioLogDir               string
	dremioConfDir              string
	dremioEndpoint             string
	dremioUsername             string
	dremioPATToken             string
	dremioRocksDBDir           string
	numberJobProfilesToCollect int
	dremioPIDDetection         bool
	collectAccelerationLogs    bool
	collectAccessLogs          bool
	collectAuditLogs           bool
	collectJVMFlags            bool
	captureHeapDump            bool
	acceptCollectionConsent    bool
	isDremioCloud              bool
	dremioCloudProjectID       string
	dremioCloudAppEndpoint     string

	// advanced variables setable by configuration or environement variable
	outputDir                   string
	tarballOutDir               string
	dremioTtopTimeSeconds       int
	dremioTtopFreqSeconds       int
	dremioJFRTimeSeconds        int
	dremioJStackFreqSeconds     int
	dremioJStackTimeSeconds     int
	dremioLogsNumDays           int
	dremioGCFilePattern         string
	dremioQueriesJSONNumDays    int
	jobProfilesNumSlowExec      int
	jobProfilesNumHighQueryCost int
	jobProfilesNumSlowPlanning  int
	jobProfilesNumRecentErrors  int
	allowInsecureSSL            bool
	collectJFR                  bool
	collectJStack               bool
	collectKVStoreReport        bool
	collectServerLogs           bool
	collectMetaRefreshLogs      bool
	collectQueriesJSON          bool
	collectDremioConfiguration  bool
	collectReflectionLogs       bool
	collectSystemTablesExport   bool
	systemTablesRowLimit        int
	collectOSConfig             bool
	collectDiskUsage            bool
	collectGCLogs               bool
	collectTtop                 bool
	collectWLM                  bool
	nodeName                    string
	restHTTPTimeout             int

	// variables
	systemtables            []string
	systemtablesdremiocloud []string
	dremioPID               int
}

func SystemTableList() []string {
	return []string{
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
}
func ReadConfFromExecLocation(overrides map[string]string) (*CollectConf, error) {
	//now read in configuration values. This will get defaults if no values are available in the configuration files or no environment variable is set

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

func ReadConf(overrides map[string]string, configDir string) (*CollectConf, error) {
	confData, err := ParseConfig(configDir, overrides)
	if err != nil {
		return &CollectConf{}, fmt.Errorf("config failed: %w", err)
	}
	simplelog.Debugf("logging parsed configuration from ddc.yaml")
	defaultCaptureSeconds := 60
	// set node name
	hostName, err := os.Hostname()
	if err != nil {
		hostName = fmt.Sprintf("unknown-%v", uuid.New())
	}

	SetViperDefaults(confData, hostName, defaultCaptureSeconds, getOutputDir(time.Now()))

	c := &CollectConf{}
	c.systemtables = SystemTableList()
	c.systemtablesdremiocloud = []string{
		"organization.clouds",
		"organization.privileges",
		"organization.projects",
		"organization.roles",
		"organization.usage",
		"project.engines",
		"project.jobs",
		"project.privileges",
		"project.reflections",
		"project.\\\"tables\\\"",
		"project.views",
		// "project.history.events",
		"project.history.jobs",
	}

	for k, v := range confData {
		if k == KeyDremioPatToken && v != "" {
			simplelog.Debugf("conf key '%v':'REDACTED'", k)
		} else {
			simplelog.Debugf("conf key '%v':'%v'", k, v)
		}
	}
	// now we can setup verbosity as we are parsing it in the ParseConfig function
	verboseString := GetString(confData, "verbose")
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

	c.dremioPIDDetection = GetBool(confData, KeyDremioPidDetection)
	c.dremioPID = GetInt(confData, KeyDremioPid)
	if c.dremioPID < 1 && c.dremioPIDDetection {
		dremioPID, err := autodetect.GetDremioPID()
		if err != nil {
			simplelog.Errorf("disabling Heap Dump Capture, Jstack and JFR collection: %v", err)
			//return &CollectConf{}, fmt.Errorf("read config stopped due to error %v", err)
		} else {
			c.dremioPID = dremioPID
		}
	}
	dremioPIDIsValid := c.dremioPID > 0

	c.acceptCollectionConsent = GetBool(confData, KeyAcceptCollectionConsent)
	c.isDremioCloud = GetBool(confData, KeyIsDremioCloud)
	c.dremioCloudProjectID = GetString(confData, KeyDremioCloudProjectID)
	c.collectAccelerationLogs = GetBool(confData, KeyCollectAccelerationLog)
	c.collectAccessLogs = GetBool(confData, KeyCollectAccessLog)
	c.collectAuditLogs = GetBool(confData, KeyCollectAuditLog)
	c.gcLogsDir = GetString(confData, KeyDremioGCLogsDir)
	c.dremioLogDir = GetString(confData, KeyDremioLogDir)
	c.nodeName = GetString(confData, KeyNodeName)
	IsAWSEfromLogDirs, err := autodetect.IsAWSEfromLogDirs()
	if err != nil {
		simplelog.Warningf("unable to determine if node is AWSE or not due to error %v", err)
	}
	if IsAWSEfromLogDirs {
		isCoord, logPath, err := autodetect.IsAWSECoordinator()
		if err != nil {
			simplelog.Errorf("unable to detect if this node %v was a coordinator so will not apply AWSE log path fix this may mean no log collection %v", c.nodeName, err)
		}
		if isCoord {
			simplelog.Debugf("AWSE coordinator node detected, using log dir %v, sylinked to %v", c.dremioLogDir, logPath)
		} else {
			simplelog.Debugf("AWSE executor node detected, using log dir %v, sylinked to %v", c.dremioLogDir, logPath)
		}
	}
	c.dremioConfDir = GetString(confData, KeyDremioConfDir)
	c.numberThreads = GetInt(confData, KeyNumberThreads)
	c.dremioEndpoint = GetString(confData, KeyDremioEndpoint)
	if c.isDremioCloud {
		if len(c.dremioCloudProjectID) != 36 {
			simplelog.Warningf("dremio cloud project id is expected to have 36 characters - the following provided id may be incorrect: %v", c.dremioCloudProjectID)
		}
		if strings.Contains(c.dremioEndpoint, "eu.dremio.cloud") {
			c.dremioEndpoint = "https://api.eu.dremio.cloud"
			c.dremioCloudAppEndpoint = "https://app.eu.dremio.cloud"
		} else if strings.Contains(c.dremioEndpoint, "dremio.cloud") {
			c.dremioEndpoint = "https://api.dremio.cloud"
			c.dremioCloudAppEndpoint = "https://app.dremio.cloud"
		} else {
			simplelog.Warningf("unexpected dremio cloud endpoint: %v - Known endpoints are https://app.dremio.cloud and https://app.eu.dremio.cloud", c.dremioEndpoint)
		}
	}
	c.dremioUsername = GetString(confData, KeyDremioUsername)
	c.disableRESTAPI = GetBool(confData, KeyDisableRESTAPI)

	c.dremioPATToken = GetString(confData, KeyDremioPatToken)
	c.dremioRocksDBDir = GetString(confData, KeyDremioRocksdbDir)
	c.collectDremioConfiguration = GetBool(confData, KeyCollectDremioConfiguration)
	c.numberJobProfilesToCollect = GetInt(confData, KeyNumberJobProfiles)
	c.captureHeapDump = GetBool(confData, KeyCaptureHeapDump) && dremioPIDIsValid

	// system diag
	c.collectOSConfig = GetBool(confData, KeyCollectOSConfig)
	c.collectDiskUsage = GetBool(confData, KeyCollectDiskUsage)
	c.collectJVMFlags = GetBool(confData, KeyCollectJVMFlags)

	// log collect
	c.tarballOutDir = GetString(confData, KeyTarballOutDir)
	c.outputDir = GetString(confData, KeyTmpOutputDir)
	c.dremioLogsNumDays = GetInt(confData, KeyDremioLogsNumDays)
	c.dremioQueriesJSONNumDays = GetInt(confData, KeyDremioQueriesJSONNumDays)
	c.dremioGCFilePattern = GetString(confData, KeyDremioGCFilePattern)
	c.collectQueriesJSON = GetBool(confData, KeyCollectQueriesJSON)
	c.collectServerLogs = GetBool(confData, KeyCollectServerLogs)
	c.collectMetaRefreshLogs = GetBool(confData, KeyCollectMetaRefreshLog)
	c.collectReflectionLogs = GetBool(confData, KeyCollectReflectionLog)
	c.collectGCLogs = GetBool(confData, KeyCollectGCLogs)
	c.gcLogsDir = GetString(confData, KeyDremioGCLogsDir)
	parsedGCLogDir, err := autodetect.FindGCLogLocation()
	if err != nil {
		if c.gcLogsDir == "" {
			simplelog.Warningf("Must set dremio-gclogs-dir manually since we are unable to retrieve gc log location from pid due to error %v", err)
		}
	}
	if parsedGCLogDir != "" {
		if c.gcLogsDir == "" {
			simplelog.Debugf("setting gc logs to %v", parsedGCLogDir)
		} else {
			simplelog.Debugf("overriding gc logs location from %v to %v due to detection of gclog directory", c.gcLogsDir, parsedGCLogDir)
		}
		c.gcLogsDir = parsedGCLogDir
	}

	// jfr config
	c.collectJFR = GetBool(confData, KeyCollectJFR) && dremioPIDIsValid
	c.dremioJFRTimeSeconds = GetInt(confData, KeyDremioJFRTimeSeconds)
	// jstack config
	c.collectJStack = GetBool(confData, KeyCollectJStack) && dremioPIDIsValid
	c.dremioJStackTimeSeconds = GetInt(confData, KeyDremioJStackTimeSeconds)
	c.dremioJStackFreqSeconds = GetInt(confData, KeyDremioJStackFreqSeconds)

	// ttop
	c.collectTtop = GetBool(confData, KeyCollectTtop)
	c.dremioTtopFreqSeconds = GetInt(confData, KeyDremioTtopFreqSeconds)
	c.dremioTtopTimeSeconds = GetInt(confData, KeyDremioTtopTimeSeconds)
	// collect rest apis
	disableRESTAPI := c.disableRESTAPI || c.dremioPATToken == ""
	if disableRESTAPI {
		simplelog.Debugf("disabling all Workload Manager, System Table, KV Store, and Job Profile collection since the --dremio-pat-token is not set")
	}
	c.allowInsecureSSL = GetBool(confData, KeyAllowInsecureSSL)
	c.collectWLM = GetBool(confData, KeyCollectWLM) && !disableRESTAPI
	c.collectSystemTablesExport = GetBool(confData, KeyCollectSystemTablesExport) && !disableRESTAPI
	c.systemTablesRowLimit = GetInt(confData, KeySystemTablesRowLimit)
	c.collectKVStoreReport = GetBool(confData, KeyCollectKVStoreReport) && !disableRESTAPI
	c.restHTTPTimeout = GetInt(confData, KeyRestHTTPTimeout)
	restclient.InitClient(c.allowInsecureSSL, c.restHTTPTimeout)

	numberJobProfilesToCollect, jobProfilesNumHighQueryCost, jobProfilesNumSlowExec, jobProfilesNumRecentErrors, jobProfilesNumSlowPlanning := CalculateJobProfileSettingsWithViperConfig(c)
	c.numberJobProfilesToCollect = numberJobProfilesToCollect
	c.jobProfilesNumHighQueryCost = jobProfilesNumHighQueryCost
	c.jobProfilesNumSlowExec = jobProfilesNumSlowExec
	c.jobProfilesNumRecentErrors = jobProfilesNumRecentErrors
	c.jobProfilesNumSlowPlanning = jobProfilesNumSlowPlanning

	return c, nil
}

func getOutputDir(now time.Time) string {
	nowStr := now.Format("20060102-150405")
	return filepath.Join(os.TempDir(), "ddc", nowStr)
}

func (c CollectConf) DisableRESTAPI() bool {
	return c.disableRESTAPI
}

func (c *CollectConf) GcLogsDir() string {
	return c.gcLogsDir
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

func (c *CollectConf) CollectOSConfig() bool {
	return c.collectOSConfig
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

func (c *CollectConf) SystemTablesRowLimit() int {
	return c.systemTablesRowLimit
}

func (c *CollectConf) CollectKVStoreReport() bool {
	return c.collectKVStoreReport
}

func (c *CollectConf) Systemtables() []string {
	return c.systemtables
}

func (c *CollectConf) SystemtablesDremioCloud() []string {
	return c.systemtablesdremiocloud
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

func (c *CollectConf) CollectJVMFlags() bool {
	return c.collectJVMFlags
}

func (c *CollectConf) CollectAuditLogs() bool {
	return c.collectAuditLogs
}

func (c *CollectConf) TtopOutDir() string {
	return filepath.Join(c.outputDir, "ttop", c.nodeName)
}

func (c *CollectConf) HeapDumpsOutDir() string { return filepath.Join(c.outputDir, "heap-dumps") }

func (c *CollectConf) JobProfilesOutDir() string {
	return filepath.Join(c.outputDir, "job-profiles", c.nodeName)
}
func (c *CollectConf) KubernetesOutDir() string { return filepath.Join(c.outputDir, "kubernetes") }
func (c *CollectConf) KVstoreOutDir() string {
	return filepath.Join(c.outputDir, "kvstore", c.nodeName)
}
func (c *CollectConf) SystemTablesOutDir() string {
	return filepath.Join(c.outputDir, "system-tables", c.nodeName)
}
func (c *CollectConf) WLMOutDir() string { return filepath.Join(c.outputDir, "wlm", c.nodeName) }

// works on all nodes but includes node name in file name
func (c *CollectConf) JFROutDir() string { return filepath.Join(c.outputDir, "jfr") }

// per node out directories
func (c *CollectConf) ConfigurationOutDir() string {
	return filepath.Join(c.outputDir, "configuration", c.nodeName)
}
func (c *CollectConf) LogsOutDir() string { return filepath.Join(c.outputDir, "logs", c.nodeName) }
func (c *CollectConf) NodeInfoOutDir() string {
	return filepath.Join(c.outputDir, "node-info", c.nodeName)
}
func (c *CollectConf) QueriesOutDir() string {
	return filepath.Join(c.outputDir, "queries", c.nodeName)
}
func (c *CollectConf) ThreadDumpsOutDir() string {
	return filepath.Join(c.outputDir, "jfr", "thread-dumps", c.nodeName)
}

func (c *CollectConf) DremioEndpoint() string {
	return SanitiseURL(c.dremioEndpoint)
}

func (c *CollectConf) DremioPATToken() string {
	return c.dremioPATToken
}

func (c *CollectConf) AcceptCollectionConsent() bool {
	return c.acceptCollectionConsent
}

func (c *CollectConf) IsDremioCloud() bool {
	return c.isDremioCloud
}

func (c *CollectConf) DremioCloudProjectID() string {
	return c.dremioCloudProjectID
}

func (c *CollectConf) DremioCloudAppEndpoint() string {
	return c.dremioCloudAppEndpoint
}

func (c *CollectConf) NodeName() string {
	return c.nodeName
}

func (c *CollectConf) TarballOutDir() string {
	return c.tarballOutDir
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

func (c *CollectConf) DremioPIDDetection() bool {
	return c.dremioPIDDetection
}

func (c *CollectConf) DremioConfDir() string {
	return c.dremioConfDir
}

func (c *CollectConf) DremioJFRTimeSeconds() int {
	return c.dremioJFRTimeSeconds
}

func (c *CollectConf) DremioTtopTimeSeconds() int {
	return c.dremioTtopTimeSeconds
}

func (c *CollectConf) DremioTtopFreqSeconds() int {
	return c.dremioTtopFreqSeconds
}

func (c *CollectConf) CollectTtop() bool {
	return c.collectTtop
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

func (c *CollectConf) RestHTTPTimeout() int {
	return c.restHTTPTimeout
}

func (c *CollectConf) DremioRocksDBDir() string {
	return c.dremioRocksDBDir
}
