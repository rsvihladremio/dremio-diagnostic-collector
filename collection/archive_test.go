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

func TestZip(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("my row"), 0600)
	if err != nil {
		t.Fatalf("unexpected error making file %v due to error %v", testFile, err)
	}
	fi, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("unexpected error getting file size for file %v due to error %v", testFile, err)
	}
	archiveFile := tmpDir + ".zip"
	err = ZipDiag(archiveFile, tmpDir, []CollectedFile{
		{
			Path: testFile,
			Size: fi.Size(),
		},
	})

	if err != nil {
		t.Fatalf("unexpected error zipping file %v due to error %v", testFile, err)
	}
	tests.ZipContainsFile(t, testFile, archiveFile)

	fakePath := tmpDir + "/does-not-exist/test.zip"
	err = ZipDiag(fakePath, tmpDir, []CollectedFile{
		{
			Path: testFile,
			Size: fi.Size(),
		},
	})
	expectedOutput := "open " + fakePath + ": no such file or directory"
	if err.Error() != expectedOutput {
		t.Fatalf("unexpected error zipping file %v due to error %v", testFile, err)
	}

}

func TestTar(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("my row"), 0600)
	if err != nil {
		t.Fatalf("unexpected error making file %v due to error %v", testFile, err)
	}
	fi, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("unexpected error getting file size for file %v due to error %v", testFile, err)
	}
	archiveFile := tmpDir + ".tar"
	err = TarDiag(archiveFile, tmpDir, []CollectedFile{
		{
			Path: testFile,
			Size: fi.Size(),
		},
	})
	if err != nil {
		t.Fatalf("unexpected error taring file %v due to error %v", testFile, err)
	}
	tests.TarContainsFile(t, testFile, archiveFile)

	fakePath := tmpDir + "/does-not-exist/test.tar"
	err = TarDiag(fakePath, tmpDir, []CollectedFile{
		{
			Path: testFile,
			Size: fi.Size(),
		},
	})
	expectedOutput := "open " + fakePath + ": no such file or directory"
	if err.Error() != expectedOutput {
		t.Fatalf("unexpected error taring file %v due to error %v", testFile, err)
	}

}

func TestGZip(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("my row"), 0600)
	if err != nil {
		t.Fatalf("unexpected error making file %v due to error %v", testFile, err)
	}
	archiveFile := tmpDir + ".gzip"

	err = GZipDiag(archiveFile, tmpDir, testFile)
	if err != nil {
		t.Fatalf("unexpected error zipping file %v due to error %v", testFile, err)
	}

	tests.GzipContainsFile(t, testFile, archiveFile)

	fakePath := tmpDir + "/does-not-exist/test.gzip"
	err = GZipDiag(fakePath, tmpDir, testFile)
	expectedOutput := "open " + fakePath + ": no such file or directory"
	if err.Error() != expectedOutput {
		t.Fatalf("unexpected error zipping file %v due to error %v", testFile, err)
	}
}
