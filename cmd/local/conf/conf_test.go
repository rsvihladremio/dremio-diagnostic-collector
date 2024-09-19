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

package conf_test

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dremio/dremio-diagnostic-collector/v3/cmd/local/conf"
	"github.com/dremio/dremio-diagnostic-collector/v3/pkg/collects"
	"github.com/dremio/dremio-diagnostic-collector/v3/pkg/shutdown"
	"github.com/dremio/dremio-diagnostic-collector/v3/pkg/simplelog"
)

var (
	tmpDir        string
	cfgFilePath   string
	overrides     map[string]string
	err           error
	cfg           *conf.CollectConf
	tarballOutDir string
)
var ts *httptest.Server

var genericConfSetup = func(cfgContent string) {
	tarballOutDir, err = os.MkdirTemp("", "testerDir")
	if err != nil {
		log.Fatalf("unable to create dir with error %v", err)
	}
	tmpDir, err = os.MkdirTemp("", "testdataabdc")
	if err != nil {
		log.Fatalf("unable to create dir with error %v", err)
	}
	cfgFilePath = filepath.Join(tmpDir, "ddc.yaml")

	if cfgContent == "" {
		ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "Hello, client")
		}))
		// Create a sample configuration file.
		cfgContent = fmt.Sprintf(`
accept-collection-consent: true
disable-rest-api: false
collect-acceleration-log: true
collect-access-log: true
collect-audit-log: true
collect-jvm-flags: true
dremio-pid-detection: false
dremio-gclogs-dir: "/path/to/gclogs"
dremio-log-dir: %v
node-name: "node1"
dremio-conf-dir: "%v"
tarball-out-dir: "%v"
number-threads: 4
dremio-endpoint: "%v"
dremio-username: "admin"
dremio-pat-token: "your_personal_access_token"
dremio-rocksdb-dir: "/path/to/dremio/rocksdb"
collect-dremio-configuration: true
number-job-profiles: 10
capture-heap-dump: true
collect-metrics: true
collect-os-config: true
collect-disk-usage: true
tmp-output-dir: "/path/to/tmp"
dremio-logs-num-days: 7
dremio-queries-json-num-days: 7
dremio-gc-file-pattern: "*.log"
collect-queries-json: true
collect-server-logs: true
collect-meta-refresh-log: true
collect-reflection-log: true
collect-gc-logs: true
collect-jfr: true
dremio-jfr-time-seconds: 60
collect-jstack: true
dremio-jstack-time-seconds: 60
dremio-jstack-freq-seconds: 10
dremio-ttop-time-seconds: 30 
dremio-ttop-freq-seconds: 5
collect-wlm: true
collect-ttop: true
collect-system-tables-export: true
collect-kvstore-report: true
collect-system-tables-timeout-seconds: 10
collect-cluster-id-timeout-seconds: 12
`, filepath.Join("testdata", "logs"), filepath.Join("testdata", "conf"), tarballOutDir, ts.URL)
	}
	cfgContent = strings.ReplaceAll(cfgContent, "\\", "\\\\")
	// Write the sample configuration to a file.
	err = os.WriteFile(cfgFilePath, []byte(cfgContent), 0600)
	if err != nil {
		log.Fatalf("unable to create conf file with error %v", err)
	}
	overrides = make(map[string]string)
}

var afterEachConfTest = func() {
	// Remove the configuration file after each test.
	err := os.Remove(cfgFilePath)
	if err != nil {
		log.Fatalf("unable to remove conf file with error %v", err)
	}
	// Remove the temporary directory after each test.
	err = os.RemoveAll(tmpDir)
	if err != nil {
		log.Fatalf("unable to remove conf dir with error %v", err)
	}
	if ts != nil {
		ts.Close()
	}
}

func TestConfReadingWithEUDremioCloud(t *testing.T) {
	genericConfSetup(`
is-dremio-cloud: true
dremio-cloud-project-id: "224653935291683895642623390599291234"
dremio-endpoint: eu.dremio.cloud
`)
	hook := shutdown.NewHook()
	defer hook.Cleanup()
	//should parse the configuration correctly
	cfg, err = conf.ReadConf(hook, overrides, cfgFilePath, collects.StandardCollection)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	if cfg == nil {
		t.Error("invalid conf")
	}

	if cfg.IsDremioCloud() != true {
		t.Error("expected is-dremio-cloud to be true")
	}
	expected := "224653935291683895642623390599291234"
	if cfg.DremioCloudProjectID() != expected {
		t.Errorf("expected dreimo-cloud-project-id to be %v but was %v", expected, cfg.DremioCloudProjectID())
	}

	if cfg.DremioEndpoint() != "https://api.eu.dremio.cloud" {
		t.Errorf("expected dremio-endpoint to be https://api.eu.dremio.cloud but was %v", cfg.DremioEndpoint())
	}

	if cfg.DremioCloudAppEndpoint() != "https://app.eu.dremio.cloud" {
		t.Errorf("expected dremio-cloud-app-endpoint to be https://app.eu.dremio.cloud but was %v", cfg.DremioCloudAppEndpoint())
	}
	afterEachConfTest()
}

