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
	"log"
	"math"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rsvihladremio/dremio-diagnostic-collector/cmd/local/conf/autodetect"
	"github.com/rsvihladremio/dremio-diagnostic-collector/cmd/simplelog"
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
	supportedExtensions     []string
	systemtables            []string
	unableToReadConfigError error
	confFiles               []string
	configIsFound           bool
	foundConfig             string
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

	c := &CollectConf{}
	c.supportedExtensions = []string{"yaml", "json", "toml", "hcl", "env", "props"}
	//kubernetesConfTypes = []string{"nodes", "sc", "pvc", "pv", "service", "endpoints", "pods", "deployments", "statefulsets", "daemonset", "replicaset", "cronjob", "job", "events", "ingress", "limitrange", "resourcequota", "hpa", "pdb", "pc"}
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
		return &CollectConf{}, fmt.Errorf("read config stopped due to error %v", err)
	}
	c.dremioPID = dremioPID
	// set default config
	viper.SetDefault("collect-acceleration-log", false)
	viper.SetDefault("collect-access-log", false)
	viper.SetDefault("dremio-log-dir", "/var/log/dremio")
	defaultThreads := getThreads(runtime.NumCPU())
	viper.SetDefault("number-threads", defaultThreads)
	viper.SetDefault("dremio-usernme", "dremio")
	viper.SetDefault("dremio-pat-token", "")
	viper.SetDefault("dremio-conf-dir", "/opt/dremio/conf")
	viper.SetDefault("dremio-rocksdb-dir", "/opt/dremio/data/db")
	viper.SetDefault("collect-dremio-configuration", true)
	viper.SetDefault("capture-heap-dump", false)
	viper.SetDefault("number-job-profiles", 25000)
	viper.SetDefault("dremio-endpoint", "http://localhost:9047")
	viper.SetDefault("tmp-output-dir", getOutputDir(time.Now()))
	viper.SetDefault("collect-metrics", true)
	viper.SetDefault("collect-disk-usage", true)
	viper.SetDefault("dremio-logs-num-days", 7)
	viper.SetDefault("dremio-queries-json-num-days", 28)
	viper.SetDefault("dremio-gc-file-pattern", "gc*.log*")
	viper.SetDefault("collect-queries-json", true)
	viper.SetDefault("collect-server-logs", true)
	viper.SetDefault("collect-meta-refresh-log", true)
	viper.SetDefault("collect-reflection-log", true)
	viper.SetDefault("collect-gc-logs", true)
	viper.SetDefault("collect-jfr", true)
	viper.SetDefault("collect-jstack", true)
	viper.SetDefault("collect-system-tables-export", true)
	viper.SetDefault("collect-wlm", true)
	viper.SetDefault("collect-kvstore-report", true)
	defaultCaptureSeconds := 60
	viper.SetDefault("dremio-jstack-time-seconds", defaultCaptureSeconds)
	viper.SetDefault("dremio-jfr-time-seconds", defaultCaptureSeconds)
	viper.SetDefault("node-metrics-collect-duration-seconds", defaultCaptureSeconds)
	viper.SetDefault("dremio-jstack-freq-seconds", 1)
	viper.SetDefault("dremio-gclogs-dir", "")
	// set node name
	hostName, err := os.Hostname()
	if err != nil {
		hostName = fmt.Sprintf("unknown-%v", uuid.New())
	}
	viper.SetDefault("node-name", hostName)

	//read viper config
	baseConfig := "ddc"
	viper.SetConfigName(baseConfig) // Name of config file (without extension)
	viper.AddConfigPath(configDir)

	for _, e := range c.supportedExtensions {
		c.confFiles = append(c.confFiles, fmt.Sprintf("%v.%v", baseConfig, e))
	}

	//searching for all known
	for _, ext := range c.supportedExtensions {
		viper.SetConfigType(ext)
		unableToReadConfigError := viper.ReadInConfig()
		if unableToReadConfigError == nil {
			c.configIsFound = true
			c.foundConfig = fmt.Sprintf("%v.%v", baseConfig, ext)
			break
		}
	}

	viper.AutomaticEnv() // Automatically read environment variables

	//TODO add back command flag binding or remove flags
	// // Only bind flags that were actually set
	// cmd.Flags().VisitAll(func(flag *pflag.Flag) {
	// 	if flag.Changed {
	// 		log.Printf("flag %v passed in binding it", flag.Name)
	// 		if err := viper.BindPFlag(flag.Name, flag); err != nil {
	// 			simplelog.Errorf("unable to bind flag %v so it will likely not be read due to error: %v", flag.Name, err)
	// 		}
	// 	}
	// })
	for k, v := range overrides {
		viper.Set(k, v)
	}

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
	defer func() {
		if err := simplelog.Close(); err != nil {
			log.Printf("unable to close log due to error %v", err)
		}
	}()
	simplelog.Infof("searched for the following optional configuration files in the current directory %v", strings.Join(c.confFiles, ", "))
	if !c.configIsFound {
		simplelog.Warningf("was unable to read any of the valid config file formats (%v) due to error '%v' - falling back to defaults, command line flags and environment variables", strings.Join(c.supportedExtensions, ","), c.unableToReadConfigError)
	} else {
		simplelog.Infof("found config file %v", c.foundConfig)
	}
	// override the flag values
	c.acceptCollectionConsent = viper.GetBool("accept-collection-consent")
	c.collectAccelerationLogs = viper.GetBool("collect-acceleration-log")
	c.collectAccessLogs = viper.GetBool("collect-access-log")
	c.gcLogsDir = viper.GetString("dremio-gclogs-dir")
	c.dremioLogDir = viper.GetString("dremio-log-dir")
	// parse in the nodeName now because it is used for dremioLogDir in awse
	c.nodeName = viper.GetString("node-name")
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
	c.dremioConfDir = viper.GetString("dremio-conf-dir")
	c.numberThreads = viper.GetInt("number-threads")
	c.dremioEndpoint = viper.GetString("dremio-endpoint")
	c.dremioUsername = viper.GetString("dremio-username")
	c.dremioPATToken = viper.GetString("dremio-pat-token")
	c.dremioRocksDBDir = viper.GetString("dremio-rocksdb-dir")
	c.collectDremioConfiguration = viper.GetBool("collect-dremio-configuration")
	c.numberJobProfilesToCollect = viper.GetInt("number-job-profiles")
	c.captureHeapDump = viper.GetBool("capture-heap-dump")

	//system diag

	c.collectNodeMetrics = viper.GetBool("collect-metrics")
	c.collectDiskUsage = viper.GetBool("collect-disk-usage")

	// log collect
	c.outputDir = viper.GetString("tmp-output-dir")
	c.dremioLogsNumDays = viper.GetInt("dremio-logs-num-days")
	c.dremioQueriesJSONNumDays = viper.GetInt("dremio-queries-json-num-days")
	c.dremioGCFilePattern = viper.GetString("dremio-gc-file-pattern")
	c.collectQueriesJSON = viper.GetBool("collect-queries-json")
	c.collectServerLogs = viper.GetBool("collect-server-logs")
	c.collectMetaRefreshLogs = viper.GetBool("collect-meta-refresh-log")
	c.collectReflectionLogs = viper.GetBool("collect-reflection-log")
	c.collectGCLogs = viper.GetBool("collect-gc-logs")
	c.gcLogsDir = viper.GetString("dremio-gclogs-dir")
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
	c.collectJFR = viper.GetBool("collect-jfr")
	c.dremioJFRTimeSeconds = viper.GetInt("dremio-jfr-time-seconds")
	// jstack config
	c.collectJStack = viper.GetBool("collect-jstack")
	c.dremioJStackTimeSeconds = viper.GetInt("dremio-jstack-time-seconds")
	c.dremioJStackFreqSeconds = viper.GetInt("dremio-jstack-freq-seconds")

	// collect rest apis
	personalAccessTokenPresent := c.dremioPATToken != ""
	if !personalAccessTokenPresent {
		simplelog.Warningf("disabling all Workload Manager, System Table, KV Store, and Job Profile collection since the --dremio-pat-token is not set")
	}
	c.collectWLM = viper.GetBool("collect-wlm") && personalAccessTokenPresent
	c.collectSystemTablesExport = viper.GetBool("collect-system-tables-export") && personalAccessTokenPresent
	c.collectKVStoreReport = viper.GetBool("collect-kvstore-report") && personalAccessTokenPresent
	// don't bother doing any of the calculation if personal access token is not present in fact zero out everything
	if !personalAccessTokenPresent {
		c.numberJobProfilesToCollect = 0
		c.jobProfilesNumHighQueryCost = 0
		c.jobProfilesNumSlowExec = 0
		c.jobProfilesNumRecentErrors = 0
		c.jobProfilesNumSlowPlanning = 0
	} else {
		// check if job profile is set
		var defaultJobProfilesNumSlowExec int
		var defaultJobProfilesNumRecentErrors int
		var defaultJobProfilesNumSlowPlanning int
		var defaultJobProfilesNumHighQueryCost int
		if c.numberJobProfilesToCollect > 0 {
			if c.numberJobProfilesToCollect < 4 {
				//so few that it is not worth being clever
				defaultJobProfilesNumSlowExec = c.numberJobProfilesToCollect
			} else {
				defaultJobProfilesNumSlowExec = int(float64(c.numberJobProfilesToCollect) * 0.4)
				defaultJobProfilesNumRecentErrors = int(float64(defaultJobProfilesNumRecentErrors) * 0.2)
				defaultJobProfilesNumSlowPlanning = int(float64(defaultJobProfilesNumSlowPlanning) * 0.2)
				defaultJobProfilesNumHighQueryCost = int(float64(defaultJobProfilesNumHighQueryCost) * 0.2)
				//grab the remainder and drop on top of defaultJobProfilesNumSlowExec
				totalAllocated := defaultJobProfilesNumSlowExec + defaultJobProfilesNumRecentErrors + defaultJobProfilesNumSlowPlanning + defaultJobProfilesNumHighQueryCost
				diff := c.numberJobProfilesToCollect - totalAllocated
				defaultJobProfilesNumSlowExec += diff
			}
			simplelog.Infof("setting default values for slow execution profiles: %v, recent error profiles %v, slow planning profiles %v, high query cost profiles %v",
				defaultJobProfilesNumSlowExec,
				defaultJobProfilesNumRecentErrors,
				defaultJobProfilesNumSlowPlanning,
				defaultJobProfilesNumHighQueryCost)
		}

		// job profile specific numbers
		c.jobProfilesNumHighQueryCost = viper.GetInt("job-profiles-num-high-query-cost")
		if c.jobProfilesNumHighQueryCost == 0 {
			c.jobProfilesNumHighQueryCost = defaultJobProfilesNumHighQueryCost
		} else if c.jobProfilesNumHighQueryCost != defaultJobProfilesNumHighQueryCost {
			simplelog.Warningf("job-profiles-num-high-query-cost changed to %v by configuration", c.jobProfilesNumHighQueryCost)
		}
		c.jobProfilesNumSlowExec = viper.GetInt("job-profiles-num-slow-exec")
		if c.jobProfilesNumSlowExec == 0 {
			c.jobProfilesNumSlowExec = defaultJobProfilesNumSlowExec
		} else if c.jobProfilesNumSlowExec != defaultJobProfilesNumSlowExec {
			simplelog.Warningf("job-profiles-num-slow-exec changed to %v by configuration", c.jobProfilesNumSlowExec)
		}

		c.jobProfilesNumRecentErrors = viper.GetInt("job-profiles-num-recent-errors")
		if c.jobProfilesNumRecentErrors == 0 {
			c.jobProfilesNumRecentErrors = defaultJobProfilesNumRecentErrors
		} else if c.jobProfilesNumRecentErrors != defaultJobProfilesNumRecentErrors {
			simplelog.Warningf("job-profiles-num-recent-errors changed to %v by configuration", c.jobProfilesNumRecentErrors)
		}
		c.jobProfilesNumSlowPlanning = viper.GetInt("job-profiles-num-slow-planning")
		if c.jobProfilesNumSlowPlanning == 0 {
			c.jobProfilesNumSlowPlanning = defaultJobProfilesNumSlowPlanning
		} else if c.jobProfilesNumSlowPlanning != defaultJobProfilesNumSlowPlanning {
			simplelog.Warningf("job-profiles-num-slow-planning changed to %v by configuration", c.jobProfilesNumSlowPlanning)
		}
		totalAllocated := defaultJobProfilesNumSlowExec + defaultJobProfilesNumRecentErrors + defaultJobProfilesNumSlowPlanning + defaultJobProfilesNumHighQueryCost
		if totalAllocated > 0 && totalAllocated != c.numberJobProfilesToCollect {
			c.numberJobProfilesToCollect = totalAllocated
			simplelog.Warningf("due to configuration parameters new total jobs profiles collected has been adjusted to %v", totalAllocated)
		}
	}
	simplelog.Infof("Current configuration: %+v", viper.AllSettings())

	return c, nil
}

func getThreads(cpus int) int {
	numCPU := math.Round(float64(cpus / 2.0))
	return int(math.Max(numCPU, 2))
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
