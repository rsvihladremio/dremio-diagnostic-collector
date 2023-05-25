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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	dockertest "github.com/ory/dockertest/v3"
	dc "github.com/ory/dockertest/v3/docker"
)

type AuthResponse struct {
	Token string `json:"token"`
}

type AuthRequest struct {
	Username string `json:"userName"`
	Password string `json:"password"`
}

type JobAPIResponse struct {
	ID string `json:"id"`
}

var dremioTestPort string

func cleanupOutput() {
	if err := os.RemoveAll(outputDir); err != nil {
		log.Printf("WARN unable to remove %v it may have to be manually cleaned up", outputDir)
	}
}

// TestMain setups up a docker runtime and we use this to spin up dremio https://github.com/ory/dockertest
func TestMain(m *testing.M) {
	// uses a sensible default on windows (tcp/http) and linux/osx (socket)
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not construct pool: %s", err)
	}
	pool.MaxWait = time.Minute * 5
	// uses pool to try to connect to Docker
	err = pool.Client.Ping()
	if err != nil {
		log.Fatalf("Could not connect to Docker: %s", err)
	}

	// pulls an image, creates a container based on it and runs it
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "dremio/dremio-ee",
		Tag:        "24.0",
		//Env:        []string{},
	}, func(config *dc.HostConfig) {
		// set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = dc.RestartPolicy{
			Name: "no",
		}
		pwd, err := os.Getwd()
		if err != nil {
			log.Fatalf("failed to get working directory: %s", err)
		}
		config.Mounts = []dc.HostMount{
			{
				Target: "/opt/dremio/conf/dremio.conf",
				Source: fmt.Sprintf("%s/testdata/conf/dremio.conf", pwd),
				Type:   "bind",
			},
		}
	})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}
	dremioTestPort = resource.GetPort("9047/tcp")
	exit, err := resource.Exec([]string{"mkdir", "/tmp/dremio-source"}, dockertest.ExecOptions{})
	if err != nil {
		log.Fatalf("could not make dremio source: %s", err)
	}
	if exit > 0 {
		log.Fatalf("unable to make dremio source due to exit code %d", exit)
	}
	err = resource.Expire(60 * 5)
	if err != nil {
		log.Fatalf("Could not set expiry on resource : %s", err)
	}

	requestURL := fmt.Sprintf("http://localhost:%v", dremioTestPort)
	dremioEndpoint = requestURL

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	if err := pool.Retry(func() error {
		res, err := http.Get(requestURL) //nolint
		if err != nil {
			log.Printf("error making http request: %s\n", err)
			return err
		}
		expectedCode := 200
		if res.StatusCode != expectedCode {
			return fmt.Errorf("expected status code %v but instead got %v. Dremio is not ready", expectedCode, res.StatusCode)
		}
		// accept EULA
		var empty bytes.Buffer
		eulaURL := fmt.Sprintf("http://localhost:%v/apiv2/eula/accept", dremioTestPort)
		res, err = http.Post(eulaURL, "application/json", &empty) //nolint
		if err != nil {
			log.Printf("error accepting EULA request: %s\n", err)
			return err
		}
		if res.StatusCode != 204 {
			return fmt.Errorf("expected status code 204 but instead got %v while trying to accept EULA", res.StatusCode)
		}
		dremioUsername = "dremio"
		authRequest := &AuthRequest{
			Username: "dremio",
			Password: "dremio123",
		}
		body, err := json.Marshal(authRequest)
		if err != nil {
			return fmt.Errorf("Error marshaling JSON: %v", err)
		}
		res, err = http.Post(fmt.Sprintf("http://localhost:%v/apiv2/login", dremioTestPort), "application/json", bytes.NewBuffer(body))
		if err != nil {
			log.Fatalf("error logging in to get token : %s\n", err)
		}
		defer res.Body.Close()
		if res.StatusCode != expectedCode {
			text, err := io.ReadAll(res.Body)
			if err != nil {
				log.Fatalf("fatal attempt to decode body from dremio auth %v and unable to read body for debugging", err)
			}
			log.Printf("body was %s", string(text))
			log.Fatalf("expected status code %v but instead got %v with message %v. Unable to get dremio PAT", expectedCode, res.StatusCode, res.Status)
		}
		var authResponse AuthResponse
		err = json.NewDecoder(res.Body).Decode(&authResponse)
		if err != nil {
			text, err := io.ReadAll(res.Body)
			if err != nil {
				log.Fatalf("fatal attempt to decode body from dremio auth %v and unable to read body for debugging", err)
			}
			log.Printf("body was %s", string(text))
			log.Fatalf("fatal attempt to decode body from dremio auth %v", err)
		}
		dremioPATToken = authResponse.Token

		nasSource := `{
			"metadataPolicy": {
				"authTTLMs":86400000,
        		"namesRefreshMs":3600000,
        		"datasetRefreshAfterMs": 3600000,
        		"datasetExpireAfterMs": 10800000,
        		"datasetUpdateMode":"PREFETCH_QUERIED",
        		"deleteUnavailableDatasets": true,
        		"autoPromoteDatasets": true
        	},
			"config": {
			  	"path": "/tmp/dremio-source/",
			  	"defaultCtasFormat": "ICEBERG"
			},
			"entityType": "source",
			"type": "NAS",
			"name": "tester"
		  }`
		req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:%v/apiv3/catalog", dremioTestPort), bytes.NewBuffer([]byte(nasSource)))
		if err != nil {
			return fmt.Errorf("unable to create data source request")
		}
		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("Authorization", "_dremio"+dremioPATToken)
		res, err = http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("unable to create data source due to error %v", err)
		}
		if res.StatusCode != 200 {
			return fmt.Errorf("expected status code 200 but instead got %v while trying to create source", res.StatusCode)
		}
		return nil
	}); err != nil {
		log.Fatalf("Could not connect to dremio: %s", err)
	}
	outputDir = "testdata/output"
	code := m.Run()
	cleanupOutput()
	// You can't defer this because os.Exit doesn't care for defer
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}
	os.Exit(code)
}

