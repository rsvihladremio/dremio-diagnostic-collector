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
		if err := ddcio.DeleteDirContents(d); err != nil {
			simplelog.Warningf("unable to delete the contents of folder %v due to error %v", d, err)
		}
	}
}

var _ = Describe("Logcollect", func() {

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
		err = os.MkdirAll(destinationQueriesJSON, 0750)
		Expect(err).To(BeNil())
		logDir, err = os.MkdirTemp("", "SOURCE*")
		Expect(err).To(BeNil())
		destinationDir, err = os.MkdirTemp("", "DESTINATION*")
		Expect(err).To(BeNil())
		logCollector = *logcollect.NewLogCollector(
			logDir,
			destinationDir,
			testGCLogsDir,
			"gc.*.log*",
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

	When("all server.logs are present", func() {
		var destinationDir string
		var testLogDir string
		var yesterdaysLog string
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
			yesterdaysLog = "server." + time.Now().AddDate(0, 0, -1).Format("2006-01-02") + ".log.gz"
			if err := ddcio.CopyFile(filepath.Join(testLogDir, "archive", "server.2022-04-30.log.gz"), filepath.Join(testLogDir, "archive", yesterdaysLog)); err != nil {
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
		var yesterdaysLog string
		var testLogDir string
		BeforeEach(func() {
			destinationDir, testLogDir = setupEnv()
			//setup logs
			if err := ddcio.CopyDir(startLogDir, testLogDir); err != nil {
				simplelog.Errorf("test should fail as we had an error setting up the test directory: %v", err)
				Expect(err).To(BeNil())
			}
			//rename archive to yesterday
			yesterdaysLog = "server." + time.Now().AddDate(0, 0, -1).Format("2006-01-02") + ".log.gz"
			if err := ddcio.CopyFile(filepath.Join(testLogDir, "archive", "server.2022-04-30.log.gz"), filepath.Join(testLogDir, "archive", yesterdaysLog)); err != nil {
				simplelog.Errorf("test should fail as we had an error setting up the test directory: %v", err)
				Expect(err).To(BeNil())
			}

			if err := os.Remove(filepath.Join(testLogDir, "server.out")); err != nil {
				simplelog.Errorf("test should fail as we had an error removing the server.out: %v", err)
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

		It("should collect all logs with age", func() {
			Expect(filepath.Join(destinationDir, "server.log.gz")).To(ContainThisFileInTheGzip(filepath.Join(testLogDir, "server.log")))
			Expect(filepath.Join(destinationDir, yesterdaysLog)).To(MatchFile(filepath.Join(testLogDir, "archive", yesterdaysLog)))
		})

		It("should ignore logs older than num days", func() {
			_, err := os.Stat(filepath.Join(destinationDir, "server.2022-04-30.log.gz"))
			Expect(os.IsNotExist(err)).To(BeTrue())
		})

	})
	When("server.log is missing", func() {
		var err error
		var destinationDir string
		var yesterdaysLog string
		var testLogDir string
		BeforeEach(func() {
			destinationDir, testLogDir = setupEnv()
			//setup logs
			if err := ddcio.CopyDir(startLogDir, testLogDir); err != nil {
				simplelog.Errorf("test should fail as we had an error setting up the test directory: %v", err)
				Expect(err).To(BeNil())
			}
			yesterdaysLog = "server." + time.Now().AddDate(0, 0, -1).Format("2006-01-02") + ".log.gz"
			if err := ddcio.CopyFile(filepath.Join(testLogDir, "archive", "server.2022-04-30.log.gz"), filepath.Join(testLogDir, "archive", yesterdaysLog)); err != nil {
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

	When("all reflection logs are present", func() {
		var err error
		var destinationDir string
		var yesterdaysLog string
		var testLogDir string
		BeforeEach(func() {
			destinationDir, testLogDir = setupEnv()
			//setup logs
			if err := ddcio.CopyDir(startLogDir, testLogDir); err != nil {
				simplelog.Errorf("test should fail as we had an error setting up the test directory: %v", err)
				Expect(err).To(BeNil())
			}
			yesterdaysLog = "reflection." + time.Now().AddDate(0, 0, -1).Format("2006-01-02") + ".log.gz"
			if err := ddcio.CopyFile(filepath.Join(testLogDir, "archive", "reflection.2022-04-30.log.gz"), filepath.Join(testLogDir, "archive", yesterdaysLog)); err != nil {
				treeOut := tests.TreeToString(filepath.Join(testLogDir, "archive"))
				simplelog.Errorf("test should fail as we had an error setting up the test directory: %v with dir of %v", err, treeOut)
				Expect(err).To(BeNil())
			}
			err = logCollector.RunCollectReflectionLogs()
		})
		AfterEach(func() {
			cleanUp(destinationDir, testLogDir)
		})

		It("should not return an error", func() {
			Expect(err).To(BeNil(), fmt.Sprintf("unexpected error %v", err))
		})

		It("should collect all logs", func() {
			Expect(filepath.Join(destinationDir, "reflection.log.gz")).To(ContainThisFileInTheGzip(filepath.Join(testLogDir, "reflection.log")), fmt.Sprintf("failed to find reflection.log in tree %v", tests.TreeToString(destinationDir)))
			Expect(filepath.Join(destinationDir, yesterdaysLog)).To(MatchFile(filepath.Join(testLogDir, "archive", yesterdaysLog)))
		})

		It("should ignore logs older than num days", func() {
			_, err := os.Stat(filepath.Join(destinationDir, "reflection.2022-04-30.log.gz"))
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
	})
	When("reflection.log archives are missing", func() {
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
			err = logCollector.RunCollectReflectionLogs()
		})
		AfterEach(func() {
			cleanUp(destinationDir, testLogDir)
		})

		It("should return an error", func() {
			Expect(err).ToNot(BeNil())
		})

		It("should collect reflection.log", func() {
			Expect(filepath.Join(destinationDir, "reflection.log.gz")).To(ContainThisFileInTheGzip(filepath.Join(testLogDir, "reflection.log")), fmt.Sprintf("failed to find reflection.log in tree %v", tests.TreeToString(destinationDir)))
		})
	})

	When("reflection.log is missing", func() {
		var err error
		var destinationDir string
		var yesterdaysLog string
		var testLogDir string
		BeforeEach(func() {
			destinationDir, testLogDir = setupEnv()
			//setup logs
			if err := ddcio.CopyDir(startLogDir, testLogDir); err != nil {
				simplelog.Errorf("test should fail as we had an error setting up the test directory: %v", err)
				Expect(err).To(BeNil())
			}

			if err := os.Remove(filepath.Join(testLogDir, "reflection.log")); err != nil {
				simplelog.Errorf("test should fail as we had an error removing the reflection.log: %v", err)
				Expect(err).To(BeNil())
			}

			yesterdaysLog = "reflection." + time.Now().AddDate(0, 0, -1).Format("2006-01-02") + ".log.gz"
			if err := ddcio.CopyFile(filepath.Join(testLogDir, "archive", "reflection.2022-04-30.log.gz"), filepath.Join(testLogDir, "archive", yesterdaysLog)); err != nil {
				simplelog.Errorf("test should fail as we had an error setting up the test directory: %v", err)
				Expect(err).To(BeNil())
			}
			err = logCollector.RunCollectReflectionLogs()
		})
		AfterEach(func() {
			cleanUp(destinationDir, testLogDir)
		})

		It("should return an error", func() {
			Expect(err).ToNot(BeNil())
		})

		It("should collect archives", func() {
			Expect(filepath.Join(destinationDir, yesterdaysLog)).To(MatchFile(filepath.Join(testLogDir, "archive", yesterdaysLog)))
		})
	})

	When("all acceleration logs are present", func() {
		var err error
		var destinationDir string
		var yesterdaysLog string
		var testLogDir string
		BeforeEach(func() {
			destinationDir, testLogDir = setupEnv()
			//setup logs
			if err := ddcio.CopyDir(startLogDir, testLogDir); err != nil {
				simplelog.Errorf("test should fail as we had an error setting up the test directory: %v", err)
				Expect(err).To(BeNil())
			}
			yesterdaysLog = "acceleration." + time.Now().AddDate(0, 0, -1).Format("2006-01-02") + ".log.gz"
			if err := ddcio.CopyFile(filepath.Join(testLogDir, "archive", "acceleration.2022-04-30.log.gz"), filepath.Join(testLogDir, "archive", yesterdaysLog)); err != nil {
				simplelog.Errorf("test should fail as we had an error setting up the test directory: %v", err)
				Expect(err).To(BeNil())
			}
			err = logCollector.RunCollectAccelerationLogs()
		})
		AfterEach(func() {
			cleanUp(destinationDir, testLogDir)
		})

		It("should not return an error", func() {
			Expect(err).To(BeNil())
		})

		It("should collect all logs", func() {
			Expect(filepath.Join(destinationDir, "acceleration.log.gz")).To(ContainThisFileInTheGzip(filepath.Join(testLogDir, "acceleration.log")), fmt.Sprintf("failed to find acceleration.log in tree %v", tests.TreeToString(destinationDir)))
			Expect(filepath.Join(destinationDir, yesterdaysLog)).To(MatchFile(filepath.Join(testLogDir, "archive", yesterdaysLog)))
		})

		It("should ignore logs older than num days", func() {
			_, err := os.Stat(filepath.Join(destinationDir, "acceleration.2022-04-30.log.gz"))
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
	})
	When("acceleration.log archives are missing", func() {
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
			err = logCollector.RunCollectAccelerationLogs()
		})
		AfterEach(func() {
			cleanUp(destinationDir, testLogDir)
		})

		It("should return an error", func() {
			Expect(err).ToNot(BeNil())
		})

		It("should collect acceleration log as a gzip", func() {
			Expect(filepath.Join(destinationDir, "acceleration.log.gz")).To(ContainThisFileInTheGzip(filepath.Join(testLogDir, "acceleration.log")), fmt.Sprintf("failed to find acceleration.log in tree %v", tests.TreeToString(destinationDir)))
		})
	})

	When("acceleration.log is missing", func() {
		var err error
		var destinationDir string
		var yesterdaysLog string
		var testLogDir string
		BeforeEach(func() {
			destinationDir, testLogDir = setupEnv()
			//setup logs
			if err := ddcio.CopyDir(startLogDir, testLogDir); err != nil {
				simplelog.Errorf("test should fail as we had an error setting up the test directory: %v", err)
				Expect(err).To(BeNil())
			}

			if err := os.Remove(filepath.Join(testLogDir, "acceleration.log")); err != nil {
				simplelog.Errorf("test should fail as we had an error removing the acceleration.log: %v", err)
				Expect(err).To(BeNil())
			}

			yesterdaysLog = "acceleration." + time.Now().AddDate(0, 0, -1).Format("2006-01-02") + ".log.gz"
			if err := ddcio.CopyFile(filepath.Join(testLogDir, "archive", "acceleration.2022-04-30.log.gz"), filepath.Join(testLogDir, "archive", yesterdaysLog)); err != nil {
				simplelog.Errorf("test should fail as we had an error setting up the test directory: %v", err)
				Expect(err).To(BeNil())
			}
			err = logCollector.RunCollectAccelerationLogs()
		})
		AfterEach(func() {
			cleanUp(destinationDir, testLogDir)
		})

		It("should return an error", func() {
			Expect(err).ToNot(BeNil())
		})

		It("should collect archives", func() {
			Expect(filepath.Join(destinationDir, yesterdaysLog)).To(MatchFile(filepath.Join(testLogDir, "archive", yesterdaysLog)))
		})
	})

	When("all access logs are present", func() {
		var err error
		var destinationDir string
		var yesterdaysLog string
		var testLogDir string
		BeforeEach(func() {
			destinationDir, testLogDir = setupEnv()
			//setup logs
			if err := ddcio.CopyDir(startLogDir, testLogDir); err != nil {
				simplelog.Errorf("test should fail as we had an error setting up the test directory: %v", err)
				Expect(err).To(BeNil())
			}
			yesterdaysLog = "access." + time.Now().AddDate(0, 0, -1).Format("2006-01-02") + ".log.gz"
			if err := ddcio.CopyFile(filepath.Join(testLogDir, "archive", "access.2022-04-30.log.gz"), filepath.Join(testLogDir, "archive", yesterdaysLog)); err != nil {
				simplelog.Errorf("test should fail as we had an error setting up the test directory: %v", err)
				Expect(err).To(BeNil())
			}
			err = logCollector.RunCollectDremioAccessLogs()
		})
		AfterEach(func() {
			cleanUp(destinationDir, testLogDir)
		})

		It("should not return an error", func() {
			Expect(err).To(BeNil())
		})

		It("should collect all logs", func() {
			Expect(filepath.Join(destinationDir, "access.log.gz")).To(ContainThisFileInTheGzip(filepath.Join(testLogDir, "access.log")), fmt.Sprintf("failed to find access.log in tree %v", tests.TreeToString(destinationDir)))
			Expect(filepath.Join(destinationDir, yesterdaysLog)).To(MatchFile(filepath.Join(testLogDir, "archive", yesterdaysLog)))
		})

		It("should ignore logs older than num days", func() {
			_, err := os.Stat(filepath.Join(destinationDir, "access.2022-04-30.log.gz"))
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
	})
	When("access.log archives are missing", func() {
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
			err = logCollector.RunCollectDremioAccessLogs()
		})
		AfterEach(func() {
			cleanUp(destinationDir, testLogDir)
		})

		It("should return an error", func() {
			Expect(err).ToNot(BeNil())
		})

		It("should collect access log as a gzip", func() {
			Expect(filepath.Join(destinationDir, "access.log.gz")).To(ContainThisFileInTheGzip(filepath.Join(testLogDir, "access.log")), fmt.Sprintf("failed to find access.log in tree %v", tests.TreeToString(destinationDir)))
		})
	})

	When("access.log is missing", func() {
		var err error
		var destinationDir string
		var yesterdaysLog string
		var testLogDir string
		BeforeEach(func() {
			destinationDir, testLogDir = setupEnv()
			//setup logs
			if err := ddcio.CopyDir(startLogDir, testLogDir); err != nil {
				simplelog.Errorf("test should fail as we had an error setting up the test directory: %v", err)
				Expect(err).To(BeNil())
			}

			if err := os.Remove(filepath.Join(testLogDir, "access.log")); err != nil {
				simplelog.Errorf("test should fail as we had an error removing the access.log: %v", err)
				Expect(err).To(BeNil())
			}

			yesterdaysLog = "access." + time.Now().AddDate(0, 0, -1).Format("2006-01-02") + ".log.gz"
			if err := ddcio.CopyFile(filepath.Join(testLogDir, "archive", "access.2022-04-30.log.gz"), filepath.Join(testLogDir, "archive", yesterdaysLog)); err != nil {
				simplelog.Errorf("test should fail as we had an error setting up the test directory: %v", err)
				Expect(err).To(BeNil())
			}
			err = logCollector.RunCollectDremioAccessLogs()
		})
		AfterEach(func() {
			cleanUp(destinationDir, testLogDir)
		})

		It("should return an error", func() {
			Expect(err).ToNot(BeNil())
		})

		It("should collect archives", func() {
			Expect(filepath.Join(destinationDir, yesterdaysLog)).To(MatchFile(filepath.Join(testLogDir, "archive", yesterdaysLog)))
		})
	})

	When("all metadata_refresh logs are present", func() {
		var err error
		var destinationDir string
		var yesterdaysLog string
		var testLogDir string
		BeforeEach(func() {
			destinationDir, testLogDir = setupEnv()
			//setup logs
			if err := ddcio.CopyDir(startLogDir, testLogDir); err != nil {
				simplelog.Errorf("test should fail as we had an error setting up the test directory: %v", err)
				Expect(err).To(BeNil())
			}
			yesterdaysLog = "metadata_refresh." + time.Now().AddDate(0, 0, -1).Format("2006-01-02") + ".log.gz"
			if err := ddcio.CopyFile(filepath.Join(testLogDir, "archive", "metadata_refresh.2022-04-30.log.gz"), filepath.Join(testLogDir, "archive", yesterdaysLog)); err != nil {
				simplelog.Errorf("test should fail as we had an error setting up the test directory: %v", err)
				Expect(err).To(BeNil())
			}
			err = logCollector.RunCollectMetadataRefreshLogs()
		})
		AfterEach(func() {
			cleanUp(destinationDir, testLogDir)
		})

		It("should not return an error", func() {
			Expect(err).To(BeNil())
		})

		It("should collect all logs", func() {
			Expect(filepath.Join(destinationDir, "metadata_refresh.log.gz")).To(ContainThisFileInTheGzip(filepath.Join(testLogDir, "metadata_refresh.log")), fmt.Sprintf("failed to find metadata_refresh.log in tree %v", tests.TreeToString(destinationDir)))
			Expect(filepath.Join(destinationDir, yesterdaysLog)).To(MatchFile(filepath.Join(testLogDir, "archive", yesterdaysLog)))
		})

		It("should ignore logs older than num days", func() {
			_, err := os.Stat(filepath.Join(destinationDir, "metadata_refresh.2022-04-30.log.gz"))
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
	})
	When("metadata_frefresh.log archives are missing", func() {
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
			err = logCollector.RunCollectMetadataRefreshLogs()
		})
		AfterEach(func() {
			cleanUp(destinationDir, testLogDir)
		})

		It("should return an error", func() {
			Expect(err).ToNot(BeNil())
		})

		It("should collect access log as a gzip", func() {
			Expect(filepath.Join(destinationDir, "metadata_refresh.log.gz")).To(ContainThisFileInTheGzip(filepath.Join(testLogDir, "metadata_refresh.log")), fmt.Sprintf("failed to find metadata_refresh.log in tree %v", tests.TreeToString(destinationDir)))
		})
	})

	When("metadata_refresh.log is missing", func() {
		var err error
		var destinationDir string
		var yesterdaysLog string
		var testLogDir string
		BeforeEach(func() {
			destinationDir, testLogDir = setupEnv()
			//setup logs
			if err := ddcio.CopyDir(startLogDir, testLogDir); err != nil {
				simplelog.Errorf("test should fail as we had an error setting up the test directory: %v", err)
				Expect(err).To(BeNil())
			}

			if err := os.Remove(filepath.Join(testLogDir, "metadata_refresh.log")); err != nil {
				simplelog.Errorf("test should fail as we had an error removing the metadata_refresh.log: %v", err)
				Expect(err).To(BeNil())
			}

			yesterdaysLog = "metadata_refresh." + time.Now().AddDate(0, 0, -1).Format("2006-01-02") + ".log.gz"
			if err := ddcio.CopyFile(filepath.Join(testLogDir, "archive", "metadata_refresh.2022-04-30.log.gz"), filepath.Join(testLogDir, "archive", yesterdaysLog)); err != nil {
				simplelog.Errorf("test should fail as we had an error setting up the test directory: %v", err)
				Expect(err).To(BeNil())
			}
			err = logCollector.RunCollectMetadataRefreshLogs()
		})
		AfterEach(func() {
			cleanUp(destinationDir, testLogDir)
		})

		It("should return an error", func() {
			Expect(err).ToNot(BeNil())
		})

		It("should collect archives", func() {
			Expect(filepath.Join(destinationDir, yesterdaysLog)).To(MatchFile(filepath.Join(testLogDir, "archive", yesterdaysLog)))
		})
	})

	When("all queries.json logs are present", func() {
		var err error
		var yesterdaysLog string
		var testLogDir string
		BeforeEach(func() {
			_, testLogDir = setupEnv()
			//setup logs
			if err := ddcio.CopyDir(startLogDir, testLogDir); err != nil {
				simplelog.Errorf("test should fail as we had an error setting up the test directory: %v", err)
				Expect(err).To(BeNil())
			}
			yesterdaysLog = "queries." + time.Now().AddDate(0, 0, -1).Format("2006-01-02") + ".json.gz"
			if err := ddcio.CopyFile(filepath.Join(testLogDir, "archive", "queries.2022-04-30.json.gz"), filepath.Join(testLogDir, "archive", yesterdaysLog)); err != nil {
				simplelog.Errorf("test should fail as we had an error setting up the test directory: %v", err)
				Expect(err).To(BeNil())
			}
			tests.Tree(testLogDir)
			err = logCollector.RunCollectQueriesJSON()
			tests.Tree(destinationQueriesJSON)
		})
		AfterEach(func() {
			cleanUp(destinationQueriesJSON, testLogDir)
		})

		It("should not return an error", func() {
			Expect(err).To(BeNil())
		})

		It("should collect all logs", func() {
			Expect(filepath.Join(destinationQueriesJSON, "queries.json.gz")).To(ContainThisFileInTheGzip(filepath.Join(testLogDir, "queries.json")), fmt.Sprintf("failed to find queries.json in tree %v", tests.TreeToString(destinationQueriesJSON)))
			Expect(filepath.Join(destinationQueriesJSON, yesterdaysLog)).To(MatchFile(filepath.Join(testLogDir, "archive", yesterdaysLog)))
		})

		It("should ignore logs older than num days", func() {
			_, err := os.Stat(filepath.Join(destinationQueriesJSON, "queries.2022-04-30.json.gz"))
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
	})

	When("queries.json archives are missing", func() {
		var err error
		var testLogDir string
		BeforeEach(func() {
			_, testLogDir = setupEnv()
			//setup logs
			if err := ddcio.CopyDir(startLogDir, testLogDir); err != nil {
				simplelog.Errorf("test should fail as we had an error setting up the test directory: %v", err)
				Expect(err).To(BeNil())
			}
			// just deleting the archive folder entirely
			if err := os.RemoveAll(filepath.Join(testLogDir, "archive")); err != nil {
				Expect(err).To(BeNil())
			}
			err = logCollector.RunCollectQueriesJSON()
		})
		AfterEach(func() {
			cleanUp(destinationQueriesJSON, testLogDir)
		})

		It("should return an error", func() {
			Expect(err).ToNot(BeNil())
		})

		It("should collect access log as a gzip", func() {
			tests.Tree(destinationQueriesJSON)
			Expect(filepath.Join(destinationQueriesJSON, "queries.json.gz")).To(ContainThisFileInTheGzip(filepath.Join(testLogDir, "queries.json")), fmt.Sprintf("failed to find queries.json in tree %v", tests.TreeToString(destinationQueriesJSON)))
		})
	})

	When("queries.json is missing", func() {
		var err error
		var yesterdaysLog string
		var testLogDir string
		BeforeEach(func() {
			_, testLogDir = setupEnv()
			//setup logs
			if err := ddcio.CopyDir(startLogDir, testLogDir); err != nil {
				simplelog.Errorf("test should fail as we had an error setting up the test directory: %v", err)
				Expect(err).To(BeNil())
			}

			if err := os.Remove(filepath.Join(testLogDir, "queries.json")); err != nil {
				simplelog.Errorf("test should fail as we had an error removing the queries.json: %v", err)
				Expect(err).To(BeNil())
			}

			yesterdaysLog = "queries." + time.Now().AddDate(0, 0, -1).Format("2006-01-02") + ".json.gz"
			if err := ddcio.CopyFile(filepath.Join(testLogDir, "archive", "queries.2022-04-30.json.gz"), filepath.Join(testLogDir, "archive", yesterdaysLog)); err != nil {
				simplelog.Errorf("test should fail as we had an error setting up the test directory: %v", err)
				Expect(err).To(BeNil())
			}
			err = logCollector.RunCollectQueriesJSON()
			tests.Tree(destinationQueriesJSON)
		})
		AfterEach(func() {
			cleanUp(destinationQueriesJSON, testLogDir)
		})

		It("should return an error", func() {
			Expect(err).ToNot(BeNil())
		})

		It("should collect queriesJSON archives", func() {
			Expect(filepath.Join(destinationQueriesJSON, yesterdaysLog)).To(MatchFile(filepath.Join(testLogDir, "archive", yesterdaysLog)))
		})
	})

	When("gc logs are present and there are more than one", func() {
		var err error
		var destinationDir string
		BeforeEach(func() {
			destinationDir, _ = setupEnv()
			err = logCollector.RunCollectGcLogs()
			Expect(err).To(BeNil())
		})
		AfterEach(func() {
			cleanUp(destinationDir)
		})
		It("should collect all gc logs as gzips", func() {
			tests.Tree(destinationDir)
			tests.Tree(testGCLogsDir)
			Expect(filepath.Join(destinationDir, "gc.0.log")).To(MatchFile(filepath.Join(testGCLogsDir, "gc.0.log")), fmt.Sprintf("failed to find gc.0.log in tree %v", tests.TreeToString(destinationDir)))
			Expect(filepath.Join(destinationDir, "gc.1.log")).To(MatchFile(filepath.Join(testGCLogsDir, "gc.1.log")), fmt.Sprintf("failed to find gc.1.log in tree %v", tests.TreeToString(destinationDir)))
			Expect(filepath.Join(destinationDir, "gc.2.log")).To(MatchFile(filepath.Join(testGCLogsDir, "gc.2.log")), fmt.Sprintf("failed to find gc.2.log in tree %v", tests.TreeToString(destinationDir)))
			Expect(filepath.Join(destinationDir, "gc.3.log")).To(MatchFile(filepath.Join(testGCLogsDir, "gc.3.log")), fmt.Sprintf("failed to find gc.3.log in tree %v", tests.TreeToString(destinationDir)))
			Expect(filepath.Join(destinationDir, "gc.4.log.current")).To(MatchFile(filepath.Join(testGCLogsDir, "gc.4.log.current")), fmt.Sprintf("gc.4.log.current to find metadata_refresh.log in tree %v", tests.TreeToString(destinationDir)))
		})
	})
})
