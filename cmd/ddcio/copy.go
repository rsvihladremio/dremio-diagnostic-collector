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

// ddcio include helper code for io operations common to ddc
package ddcio

import (
	"io"
	"os"
	"path"

	"github.com/rsvihladremio/dremio-diagnostic-collector/cmd/simplelog"
)

func CopyFile(srcPath, dstPath string) error {
	// Open the source file
	srcFile, err := os.Open(path.Clean(srcPath))
	if err != nil {
		return err
	}
	defer func() {
		if err := srcFile.Close(); err != nil {
			simplelog.Warningf("unable to close %v due to error %v", path.Clean(srcPath), err)
		}
	}()

	// Create the destination file
	dstFile, err := os.Create(path.Clean(dstPath))
	if err != nil {
		return err
	}
	defer func() {
		if err := dstFile.Close(); err != nil {
			simplelog.Errorf("unable to close file %v due to error %v", path.Clean(dstPath), err)
			os.Exit(1)
		}
	}()

	// Copy the contents of the source file to the destination file
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	// Flush the written data to disk
	err = dstFile.Sync()
	if err != nil {
		return err
	}

	return nil
}
