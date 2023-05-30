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
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rsvihladremio/dremio-diagnostic-collector/cmd/local/logcollect"
	"github.com/rsvihladremio/dremio-diagnostic-collector/cmd/simplelog"
	"github.com/rsvihladremio/dremio-diagnostic-collector/pkg/ddcio"
	. "github.com/rsvihladremio/dremio-diagnostic-collector/pkg/matchers"
)

var _ = Describe("Logcollect", func() {
	var dremioLogDays = 2
	var dremioQueriesJsonDays = 4
	var startLogDir = filepath.Join("testdata", "logDirWithAllLogs")
	var testLogDir = filepath.Join("testdata", "logDir")
	var destinationDir = filepath.Join("testdata", "destinationDir")
	var testGCLogsDir = filepath.Join("testdata", "gcLogDir")
	var destinationQueriesJson = filepath.Join("testdata", "queriesOutDir")
	var logCollector logcollect.Collector
	BeforeEach(func() {
		logCollector = *logcollect.NewLogCollector(
			testLogDir,
			destinationDir,
			testGCLogsDir,
			"gc*.log",
			destinationQueriesJson,
			dremioQueriesJsonDays,
			dremioLogDays,
		)
	})

	AfterEach(func() {
		if err := ddcio.DeleteDirContents(destinationDir); err != nil {
			simplelog.Warningf("unable to delete the contents of folder %v due to error %v", destinationDir, err)
		}
		if err := ddcio.DeleteDirContents(destinationQueriesJson); err != nil {
			simplelog.Warningf("unable to delete the contents of folder %v due to error %v", destinationQueriesJson, err)
		}
	})

	Describe("collecting server.log logs", func() {
		Context("all logs are present", func() {
			if err := ddcio.CopyDir(startLogDir, testLogDir); err != nil {
				simplelog.Errorf("test should fail as we had an error setting up the test directory: %v", err)
			}
			It("should collect all logs", func() {
				err := logCollector.RunCollectDremioServerLog()
				Expect(err).To(BeNil())
				Expect(filepath.Join(destinationDir, "server.log")).To(MatchFile(filepath.Join(testLogDir, "server.log")))
				Expect(filepath.Join(destinationDir, "server.out")).To(MatchFile(filepath.Join(testLogDir, "server.out")))
				Expect(filepath.Join(destinationDir, "server.2023-04-30.log")).To(ContainFileInGzip(filepath.Join(testLogDir, "archive", "server.2023-04-30.log")))
			})
		})

		Context("server.out is missing")
		Context("server.log is missing")
		Context("server.log archives are missing")

	})

	Describe("collecting acceleration.log logs", func() {
		Context("all logs are present", func() {
			It("should collect all logs", func() {

			})
		})
		Context("acceleration.log archives are missing")
		Context("acceleration.log is missing")

	})

	Describe("collecting reflection.log logs", func() {
		Context("all logs are present", func() {
			It("should collect all logs", func() {

			})
		})

		Context("reflection.log archives are missing")
		Context("reflection.log is missing")
	})
})