func TestConfCanUseTarballOutputDirWithAllowedFiles(t *testing.T) {
	// allowed files are ddc, ddc.log, ddc.yaml
	outDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(outDir, "ddc.yaml"), []byte("test: 1"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "ddc"), []byte("myfile"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "ddc.log"), []byte("my log"), 0600); err != nil {
		t.Fatal(err)
	}
	logDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(logDir, "server.log"), []byte("my log"), 0600); err != nil {
		t.Fatal(err)
	}
	confDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(confDir, "dremio.conf"), []byte("my log"), 0600); err != nil {
		t.Fatal(err)
	}
	yamlText := fmt.Sprintf("tarball-out-dir: %v\ndremio-log-dir: %v\ndremio-conf-dir: %v\n", outDir, logDir, confDir)
	tmpDir, err = os.MkdirTemp("", "testdataabdc")
	if err != nil {
		log.Fatalf("unable to create dir with error %v", err)
	}
	cfgFilePath := filepath.Join(tmpDir, "ddc.yaml")
	if err := os.WriteFile(cfgFilePath, []byte(yamlText), 0600); err != nil {
		t.Fatal(err)
	}
	hook := shutdown.NewHook()
	defer hook.Cleanup()
	defer os.Remove(cfgFilePath)
	_, err = conf.ReadConf(hook, overrides, cfgFilePath, collects.StandardCollection)
	if err != nil {
		t.Errorf("should not have error: %v", err)
	}
}
func TestConfCannotUseTarballOutputDirWithFiles(t *testing.T) {
	//allowed files are ddc, ddc.log, ddc.yaml
	outDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(outDir, "ddc.yaml"), []byte("test: 1"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "ddc"), []byte("myfile"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "ddc.log"), []byte("my log"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "server.log"), []byte("my log"), 0600); err != nil {
		t.Fatal(err)
	}
	yamlText := fmt.Sprintf("tarball-out-dir: %v\ndremio-log-dir: %v\n", outDir, outDir)
	tmpDir, err = os.MkdirTemp("", "testdataabdc")
	if err != nil {
		log.Fatalf("unable to create dir with error %v", err)
	}
	cfgFilePath := filepath.Join(tmpDir, "ddc.yaml")
	if err := os.WriteFile(cfgFilePath, []byte(yamlText), 0600); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(cfgFilePath)
	hook := shutdown.NewHook()
	defer hook.Cleanup()
	cfg, err = conf.ReadConf(hook, overrides, cfgFilePath, collects.StandardCollection)
	if err == nil {
		t.Error("should have an error")
	}
	expected := fmt.Sprintf("cannot use directory '%v' for tarball output as it contains 1 entries: ([server.log])", outDir)
	if err.Error() != expected {
		t.Errorf("expected %v actual %v", expected, err.Error())
	}
}

func TestConfReadingWithDremioCloud(t *testing.T) {
	genericConfSetup(`
is-dremio-cloud: true
dremio-cloud-project-id: "224653935291683895642623390599291234"
dremio-endpoint: dremio.cloud
`)
	//should parse the configuration correctly
	hook := shutdown.NewHook()
	defer hook.Cleanup()
	cfg, err = conf.ReadConf(hook, overrides, cfgFilePath, collects.StandardCollection)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	if cfg == nil {
		t.Error("invalid conf")
	}

	if cfg.IsDremioCloud() != true {
		t.Error("expected is-dremio-cloud to be true")
	}

	expected := "224653935291683895642623390599291234"
	if cfg.DremioCloudProjectID() != expected {
		t.Errorf("expected dreimo-cloud-project-id to be %v but was %v", expected, cfg.DremioCloudProjectID())
	}

	if cfg.DremioEndpoint() != "https://api.dremio.cloud" {
		t.Errorf("expected dremio-endpoint to be https://api.dremio.cloud but was %v", cfg.DremioEndpoint())
	}

	if cfg.DremioCloudAppEndpoint() != "https://app.dremio.cloud" {
		t.Errorf("expected dremio-cloud-app-endpoint to be https://app.dremio.cloud but was %v", cfg.DremioCloudAppEndpoint())
	}
	afterEachConfTest()
}
func TestConfReadingWithAValidConfigurationFile(t *testing.T) {
	genericConfSetup("")
	//should parse the configuration correctly
	hook := shutdown.NewHook()
	defer hook.Cleanup()
	cfg, err = conf.ReadConf(hook, overrides, cfgFilePath, collects.StandardCollection)
	if err != nil {
		b, yamlErr := os.ReadFile(cfgFilePath)
		if yamlErr != nil {
			t.Fatal(err)
		}
		t.Errorf("unexpected error %v - yaml %s", err, b)
	}
	if cfg == nil {
		t.Error("invalid conf")
	}

	if cfg.DisableRESTAPI() != false {
		t.Errorf("Expected DisableRESTAPI to be true, got false")
	}

	if cfg.CollectAccelerationLogs() != true {
		t.Errorf("Expected CollectAccelerationLogs to be true, got false")
	}

	if cfg.CollectOSConfig() != true {
		t.Errorf("Expected CollectJVMConf to be true, got false")
	}

	if cfg.CollectJVMFlags() != true {
		t.Errorf("Expected CollectJVMConf to be true, got false")
	}
	if cfg.CollectAccessLogs() != true {
		t.Errorf("Expected CollectAccessLogs to be true, got false")
	}

	if cfg.CollectAuditLogs() != true {
		t.Errorf("Expected CollectAuditLogs to be true, got false")
	}

	if cfg.CollectDiskUsage() != true {
		t.Errorf("Expected CollectDiskUsage to be true, got false")
	}

	if cfg.CollectDremioConfiguration() != true {
		t.Errorf("Expected CollectDremioConfiguration to be true, got false")
	}

	if cfg.CollectKVStoreReport() != true {
		t.Errorf("Expected CollectKVStoreReport to be true, got false")
	}

	if cfg.CollectMetaRefreshLogs() != true {
		t.Errorf("Expected CollectMetaRefreshLogs to be true, got false")
	}

	if cfg.CollectQueriesJSON() != true {
		t.Errorf("Expected CollectQueriesJSON to be true, got false")
	}

	if cfg.CollectReflectionLogs() != true {
		t.Errorf("Expected CollectReflectionLogs to be true, got false")
	}

	if cfg.CollectServerLogs() != true {
		t.Errorf("Expected CollectServerLogs to be true, got false")
	}

	if cfg.CollectSystemTablesExport() != true {
		t.Errorf("Expected CollectSystemTablesExport to be true, got false")
	}

	if cfg.CollectWLM() != true {
		t.Errorf("Expected CollectWLM to be true, got false")
	}

	if cfg.CollectTtop() != true {
		t.Errorf("Expected CollectTtop to be true, got false")
	}

	if cfg.DremioTtopTimeSeconds() != 30 {
		t.Errorf("Expected to have 30 seconds for ttop time but was %v", cfg.DremioTtopTimeSeconds())
	}

	if cfg.DremioTtopFreqSeconds() != 5 {
		t.Errorf("Expected to have 5 seconds for ttop freq but was %v", cfg.DremioTtopFreqSeconds())
	}
	testConf := filepath.Join("testdata", "conf")
	if cfg.DremioConfDir() != testConf {
		t.Errorf("Expected DremioConfDir to be '%v', got '%s'", testConf, cfg.DremioConfDir())
	}
	if cfg.TarballOutDir() != tarballOutDir {
		t.Errorf("expected /my-tarball-dir but was %v", cfg.TarballOutDir())
	}
	if cfg.DremioPIDDetection() != false {
		t.Errorf("expected dremio-pid-detection to be disabled")
	}

	if cfg.CollectSystemTablesTimeoutSeconds() != 10 {
		t.Errorf("expected timeout of 10 seconds for system tables collection")
	}

	if cfg.CollectClusterIDTimeoutSeconds() != 12 {
		t.Errorf("expected timeout of 12 seconds for cluster id collection")
	}

	afterEachConfTest()
}

