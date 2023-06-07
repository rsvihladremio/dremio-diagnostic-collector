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

package matchers_test

import (
	. "github.com/dremio/dremio-diagnostic-collector/pkg/matchers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("File Matchers", func() {
	Context("MatchFile", func() {
		It("should match files with the same content", func() {
			file1 := "testdata/file1.txt"
			file2 := "testdata/file2.txt"
			expectedFile := "testdata/expected.txt"

			// Expect file1 to match the expected file
			Expect(file1).To(MatchFile(expectedFile))

			// Expect file2 to match the expected file
			Expect(file2).To(MatchFile(expectedFile))
		})

		It("should not match files with different content", func() {
			file1 := "testdata/file1.txt"
			file3 := "testdata/file3.txt"
			expectedFile := "testdata/expected.txt"

			// Expect file1 to match the expected file
			Expect(file1).To(MatchFile(expectedFile))

			// Expect file3 not to match the expected file
			Expect(file3).NotTo(MatchFile(expectedFile))
		})
	})
})
