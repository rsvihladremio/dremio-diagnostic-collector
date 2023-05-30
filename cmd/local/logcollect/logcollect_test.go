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

package logcollect_test

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rsvihladremio/dremio-diagnostic-collector/cmd/local/logcollect"
	"github.com/rsvihladremio/dremio-diagnostic-collector/cmd/simplelog"
	"github.com/rsvihladremio/dremio-diagnostic-collector/pkg/ddcio"
	. "github.com/rsvihladremio/dremio-diagnostic-collector/pkg/matchers"
	"github.com/rsvihladremio/dremio-diagnostic-collector/pkg/tests"
)

func cleanUp(dirs ...string) {
	for _, d := range dirs {
		simplelog.Infof("deleting %v", d)
		if err := os.RemoveAll(d); err != nil {
			simplelog.Warningf("unable to delete the contents of folder %v due to error %v", d, err)
		}
	}
	fmt.Println("--------------\n----------------")
}

var _ = Describe("Logcollect", func() {

	var yesterdaysLog string
	var logCollector logcollect.Collector
	var destinationQueriesJSON string
	var startLogDir string
	var testGCLogsDir string
	var dremioLogDays int
	var dremioQueriesJSONDays int

	var setupEnv = func() (destinationDir, logDir string) {
		dremioLogDays = 2
		dremioQueriesJSONDays = 4
		startLogDir = filepath.Join("testdata", "logDirWithAllLogs")
		testGCLogsDir = filepath.Join("testdata", "gcLogDir")
		destinationQueriesJSON = filepath.Join("testdata", "queriesOutDir")
		var err error
		logDir, err = os.MkdirTemp("", "SOURCE*")
		Expect(err).To(BeNil())
		destinationDir, err = os.MkdirTemp("", "DESTINATION*")
		Expect(err).To(BeNil())
		logCollector = *logcollect.NewLogCollector(
			logDir,
			destinationDir,
			testGCLogsDir,
			"gc*.log",
			destinationQueriesJSON,
			dremioQueriesJSONDays,
			dremioLogDays,
		)
		return destinationDir, logDir
	}

	AfterEach(func() {
		if err := ddcio.DeleteDirContents(destinationQueriesJSON); err != nil {
			simplelog.Warningf("unable to delete the contents of folder %v due to error %v", destinationQueriesJSON, err)
		}
	})

	When("all logs are present", func() {
		var destinationDir string
		var testLogDir string

		AfterEach(func() {
			cleanUp(destinationDir, testLogDir)
		})
		It("should collect all logs", func() {
			destinationDir, testLogDir = setupEnv()
			//setup logs
			if err := ddcio.CopyDir(startLogDir, testLogDir); err != nil {
				simplelog.Errorf("test should fail as we had an error setting up the test directory: %v", err)
				Expect(err).To(BeNil())
			}

			//rename archive to yesterday
			yesterdaysLog = "server.log." + time.Now().AddDate(0, 0, -1).Format("2006-01-02") + ".gz"
			if err := os.Rename(filepath.Join(testLogDir, "archive", "server.log.2022-04-30.gz"), filepath.Join(testLogDir, "archive", yesterdaysLog)); err != nil {
				simplelog.Errorf("test should fail as we had an error setting up the test directory: %v", err)
				Expect(err).To(BeNil())
			}

			tests.Tree(testLogDir)
			err := logCollector.RunCollectDremioServerLog()
			Expect(err).To(BeNil())
			tests.Tree(destinationDir)
			Expect(filepath.Join(destinationDir, "server.log.gz")).To(ContainThisFileInTheGzip(filepath.Join(testLogDir, "server.log")))
			Expect(filepath.Join(destinationDir, "server.out")).To(MatchFile(filepath.Join(testLogDir, "server.out")))
			Expect(filepath.Join(destinationDir, yesterdaysLog)).To(MatchFile(filepath.Join(testLogDir, "archive", yesterdaysLog)))
		})
	})

	When("server.out is missing", func() {
		var err error
		var destinationDir string
		var testLogDir string
		BeforeEach(func() {
			destinationDir, testLogDir = setupEnv()
			//setup logs
			if err := ddcio.CopyDir(startLogDir, testLogDir); err != nil {
				simplelog.Errorf("test should fail as we had an error setting up the test directory: %v", err)
				Expect(err).To(BeNil())
			}
			//rename archive to yesterday
			yesterdaysLog = "server.log." + time.Now().AddDate(0, 0, -1).Format("2006-01-02") + ".gz"
			if err := os.Rename(filepath.Join(testLogDir, "archive", "server.log.2022-04-30.gz"), filepath.Join(testLogDir, "archive", yesterdaysLog)); err != nil {
				simplelog.Errorf("test should fail as we had an error setting up the test directory: %v", err)
				Expect(err).To(BeNil())
			}

			if err := os.Remove(filepath.Join(testLogDir, "server.out")); err != nil {
				simplelog.Errorf("test should fail as we had an error removin the server.out: %v", err)
				Expect(err).To(BeNil())
			}
			err = logCollector.RunCollectDremioServerLog()

		})
		AfterEach(func() {
			cleanUp(destinationDir, testLogDir)
		})

		It("should return an error", func() {
			Expect(err).ToNot(BeNil())
		})

		It("should collect all logs", func() {
			Expect(filepath.Join(destinationDir, "server.log.gz")).To(ContainThisFileInTheGzip(filepath.Join(testLogDir, "server.log")))
			Expect(filepath.Join(destinationDir, yesterdaysLog)).To(MatchFile(filepath.Join(testLogDir, "archive", yesterdaysLog)))
		})

	})
	When("server.log is missing", func() {
		var err error
		var destinationDir string
		var testLogDir string
		BeforeEach(func() {
			destinationDir, testLogDir = setupEnv()
			//setup logs
			if err := ddcio.CopyDir(startLogDir, testLogDir); err != nil {
				simplelog.Errorf("test should fail as we had an error setting up the test directory: %v", err)
				Expect(err).To(BeNil())
			}
			yesterdaysLog = "server.log." + time.Now().AddDate(0, 0, -1).Format("2006-01-02") + ".gz"
			if err := os.Rename(filepath.Join(testLogDir, "archive", "server.log.2022-04-30.gz"), filepath.Join(testLogDir, "archive", yesterdaysLog)); err != nil {
				simplelog.Errorf("test should fail as we had an error setting up the test directory: %v", err)
				Expect(err).To(BeNil())
			}
			if err := os.Remove(filepath.Join(testLogDir, "server.log")); err != nil {
				simplelog.Errorf("test should fail as we had an error removing the server.log: %v", err)
				Expect(err).To(BeNil())
			}
			err = logCollector.RunCollectDremioServerLog()
		})
		AfterEach(func() {
			cleanUp(destinationDir, testLogDir)
		})

		It("should return an error", func() {
			Expect(err).ToNot(BeNil())
		})

		It("should collect all logs still present", func() {
			Expect(filepath.Join(destinationDir, "server.out")).To(MatchFile(filepath.Join(testLogDir, "server.out")))
			Expect(filepath.Join(destinationDir, yesterdaysLog)).To(MatchFile(filepath.Join(testLogDir, "archive", yesterdaysLog)))
		})

	})
	When("server.log archives are missing", func() {
		var err error
		var destinationDir string
		var testLogDir string
		BeforeEach(func() {
			destinationDir, testLogDir = setupEnv()
			//setup logs
			if err := ddcio.CopyDir(startLogDir, testLogDir); err != nil {
				simplelog.Errorf("test should fail as we had an error setting up the test directory: %v", err)
				Expect(err).To(BeNil())
			}

			// just deleting the archive folder entirely
			if err := os.RemoveAll(filepath.Join(testLogDir, "archive")); err != nil {
				Expect(err).To(BeNil())
			}
			err = logCollector.RunCollectDremioServerLog()
		})
		AfterEach(func() {
			cleanUp(destinationDir, testLogDir)
		})

		It("should return an error", func() {
			Expect(err).ToNot(BeNil())
		})

		It("should collect all logs still present", func() {
			Expect(filepath.Join(destinationDir, "server.log.gz")).To(ContainThisFileInTheGzip(filepath.Join(testLogDir, "server.log")), fmt.Sprintf("failed to find server.log in tree %v", tests.TreeToString(destinationDir)))
			Expect(filepath.Join(destinationDir, "server.out")).To(MatchFile(filepath.Join(testLogDir, "server.out")), fmt.Sprintf("failed to find server.out in tree %v", tests.TreeToString(destinationDir)))
		})

	})

	// Describe("collecting acceleration.log logs", func() {
	// 	Context("all logs are present", func() {
	// 		It("should collect all logs", func() {

	// 		})
	// 	})
	// 	Context("acceleration.log archives are missing")
	// 	Context("acceleration.log is missing")

	// })

	// Describe("collecting reflection.log logs", func() {
	// 	Context("all logs are present", func() {
	// 		It("should collect all logs", func() {

	// 		})
	// 	})

	// 	Context("reflection.log archives are missing")
	// 	Context("reflection.log is missing")
	// })
})
