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

// queriesjson package contains the logic for collecting queries.json information
package queriesjson

import (
	"testing"
)

func TestGetSlowExecJobs_empty(t *testing.T) {
	queriesrows_empty := []QueriesRow{}
	numSlowExecJobs_empty := 10
	slowexecqueriesrows_empty := GetSlowExecJobs(queriesrows_empty, numSlowExecJobs_empty)
	if len(slowexecqueriesrows_empty) != 0 {
		t.Errorf("Error")
	}
}

func TestGetSlowExecJobs_small(t *testing.T) {
	var row1 = new(QueriesRow)
	row1.QueryID = "Row1"
	row1.QueryType = "REST"
	row1.QueryCost = 500
	row1.ExecutionPlanningTime = 5
	row1.RunningTime = 100
	row1.Start = 11111
	row1.Outcome = "FAILED"

	var row2 = new(QueriesRow)
	row2.QueryID = "Row2"
	row2.QueryType = "ODBC"
	row2.QueryCost = 10
	row2.ExecutionPlanningTime = 500
	row2.RunningTime = 1
	row2.Start = 22222
	row2.Outcome = "FAILED"

	var row3 = new(QueriesRow)
	row3.QueryID = "Row3"
	row3.QueryType = "META"
	row3.QueryCost = 1000
	row3.ExecutionPlanningTime = 1
	row3.RunningTime = 50
	row3.Start = 33333
	row3.Outcome = "CANCELLED"

	var row4 = new(QueriesRow)
	row4.QueryID = "Row4"
	row4.QueryType = "REFLECTION"
	row4.QueryCost = 10
	row4.ExecutionPlanningTime = 100
	row4.RunningTime = 10
	row4.Start = 44444
	row4.Outcome = "FINISHED"

	var row5 = new(QueriesRow)
	row5.QueryID = "Row5"
	row5.QueryType = "UI"
	row5.QueryCost = 99
	row5.ExecutionPlanningTime = 1000
	row5.RunningTime = 25
	row5.Start = 55555
	row5.Outcome = "FAILED"
	queriesrows := []QueriesRow{*row1, *row2, *row3, *row4, *row5}

	// Slow Planning
	numSlowPlanningJobs := 10
	slowplanqueriesrows := GetSlowPlanningJobs(queriesrows, numSlowPlanningJobs)
	if len(slowplanqueriesrows) != 5 {
		t.Errorf("Error")
	}

	numSlowPlanningJobs = 3
	slowplanqueriesrows = GetSlowPlanningJobs(queriesrows, numSlowPlanningJobs)
	if len(slowplanqueriesrows) != 3 {
		t.Errorf("Error")
	}
	if slowplanqueriesrows[0] != *row5 {
		t.Errorf("Error")
	}
	if slowplanqueriesrows[1] != *row2 {
		t.Errorf("Error")
	}
	if slowplanqueriesrows[2] != *row4 {
		t.Errorf("Error")
	}

	// Slow Execution
	numSlowExecJobs := 10
	slowexecqueriesrows := GetSlowExecJobs(queriesrows, numSlowExecJobs)
	if len(slowexecqueriesrows) != 5 {
		t.Errorf("Error")
	}

	numSlowExecJobs = 3
	slowexecqueriesrows = GetSlowExecJobs(queriesrows, numSlowExecJobs)
	if len(slowexecqueriesrows) != 3 {
		t.Errorf("Error")
	}
	if slowexecqueriesrows[0] != *row1 {
		t.Errorf("Error")
	}
	if slowexecqueriesrows[1] != *row3 {
		t.Errorf("Error")
	}
	if slowexecqueriesrows[2] != *row5 {
		t.Errorf("Error")
	}

	// High Cost
	numHighQueryCostJobs := 10
	highcostqueriesrows := GetHighCostJobs(queriesrows, numHighQueryCostJobs)
	if len(highcostqueriesrows) != 5 {
		t.Errorf("Error")
	}

	numHighQueryCostJobs = 3
	highcostqueriesrows = GetHighCostJobs(queriesrows, numHighQueryCostJobs)
	if len(highcostqueriesrows) != 3 {
		t.Errorf("Error")
	}
	if highcostqueriesrows[0] != *row3 {
		t.Errorf("Error")
	}
	if highcostqueriesrows[1] != *row1 {
		t.Errorf("Error")
	}
	if highcostqueriesrows[2] != *row5 {
		t.Errorf("Error")
	}

	// Recent Errors
	numRecentErrorJobs := 10
	errorqueriesrows := GetRecentErrorJobs(queriesrows, numRecentErrorJobs)
	if len(errorqueriesrows) != 3 {
		t.Errorf("Error")
	}

	numRecentErrorJobs = 2
	errorqueriesrows = GetRecentErrorJobs(queriesrows, numRecentErrorJobs)
	if len(errorqueriesrows) != 2 {
		t.Errorf("Error")
	}
	if errorqueriesrows[0] != *row5 {
		t.Errorf("Error")
	}
	if errorqueriesrows[1] != *row2 {
		t.Errorf("Error")
	}
}

