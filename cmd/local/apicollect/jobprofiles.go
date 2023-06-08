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

// apicollect provides all the methods that collect via the API, this is a substantial part of the activities of DDC so it gets it's own package
package apicollect

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/dremio/dremio-diagnostic-collector/cmd/local/conf"
	"github.com/dremio/dremio-diagnostic-collector/cmd/local/queriesjson"
	"github.com/dremio/dremio-diagnostic-collector/cmd/local/restclient"
	"github.com/dremio/dremio-diagnostic-collector/cmd/simplelog"
	"github.com/dremio/dremio-diagnostic-collector/pkg/ddcio"
	"github.com/dremio/dremio-diagnostic-collector/pkg/threading"
)

func getNumberOfJobProfilesCollected(c *conf.CollectConf) (tried, collected int, err error) {
	files, err := os.ReadDir(c.QueriesOutDir())
	if err != nil {
		return 0, 0, err
	}
	queriesjsons := []string{}
	for _, file := range files {
		queriesjsons = append(queriesjsons, path.Join(c.QueriesOutDir(), file.Name()))
	}

	if len(queriesjsons) == 0 {
		simplelog.Warning("no queries.json files found. This is probably an executor, so we are skipping collection of Job Profiles")
		return
	}

	queriesrows := queriesjson.CollectQueriesJSON(queriesjsons)
	profilesToCollect := map[string]string{}

	slowplanqueriesrows := queriesjson.GetSlowPlanningJobs(queriesrows, c.JobProfilesNumSlowPlanning())
	queriesjson.AddRowsToSet(slowplanqueriesrows, profilesToCollect)

	slowexecqueriesrows := queriesjson.GetSlowExecJobs(queriesrows, c.JobProfilesNumSlowExec())
	queriesjson.AddRowsToSet(slowexecqueriesrows, profilesToCollect)

	highcostqueriesrows := queriesjson.GetHighCostJobs(queriesrows, c.JobProfilesNumHighQueryCost())
	queriesjson.AddRowsToSet(highcostqueriesrows, profilesToCollect)

	errorqueriesrows := queriesjson.GetRecentErrorJobs(queriesrows, c.JobProfilesNumRecentErrors())
	queriesjson.AddRowsToSet(errorqueriesrows, profilesToCollect)

	simplelog.Infof("jobProfilesNumSlowPlanning: %v", c.JobProfilesNumSlowPlanning())
	simplelog.Infof("jobProfilesNumSlowExec: %v", c.JobProfilesNumSlowExec())
	simplelog.Infof("jobProfilesNumHighQueryCost: %v", c.JobProfilesNumHighQueryCost())
	simplelog.Infof("jobProfilesNumRecentErrors: %v", c.JobProfilesNumRecentErrors())
	tried = len(profilesToCollect)
	if len(profilesToCollect) > 0 {
		simplelog.Infof("Downloading %v job profiles...", len(profilesToCollect))
		downloadThreadPool := threading.NewThreadPoolWithJobQueue(c.NumberThreads(), len(profilesToCollect))
		for key := range profilesToCollect {
			//because we are looping
			keyToDownload := key
			downloadThreadPool.AddJob(func() error {
				err := downloadJobProfile(c, keyToDownload)
				if err != nil {
					simplelog.Error(err.Error()) // Print instead of Error
				}
				return nil
			})
			collected++
		}
		if err := downloadThreadPool.ProcessAndWait(); err != nil {
			simplelog.Errorf("job profile download thread pool wait error %v", err)
		}
	} else {
		simplelog.Info("No job profiles to collect exiting...")
	}
	return tried, collected, nil
}

func RunCollectJobProfiles(c *conf.CollectConf) error {
	simplelog.Info("Collecting Job Profiles...")
	err := ValidateAPICredentials(c)
	if err != nil {
		return err
	}
	tried, collected, err := getNumberOfJobProfilesCollected(c)
	if err != nil {
		return err
	}
	simplelog.Infof("After eliminating duplicates we are tried to collect %v profiles", tried)
	simplelog.Infof("Downloaded %v job profiles", collected)
	return nil
}

func downloadJobProfile(c *conf.CollectConf, jobid string) error {
	apipath := "/apiv2/support/" + jobid + "/download"
	filename := jobid + ".zip"
	url := c.DremioEndpoint() + apipath
	headers := map[string]string{"Accept": "application/octet-stream"}
	body, err := restclient.APIRequest(url, c.DremioPATToken(), "POST", headers)
	if err != nil {
		return err
	}
	sb := string(body)
	jobProfileFile := path.Clean(path.Join(c.JobProfilesOutDir(), filename))
	file, err := os.Create(path.Clean(jobProfileFile))
	if err != nil {
		return fmt.Errorf("unable to create file %s due to error %v", filename, err)
	}
	defer ddcio.EnsureClose(filepath.Clean(jobProfileFile), file.Close)
	_, err = fmt.Fprint(file, sb)
	if err != nil {
		return fmt.Errorf("unable to create file %s due to error %v", filename, err)
	}
	return nil
}