func TestConfReadWithDisabledRestAPIResultsInDisabledWLMJobProfileAndKVReport(t *testing.T) {
	yaml := fmt.Sprintf(`
dremio-log-dir: %v
dremio-conf-dir: %v
disable-rest-api: true
number-threads: 4
dremio-endpoint: "http://localhost:9047"
dremio-username: "admin"
dremio-pat-token: "your_personal_access_token"
number-job-profiles: 10
collect-wlm: true
collect-system-tables-export: true
collect-kvstore-report: true
`, filepath.Join("testdata", "logs"), filepath.Join("testdata", "conf"))
	genericConfSetup(yaml)
	defer afterEachConfTest()
	hook := shutdown.NewHook()
	defer hook.Cleanup()
	cfg, err = conf.ReadConf(hook, overrides, cfgFilePath, collects.StandardCollection)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	if cfg == nil {
		t.Fatal("invalid conf")
	}
	if cfg.CollectSystemTablesExport() == true {
		t.Error("expected collect system tables export to be false")
	}
	if cfg.CollectWLM() == true {
		t.Error("expected collect wlm to be false")
	}
	if cfg.CollectKVStoreReport() == true {
		t.Error("expected collect wlm to be false")
	}
	if cfg.NumberJobProfilesToCollect() != 0 {
		t.Errorf("expected number job profiles was %v but expected 0", cfg.NumberJobProfilesToCollect())
	}
	if cfg.JobProfilesNumHighQueryCost() != 0 {
		t.Errorf("expected number high query cost job profiles was %v but expected 0", cfg.JobProfilesNumHighQueryCost())
	}
	if cfg.JobProfilesNumRecentErrors() != 0 {
		t.Errorf("expected number high query cost job profiles was %v but expected 0", cfg.JobProfilesNumRecentErrors())
	}
	if cfg.JobProfilesNumSlowExec() != 0 {
		t.Errorf("expected number high query cost job profiles was %v but expected 0", cfg.JobProfilesNumSlowExec())
	}
	if cfg.JobProfilesNumSlowPlanning() != 0 {
		t.Errorf("expected number high query cost job profiles was %v but expected 0", cfg.JobProfilesNumSlowPlanning())
	}
}

func TestConfReadingWhenLoggingParsingOfDdcYAML(t *testing.T) {
	genericConfSetup("")
	testLog := filepath.Join(t.TempDir(), "ddc.log")
	simplelog.InitLoggerWithFile(testLog)
	//should log redacted when token is present
	hook := shutdown.NewHook()
	defer hook.Cleanup()
	cfg, err = conf.ReadConf(hook, overrides, cfgFilePath, collects.StandardCollection)
	if err != nil {
		t.Fatalf("expected no error but had %v", err)
	}
	if cfg == nil {
		t.Error("expected a valid CollectConf but it is nil")
	}
	if err := simplelog.Close(); err != nil {
		t.Fatal(err)
	}
	defer simplelog.InitLogger()
	b, err := os.ReadFile(testLog)
	if err != nil {
		t.Fatal(err)
	}
	out := string(b)

	if !strings.Contains(out, "conf key 'dremio-pat-token':'REDACTED'") {
		t.Errorf("expected dremio-pat-token to be redacted in '%v' but it was not", out)
	}
	afterEachConfTest()
}

func TestURLsuffix(t *testing.T) {
	testURL := "http://localhost:9047/some/path/"
	expected := "http://localhost:9047/some/path"
	actual := conf.SanitiseURL(testURL)
	if expected != actual {
		t.Errorf("\nexpected: %v\nactual: %v\n'", expected, actual)
	}

	testURL = "http://localhost:9047/some/path"
	expected = "http://localhost:9047/some/path"
	actual = conf.SanitiseURL(testURL)
	if expected != actual {
		t.Errorf("\nexpected: %v\nactual: %v\n'", expected, actual)
	}

}

func TestClusterStatsDirectory(t *testing.T) {
	genericConfSetup("")
	hook := shutdown.NewHook()
	defer hook.Cleanup()
	cfg, err = conf.ReadConf(hook, overrides, cfgFilePath, collects.StandardCollection)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	if cfg == nil {
		t.Error("invalid conf")
	}
	outDir := cfg.ClusterStatsOutDir()
	expected := filepath.Join("cluster-stats", "node1")
	if !strings.HasSuffix(outDir, expected) {
		t.Errorf("expected %v to end with %v", outDir, expected)
	}
}