func TestCreateAllDirs(t *testing.T) {
	err := createAllDirs()
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
}

func TestCollectWlm(t *testing.T) {
	err := runCollectWLM()
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
}

func TestCollectKVReport(t *testing.T) {
	err := runCollectKvReport()
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
}

// TODO figure out why this is failing
// func TestCollectDremioSystemTables(t *testing.T) {
// 	err := collectDremioSystemTables()
// 	if err != nil {
// 		t.Errorf("unexpected error %v", err)
// 	}
// }

func TestDownloadJobProfile(t *testing.T) {
	sql := `{
		"sql": "CREATE TABLE tester.table1 AS SELECT \"a\", \"b\" FROM (values (CAST(1 AS INTEGER), CAST(2 AS INTEGER))) as t(\"a\", \"b\")"
	}`
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:%v/api/v3/sql/", dremioTestPort), bytes.NewBuffer([]byte(sql)))
	if err != nil {
		t.Fatalf("unable to create table request %v", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "_dremio"+dremioPATToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("unable to create table %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode > 299 {
		text, err := io.ReadAll(res.Body)
		if err != nil {
			log.Fatalf("fatal attempt to make job api call %v and unable to read body for debugging", err)
		}
		log.Printf("body was %s", string(text))
		t.Fatalf("expected status code greater than 299 but instead got %v while trying to create source", res.StatusCode)
	}
	var jobResponse JobAPIResponse
	err = json.NewDecoder(res.Body).Decode(&jobResponse)
	if err != nil {
		text, err := io.ReadAll(res.Body)
		if err != nil {
			log.Fatalf("fatal attempt to decode body from dremio job api call %v and unable to read body for debugging", err)
		}
		log.Printf("body was %s", string(text))
		log.Fatalf("fatal attempt to decode body from dremio job api %v", err)
	}
	time.Sleep(10 * time.Second)
	jobid := jobResponse.ID
	err = downloadJobProfile(jobid)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
}

func TestValidateAPICredentials(t *testing.T) {
	err := validateAPICredentials()
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
}

func TestValidateCollectJobProfiles(t *testing.T) {
	err := runCollectJobProfiles()
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
}

func TestParseGCLogFromFlags(t *testing.T) {
	processFlags := `1:
VM Arguments:
jvm_args: -Djava.util.logging.config.class=org.slf4j.bridge.SLF4JBridgeHandler -Djava.library.path=/opt/dremio/lib -XX:+PrintGCDetails -XX:+PrintGCDateStamps -Ddremio.plugins.path=/opt/dremio/plugins -Xmx4096m -XX:MaxDirectMemorySize=2048m -XX:+HeapDumpOnOutOfMemoryError -XX:HeapDumpPath=/var/log/dremio -Dio.netty.maxDirectMemory=0 -Dio.netty.tryReflectionSetAccessible=true -DMAPR_IMPALA_RA_THROTTLE -DMAPR_MAX_RA_STREAMS=400 -XX:+UseG1GC -Ddremio.log.path=/opt/dremio/data/log -Xloggc:/opt/dremio/data/log/gc.log -XX:+UseGCLogFileRotation -XX:NumberOfGCLogFiles=5 -XX:GCLogFileSize=4000k -XX:+PrintGCDetails -XX:+PrintGCTimeStamps -XX:+PrintGCDateStamps -XX:+PrintClassHistogramBeforeFullGC -XX:+PrintClassHistogramAfterFullGC -XX:+HeapDumpOnOutOfMemoryError -XX:HeapDumpPath=/opt/dremio/data -XX:+UseG1GC -XX:G1HeapRegionSize=32M -XX:MaxGCPauseMillis=500 -XX:InitiatingHeapOccupancyPercent=25 -XX:+PrintAdaptiveSizePolicy -XX:+PrintReferenceGC -XX:ErrorFile=/opt/dremio/data/hs_err_pid%p.log -Dzookeeper=zk-hs:2181 -Dservices.coordinator.enabled=false -Dservices.coordinator.master.enabled=false -Dservices.coordinator.master.embedded-zookeeper.enabled=false -Dservices.executor.enabled=true -Dservices.conduit.port=45679 -Dservices.node-tag=default -XX:+PrintClassHistogramBeforeFullGC -XX:+PrintClassHistogramAfterFullGC 
java_command: com.dremio.dac.daemon.DremioDaemon
java_class_path (initial): /opt/dremio/conf:/opt/dremio/jars/dremio-services-coordinator-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-hive-function-registry-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-ce-sabot-serializer-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-hive2-plugin-launcher-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-ee-services-credentials-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-ee-services-accesscontrol-common-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-ce-sabot-scheduler-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-dac-common-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-services-usersessions-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-ee-services-sysflight-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-protocol-20.0.0-202201050826310141-8cc7162b-proto.jar:/opt/dremio/jars/dremio-services-telemetry-impl-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-services-jobtelemetry-client-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-hive3-plugin-launcher-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-ce-services-cachemanager-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-ee-dac-tools-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-services-base-rpc-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-services-datastore-20.0.0-202201050826310141-8cc7162b-proto.jar:/opt/dremio/jars/dremio-sabot-logical-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-services-transientstore-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-services-resourcescheduler-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-dac-daemon-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-ee-services-namespace-20.0.0-202201050826310141-8cc7162b-tests.jar:/opt/dremio/j
Launcher Type: SUN_STANDARD`
	gcLogLocation, err := ParseGCLogFromFlags(processFlags)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	expected := "/opt/dremio/data/log"
	if gcLogLocation != expected {
		t.Errorf("expected '%v' but was '%v'", expected, gcLogLocation)
	}
}

func TestParseGCLogFromFlagsWithExtraLogFileLine(t *testing.T) {
	processFlags := `kubectl exec -it -n default -c dremio-master-coordinator dremio-master-0 -- jcmd 1 VM.command_line | grep Xlog
jvm_args: -Djava.util.logging.config.class=org.slf4j.bridge.SLF4JBridgeHandler -Djava.library.path=/opt/dremio/lib -XX:+PrintGCDetails -XX:+PrintGCDateStamps -Xloggc:/var/log/dremio/server.gc -Ddremio.log.path=/var/log/dremio -Ddremio.plugins.path=/opt/dremio/plugins -Xmx6144m -XX:MaxDirectMemorySize=2048m -XX:+HeapDumpOnOutOfMemoryError -XX:HeapDumpPath=/var/log/dremio -Dio.netty.maxDirectMemory=0 -Dio.netty.tryReflectionSetAccessible=true -DMAPR_IMPALA_RA_THROTTLE -DMAPR_MAX_RA_STREAMS=400 -XX:+UseG1GC -Ddremio.log.path=/opt/dremio/data/log -Xloggc:/opt/dremio/data/log/gc.log -XX:NumberOfGCLogFiles=5 -XX:GCLogFileSize=4000k -XX:+PrintGCDetails -XX:+PrintGCTimeStamps -XX:+PrintGCDateStamps -XX:+PrintClassHistogramBeforeFullGC -XX:+PrintClassHistogramAfterFullGC -XX:+HeapDumpOnOutOfMemoryError -XX:HeapDumpPath=/opt/dremio/data -XX:+UseG1GC -XX:G1HeapRegionSize=32M -XX:MaxGCPauseMillis=500 -XX:InitiatingHeapOccupancyPercent=25 -XX:+PrintAdaptiveSizePolicy -XX:+PrintReferenceGC -XX:ErrorFile=/opt/dremio/data/hs_err_pid%p.log -Dzookeeper=zk-hs:2181 -Dservices.coordinator.enabled=true -Dservices.coordinator.master.enabled=true -Dservices.coordinator.master.embedded-zookeeper.enabled=false -Dservices.executor.enabled=false -Dservices.conduit.port=45679 -XX:+PrintClassHistogramBeforeFullGC -XX:+PrintClassHistogramAfterFullGC
`
	gcLogLocation, err := ParseGCLogFromFlags(processFlags)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	expected := "/opt/dremio/data/log"
	if gcLogLocation != expected {
		t.Errorf("expected '%v' but was '%v'", expected, gcLogLocation)
	}
}

// func TestFindGCLocation(t *testing.T) {
// 	expectedOutput := "1 com.dremio.dac.daemon.DremioDaemon\n2 myfoo\n3 nothing"
// 	var returnValues [][]interface{}
// 	e := []interface{}{expectedOutput, nil}
// 	returnValues = append(returnValues, e)
// 	processFlags := `1:
//     VM Arguments:
//     jvm_args: -Djava.util.logging.config.class=org.slf4j.bridge.SLF4JBridgeHandler -Djava.library.path=/opt/dremio/lib -XX:+PrintGCDetails -XX:+PrintGCDateStamps -Ddremio.plugins.path=/opt/dremio/plugins -Xmx4096m -XX:MaxDirectMemorySize=2048m -XX:+HeapDumpOnOutOfMemoryError -XX:HeapDumpPath=/var/log/dremio -Dio.netty.maxDirectMemory=0 -Dio.netty.tryReflectionSetAccessible=true -DMAPR_IMPALA_RA_THROTTLE -DMAPR_MAX_RA_STREAMS=400 -XX:+UseG1GC -Ddremio.log.path=/opt/dremio/data/log -Xloggc:/opt/dremio/data/log/gc.log -XX:+UseGCLogFileRotation -XX:NumberOfGCLogFiles=5 -XX:GCLogFileSize=4000k -XX:+PrintGCDetails -XX:+PrintGCTimeStamps -XX:+PrintGCDateStamps -XX:+PrintClassHistogramBeforeFullGC -XX:+PrintClassHistogramAfterFullGC -XX:+HeapDumpOnOutOfMemoryError -XX:HeapDumpPath=/opt/dremio/data -XX:+UseG1GC -XX:G1HeapRegionSize=32M -XX:MaxGCPauseMillis=500 -XX:InitiatingHeapOccupancyPercent=25 -XX:+PrintAdaptiveSizePolicy -XX:+PrintReferenceGC -XX:ErrorFile=/opt/dremio/data/hs_err_pid%p.log -Dzookeeper=zk-hs:2181 -Dservices.coordinator.enabled=false -Dservices.coordinator.master.enabled=false -Dservices.coordinator.master.embedded-zookeeper.enabled=false -Dservices.executor.enabled=true -Dservices.conduit.port=45679 -Dservices.node-tag=default -XX:+PrintClassHistogramBeforeFullGC -XX:+PrintClassHistogramAfterFullGC
//     java_command: com.dremio.dac.daemon.DremioDaemon
//     java_class_path (initial): /opt/dremio/conf:/opt/dremio/jars/dremio-services-coordinator-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-hive-function-registry-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-ce-sabot-serializer-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-hive2-plugin-launcher-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-ee-services-credentials-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-ee-services-accesscontrol-common-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-ce-sabot-scheduler-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-dac-common-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-services-usersessions-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-ee-services-sysflight-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-protocol-20.0.0-202201050826310141-8cc7162b-proto.jar:/opt/dremio/jars/dremio-services-telemetry-impl-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-services-jobtelemetry-client-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-hive3-plugin-launcher-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-ce-services-cachemanager-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-ee-dac-tools-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-services-base-rpc-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-services-datastore-20.0.0-202201050826310141-8cc7162b-proto.jar:/opt/dremio/jars/dremio-sabot-logical-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-services-transientstore-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-services-resourcescheduler-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-dac-daemon-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-ee-services-namespace-20.0.0-202201050826310141-8cc7162b-tests.jar:/opt/dremio/j
//     Launcher Type: SUN_STANDARD`
// 	e = []interface{}{processFlags, nil}
// 	returnValues = append(returnValues, e)
// 	mockCollector := &MockCollector{
// 		Returns: returnValues,
// 	}

// 	location, err := findGCLogLocation()
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	expected := "/opt/dremio/data/log/gc.log*"
// 	if location != expected {
// 		t.Errorf("expected '%v' but was '%v'", expected, location)
// 	}
// }
