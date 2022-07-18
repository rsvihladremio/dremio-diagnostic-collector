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
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
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
	tr, err := zip.OpenReader(archiveFile)
	if err != nil {
		t.Fatalf("unexpected error getting opening file %v due to error %v", testFile, err)
	}
	defer tr.Close()
	var found bool
	var buf bytes.Buffer
	for _, f := range tr.File {
		fmt.Printf("Contents of %s:\n", f.Name)
		if f.Name == "/test.txt" {
			found = true
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		for {
			_, err := io.CopyN(&buf, rc, 1024)
			if err != nil {
				if err == io.EOF {
					break
				}
				t.Fatal(err)
			}
		}
		rc.Close()
	}

	if !found {
		t.Error("expected to find the newly archived file but did not")
	}
	row := buf.String()
	if row != "my row" {
		t.Errorf("expected content to have 'my row' but was %v", row)
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
	f, err := os.Open(archiveFile)
	if err != nil {
		t.Fatalf("unexpected error untaring file %v due to error %v", testFile, err)
	}

	tr := tar.NewReader(f)
	var found bool
	var buf bytes.Buffer
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			t.Fatal(err)
		}
		fmt.Printf("Contents of %s:\n", hdr.Name)
		if hdr.Name == "/test.txt" {
			found = true
		}
		for {
			_, err := io.CopyN(&buf, tr, 1024)
			if err != nil {
				if err == io.EOF {
					break
				}
				t.Fatal(err)
			}
		}
		fmt.Println()
	}
	if !found {
		t.Error("expected to find the newly archived file but did not")
	}
	row := buf.String()
	if row != "my row" {
		t.Errorf("expected content to have 'my row' but was %v", row)
	}
}

func TestGZip(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("my row"), 0600)
	if err != nil {
		t.Fatalf("unexpected error making file %v due to error %v", testFile, err)
	}

	archiveFile := tmpDir + ".zip"
	err = GZipDiag(archiveFile, tmpDir, testFile)

	if err != nil {
		t.Fatalf("unexpected error zipping file %v due to error %v", testFile, err)
	}

	f, err := os.Open(archiveFile)
	if err != nil {
		t.Fatalf("unexpected error untaring file %v due to error %v", testFile, err)
	}
	tr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("unexpected error getting opening file %v due to error %v", testFile, err)
	}
	defer tr.Close()
	var buf bytes.Buffer
	for {
		_, err := io.CopyN(&buf, tr, 1024)
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal(err)
		}
	}
	row := buf.String()
	if row != "my row" {
		t.Errorf("expected content to have 'my row' but was %v", row)
	}
}