func TestParsePSForConfig(t *testing.T) {
	ps := `   /opt/java/openjdk/bin/java -Djava.util.logging.config.class=org.slf4j.bridge.SLF4JBridgeHandler -Djava.library.path=/opt/dremio/lib -XX:+PrintGCDetails -XX:+PrintGCDateStamps -Ddremio.plugins.path=/opt/dremio/plugins -Xmx2048m -XX:MaxDirectMemorySize=2048m -XX:+HeapDumpOnOutOfMemoryError -XX:HeapDumpPath=/var/log/dremio -Dio.netty.maxDirectMemory=0 -Dio.netty.tryReflectionSetAccessible=true -DMAPR_IMPALA_RA_THROTTLE -DMAPR_MAX_RA_STREAMS=400 -XX:+UseG1GC -Ddremio.log.path=/opt/dremio/data/WRONG -Xloggc:/opt/dremio/data/logs/gc.log -XX:+PrintGCDetails -XX:+PrintGCDateStamps -XX:+PrintTenuringDistribution -XX:+PrintGCCause -XX:+UseGCLogFileRotation -XX:NumberOfGCLogFiles=10 -XX:GCLogFileSize=5M -Dzookeeper=zk-hs:2181 -Dservices.coordinator.enabled=true -Dservices.coordinator.master.enabled=true -Dservices.coordinator.master.embedded-zookeeper.enabled=false -Dservices.executor.enabled=false -Dservices.conduit.port=45679 -Ddremio.admin-only-mode=false -XX:+PrintClassHistogramBeforeFullGC -XX:+PrintClassHistogramAfterFullGC -cp /opt/dremio/conf:/opt/dremio/jars/*:/opt/dremio/jars/ext/*:/opt/dremio/jars/3rdparty/*:/opt/java/openjdk/lib/tools.jar com.dremio.dac.daemon.DremioDaemon DREMIO_PLUGINS_DIR=/opt/dremio/plugins KUBERNETES_SERVICE_PORT_HTTPS=443 KUBERNETES_SERVICE_PORT=443 DREMIO_LOG_DIR=/var/log/dremio JAVA_MAJOR_VERSION=8 DREMIO_IN_CONTAINER=1 HOSTNAME=dremio-master-0 LANGUAGE=en_US:en JAVA_HOME=/opt/java/openjdk AWS_CREDENTIAL_PROFILES_FILE=/opt/dremio/aws/credentials DREMIO_CLIENT_PORT_32010_TCP_PROTO=tcp MALLOC_ARENA_MAX=4 ZK_CS_PORT_2181_TCP_ADDR=192.10.1.1 DREMIO_GC_LOGS_ENABLED=yes DREMIO_CLASSPATH=/opt/dremio/conf:/opt/dremio/jars/*:/opt/dremio/jars/ext/*:/opt/dremio/jars/3rdparty/*:/opt/java/openjdk/lib/tools.jar DREMIO_MAX_HEAP_MEMORY_SIZE_MB=2048 DREMIO_CLIENT_PORT_9047_TCP_PORT=9047 PWD=/opt/dremio JAVA_VERSION_STRING=1.8.0_372 DREMIO_JAVA_SERVER_EXTRA_OPTS=-Ddremio.log.path=/opt/dremio/data/logs -Xloggc:/opt/dremio/data/logs/gc.log -XX:+PrintGCDetails -XX:+PrintGCDateStamps -XX:+PrintTenuringDistribution -XX:+PrintGCCause -XX:+UseGCLogFileRotation -XX:NumberOfGCLogFiles=10 -XX:GCLogFileSize=5M -Dzookeeper=zk-hs:2181 -Dservices.coordinator.enabled=true -Dservices.coordinator.master.enabled=true -Dservices.coordinator.master.embedded-zookeeper.enabled=false -Dservices.executor.enabled=false -Dservices.conduit.port=45679 DREMIO_MAX_DIRECT_MEMORY_SIZE_MB=2048 ZK_CS_PORT_2181_TCP_PROTO=tcp MALLOC_MMAP_MAX_=65536 DREMIO_CLIENT_PORT_32010_TCP_ADDR=192.10.1.13 DREMIO_CLIENT_PORT_31010_TCP_PROTO=tcp DREMIO_CONF_DIR=/opt/dremio/conf TZ=UTC ZK_CS_PORT=tcp://10.43.15.147:2181 DREMIO_ENV_SCRIPT=dremio-env DREMIO_CLIENT_PORT_31010_TCP_ADDR=192.10.1.1 HOME=/var/lib/dremio/dremio LANG=en_US.UTF-8 KUBERNETES_PORT_443_TCP=tcp://192.10.1.1:443 ZK_CS_PORT_2181_TCP_PORT=2181 DREMIO_CLIENT_PORT_9047_TCP_PROTO=tcp LOG_TO_CONSOLE=0 DREMIO_ADMIN_ONLY=false DREMIO_CLIENT_PORT=tcp://192.10.1.13:31010 DREMIO_CLIENT_SERVICE_HOST=192.10.1.13 DREMIO_HOME=/opt/dremio ZK_CS_SERVICE_PORT_CLIENT=2181 DREMIO_CLIENT_SERVICE_PORT_WEB=9047 ZK_CS_SERVICE_PORT=2181 DREMIO_CLIENT_PORT_31010_TCP=tcp://192.10.1.13:31010 DREMIO_CLIENT_SERVICE_PORT_CLIENT=31010 DREMIO_CLIENT_PORT_9047_TCP=tcp://192.10.1.13:9047 DREMIO_PID_DIR=/var/run/dremio DREMIO_CLIENT_SERVICE_PORT=31010 MALLOC_TRIM_THRESHOLD_=131072 DREMIO_GC_OPTS=-XX:+UseG1GC SHLVL=0 DREMIO_CLIENT_PORT_31010_TCP_PORT=31010 DREMIO_GC_LOG_TO_CONSOLE=yes KUBERNETES_PORT_443_TCP_PROTO=tcp is_cygwin=false MALLOC_MMAP_THRESHOLD_=131072 KUBERNETES_PORT_443_TCP_ADDR=10.43.0.1 KUBERNETES_SERVICE_HOST=10.43.0.1 LC_ALL=en_US.UTF-8 AWS_SHARED_CREDENTIALS_FILE=/opt/dremio/aws/credentials KUBERNETES_PORT=tcp://10.43.0.1:443 DREMIO_CLIENT_PORT_9047_TCP_ADDR=192.10.1.13 KUBERNETES_PORT_443_TCP_PORT=443 PATH=/opt/java/openjdk/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin MALLOC_TOP_PAD_=131072 DREMIO_JAVA_OPTS=-Djava.util.logging.config.class=org.slf4j.bridge.SLF4JBridgeHandler -Djava.library.path=/opt/dremio/lib -XX:+PrintGCDetails -XX:+PrintGCDateStamps -Ddremio.plugins.path=/opt/dremio/plugins -Xmx2048m -XX:MaxDirectMemorySize=2048m -XX:+HeapDumpOnOutOfMemoryError -XX:HeapDumpPath=/var/log/dremio -Dio.netty.maxDirectMemory=0 -Dio.netty.tryReflectionSetAccessible=true -DMAPR_IMPALA_RA_THROTTLE -DMAPR_MAX_RA_STREAMS=400 -XX:+UseG1GC -Ddremio.log.path=/opt/dremio/data/logs -Xloggc:/opt/dremio/data/logs/gc.log -XX:+PrintGCDetails -XX:+PrintGCDateStamps -XX:+PrintTenuringDistribution -XX:+PrintGCCause -XX:+UseGCLogFileRotation -XX:NumberOfGCLogFiles=10 -XX:GCLogFileSize=5M -Dzookeeper=zk-hs:2181 -Dservices.coordinator.enabled=true -Dservices.coordinator.master.enabled=true -Dservices.coordinator.master.embedded-zookeeper.enabled=false -Dservices.executor.enabled=false -Dservices.conduit.port=45679 -Ddremio.admin-only-mode=false -XX:+PrintClassHistogramBeforeFullGC -XX:+PrintClassHistogramAfterFullGC DREMIO_CLIENT_PORT_32010_TCP=tcp://192.10.1.1:32010 ZK_CS_SERVICE_HOST=192.10.1.1 DREMIO_CLIENT_SERVICE_PORT_FLIGHT=32010 DREMIO_LOG_TO_CONSOLE=1 DREMIO_CLIENT_PORT_32010_TCP_PORT=32010 JAVA_VERSION=jdk8u372-b07 ZK_CS_PORT_2181_TCP=tcp://192.10.1.1:2181`
	conf, err := conf.ParsePSForConfig(ps)
	if err != nil {
		t.Fatal(err)
	}
	if conf.ConfDir != "/opt/dremio/conf" {
		t.Errorf("expected /opt/dremio/conf but was %v", conf.ConfDir)
	}

	if conf.LogDir != "/opt/dremio/data/logs" {
		t.Errorf("expected /opt/dremio/data/logs but was %v", conf.LogDir)
	}

	if conf.Home != "/opt/dremio" {
		t.Errorf("expected /opt/dremio but was %q", conf.Home)
	}
}

