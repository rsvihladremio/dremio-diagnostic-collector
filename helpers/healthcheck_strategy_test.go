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

// helpers package provides different functionality

package helpers

import (
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"
)

// Tests the constructor is setting a basedir dir
func TestBaseDirHC(t *testing.T) {
	ddcfs := NewFakeFileSystem()
	testStrat := NewHCCopyStrategy(ddcfs)
	expected := time.Now().Format("20060102-150405-DDC")
	actual := testStrat.BaseDir
	// Check the base dir is set on creation
	if expected != actual {
		t.Errorf("ERROR: base directory name on create: \nexpected:\t%v\nactual:\t\t%v\n", expected, actual)
	}
	// Check the base dir is returned post creation
	check := testStrat.GetBaseDir()
	if check != actual {
		t.Errorf("ERROR: base directory name check: \nactual:\t%v\ncheck:\t%v\n", actual, check)
	}
}

// Tests the constructor is setting a temp dir
func TestTmpDirHC(t *testing.T) {
	ddcfs := NewFakeFileSystem()
	testStrat := NewHCCopyStrategy(ddcfs)
	expected := filepath.Join("tmp", "dir1", "random")
	actual := testStrat.TmpDir
	// Check the base dir is set on creation
	if expected != actual {
		t.Errorf("ERROR: tmp directory on create: \nexpected:\t%v\nactual:\t\t%v\n", expected, actual)
	}
	// Check the base dir is returned post creation
	check := testStrat.GetTmpDir()
	if check != actual {
		t.Errorf("ERROR: tmp directory name check: \nactual:\t%v\ncheck:\t%v\n", actual, check)
	}

}

// Tests the method returns the correct path
func TestGetPathHC(t *testing.T) {
	ddcfs := NewFakeFileSystem()
	testStrat := NewHCCopyStrategy(ddcfs)
	// Test path for coordinators
	expected := filepath.Join("tmp", "dir1", "random", testStrat.BaseDir, "log", "node1-C")
	actual, _ := testStrat.CreatePath(ddcfs, "log", "node1", "coordinator")
	if expected != actual {
		t.Errorf("\nERROR: returned path: \nexpected:\t%v\nactual:\t\t%v\n", expected, actual)
	}
	// Test path for executors
	expected = filepath.Join("tmp", "dir1", "random", testStrat.BaseDir, "log", "node1-E")
	actual, _ = testStrat.CreatePath(ddcfs, "log", "node1", "executors")
	if expected != actual {
		t.Errorf("\nERROR: returned path: \nexpected:\t%v\nactual:\t\t%v\n", expected, actual)
	}
}

// Tests the method returns the correct path
func TestGzipFilesHC(t *testing.T) {
	testSrc := filepath.Join("testdata", "archive.txt")
	ddcfs := NewRealFileSystem()
	testStrat := NewHCCopyStrategy(ddcfs)
	tmpDir := t.TempDir()
	testDst := filepath.Join(tmpDir, "archive.txt")
	bytesRead, err := ioutil.ReadFile(testSrc)
	if err != nil {
		t.Error(err)
	}
	err = ioutil.WriteFile(testDst, bytesRead, 0600)
	if err != nil {
		t.Error(err)
	}
	err = testStrat.GzipAllFiles(ddcfs, tmpDir)
	if err != nil {
		t.Errorf("\nERROR: gzip file: \nexpected:\t%v\nactual:\t\t%v\n", nil, err)
	}
	expectedPath := filepath.Join(tmpDir, "archive.txt.gz")
	expected := "archive.txt.gz"
	stat, err := ddcfs.Stat(expectedPath)
	if err != nil {
		t.Errorf("\nERROR: gzip files: \nERROR:\n%v\n", err)

	}
	if stat.Name() != expected {
		t.Errorf("\nERROR: gzip name incorrect: \nexpected:\t%v\nactual:\t\t%v\n", expected, stat.Name())
	}
	expSize := int64(62)
	if stat.Size() != expSize {
		t.Errorf("\nERROR: gzip size incorrect: \nexpected:\t%v\nactual:\t\t%d\n", expSize, stat.Size())
	}

}

// Test archiving of a file (which is also tested elsewhere) but in addition
// it tests the call via the selected strategy
func TestArchiveDiagHC(t *testing.T) {
	ddcfs := NewRealFileSystem()
	testStrat := NewHCCopyStrategy(ddcfs)
	tmpDir := t.TempDir()
	testFileRaw := filepath.Join("testdata", "test.txt")
	testFile, err := filepath.Abs(testFileRaw)
	if err != nil {
		t.Fatalf("not able to get absolute path for test file %v", err)
	}
	fi, err := ddcfs.Stat(testFile)
	if err != nil {
		t.Fatalf("unexpected error getting file size for file %v due to error %v", testFile, err)
	}
	archiveFile := tmpDir + ".zip"
	testDataDir, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatalf("not able to get absolute path for testdata dir %v", err)
	}
	testFiles := []CollectedFile{
		{
			Path: testFile,
			Size: int64(fi.Size()),
		},
	}
	// Test Archive, pushes a teal test file into a zip archive
	err = testStrat.ArchiveDiag(ddcfs, archiveFile, testDataDir, testFiles)
	if err != nil {
		t.Errorf("\nERROR: gzip file: \nexpected:\t%v\nactual:\t\t%v\n", nil, err)
	}
}
