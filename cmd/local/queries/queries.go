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

// queries package contains the logic for collecting queries.json information
package queries

import (
	"bufio"
	"compress/gzip"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
)

type QueriesRow struct {
	QueryID string `json:"queryId"`
	// Context       string `json:"context"`
	QueryText string  `json:"queryText"`
	Start     float64 `json:"start"`
	// Finish        int64  `json:"finish"`
	Outcome string `json:"outcome"`
	// OutcomeReason string `json:"outcomeReason"`
	// Username      string `json:"username"`
	// InputRecords  int    `json:"inputRecords"`
	// InputBytes    int    `json:"inputBytes"`
	// OutputRecords int    `json:"outputRecords"`
	// OutputBytes   int    `json:"outputBytes"`
	// RequestType   string `json:"requestType"`
	QueryType string `json:"queryType"`
	// ParentsList             []any  `json:"parentsList"`
	// Accelerated             bool   `json:"accelerated"`
	// ReflectionRelationships []any  `json:"reflectionRelationships"`
	QueryCost float64 `json:"queryCost"`
	// QueueName               string `json:"queueName"`
	// PoolWaitTime            int    `json:"poolWaitTime"`
	// PendingTime             int    `json:"pendingTime"`
	// MetadataRetrievalTime   int    `json:"metadataRetrievalTime"`
	// PlanningTime            int    `json:"planningTime"`
	// EngineStartTime         int    `json:"engineStartTime"`
	// QueuedTime              int    `json:"queuedTime"`
	ExecutionPlanningTime float64 `json:"executionPlanningTime"`
	// StartingTime            int    `json:"startingTime"`
	RunningTime float64 `json:"runningTime"`
	// EngineName              string `json:"engineName"`
	// AttemptCount            int    `json:"attemptCount"`
	// Submitted               int64  `json:"submitted"`
	// MetadataRetrieval       int64  `json:"metadataRetrieval"`
	// PlanningStart           int    `json:"planningStart"`
	// QueryEnqueued           int64  `json:"queryEnqueued"`
	// EngineStart             int64  `json:"engineStart"`
	// ExecutionPlanningStart  int64  `json:"executionPlanningStart"`
	// ExecutionStart          int64  `json:"executionStart"`
	// ScannedDatasets         []any  `json:"scannedDatasets"`
	// ExecutionNodes          []struct {
	// 	NodeID     string `json:"nodeId"`
	// 	Hostname   string `json:"hostname"`
	// 	MaxMemUsed int    `json:"maxMemUsed"`
	// } `json:"executionNodes"`
	// ExecutionCPUTimeNs   int   `json:"executionCpuTimeNs"`
	// SetupTimeNs          int   `json:"setupTimeNs"`
	// WaitTimeNs           int   `json:"waitTimeNs"`
	// MemoryAllocated      int   `json:"memoryAllocated"`
	// StartingStart        int64 `json:"startingStart"`
	// IsTruncatedQueryText bool  `json:"isTruncatedQueryText"`
}

func ReadGzFile(filename string) ([]QueriesRow, error) {
	queriesrows := []QueriesRow{}
	file, err := os.Open(filename)
	if err != nil {
		return queriesrows, err
	}
	defer errCheck(file.Close)

	fz, err := gzip.NewReader(file)
	if err != nil {
		return queriesrows, err
	}
	defer errCheck(fz.Close)

	scanner := *bufio.NewScanner(fz)

	i := 0
	for scanner.Scan() {
		line := scanner.Text()
		row, err := parseLine(line, i)
		if err != nil {
			fmt.Println(err.Error() + `: ` + filename)
		}
		queriesrows = append(queriesrows, row)
		i++
	}
	return queriesrows, err
}

func ReadJsonFile(filename string) ([]QueriesRow, error) {
	// Source: https://gist.github.com/kendellfab/7417164
	queriesrows := []QueriesRow{}
	file, err := os.Open(filename)
	if err != nil {
		fmt.Println(err.Error() + `: ` + filename)
		return queriesrows, err
	}
	defer errCheck(file.Close)
	scanner := *bufio.NewScanner(file)
	i := 0
	for scanner.Scan() {
		line := scanner.Text()
		row, err := parseLine(line, i)
		if err != nil {
			fmt.Println(err.Error() + `: ` + filename)
		}
		queriesrows = append(queriesrows, row)
		i++
	}
	return queriesrows, err
}

