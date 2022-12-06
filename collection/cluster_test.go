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
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rsvihladremio/dremio-diagnostic-collector/cli"
	"github.com/rsvihladremio/dremio-diagnostic-collector/helpers"
	"github.com/rsvihladremio/dremio-diagnostic-collector/tests"
)

type ExpectedJSON struct {
	APIVersion string
	Kind       string
	Value      int
}

func TestClusterCopyJSON(t *testing.T) {
	tmpDir := t.TempDir()
	// Read a file bytes
	testjson := filepath.Join("testdata", "test.json")
	actual, err := os.ReadFile(testjson)
	if err != nil {
		log.Printf("ERROR: when reading json file\n%v\nerror returned was:\n %v", actual, err)
	}

	afile := filepath.Join(tmpDir, "actual.json")
	// Write a file with the same bytes
	err = os.WriteFile(afile, actual, DirPerms)
	if err != nil {
		t.Errorf("ERROR: trying to write file %v, error was %v", afile, err)
	}

	expected := ExpectedJSON{
		APIVersion: "v1",
		Kind:       "Data",
		Value:      100,
	}

	// Create a model file
	efile := filepath.Join(tmpDir, "expected.json")
	edata, _ := json.MarshalIndent(expected, "", "    ")
	err = os.WriteFile(efile, edata, DirPerms)
	if err != nil {
		t.Errorf("ERROR: trying to write file %v, error was %v", efile, err)
	}
	// Read back files and compare
	acheck, err := os.ReadFile(afile)
	if err != nil {
		t.Errorf("ERROR: trying to read file %v, error was %v", afile, err)
	}
	echeck, err := os.ReadFile(efile)
	if err != nil {
		t.Errorf("ERROR: trying to read file %v, error was %v", efile, err)
	}

	expStr := strings.ReplaceAll((string(echeck)), `\r\n`, `\n`)
	actStr := strings.ReplaceAll((string(acheck)), `\r\n`, `\n`)

	if expStr != actStr {
		t.Errorf("\nERROR: \nexpected:\t%q\nactual:\t\t%q\n", expStr, actStr)
	}

	/*if !reflect.DeepEqual(acheck, echeck) {
		t.Errorf("\nERROR: \nexpected:\t%q\nactual:\t\t%q\n", string(acheck), string(echeck))
	}*/
}

func TestClusterZipJSON(t *testing.T) {
	tmpDir := t.TempDir()
	var afiles []helpers.CollectedFile
	testjson := filepath.Join("testdata", "test.json")
	actual, err := os.ReadFile(testjson)
	if err != nil {
		log.Printf("ERROR: when reading json file\n%v\nerror returned was:\n %v", actual, err)
	}

	afilepath := filepath.Join(tmpDir, "actual.json")
	// Write a file with the same bytes
	err = os.WriteFile(afilepath, actual, DirPerms)
	if err != nil {
		t.Errorf("ERROR: trying to write file %v, error was %v", afilepath, err)
	}

	// Wrap the test file into a struct to pass into the archive call
	//afilepath := filepath.Join("testdata", "test.json")
	afilesize, err := os.Stat(afilepath)
	if err != nil {
		t.Errorf("ERROR: trying to stat file %v, error was %v", afilepath, err)
	}
	afile := helpers.CollectedFile{
		Path: afilepath,
		Size: afilesize.Size(),
	}
	afiles = append(afiles, afile)

	// Make the test zip file
	testZip := filepath.Join(tmpDir, "test.zip")
	err = helpers.ArchiveDiagFromList(testZip, tmpDir, afiles)
	if err != nil {
		t.Errorf("ERROR: trying to archive files into file %v, error was %v", testZip, err)
	}

	// Create model data
	expected := ExpectedJSON{
		APIVersion: "v1",
		Kind:       "Data",
		Value:      100,
	}

	// Create a model file
	efile := filepath.Join(tmpDir, "expected.json")
	edata, _ := json.MarshalIndent(expected, "", "    ")
	err = os.WriteFile(efile, edata, DirPerms)
	if err != nil {
		t.Errorf("ERROR: trying to write file %v, error was %v", efile, err)
	}

	// Read back files and compare
	tests.ZipContainsFile(t, afile.Path, testZip)

}

func TestK8sZipsJSON(t *testing.T) {
	tmpDir := t.TempDir()
	var afiles []helpers.CollectedFile
	zipFile := filepath.Join(tmpDir, "test.zip")

	// Make a complete list of collected files
	found, err := findAllFiles("testdata/kubernetes")
	if err != nil {
		t.Errorf("ERROR: trying to find files, error was %v", err)
	}
	// Get all file sizes
	afiles, err = createFileList(found)
	if err != nil {
		t.Errorf("ERROR: trying to get file size, error was %v", err)
	}

	for _, file := range afiles {

		filedata, err := os.ReadFile(file.Path)
		if err != nil {
			log.Printf("ERROR: when reading json file\n%v\nerror returned was:\n %v", file.Path, err)
		}
		tmpFile := filepath.Join(tmpDir, filepath.Base(file.Path))
		// Write a file with the same bytes
		err = os.WriteFile(tmpFile, filedata, DirPerms)
		if err != nil {
			t.Errorf("ERROR: trying to write file %v, error was %v", file.Path, err)
		}
	}

	// Make the test zip file
	err = helpers.ArchiveDiagFromList(zipFile, "", afiles)
	if err != nil {
		t.Errorf("ERROR: trying to zip files, error was %v", err)
	}

	for _, file := range afiles {
		// Read back files and compare
		t.Logf("INFO: checking archive %v for file %v", zipFile, file.Path)
		tests.ZipContainsFile(t, file.Path, zipFile)
	}

}

func findAllFiles(path string) ([]string, error) {
	cmd := cli.Cli{}
	f := []string{}
	out, err := cmd.Execute("find", path, "-type", "f")
	if err != nil {
		return f, err
	}
	f = strings.Split(out, "\n")
	return f, nil
}

func createFileList(foundFiles []string) (files []helpers.CollectedFile, err error) {
	for _, file := range foundFiles {
		if file == "" {
			break
		}
		g, err := os.Stat(file)
		if err != nil {
			return nil, err
		}
		files = append(files, helpers.CollectedFile{
			Path: file,
			Size: g.Size(),
		})
	}
	return files, err
}
