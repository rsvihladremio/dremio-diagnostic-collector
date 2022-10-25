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
	"os"
	"path/filepath"
	"testing"

	"github.com/rsvihladremio/dremio-diagnostic-collector/tests"
)

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