func TestParseLine(t *testing.T) {
	s := "123"
	actual, err := parseLine(s, 1)
	if err == nil {
		t.Errorf("ERROR")
	}
	expected := *new(QueriesRow)
	if expected != actual {
		t.Errorf("ERROR")
	}
}

func TestParseLine_Empty(t *testing.T) {
	s := ""
	actual, err := parseLine(s, 1)
	if err == nil {
		t.Errorf("ERROR")
	}
	expected := *new(QueriesRow)
	if expected != actual {
		t.Errorf("ERROR")
	}
}

func TestParseLine_ValidJson(t *testing.T) {
	s := `{
		"queryId":"1b9b9629-8289-b46c-c765-455d24da7800",
		"start":100,
		"outcome":"COMPLETED",
		"queryType":"METADATA_REFRESH",
		"queryCost":5.1003501E7,
		"planningTime":0,
		"executionPlanningTime":340,
		"runningTime":4785
	}`
	actual, err := parseLine(s, 1)
	if err != nil {
		t.Errorf("There should be no error here")
	}
	var expected = new(QueriesRow)
	expected.QueryID = "1b9b9629-8289-b46c-c765-455d24da7800"
	expected.QueryType = "METADATA_REFRESH"
	expected.QueryCost = 5.1003501e7
	expected.ExecutionPlanningTime = 340
	expected.RunningTime = 4785
	expected.Start = 100
	expected.Outcome = "COMPLETED"
	if *expected != actual {
		t.Errorf("ERROR")
	}
}

func TestParseLine_EmptyJson(t *testing.T) {
	s := "{}"
	actual, err := parseLine(s, 1)
	if err == nil {
		t.Errorf("ERROR")
	}
	expected := *new(QueriesRow)
	if expected != actual {
		t.Errorf("ERROR")
	}
}

func TestParseLine_ValidJsonWithMissingFields(t *testing.T) {
	s := `{
		"queryId":"1b9b9629-8289-b46c-c765-455d24da7800"
	}`
	actual, err := parseLine(s, 1)
	if err == nil {
		t.Errorf("There should be an error here")
	}
	expected := *new(QueriesRow)
	if expected != actual {
		t.Errorf("ERROR")
	}
}

func TestMin(t *testing.T) {
	actual := min(1, 2)
	expected := 1
	if expected != actual {
		t.Errorf("ERROR")
	}
	actual = min(2, 1)
	if expected != actual {
		t.Errorf("ERROR")
	}
	actual = min(1, 1)
	if expected != actual {
		t.Errorf("ERROR")
	}
}

func TestAddRowsToSet(t *testing.T) {
	// TODO
}

func TestReadJSONFile(t *testing.T) {
	filename := "../../testdata/queries/bad_queries.json"
	actual, err := ReadJSONFile(filename)
	if err != nil {
		t.Errorf("There should be an error here")
	}
	if len(actual) != 0 {
		t.Errorf("ERROR")
	}
}

func TestReadGzippedJSONFile(t *testing.T) {
	filename := "../../testdata/queries/queries.json.gz"
	actual, err := ReadGzFile(filename)
	if err != nil {
		t.Errorf("There should not be an error here")
	}
	if len(actual) != 3 {
		t.Errorf("The zipped queries.json should produce 3 entries")
	}
	expected_start_of_index0 := 100.0
	if actual[0].Start != expected_start_of_index0 {
		t.Errorf("ERROR")
	}
}