func parseLine(line string, i int) (QueriesRow, error) {
	dat := make(map[string]interface{})
	err := json.Unmarshal([]byte(line), &dat)
	if err != nil {
		log.Println(err)
		log.Println("queries.json line #", i, line)
	}
	var row = new(QueriesRow)
	row.QueryID = dat["queryId"].(string)
	row.QueryType = dat["queryType"].(string)
	row.QueryCost = dat["queryCost"].(float64)
	row.ExecutionPlanningTime = dat["executionPlanningTime"].(float64)
	row.RunningTime = dat["runningTime"].(float64)
	row.Start = dat["start"].(float64)
	row.Outcome = dat["outcome"].(string)
	queriesrow := *row
	return queriesrow, err
}

func GetRecentErrorJobs(queriesrows []QueriesRow, limit int) []QueriesRow {
	errorqueriesrows := []QueriesRow{}

	for i := range queriesrows {
		if queriesrows[i].Outcome == "FAILED" {
			errorqueriesrows = append(errorqueriesrows, queriesrows[i])
		}
	}

	totalrows := len(errorqueriesrows)
	sort.Slice(errorqueriesrows, func(i, j int) bool {
		return errorqueriesrows[i].Start > errorqueriesrows[j].Start
	})
	return errorqueriesrows[:min(totalrows, limit)]
}

func GetSlowExecJobs(queriesrows []QueriesRow, limit int) []QueriesRow {

	totalrows := len(queriesrows)
	sort.Slice(queriesrows, func(i, j int) bool {
		return queriesrows[i].RunningTime > queriesrows[j].RunningTime
	})
	return queriesrows[:min(totalrows, limit)]
}

func GetSlowPlanningJobs(queriesrows []QueriesRow, limit int) []QueriesRow {

	totalrows := len(queriesrows)
	sort.Slice(queriesrows, func(i, j int) bool {
		return queriesrows[i].ExecutionPlanningTime > queriesrows[j].ExecutionPlanningTime
	})
	return queriesrows[:min(totalrows, limit)]
}

func GetHighCostJobs(queriesrows []QueriesRow, limit int) []QueriesRow {

	totalrows := len(queriesrows)
	sort.Slice(queriesrows, func(i, j int) bool {
		return queriesrows[i].QueryCost > queriesrows[j].QueryCost
	})

	return queriesrows[:min(totalrows, limit)]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func AddRowsToSet(queriesrows []QueriesRow, profilesToCollect map[string]string) map[string]string {
	for _, row := range queriesrows {
		jobid := row.QueryID
		profilesToCollect[jobid] = ""
	}
	return profilesToCollect
}

func CollectQueriesJson(queriesjsons []string) []QueriesRow {

	queriesrows := []QueriesRow{}
	for _, queriesjson := range queriesjsons {
		log.Println("Attempting to open queries.json file", queriesjson)
		rows := []QueriesRow{}
		var err error

		if strings.HasSuffix(queriesjson, ".gz") {
			rows, err = ReadGzFile(queriesjson)
			if err != nil {
				log.Println("ERROR", err)
				continue
			}
			queriesrows = append(queriesrows, rows...)
		} else if strings.HasSuffix(queriesjson, ".json") {
			rows, err = ReadJsonFile(queriesjson)
			if err != nil {
				log.Println("ERROR", err)
				continue
			}
			queriesrows = append(queriesrows, rows...)
		} else {
			log.Println("ERROR", "File is neither JSON or GZIP format.")
		}
		log.Println("Found", strconv.Itoa(len(rows)), "new rows in", queriesjson)
	}
	log.Println("Collected a total of", strconv.Itoa(len(queriesrows)), "rows of queries.json")
	return queriesrows
}

func writeToCSV(queriesrows []QueriesRow, filter string, limit int) {
	// Can be used for testing or debugging

	file, err := os.Create("job_ids_go_" + filter + strconv.Itoa(limit) + ".csv")
	if err != nil {
		panic(err)
	}
	w := csv.NewWriter(file)
	err = w.Write([]string{"job_id"})
	if err != nil {
		panic(err)
	}
	sortingmetric := -1
	for _, row := range queriesrows {
		if filter == "slowplanqueriesrows" {
			sortingmetric = int(row.ExecutionPlanningTime)
		} else if filter == "slowexecqueriesrows" {
			sortingmetric = int(row.RunningTime)
		} else if filter == "highcostqueriesrows" {
			sortingmetric = int(row.QueryCost)
		} else if filter == "errorqueriesrows" {
			sortingmetric = int(row.Start)
		} else {
			log.Println("unknown filter", filter)
			break
		}
		err := w.Write([]string{fmt.Sprintf("%v", row.QueryID), fmt.Sprintf("%d", sortingmetric)})
		if err != nil {
			panic(err)
		}
	}
	w.Flush()

}

func errCheck(f func() error) {
	err := f()
	if err != nil {
		fmt.Println("Received error:", err)
	}
}