func TestParsePSForConfigWithNewLines(t *testing.T) {
	ps := "/opt/java/openjdk/bin/java -Djava.util.logging.config.class=org.slf4j.bridge.SLF4JBridgeHandler -Djava.library.path=/opt/dremio/lib -XX:+PrintGCDetails -XX:+PrintGCDateStamps -Ddremio.plugins.path=/opt/dremio/plugins -Xmx2048m -XX:MaxDirectMemorySize=2048m -XX:+HeapDumpOnOutOfMemoryError -XX:HeapDumpPath=/var/log/dremio -Dio.netty.maxDirectMemory=0 -Dio.netty.tryReflectionSetAccessible=true -DMAPR_IMPALA_RA_THROTTLE -DMAPR_MAX_RA_STREAMS=400 -XX:+UseG1GC -Ddremio.log.path=/opt/dremio/data/logs\n -Xloggc:/opt/dremio/data/logs/gc.log -XX:+PrintGCDetails -XX:+PrintGCDateStamps -XX:+PrintTenuringDistribution -XX:+PrintGCCause -XX:+UseGCLogFileRotation -XX:NumberOfGCLogFiles=10 -XX:GCLogFileSize=5M -Dzookeeper=zk-hs:2181 -Dservices.coordinator.enabled=true -Dservices.coordinator.master.enabled=true -Dservices.coordinator.master.embedded-zookeeper.enabled=false -Dservices.executor.enabled=false -Dservices.conduit.port=45679 -Ddremio.admin-only-mode=false -XX:+PrintClassHistogramBeforeFullGC -XX:+PrintClassHistogramAfterFullGC -cp /opt/dremio/conf:/opt/dremio/jars/*:/opt/dremio/jars/ext/*:/opt/dremio/jars/3rdparty/*:/opt/java/openjdk/lib/tools.jar com.dremio.dac.daemon.DremioDaemon DREMIO_PLUGINS_DIR=/opt/dremio/plugins KUBERNETES_SERVICE_PORT_HTTPS=443 KUBERNETES_SERVICE_PORT=443 DREMIO_LOG_DIR=/var/log/dremio\n JAVA_MAJOR_VERSION=8 DREMIO_IN_CONTAINER=1 HOSTNAME=dremio-master-0 LANGUAGE=en_US:en JAVA_HOME=/opt/java/openjdk AWS_CREDENTIAL_PROFILES_FILE=/opt/dremio/aws/credentials DREMIO_CLIENT_PORT_32010_TCP_PROTO=tcp MALLOC_ARENA_MAX=4 ZK_CS_PORT_2181_TCP_ADDR=192.10.1.1 DREMIO_GC_LOGS_ENABLED=yes DREMIO_CLASSPATH=/opt/dremio/conf:/opt/dremio/jars/*:/opt/dremio/jars/ext/*:/opt/dremio/jars/3rdparty/*:/opt/java/openjdk/lib/tools.jar DREMIO_MAX_HEAP_MEMORY_SIZE_MB=2048 DREMIO_CLIENT_PORT_9047_TCP_PORT=9047 PWD=/opt/dremio JAVA_VERSION_STRING=1.8.0_372 DREMIO_JAVA_SERVER_EXTRA_OPTS=-Ddremio.log.path=/opt/dremio/data/logs\n -Xloggc:/opt/dremio/data/logs/gc.log -XX:+PrintGCDetails -XX:+PrintGCDateStamps -XX:+PrintTenuringDistribution -XX:+PrintGCCause -XX:+UseGCLogFileRotation -XX:NumberOfGCLogFiles=10 -XX:GCLogFileSize=5M -Dzookeeper=zk-hs:2181 -Dservices.coordinator.enabled=true -Dservices.coordinator.master.enabled=true -Dservices.coordinator.master.embedded-zookeeper.enabled=false -Dservices.executor.enabled=false -Dservices.conduit.port=45679 DREMIO_MAX_DIRECT_MEMORY_SIZE_MB=2048 ZK_CS_PORT_2181_TCP_PROTO=tcp MALLOC_MMAP_MAX_=65536 DREMIO_CLIENT_PORT_32010_TCP_ADDR=192.10.1.13 DREMIO_CLIENT_PORT_31010_TCP_PROTO=tcp DREMIO_CONF_DIR=/opt/dremio/conf\n TZ=UTC ZK_CS_PORT=tcp://10.43.15.147:2181 DREMIO_ENV_SCRIPT=dremio-env DREMIO_CLIENT_PORT_31010_TCP_ADDR=192.10.1.1 HOME=/var/lib/dremio/dremio LANG=en_US.UTF-8 KUBERNETES_PORT_443_TCP=tcp://192.10.1.1:443 ZK_CS_PORT_2181_TCP_PORT=2181 DREMIO_CLIENT_PORT_9047_TCP_PROTO=tcp LOG_TO_CONSOLE=0 DREMIO_ADMIN_ONLY=false DREMIO_CLIENT_PORT=tcp://192.10.1.13:31010 DREMIO_CLIENT_SERVICE_HOST=192.10.1.13 DREMIO_HOME=/opt/dremio\n ZK_CS_SERVICE_PORT_CLIENT=2181 DREMIO_CLIENT_SERVICE_PORT_WEB=9047 ZK_CS_SERVICE_PORT=2181 DREMIO_CLIENT_PORT_31010_TCP=tcp://192.10.1.13:31010 DREMIO_CLIENT_SERVICE_PORT_CLIENT=31010 DREMIO_CLIENT_PORT_9047_TCP=tcp://192.10.1.13:9047 DREMIO_PID_DIR=/var/run/dremio DREMIO_CLIENT_SERVICE_PORT=31010 MALLOC_TRIM_THRESHOLD_=131072 DREMIO_GC_OPTS=-XX:+UseG1GC SHLVL=0 DREMIO_CLIENT_PORT_31010_TCP_PORT=31010 DREMIO_GC_LOG_TO_CONSOLE=yes KUBERNETES_PORT_443_TCP_PROTO=tcp is_cygwin=false MALLOC_MMAP_THRESHOLD_=131072 KUBERNETES_PORT_443_TCP_ADDR=10.43.0.1 KUBERNETES_SERVICE_HOST=10.43.0.1 LC_ALL=en_US.UTF-8 AWS_SHARED_CREDENTIALS_FILE=/opt/dremio/aws/credentials KUBERNETES_PORT=tcp://10.43.0.1:443 DREMIO_CLIENT_PORT_9047_TCP_ADDR=192.10.1.13 KUBERNETES_PORT_443_TCP_PORT=443 PATH=/opt/java/openjdk/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin MALLOC_TOP_PAD_=131072 DREMIO_JAVA_OPTS=-Djava.util.logging.config.class=org.slf4j.bridge.SLF4JBridgeHandler -Djava.library.path=/opt/dremio/lib -XX:+PrintGCDetails -XX:+PrintGCDateStamps -Ddremio.plugins.path=/opt/dremio/plugins -Xmx2048m -XX:MaxDirectMemorySize=2048m -XX:+HeapDumpOnOutOfMemoryError -XX:HeapDumpPath=/var/log/dremio -Dio.netty.maxDirectMemory=0 -Dio.netty.tryReflectionSetAccessible=true -DMAPR_IMPALA_RA_THROTTLE -DMAPR_MAX_RA_STREAMS=400 -XX:+UseG1GC -Ddremio.log.path=/opt/dremio/data/logs -Xloggc:/opt/dremio/data/logs/gc.log -XX:+PrintGCDetails -XX:+PrintGCDateStamps -XX:+PrintTenuringDistribution -XX:+PrintGCCause -XX:+UseGCLogFileRotation -XX:NumberOfGCLogFiles=10 -XX:GCLogFileSize=5M -Dzookeeper=zk-hs:2181 -Dservices.coordinator.enabled=true -Dservices.coordinator.master.enabled=true -Dservices.coordinator.master.embedded-zookeeper.enabled=false -Dservices.executor.enabled=false -Dservices.conduit.port=45679 -Ddremio.admin-only-mode=false -XX:+PrintClassHistogramBeforeFullGC -XX:+PrintClassHistogramAfterFullGC DREMIO_CLIENT_PORT_32010_TCP=tcp://192.10.1.1:32010 ZK_CS_SERVICE_HOST=192.10.1.1 DREMIO_CLIENT_SERVICE_PORT_FLIGHT=32010 DREMIO_LOG_TO_CONSOLE=1 DREMIO_CLIENT_PORT_32010_TCP_PORT=32010 JAVA_VERSION=jdk8u372-b07 ZK_CS_PORT_2181_TCP=tcp://192.10.1.1:2181"
	conf, err := conf.ParsePSForConfig(ps)
	if err != nil {
		t.Fatal(err)
	}
	if conf.ConfDir != "/opt/dremio/conf" {
		t.Errorf("expected /opt/dremio/conf but was %v", conf.ConfDir)
	}

	if conf.LogDir != "/opt/dremio/data/logs" {
		t.Errorf("expected /opt/dremio/data/logs but was %v", conf.LogDir)
	}

	if conf.Home != "/opt/dremio" {
		t.Errorf("expected /opt/dremio but was %q", conf.Home)
	}
}

