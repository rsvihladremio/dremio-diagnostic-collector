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

//package tests provides helper functions and mocks for running tests
package tests

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"
)

func ZipContainsFile(t *testing.T, expectedFile, zipArchive string) {
	t.Helper()
	cleanedExpectedFile := filepath.Clean(expectedFile)

	tr, err := zip.OpenReader(zipArchive)
	if err != nil {
		t.Fatalf("unexpected error getting opening file %v due to error %v", cleanedExpectedFile, err)
	}
	defer tr.Close()
	var found bool
	var buf bytes.Buffer
	for _, f := range tr.File {
		fmt.Printf("Contents of %s:\n", f.Name)
		if f.Name == "/"+filepath.Base(cleanedExpectedFile) {
			found = true
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			err := rc.Close()
			if err != nil {
				log.Printf("WARN: unable to close zip file %v", err)
			}
		}()
		for {
			_, err := io.CopyN(&buf, rc, 1024)
			if err != nil {
				if err == io.EOF {
					break
				}
				t.Fatal(err)
			}
		}

	}

	if !found {
		t.Error("expected to find the newly archived file but did not")
	}
	expectedText, err := os.ReadFile(cleanedExpectedFile)
	if err != nil {
		t.Fatal(err)
	}
	row := buf.String()
	if row != string(expectedText) {
		t.Errorf("expected content to have '%v' but was %v", string(expectedText), row)
	}
}

func GzipContainsFile(t *testing.T, expectedFile, gzipArchive string) {
	t.Helper()
	cleanedExpectedFile := filepath.Clean(expectedFile)
	cleanedArchiveFile := filepath.Clean(gzipArchive)
	f, err := os.Open(cleanedArchiveFile)
	if err != nil {
		t.Fatalf("unexpected error ungziping file %v due to error %v", cleanedExpectedFile, err)
	}
	tr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("unexpected error getting opening file %v due to error %v", cleanedExpectedFile, err)
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
	expectedText, err := os.ReadFile(cleanedExpectedFile)
	if err != nil {
		t.Fatal(err)
	}
	row := buf.String()
	if row != string(expectedText) {
		t.Errorf("expected content to have '%v' but was %v", string(expectedText), row)
	}
}

func extraGZip(t *testing.T, gzipArchive string) string {
	t.Helper()
	cleanedArchiveFile := filepath.Clean(gzipArchive)
	f, err := os.Open(cleanedArchiveFile)
	if err != nil {
		t.Fatalf("unexpected error ungziping file %v due to error %v", cleanedArchiveFile, err)
	}
	tr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("unexpected error getting opening file %v due to error %v", cleanedArchiveFile, err)
	}
	defer tr.Close()
	tarFile := filepath.Clean(cleanedArchiveFile + "tar")
	newFile, err := os.Create(tarFile)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err = newFile.Close()
		if err != nil {
			t.Logf("unable to close %v to due error %v", tarFile, err)
		}
	}()

	for {
		_, err := io.CopyN(newFile, tr, 1024)
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal(err)
		}
	}
	return tarFile
}

func TgzContainsFile(t *testing.T, expectedFile, archiveFile string) {
	t.Helper()
	tarFile := extraGZip(t, archiveFile)
	TarContainsFile(t, expectedFile, tarFile)
}

func TarContainsFile(t *testing.T, expectedFile, archiveFile string) {
	t.Helper()
	cleanedExpectedFile := filepath.Clean(expectedFile)
	cleanedArchiveFile := filepath.Clean(archiveFile)
	f, err := os.Open(cleanedArchiveFile)
	if err != nil {
		t.Fatalf("unexpected error untaring file %v due to error %v", archiveFile, err)
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
		if hdr.Name == "/"+filepath.Base(cleanedExpectedFile) {
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
		t.Errorf("expected to find the newly archived %v file but did not", cleanedExpectedFile)
	}
	expectedText, err := os.ReadFile(cleanedExpectedFile)
	if err != nil {
		t.Fatal(err)
	}
	row := buf.String()
	if row != string(expectedText) {
		t.Errorf("expected content to have %v but was %v", string(expectedText), row)
	}
}
