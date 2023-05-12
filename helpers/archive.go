//	Copyright 2023 Dremio Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//This module controls archiving files that are collected

package helpers

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/rsvihladremio/dremio-diagnostic-collector/cli"
)

// archiveDiagDirectory will detect the extension asked for and use the correct archival library
// to archive the old directory. It supports: .tgz, .tar.gz and .zip extensions
func ArchiveDiagDirectory(outputFile, outputDir string, fileList []CollectedFile) error {

	// Make a complete list of collected files
	found, err := findAllFiles(outputDir)
	if err != nil {
		return err
	}
	// Get all file sizes
	files, err := createFileList(found)
	if err != nil {
		return err
	}

	ext := filepath.Ext(outputFile)
	if ext == ".zip" {
		if err := ZipDiag(outputFile, outputDir, files); err != nil {
			return fmt.Errorf("unable to write zip file %v due to error %v", outputFile, err)
		}
	} else if strings.HasSuffix(outputFile, "tar.gz") || ext == ".tgz" {
		tempFile := strings.Join([]string{strings.TrimSuffix(outputFile, ext), "tar"}, ".")
		if err := TarDiag(tempFile, outputDir, files); err != nil {
			return fmt.Errorf("unable to write tar file %v due to error %v", outputFile, err)
		}
		defer func() {
			if err := os.Remove(tempFile); err != nil {
				log.Printf("WARN unable to delete file '%v' due to '%v'", tempFile, err)
			}
		}()
		if err := GZipDiag(outputFile, outputDir, tempFile); err != nil {
			return fmt.Errorf("unable to write gz file %v due to error %v", outputFile, err)
		}
	} else if ext == ".tar" {
		if err := TarDiag(outputFile, outputDir, files); err != nil {
			return fmt.Errorf("unable to write tar file %v due to error %v", outputFile, err)
		}
	}
	return nil
}

func ArchiveDiagFromList(outputFile, outputDir string, fileList []CollectedFile) error {

	ext := filepath.Ext(outputFile)
	if ext == ".zip" {
		if err := ZipDiag(outputFile, outputDir, fileList); err != nil {
			return fmt.Errorf("unable to write zip file %v due to error %v", outputFile, err)
		}
	} else if strings.HasSuffix(outputFile, "tar.gz") || ext == ".tgz" {
		tempFile := strings.Join([]string{strings.TrimSuffix(outputFile, ext), "tar"}, ".")
		if err := TarDiag(tempFile, outputDir, fileList); err != nil {
			return fmt.Errorf("unable to write tar file %v due to error %v", outputFile, err)
		}
		defer func() {
			if err := os.Remove(tempFile); err != nil {
				log.Printf("WARN unable to delete file '%v' due to '%v'", tempFile, err)
			}
		}()
		if err := GZipDiag(outputFile, outputDir, tempFile); err != nil {
			return fmt.Errorf("unable to write gz file %v due to error %v", outputFile, err)
		}
	} else if ext == ".tar" {
		if err := TarDiag(outputFile, outputDir, fileList); err != nil {
			return fmt.Errorf("unable to write tar file %v due to error %v", outputFile, err)
		}
	}
	return nil
}