func TestLoggingDirsHaveExpectedFiles(t *testing.T) {
	yaml := fmt.Sprintf(`
dremio-log-dir: %v
dremio-conf-dir: %v
`, filepath.Join("testdata", "badlogs"), filepath.Join("testdata", "conf"))
	genericConfSetup(yaml)
	hook := shutdown.NewHook()
	defer hook.Cleanup()
	// we expect an error since "badlogs" doesnt have the right files
	cfg, err = conf.ReadConf(hook, overrides, cfgFilePath, collects.StandardCollection)
	if cfg == nil {
		t.Error("expected a valid CollectConf but it is nil")
	}
	// typically the config directory "badlogs" does not have the right files, so
	// we fall back to the auto-detect which in this case comes up empty (expected)
	expected := `invalid dremio log dir '', update ddc.yaml and fix it: directory does not exist`
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("\nexpected:\n%v\nactual:\n%v", expected, err.Error())
	}

	// reset config (so we point to the normal logs dir)
	genericConfSetup("")
	// we don't expect an error since "logs" has the right files
	cfg, err = conf.ReadConf(hook, overrides, cfgFilePath, collects.StandardCollection)
	if err != nil {
		t.Fatal(err)
	}
	if cfg == nil {
		t.Error("expected a valid CollectConf but it is nil")
	}

	afterEachConfTest()
}

