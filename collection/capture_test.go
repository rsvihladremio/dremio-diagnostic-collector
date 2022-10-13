/*
   Copyright 2022 Ryan SVIHLA

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

//collection package provides the interface for collection implementation and the actual collection execution
package collection

import (
	"log"
	"reflect"
	"testing"
)

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
	expected := "/opt/dremio/data/log/gc.log"
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
	expected := "/opt/dremio/data/log/gc.log"
	if gcLogLocation != expected {
		t.Errorf("expected '%v' but was '%v'", expected, gcLogLocation)
	}
}

func TestGetDremioPID(t *testing.T) {
	pidList := `1 com.dremio.dac.daemon.DremioDaemon
383 sun.tools.jcmd.JCmd -l
`
	pid, err := GetDremioPID(pidList)
	if err != nil {
		t.Fatal(err)
	}
	if pid != 1 {
		t.Errorf("expected 1 but was %v", pid)
	}
}

type MockCollector struct {
	Returns     [][]interface{}
	Calls       []map[string]interface{}
	CallCounter int
}

func (m *MockCollector) CopyFromHost(hostString string, isCoordinator bool, source, destination string) (out string, err error) {
	args := make(map[string]interface{})
	args["hostString"] = hostString
	args["isCoordinator"] = isCoordinator
	args["source"] = source
	args["destination"] = destination
	m.Calls = append(m.Calls, args)
	response := m.Returns[m.CallCounter]
	m.CallCounter++
	return response[0].(string), response[1].(error)
}
func (m *MockCollector) FindHosts(searchTerm string) (podName []string, err error) {
	args := make(map[string]interface{})
	args["searchTerm"] = searchTerm
	m.Calls = append(m.Calls, args)
	response := m.Returns[m.CallCounter]
	m.CallCounter++
	return response[0].([]string), response[1].(error)
}
func (m *MockCollector) HostExecute(hostString string, isCoordinator bool, args ...string) (stdOut string, err error) {
	capturedArgs := make(map[string]interface{})
	capturedArgs["hostString"] = hostString
	capturedArgs["isCoordinator"] = isCoordinator
	capturedArgs["args"] = args
	m.Calls = append(m.Calls, capturedArgs)
	response := m.Returns[m.CallCounter]
	m.CallCounter++
	if response[1] == nil {
		return response[0].(string), nil

	}
	return response[0].(string), response[1].(error)
}
func TestFindFiles(t *testing.T) {

	expectedOutput := "/opt/file1\n/opt/file2\n"
	var returnValues [][]interface{}
	e := []interface{}{expectedOutput, nil}
	returnValues = append(returnValues, e)
	mockCollector := &MockCollector{
		Returns: returnValues,
	}
	myHost := "thishost"
	conf := HostCaptureConfiguration{
		Logger:                    &log.Logger{},
		IsCoordinator:             false,
		Collector:                 mockCollector,
		Host:                      myHost,
		OutputLocation:            "",
		DremioConfDir:             "",
		DremioLogDir:              "",
		DurationDiagnosticTooling: 0,
		LogAge:                    5,
	}
	searchStr := "/opt/file*"
	files, err := findFiles(conf, searchStr, true)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(files, []string{"/opt/file1", "/opt/file2"}) {
		t.Errorf("expected '%v' but was '%v'", expectedOutput, files)
	}
	if mockCollector.CallCounter != 1 {
		t.Fatalf("expected 1 call but was %v", mockCollector.CallCounter)
	}
	if len(mockCollector.Calls) != 1 {
		t.Fatalf("expected 1 call but was %v", len(mockCollector.Calls))
	}
	calls := mockCollector.Calls[0]
	if calls["hostString"] != myHost {
		t.Errorf("expected %v but was %v", myHost, calls["hostString"])
	}
	if calls["isCoordinator"] != conf.IsCoordinator {
		t.Errorf("expected %v but was %v", conf.IsCoordinator, calls["isCoordinator"])
	}

	expectedArgs := []string{"bash", "-c", "find /opt/file* -maxdepth 3 -type f -mtime 5"}
	if !reflect.DeepEqual(calls["args"], expectedArgs) {
		t.Errorf("expected %v but was %v", expectedArgs, calls["args"])
	}
}
func TestGetStartupFlags(t *testing.T) {
	expectedOutput := "my startup flags"
	var returnValues [][]interface{}
	e := []interface{}{expectedOutput, nil}
	returnValues = append(returnValues, e)
	mockCollector := &MockCollector{
		Returns: returnValues,
	}
	myHost := "thishost"
	conf := HostCaptureConfiguration{
		Logger:                    &log.Logger{},
		IsCoordinator:             false,
		Collector:                 mockCollector,
		Host:                      myHost,
		OutputLocation:            "",
		DremioConfDir:             "",
		DremioLogDir:              "",
		DurationDiagnosticTooling: 0,
		LogAge:                    5,
	}
	pid := 1
	flags, err := GetStartupFlags(conf, pid)
	if err != nil {
		t.Fatal(err)
	}
	if flags != expectedOutput {
		t.Errorf("expected '%v' but was '%v'", expectedOutput, flags)
	}
	if mockCollector.CallCounter != 1 {
		t.Fatalf("expected 1 call but was %v", mockCollector.CallCounter)
	}
	if len(mockCollector.Calls) != 1 {
		t.Fatalf("expected 1 call but was %v", len(mockCollector.Calls))
	}
	calls := mockCollector.Calls[0]
	if calls["hostString"] != myHost {
		t.Errorf("expected %v but was %v", myHost, calls["hostString"])
	}
	if calls["isCoordinator"] != conf.IsCoordinator {
		t.Errorf("expected %v but was %v", conf.IsCoordinator, calls["isCoordinator"])
	}

	expectedArgs := []string{"ps", "-f", "1"}
	if !reflect.DeepEqual(calls["args"], expectedArgs) {
		t.Errorf("expected %v but was %v", expectedArgs, calls["args"])
	}
}

func TestFindGCLocation(t *testing.T) {
	expectedOutput := "1 com.dremio.dac.daemon.DremioDaemon\n2 myfoo\n3 nothing"
	var returnValues [][]interface{}
	e := []interface{}{expectedOutput, nil}
	returnValues = append(returnValues, e)
	processFlags := `1:
    VM Arguments:
    jvm_args: -Djava.util.logging.config.class=org.slf4j.bridge.SLF4JBridgeHandler -Djava.library.path=/opt/dremio/lib -XX:+PrintGCDetails -XX:+PrintGCDateStamps -Ddremio.plugins.path=/opt/dremio/plugins -Xmx4096m -XX:MaxDirectMemorySize=2048m -XX:+HeapDumpOnOutOfMemoryError -XX:HeapDumpPath=/var/log/dremio -Dio.netty.maxDirectMemory=0 -Dio.netty.tryReflectionSetAccessible=true -DMAPR_IMPALA_RA_THROTTLE -DMAPR_MAX_RA_STREAMS=400 -XX:+UseG1GC -Ddremio.log.path=/opt/dremio/data/log -Xloggc:/opt/dremio/data/log/gc.log -XX:+UseGCLogFileRotation -XX:NumberOfGCLogFiles=5 -XX:GCLogFileSize=4000k -XX:+PrintGCDetails -XX:+PrintGCTimeStamps -XX:+PrintGCDateStamps -XX:+PrintClassHistogramBeforeFullGC -XX:+PrintClassHistogramAfterFullGC -XX:+HeapDumpOnOutOfMemoryError -XX:HeapDumpPath=/opt/dremio/data -XX:+UseG1GC -XX:G1HeapRegionSize=32M -XX:MaxGCPauseMillis=500 -XX:InitiatingHeapOccupancyPercent=25 -XX:+PrintAdaptiveSizePolicy -XX:+PrintReferenceGC -XX:ErrorFile=/opt/dremio/data/hs_err_pid%p.log -Dzookeeper=zk-hs:2181 -Dservices.coordinator.enabled=false -Dservices.coordinator.master.enabled=false -Dservices.coordinator.master.embedded-zookeeper.enabled=false -Dservices.executor.enabled=true -Dservices.conduit.port=45679 -Dservices.node-tag=default -XX:+PrintClassHistogramBeforeFullGC -XX:+PrintClassHistogramAfterFullGC 
    java_command: com.dremio.dac.daemon.DremioDaemon
    java_class_path (initial): /opt/dremio/conf:/opt/dremio/jars/dremio-services-coordinator-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-hive-function-registry-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-ce-sabot-serializer-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-hive2-plugin-launcher-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-ee-services-credentials-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-ee-services-accesscontrol-common-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-ce-sabot-scheduler-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-dac-common-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-services-usersessions-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-ee-services-sysflight-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-protocol-20.0.0-202201050826310141-8cc7162b-proto.jar:/opt/dremio/jars/dremio-services-telemetry-impl-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-services-jobtelemetry-client-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-hive3-plugin-launcher-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-ce-services-cachemanager-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-ee-dac-tools-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-services-base-rpc-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-services-datastore-20.0.0-202201050826310141-8cc7162b-proto.jar:/opt/dremio/jars/dremio-sabot-logical-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-services-transientstore-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-services-resourcescheduler-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-dac-daemon-20.0.0-202201050826310141-8cc7162b.jar:/opt/dremio/jars/dremio-ee-services-namespace-20.0.0-202201050826310141-8cc7162b-tests.jar:/opt/dremio/j
    Launcher Type: SUN_STANDARD`
	e = []interface{}{processFlags, nil}
	returnValues = append(returnValues, e)
	mockCollector := &MockCollector{
		Returns: returnValues,
	}
	myHost := "thishost"
	conf := HostCaptureConfiguration{
		Logger:                    &log.Logger{},
		IsCoordinator:             false,
		Collector:                 mockCollector,
		Host:                      myHost,
		OutputLocation:            "",
		DremioConfDir:             "",
		DremioLogDir:              "",
		DurationDiagnosticTooling: 0,
		LogAge:                    5,
	}
	location, err := findGCLogLocation(conf)
	if err != nil {
		t.Fatal(err)
	}
	expected := "/opt/dremio/data/log/gc.log*"
	if location != expected {
		t.Errorf("expected '%v' but was '%v'", expected, location)
	}
}
