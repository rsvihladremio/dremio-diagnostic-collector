package cmd

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
	fi, err := os.Open(filename)
	if err != nil {
		return queriesrows, err
	}
	defer fi.Close()

	fz, err := gzip.NewReader(fi)
	if err != nil {
		return queriesrows, err
	}
	defer fz.Close()

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
	inFile, err := os.Open(filename)
	if err != nil {
		fmt.Println(err.Error() + `: ` + filename)
		return queriesrows, err
	}
	defer inFile.Close()
	scanner := *bufio.NewScanner(inFile)
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

func getRecentErrorJobs(queriesrows []QueriesRow, limit int) []QueriesRow {
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

func getSlowExecJobs(queriesrows []QueriesRow, limit int) []QueriesRow {

	totalrows := len(queriesrows)
	sort.Slice(queriesrows, func(i, j int) bool {
		return queriesrows[i].RunningTime > queriesrows[j].RunningTime
	})
	return queriesrows[:min(totalrows, limit)]
}

func getSlowPlanningJobs(queriesrows []QueriesRow, limit int) []QueriesRow {

	totalrows := len(queriesrows)
	sort.Slice(queriesrows, func(i, j int) bool {
		return queriesrows[i].ExecutionPlanningTime > queriesrows[j].ExecutionPlanningTime
	})
	return queriesrows[:min(totalrows, limit)]
}

func getHighCostJobs(queriesrows []QueriesRow, limit int) []QueriesRow {

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

func addRowsToSet(queriesrows []QueriesRow, profilesToCollect map[string]string) map[string]string {
	for _, row := range queriesrows {
		jobid := row.QueryID
		profilesToCollect[jobid] = ""
	}
	return profilesToCollect
}

func collectQueriesJson(queriesjsons []string) []QueriesRow {

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

	file, _ := os.Create("job_ids_go_" + filter + strconv.Itoa(limit) + ".csv")
	w := csv.NewWriter(file)
	w.Write([]string{"job_id"})
	for _, row := range queriesrows {
		if filter == "slowplanqueriesrows" {
			err := w.Write([]string{fmt.Sprintf("%v", row.QueryID), fmt.Sprintf("%d", int(row.ExecutionPlanningTime))})
			if err != nil {
				panic(err)
			}
		}
		if filter == "slowexecqueriesrows" {
			err := w.Write([]string{fmt.Sprintf("%v", row.QueryID), fmt.Sprintf("%d", int(row.RunningTime))})
			if err != nil {
				panic(err)
			}
		}
		if filter == "highcostqueriesrows" {
			err := w.Write([]string{fmt.Sprintf("%v", row.QueryID), fmt.Sprintf("%d", int(row.QueryCost))})
			if err != nil {
				panic(err)
			}
		}
		if filter == "errorqueriesrows" {
			err := w.Write([]string{fmt.Sprintf("%v", row.QueryID), fmt.Sprintf("%d", int(row.Start))})
			if err != nil {
				panic(err)
			}
		}
	}
	w.Flush()

}