func TestEnvVarsForLogging(t *testing.T) {
	ps := `   /opt/java/openjdk/bin/java -Djava.util.logging.config.class=org.slf4j.bridge.SLF4JBridgeHandler -Djava.library.path=/opt/dremio/lib -XX:+PrintGCDetails -XX:+PrintGCDateStamps -Ddremio.plugins.path=/opt/dremio/plugins -Xmx2048m -XX:MaxDirectMemorySize=2048m -XX:+HeapDumpOnOutOfMemoryError -XX:HeapDumpPath=/var/log/dremio/blah -Dio.netty.maxDirectMemory=0 -Dio.netty.tryReflectionSetAccessible=true -DMAPR_IMPALA_RA_THROTTLE -DMAPR_MAX_RA_STREAMS=400 -XX:+UseG1GC -Xloggc:/opt/dremio/data/logs/gc.log -XX:+PrintGCDetails -XX:+PrintGCDateStamps -XX:+PrintTenuringDistribution -XX:+PrintGCCause -XX:+UseGCLogFileRotation -XX:NumberOfGCLogFiles=10 -XX:GCLogFileSize=5M -Dzookeeper=zk-hs:2181 -Dservices.coordinator.enabled=true -Dservices.coordinator.master.enabled=true -Dservices.coordinator.master.embedded-zookeeper.enabled=false -Dservices.executor.enabled=false -Dservices.conduit.port=45679 -Ddremio.admin-only-mode=false -XX:+PrintClassHistogramBeforeFullGC -XX:+PrintClassHistogramAfterFullGC -cp /opt/dremio/conf:/opt/dremio/jars/*:/opt/dremio/jars/ext/*:/opt/dremio/jars/3rdparty/*:/opt/java/openjdk/lib/tools.jar com.dremio.dac.daemon.DremioDaemon DREMIO_PLUGINS_DIR=/opt/dremio/plugins KUBERNETES_SERVICE_PORT_HTTPS=443 KUBERNETES_SERVICE_PORT=443 DREMIO_LOG_DIR=/var/log/dremio/backup JAVA_MAJOR_VERSION=8 DREMIO_IN_CONTAINER=1 HOSTNAME=dremio-master-0 LANGUAGE=en_US:en JAVA_HOME=/opt/java/openjdk AWS_CREDENTIAL_PROFILES_FILE=/opt/dremio/aws/credentials DREMIO_CLIENT_PORT_32010_TCP_PROTO=tcp MALLOC_ARENA_MAX=4 ZK_CS_PORT_2181_TCP_ADDR=192.10.1.1 DREMIO_GC_LOGS_ENABLED=yes DREMIO_CLASSPATH=/opt/dremio/conf:/opt/dremio/jars/*:/opt/dremio/jars/ext/*:/opt/dremio/jars/3rdparty/*:/opt/java/openjdk/lib/tools.jar DREMIO_MAX_HEAP_MEMORY_SIZE_MB=2048 DREMIO_CLIENT_PORT_9047_TCP_PORT=9047 PWD=/opt/dremio JAVA_VERSION_STRING=1.8.0_372 -Xloggc:/opt/dremio/data/logs/gc.log -XX:+PrintGCDetails -XX:+PrintGCDateStamps -XX:+PrintTenuringDistribution -XX:+PrintGCCause -XX:+UseGCLogFileRotation -XX:NumberOfGCLogFiles=10 -XX:GCLogFileSize=5M -Dzookeeper=zk-hs:2181 -Dservices.coordinator.enabled=true -Dservices.coordinator.master.enabled=true -Dservices.coordinator.master.embedded-zookeeper.enabled=false -Dservices.executor.enabled=false -Dservices.conduit.port=45679 DREMIO_MAX_DIRECT_MEMORY_SIZE_MB=2048 ZK_CS_PORT_2181_TCP_PROTO=tcp MALLOC_MMAP_MAX_=65536 DREMIO_CLIENT_PORT_32010_TCP_ADDR=192.10.1.13 DREMIO_CLIENT_PORT_31010_TCP_PROTO=tcp DREMIO_CONF_DIR=/opt/dremio/conf TZ=UTC ZK_CS_PORT=tcp://10.43.15.147:2181 DREMIO_ENV_SCRIPT=dremio-env DREMIO_CLIENT_PORT_31010_TCP_ADDR=192.10.1.1 HOME=/var/lib/dremio/dremio LANG=en_US.UTF-8 KUBERNETES_PORT_443_TCP=tcp://192.10.1.1:443 ZK_CS_PORT_2181_TCP_PORT=2181 DREMIO_CLIENT_PORT_9047_TCP_PROTO=tcp LOG_TO_CONSOLE=0 DREMIO_ADMIN_ONLY=false DREMIO_CLIENT_PORT=tcp://192.10.1.13:31010 DREMIO_CLIENT_SERVICE_HOST=192.10.1.13 DREMIO_HOME=/opt/dremio ZK_CS_SERVICE_PORT_CLIENT=2181 DREMIO_CLIENT_SERVICE_PORT_WEB=9047 ZK_CS_SERVICE_PORT=2181 DREMIO_CLIENT_PORT_31010_TCP=tcp://192.10.1.13:31010 DREMIO_CLIENT_SERVICE_PORT_CLIENT=31010 DREMIO_CLIENT_PORT_9047_TCP=tcp://192.10.1.13:9047 DREMIO_PID_DIR=/var/run/dremio DREMIO_CLIENT_SERVICE_PORT=31010 MALLOC_TRIM_THRESHOLD_=131072 DREMIO_GC_OPTS=-XX:+UseG1GC SHLVL=0 DREMIO_CLIENT_PORT_31010_TCP_PORT=31010 DREMIO_GC_LOG_TO_CONSOLE=yes KUBERNETES_PORT_443_TCP_PROTO=tcp is_cygwin=false MALLOC_MMAP_THRESHOLD_=131072 KUBERNETES_PORT_443_TCP_ADDR=10.43.0.1 KUBERNETES_SERVICE_HOST=10.43.0.1 LC_ALL=en_US.UTF-8 AWS_SHARED_CREDENTIALS_FILE=/opt/dremio/aws/credentials KUBERNETES_PORT=tcp://10.43.0.1:443 DREMIO_CLIENT_PORT_9047_TCP_ADDR=192.10.1.13 KUBERNETES_PORT_443_TCP_PORT=443 PATH=/opt/java/openjdk/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin MALLOC_TOP_PAD_=131072 DREMIO_JAVA_OPTS=-Djava.util.logging.config.class=org.slf4j.bridge.SLF4JBridgeHandler -Djava.library.path=/opt/dremio/lib -XX:+PrintGCDetails -XX:+PrintGCDateStamps -Ddremio.plugins.path=/opt/dremio/plugins -Xmx2048m -XX:MaxDirectMemorySize=2048m -XX:+HeapDumpOnOutOfMemoryError -XX:HeapDumpPath=/var/log/dremio -Dio.netty.maxDirectMemory=0 -Dio.netty.tryReflectionSetAccessible=true -DMAPR_IMPALA_RA_THROTTLE -DMAPR_MAX_RA_STREAMS=400 -XX:+UseG1GC -Xloggc:/opt/dremio/data/logs/gc.log -XX:+PrintGCDetails -XX:+PrintGCDateStamps -XX:+PrintTenuringDistribution -XX:+PrintGCCause -XX:+UseGCLogFileRotation -XX:NumberOfGCLogFiles=10 -XX:GCLogFileSize=5M -Dzookeeper=zk-hs:2181 -Dservices.coordinator.enabled=true -Dservices.coordinator.master.enabled=true -Dservices.coordinator.master.embedded-zookeeper.enabled=false -Dservices.executor.enabled=false -Dservices.conduit.port=45679 -Ddremio.admin-only-mode=false -XX:+PrintClassHistogramBeforeFullGC -XX:+PrintClassHistogramAfterFullGC DREMIO_CLIENT_PORT_32010_TCP=tcp://192.10.1.1:32010 ZK_CS_SERVICE_HOST=192.10.1.1 DREMIO_CLIENT_SERVICE_PORT_FLIGHT=32010 DREMIO_LOG_TO_CONSOLE=1 DREMIO_CLIENT_PORT_32010_TCP_PORT=32010 JAVA_VERSION=jdk8u372-b07 ZK_CS_PORT_2181_TCP=tcp://192.10.1.1:2181`
	expected := "/var/log/dremio/backup"
	genericConfSetup("")

	// Parse the ps line for logs, expect fallback to env var
	psConf, err := conf.ParsePSForConfig(ps)
	if err != nil {
		t.Fatal(err)
	}
	// we expect the Log dir to not have picked things up from the PS config above, instead reading if from the ENV var
	if psConf.LogDir != expected {
		t.Errorf("\nexpected:\n%v\nactual:\n%v", expected, psConf.LogDir)
	}

	afterEachConfTest()
}
