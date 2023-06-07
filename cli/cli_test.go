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

package cli_test

import (
	"path/filepath"
	"runtime"
	"strings"

	"github.com/dremio/dremio-diagnostic-collector/cli"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cli", func() {
	var (
		c              *cli.Cli
		outputHandler  cli.OutputHandler
		executedOutput string
	)

	BeforeEach(func() {
		c = &cli.Cli{}
		executedOutput = ""
		outputHandler = func(line string) {
			executedOutput += line + "\n"
		}
	})

	Describe("ExecuteAndStreamOutput", func() {
		Context("with a valid command", func() {
			It("should stream the command output", func() {
				err := c.ExecuteAndStreamOutput(outputHandler, "ls", "-v")
				Expect(err).To(BeNil())
				Expect(strings.TrimSpace(executedOutput)).ToNot(BeEmpty())
			})
		})

		Context("with a command that produces stderr", func() {
			It("should stream the command error output", func() {
				err := c.ExecuteAndStreamOutput(outputHandler, "cat", "nonexistentfile")
				Expect(err).ToNot(BeNil())
				Expect(strings.TrimSpace(executedOutput)).To(ContainSubstring("No such file or directory"))
			})
		})

		Context("with an invalid command", func() {
			It("should return an error", func() {
				err := c.ExecuteAndStreamOutput(outputHandler, "22JIDJMJMHHF")
				Expect(err).To(HaveOccurred())
				Expect(err).To(BeAssignableToTypeOf(cli.UnableToStartErr{}))
				expectedErr := "unable to start command '22JIDJMJMHHF' due to error"
				Expect(strings.Contains(err.Error(), expectedErr)).To(BeTrue())
			})
		})
	})

	Describe("Execute", func() {
		Context("when the command is valid", func() {
			It("should execute the command and return the output", func() {
				var expectedOut string
				var out string
				var err error
				if runtime.GOOS == "windows" {
					out, err = c.Execute("cmd.exe", "/c", "dir", "/B", filepath.Join("testdata", "ls"))
					expectedOut = "file1\r\nfile2\r\n"
				} else {
					out, err = c.Execute("ls", "-a", filepath.Join("testdata", "ls"))
					expectedOut = "file1\nfile2\n"
				}
				Expect(err).NotTo(HaveOccurred())
				Expect(strings.Contains(out, expectedOut)).To(BeTrue())
			})
		})

		Context("when no arguments are provided for the command", func() {
			It("should execute the command and return the output", func() {
				var expectedOut string
				var out string
				var err error
				if runtime.GOOS == "windows" {
					out, err = c.Execute("cmd.exe")
					expectedOut = "Microsoft"
				} else {
					out, err = c.Execute("ls")
					expectedOut = "cli.go"
				}
				Expect(err).NotTo(HaveOccurred())
				Expect(strings.Contains(out, expectedOut)).To(BeTrue())
			})
		})

		Context("when the command is invalid", func() {
			It("should return an error", func() {
				_, err := c.Execute("22JIDJMJMHHF")
				Expect(err).To(HaveOccurred())
				Expect(err).To(BeAssignableToTypeOf(cli.UnableToStartErr{}))
				expectedErr := "unable to start command '22JIDJMJMHHF' due to error"
				Expect(strings.Contains(err.Error(), expectedErr)).To(BeTrue())
			})
		})
	})
})
