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

type MockCopyStrategy struct {
	Returns []string
	Calls   []string
}

type MockStrategy struct {
	StrategyName string
	BaseDir      string
	TmpDir       string
	ZipPath      string
	Source       string
	NodeType     string
}

func (s *MockStrategy) GetBaseDir() string {
	dir := s.BaseDir
	return dir
}

// Returns the base dir
func (s *MockStrategy) GetTmpDir() string {
	dir := s.TmpDir
	return dir
}

func (s *MockStrategy) GetZipPath() string {
	dir := s.ZipPath
	return dir
}

func (s *MockStrategy) CreatePath(ddcfs helpers.Filesystem, fileType, source, nodeType string) (path string, err error) {
	baseDir := s.BaseDir
	var isK8s bool
	if strings.Contains(source, "dremio-master") || strings.Contains(source, "dremio-executor") || strings.Contains(source, "dremio-coordinator") {
		isK8s = true
	}
	if !isK8s {
		if nodeType == "coordinator" {
			path = filepath.Join(baseDir, fileType, source+"-C")
		} else {
			path = filepath.Join(baseDir, fileType, source+"-E")
		}
	} else {
		path = filepath.Join(baseDir, fileType, source)
	}
	return path, nil
}

type MockCapCollector struct {
	Returns []string
	Calls   []string
}

type MockCapCopy struct {
	HostString    string
	IsCoordinator bool
	Source        string
	Destination   string
}

func (m *MockCapCollector) FindHosts(searchTerm string) (response []string, err error) {
	response = strings.Split(searchTerm, "-")
	if len(response) > 1 && response[1] != "ok" {
		err = fmt.Errorf("ERROR: no hosts found matching %v", searchTerm)
	}
	return response, err
}

func (m *MockCapCollector) CopyFromHost(hostString string, isCoordinator bool, source, destination string) (response string, err error) {
	copyCall := MockCapCopy{
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

func (m *MockCapCollector) HostExecute(hostString string, isCoordinator bool, args ...string) (response string, err error) {

	fullCmd := strings.Join(args, " ")

	response = "Mock execute for " + hostString + " command: " + fullCmd
	return response, err
}

func TestFindHostsCoordinators(t *testing.T) {
	var returnValues []string
	var callValues []string
	callValues = append(callValues, "dremio-coordinator-1", "dremio-eecutor-0", "dremio-executor-1")
	tmpDir := t.TempDir()

	mockStrategy := &MockStrategy{
		StrategyName: "healthcheck",
		BaseDir:      tmpDir,
	}

	mockCollector := &MockCapCollector{
		Calls:   callValues,
		Returns: returnValues,
	}
	logOutput := os.Stdout
	fakeFS := helpers.NewFakeFileSystem()
	fakeTmp, _ := fakeFS.MkdirTemp("fake", "fake")
	fakeArgs := Args{
		DDCfs:                     fakeFS,
		CoordinatorStr:            "10.1.2.3-nok",
		ExecutorsStr:              "10.2.3.4-nok",
		OutputLoc:                 fakeTmp,
		DremioConfDir:             "/opt/dremio/conf",
		DremioLogDir:              "/var/log/dremio",
		DremioGcDir:               "/var/log/dremio",
		GCLogOverride:             "",
		DurationDiagnosticTooling: 5,
		LogAge:                    1,
		CopyStrategy:              mockStrategy,
	}

	// Test for incorrect host
	fakeArgs.CoordinatorStr = "dremio-master-99"
	expected := "ERROR: no hosts found matching dremio-master-99"
	err := Execute(mockCollector, fakeArgs.CopyStrategy, logOutput, fakeArgs)
	if err.Error() != expected {
		t.Errorf("ERROR: expected: %v, got: %v", expected, err)
	}

}

func TestFindHostsExecutors(t *testing.T) {
	var returnValues []string
	var callValues []string
	callValues = append(callValues, "dremio-coordinator-1", "dremio-eecutor-0", "dremio-executor-1")
	tmpDir := t.TempDir()

	mockStrategy := &MockStrategy{
		StrategyName: "healthcheck",
		BaseDir:      tmpDir,
	}

	mockCollector := &MockCapCollector{
		Calls:   callValues,
		Returns: returnValues,
	}
	logOutput := os.Stdout
	fakeFS := helpers.NewFakeFileSystem()
	fakeTmp, _ := fakeFS.MkdirTemp("fake", "fake")
	fakeArgs := Args{
		DDCfs:                     fakeFS,
		CoordinatorStr:            "10.1.2.3-ok",
		ExecutorsStr:              "10.2.3.4-nok",
		OutputLoc:                 fakeTmp,
		DremioConfDir:             "/opt/dremio/conf",
		DremioLogDir:              "/var/log/dremio",
		DremioGcDir:               "/var/log/dremio",
		GCLogOverride:             "",
		DurationDiagnosticTooling: 5,
		LogAge:                    1,
		CopyStrategy:              mockStrategy,
	}

	fakeArgs.ExecutorsStr = "dremio-executor-99"
	expected := "ERROR: no hosts found matching dremio-executor-99"
	err := Execute(mockCollector, fakeArgs.CopyStrategy, logOutput, fakeArgs)
	if err.Error() != expected {
		t.Errorf("ERROR: expected: %v, got: %v", expected, err)
	}

}

/*
func TestFailIOstat(t *testing.T) {
	var returnValues []string
	var callValues []string
	callValues = append(callValues, "dremio-coordinator-1", "dremio-eecutor-0", "dremio-executor-1")
	tmpDir := t.TempDir()

	mockStrategy := &MockStrategy{
		StrategyName: "healthcheck",
		BaseDir:      tmpDir,
	}

	mockCollector := &MockCapCollector{
		Calls:   callValues,
		Returns: returnValues,
	}
	logOutput := os.Stdout
	fakeFS := helpers.NewFakeFileSystem()
	fakeTmp, _ := fakeFS.MkdirTemp("fake", "fake")
	fakeArgs := Args{
		DDCfs:                     fakeFS,
		CoordinatorStr:            "10.1.2.3-ok",
		ExecutorsStr:              "10.2.3.4-ok",
		OutputLoc:                 fakeTmp,
		DremioConfDir:             "/opt/dremio/conf",
		DremioLogDir:              "/var/log/dremio",
		DremioGcDir:               "/var/log/dremio",
		GCLogOverride:             "",
		DurationDiagnosticTooling: 5,
		LogAge:                    1,
		CopyStrategy:              mockStrategy,
	}

	fakeArgs.ExecutorsStr = "dremio-ok"
	expected := "Mock execute for ok command: iostat -y -x -d -c -t 1 5"
	err := Execute(mockCollector, fakeArgs.CopyStrategy, logOutput, fakeArgs)
	if err.Error() != expected {
		t.Errorf("ERROR: expected: %v, got: %v", expected, err)
	}

}
*/

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
