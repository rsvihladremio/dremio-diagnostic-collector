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

// collection package provides the interface for collection implementation and the actual collection execution
package collection

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rsvihladremio/dremio-diagnostic-collector/helpers"
	"github.com/rsvihladremio/dremio-diagnostic-collector/tests"
)

type MockCollector2 struct {
	Returns []string
	Calls   []string
	//CallCounter int
}

type MockCopy2 struct {
	HostString    string
	IsCoordinator bool
	Source        string
	Destination   string
}

func (m *MockCollector2) FindHosts(searchTerm string) (response []string, err error) {
	if searchTerm == "dremio" {
		response = append(response, "dremio-coordinator-0", "dremio-executor-0", "dremio-executor-1")
	} else {
		response = append(response, "no results")
		err = fmt.Errorf("ERROR: no hosts found matching %v", searchTerm)
	}
	return response, err
}

func (m *MockCollector2) CopyFromHost(hostString string, isCoordinator bool, source, destination string) (response string, err error) {
	copyCall := MockCopy2{
		HostString:    hostString,
		IsCoordinator: isCoordinator,
		Source:        source,
		Destination:   destination,
	}
	if copyCall.Source == "/var/log/dremio" {
		response = "INFO: logs copied from /var/log/dremio1"
	} else if copyCall.Source == "/var/log/missing" {
		response = "WARN: No logs found at /var/log/missing"
	} else {
		response = "no files found"
		err = fmt.Errorf("ERROR: no files found for %v", copyCall.Source)
	}
	return response, err
}

func (m *MockCollector2) HostExecute(hostString string, isCoordinator bool, args ...string) (response string, err error) {
	findConf := []string{"find", "/opt/dremio/conf/"}
	findLog := []string{"find", "/var/log/dremio/"}
	mockConfFiles := "/opt/dremio/dremio.conf\n/opt/dremio/dremio.env"
	mockLogFiles := "/var/log/dremio/server.out\n/var/log/dremio/server.log"
	fullCmd := strings.Join(args, " ")

	// conf files or log files
	if args[0] == findConf[0] && args[1] == findConf[1] {
		response = mockConfFiles
	} else if args[0] == findLog[0] && args[1] == findLog[1] {
		response = mockLogFiles
	} else {
		response = "no results"
		err = fmt.Errorf("ERROR: host %v command failed for %v", hostString, fullCmd)
	}
	return response, err
}

func TestExecute(t *testing.T) {
	var returnValues []string
	var callValues []string
	callValues = append(callValues, "dremio-coordinator-1", "dremio-eecutor-0", "dremio-executor-1")
	mockCollector := &MockCollector2{
		Calls:   callValues,
		Returns: returnValues,
	}
	logOutput := os.Stdout
	fakeFs := helpers.FakeFileSystem{}
	fakeTmp, _ := fakeFs.MkdirTemp("dremio", "*")
	fakeArgs := Args{
		Cfs:                       helpers.FileSystem(fakeFs),
		CoordinatorStr:            "10.1.2.3",
		ExecutorsStr:              "10.2.3.4",
		OutputLoc:                 fakeTmp,
		DremioConfDir:             "/opt/dremio/conf",
		DremioLogDir:              "/var/log/dremio",
		DremioGcDir:               "/var/log/dremio",
		GCLogOverride:             "",
		DurationDiagnosticTooling: 5,
		LogAge:                    1,
	}
	expected := "ERROR: no hosts found matching 10.1.2.3"
	err := Execute(mockCollector, logOutput, fakeArgs)
	if err.Error() != expected {
		t.Errorf("ERROR: expected: %v, got: %v", expected, err)
	}

	fakeArgs.CoordinatorStr = "dremio-coordinator-99"
	expected = "ERROR: no hosts found matching dremio-coordinator-99"
	err = Execute(mockCollector, logOutput, fakeArgs)
	if err.Error() != expected {
		t.Errorf("ERROR: expected: %v, got: %v", expected, err)
	}

	fakeArgs.CoordinatorStr = "dremio"
	//fakeArgs.ExecutorsStr = "dremio-executor-99"
	expected = "ERROR: no hosts found matching 10.2.3.4"
	err = Execute(mockCollector, logOutput, fakeArgs)
	if err.Error() != expected {
		t.Errorf("ERROR: expected: %v, got: %v", expected, err)
	}

	fakeArgs.CoordinatorStr = "dremio"
	fakeArgs.ExecutorsStr = "dremio-executor-99"
	expected = "ERROR: no hosts found matching dremio-executor-99"
	err = Execute(mockCollector, logOutput, fakeArgs)
	if err.Error() != expected {
		t.Errorf("ERROR: expected: %v, got: %v", expected, err)
	}

}

func TestArchive(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	str := "my row"
	err := os.WriteFile(testFile, []byte(str), 0600)
	if err != nil {
		t.Fatal(err)
	}
	files := []CollectedFile{
		{
			Path: testFile,
			Size: int64(len(str)),
		},
	}
	//testing zip
	archiveFile := filepath.Join(tmpDir, "test.zip")
	err = archiveDiagDirectory(archiveFile, tmpDir, files)
	if err != nil {
		t.Fatal(err)
	}
	tests.ZipContainsFile(t, testFile, archiveFile)

	//testing tar
	archiveFile = filepath.Join(tmpDir, "test.tar")
	err = archiveDiagDirectory(archiveFile, tmpDir, files)
	if err != nil {
		t.Fatal(err)
	}
	tests.TarContainsFile(t, testFile, archiveFile)

	//testing tar gunzip
	archiveFile = filepath.Join(tmpDir, "test.tar.gz")
	err = archiveDiagDirectory(archiveFile, tmpDir, files)
	if err != nil {
		t.Fatal(err)
	}
	tests.TgzContainsFile(t, testFile, archiveFile)

	//testing tgz
	archiveFile = filepath.Join(tmpDir, "test.tgz")
	err = archiveDiagDirectory(archiveFile, tmpDir, files)
	if err != nil {
		t.Fatal(err)
	}
	tests.TgzContainsFile(t, testFile, archiveFile)
}
