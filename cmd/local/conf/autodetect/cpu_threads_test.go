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

package autodetect_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/rsvihladremio/dremio-diagnostic-collector/cmd/local/conf/autodetect"
)

var _ = Describe("GetDefaultThreadsFromCPUs", func() {
	Context("Given the number of CPUs", func() {
		It("should return the default number of threads", func() {
			var numCPUs int
			var expectedNumThreads int

			By("having number of CPUs as 1")
			numCPUs = 1
			expectedNumThreads = 2 // The minimum number of threads should be 2
			Expect(autodetect.GetDefaultThreadsFromCPUs(numCPUs)).To(Equal(expectedNumThreads))

			By("having number of CPUs as 4")
			numCPUs = 4
			expectedNumThreads = 2 // 4/2 = 2, which is not less than 2
			Expect(autodetect.GetDefaultThreadsFromCPUs(numCPUs)).To(Equal(expectedNumThreads))

			By("having number of CPUs as 6")
			numCPUs = 6
			expectedNumThreads = 3 // 6/2 = 3
			Expect(autodetect.GetDefaultThreadsFromCPUs(numCPUs)).To(Equal(expectedNumThreads))

			By("having an odd number of CPUs as 5")
			numCPUs = 5
			expectedNumThreads = 2 // 5/2 = 2.5 (rounded down to 2 by Go's integer division)
			Expect(autodetect.GetDefaultThreadsFromCPUs(numCPUs)).To(Equal(expectedNumThreads))

			By("having large number of CPUs as 1000")
			numCPUs = 1000
			expectedNumThreads = 500 // 1000/2 = 500
			Expect(autodetect.GetDefaultThreadsFromCPUs(numCPUs)).To(Equal(expectedNumThreads))
		})
	})

	Context("Given negative or zero CPUs", func() {
		It("should return at least 2 threads", func() {
			var numCPUs int
			expectedNumThreads := 2 // In these cases, the function should return 2 threads

			By("having number of CPUs as 0")
			numCPUs = 0
			Expect(autodetect.GetDefaultThreadsFromCPUs(numCPUs)).To(Equal(expectedNumThreads))

			By("having negative number of CPUs")
			numCPUs = -4
			Expect(autodetect.GetDefaultThreadsFromCPUs(numCPUs)).To(Equal(expectedNumThreads))
		})
	})
})
