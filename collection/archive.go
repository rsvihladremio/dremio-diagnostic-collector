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
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

func TarDiag(tarFileName string, baseDir string, files []CollectedFile) error {
	// Create a buffer to write our archive to.

	tarFile, err := os.Create(filepath.Clean(tarFileName))
	if err != nil {
		return err
	}
	// Create a new tar archive.
	tw := tar.NewWriter(tarFile)

	defer func() {
		err := tw.Close()
		if err != nil {
			log.Printf("unable to close file %v due to error %v", tarFileName, err)
		}
	}()
	for _, collectedFile := range files {
		file := collectedFile.Path
		fi, err := os.Stat(filepath.Clean(file))
		if err != nil {
			return err
		}
		if fi.IsDir() {
			continue
		}
		log.Printf("taring file %v", file)
		fileInfo, err := os.Stat(file)
		if err != nil {
			return err
		}
		rf, err := os.Open(filepath.Clean(file))
		if err != nil {
			return err
		}
		hdr := &tar.Header{
			Name: file[len(baseDir):],
			Mode: 0600,
			Size: fileInfo.Size(),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		_, err = io.Copy(tw, rf)
		if err != nil {
			return err
		}
	}
	if err := tw.Close(); err != nil {
		return err
	}
	return nil
}

func GZipDiag(zipFileName string, baseDir string, file string) error {
	// Create a buffer to write our archive to.
	zipFile, err := os.Create(filepath.Clean(zipFileName))
	if err != nil {
		return err
	}
	// Create a new gzip archive.
	w := gzip.NewWriter(zipFile)
	defer func() {
		err := w.Close()
		if err != nil {
			log.Printf("unable to close file %v due to error %v", zipFileName, err)
		}
	}()
	log.Printf("gzipping file %v into %v", file, zipFileName)
	rf, err := os.Open(filepath.Clean(file))
	if err != nil {
		return err
	}
	_, err = io.Copy(w, rf)
	if err != nil {
		return err
	}
	return nil
}

func ZipDiag(zipFileName string, baseDir string, files []CollectedFile) error {
	// Create a buffer to write our archive to.
	zipFile, err := os.Create(filepath.Clean(zipFileName))
	if err != nil {
		return err
	}
	// Create a new zip archive.
	w := zip.NewWriter(zipFile)
	defer func() {
		err := w.Close()
		if err != nil {
			log.Printf("unable to close file %v due to error %v", zipFileName, err)
		}
	}()
	// Add some files to the archive.
	for _, collectedFile := range files {
		file := collectedFile.Path
		fi, err := os.Stat(filepath.Clean(file))
		if err != nil {
			return fmt.Errorf("error while checking path %v with error %v", filepath.Clean(file), err)
		}
		if fi.IsDir() {
			continue
		}
		log.Printf("zipping file %v", file)
		f, err := w.Create(file[len(baseDir):])
		if err != nil {
			return err
		}
		rf, err := os.Open(filepath.Clean(file))
		if err != nil {
			return err
		}
		_, err = io.Copy(f, rf)
		if err != nil {
			return err
		}
	}
	return nil
}
