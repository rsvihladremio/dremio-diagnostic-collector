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

package ddcio_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rsvihladremio/dremio-diagnostic-collector/pkg/ddcio"
)

func TestCompareFiles(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CompareFiles Suite")
}

var _ = Describe("CompareFiles", func() {
	Context("when comparing files with the same content", func() {
		It("should return true", func() {
			file1 := "testdata/file1.txt"
			file2 := "testdata/file1_copy.txt"
			areSame, err := ddcio.CompareFiles(file1, file2)
			Expect(err).To(BeNil())
			Expect(areSame).To(BeTrue())
		})
	})

	Context("when comparing files with different content", func() {
		It("should return false", func() {
			file1 := "testdata/file1.txt"
			file2 := "testdata/file2.txt"

			areSame, err := ddcio.CompareFiles(file1, file2)
			Expect(err).To(BeNil())
			Expect(areSame).To(BeFalse())
		})
	})

	Context("when comparing non-existent files", func() {
		It("should return an error", func() {
			file1 := "testdata/nonexistent1.txt"
			file2 := "testdata/nonexistent2.txt"

			areSame, err := ddcio.CompareFiles(file1, file2)
			Expect(err).NotTo(BeNil())
			Expect(areSame).To(BeFalse())
		})
	})
})