func TarDiag(tarFileName string, baseDir string, files []CollectedFile) error {
	// Create a buffer to write our archive to.
	tarFile, err := os.Create(filepath.Clean(tarFileName))
	if err != nil {
		return err
	}
	defer func() {
		err := tarFile.Close()
		if err != nil {
			log.Printf("unable to close file %v due to error %v", tarFileName, err)
		}
	}()
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
		rf, err := os.Open(filepath.Clean(file))
		if err != nil {
			return err
		}
		defer func() {
			err := rf.Close()
			if err != nil {
				log.Printf("unable to close file %v due to error %v", tarFileName, err)
			}
		}()
		fileWithoutDir := file[len(baseDir):]
		fileWithoutDir = strings.Trim(fileWithoutDir, string(filepath.Separator))
		hdr := &tar.Header{
			Name:    fileWithoutDir,
			Mode:    0600,
			Size:    fi.Size(),
			ModTime: fi.ModTime().UTC(),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		fileBytes, err := os.ReadFile(collectedFile.Path)

		if err != nil {
			return err
		}
		_, err = tw.Write(fileBytes)
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
	defer func() {
		err := zipFile.Close()
		if err != nil {
			log.Printf("unable to close file %v due to error %v", zipFileName, err)
		}

	}()
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
	defer func() {
		err := rf.Close()
		if err != nil {
			log.Printf("unable to close file %v due to error %v", zipFileName, err)
		}
	}()
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
	defer func() {
		err := zipFile.Close()
		if err != nil {
			log.Printf("unable to close file %v due to error %v", zipFileName, err)
		}
	}()
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
			// log instead of return error to make non-fatal. We might be missing some files in
			// the archive but as long as we log it, something is better than nothing
			log.Printf("error while checking path %v with error %v", filepath.Clean(file), err)
		} else {
			if fi.IsDir() {
				continue
			}
			log.Printf("zipping file %v", file)
			// Trim off the tmp dir to leave file and path you need in the archive
			//fileWithoutDir := strings.TrimPrefix(file, fmt.Sprintf("%v%v", baseDir, filepath.Separator))
			fileWithoutDir := file[len(baseDir):]
			fileWithoutDir = strings.Trim(fileWithoutDir, string(filepath.Separator))
			header, err := zip.FileInfoHeader(fi)
			if err != nil {
				return err
			}
			header.Modified = fi.ModTime().UTC()
			header.Name = filepath.Clean(fileWithoutDir)
			f, err := w.CreateHeader(header)
			if err != nil {
				return err
			}
			rf, err := os.Open(filepath.Clean(file))
			if err != nil {
				return err
			}
			defer func() {
				err := rf.Close()
				if err != nil {
					log.Printf("WARN unable to close file %v due to error %v", file, err)
				}
			}()
			fileBytes, err := os.ReadFile(collectedFile.Path)
			if err != nil {
				return err
			}
			_, err = f.Write(fileBytes)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func GzipAllFiles(path string) (files []CollectedFile, err error) {
	var foundFiles []string
	if runtime.GOOS == "windows" {
		// Currently windows gzipping isnt supported
		return nil, nil
	}
	foundFiles, err = findAllFiles(path)
	if err != nil {
		return nil, err
	}

	for _, file := range foundFiles {
		if file == "" {
			break
		}
		zf := file + ".gz"
		err = gZipFile(zf, file)
		if err != nil {
			return nil, err
		}
	}

	foundFiles, err = findGzFiles(path)
	if err != nil {
		return nil, err
	}

	for _, file := range foundFiles {
		if file == "" {
			break
		}
		g, _ := os.Stat(file)
		files = append(files, CollectedFile{
			Path: file,
			Size: g.Size(),
		})
	}
	return files, err
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

func createFileList(foundFiles []string) (files []CollectedFile, err error) {
	for _, file := range foundFiles {
		if file == "" {
			break
		}
		g, err := os.Stat(file)
		if err != nil {
			return nil, err
		}
		files = append(files, CollectedFile{
			Path: file,
			Size: g.Size(),
		})
	}
	return files, err
}

func findGzFiles(path string) ([]string, error) {
	cmd := cli.Cli{}
	f := []string{}
	out, err := cmd.Execute("find", path, "-type", "f", "-name", "*.gz")
	if err != nil {
		return f, err
	}
	f = strings.Split(out, "\n")
	return f, nil
}

func gZipFile(zipFileName, file string) error {
	// Create a buffer to write our archive to.
	zipFile, err := os.Create(filepath.Clean(zipFileName))
	if err != nil {
		return err
	}
	defer func() {
		err := zipFile.Close()
		if err != nil {
			log.Printf("unable to close file %v due to error %v", zipFileName, err)
		}

	}()
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
	defer func() {
		err := rf.Close()
		if err != nil {
			log.Printf("unable to close file %v due to error %v", zipFileName, err)
		}
		err = os.Remove(rf.Name())
		if err != nil {
			log.Printf("unable to remove file %v due to error %v", rf, err)
		}
	}()
	_, err = io.Copy(w, rf)
	if err != nil {
		return err
	}
	return nil
}
