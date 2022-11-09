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
	"io/fs"
	"log"
	"os"
)

var logger log.Logger

// Launch check of files from a list (array)
func fileCheck(files []string, limit int64, excl string) (error, []string) {
	var skipped []string
	for _, file := range files {
		fstat, err := os.Stat(file)
		if err != nil {
			return err, skipped
		}
		// Check the file being listed for collection is within size
		err = checkFileSize(fstat, limit)
		if err != nil {
			logger.Println(err)
			skipped = append(skipped, file)
		}

		// Check the file being listed for collection is not a match to an excluded name
		err = checkFileExclusion(fstat, excl)
		if err != nil {
			logger.Println(err)
			skipped = append(skipped, file)
		}

	}
	return nil, skipped
}

// Check file size and limit
func checkFileSize(fstat fs.FileInfo, limit int64) (err error) {
	if fstat.Size() > limit {
		err = fmt.Errorf("WARN: file %v is greater than the limit of %v. Skipping collection", fstat.Name(), limit)
	}
	return err
}

// Check file exclusion
func checkFileExclusion(fstat fs.FileInfo, excl string) (err error) {
	if fstat.Name() == excl {
		err = fmt.Errorf("WARN: file %v was excluded from collection as requested", fstat.Name())
	}
	return err
}
