package cmd

import (
	"testing"
)

// func TestGetSlowExecJobs_empty(t *testing.T) {
// 	queriesrows_empty := []QueriesRow{}
// 	numSlowExecJobs_empty := 10
// 	slowexecqueriesrows_empty := getSlowExecJobs(queriesrows_empty, numSlowExecJobs_empty)
// 	log.Println(slowexecqueriesrows_empty)
// 	if len(slowexecqueriesrows_empty) != 0 {
// 		t.Errorf("Error")
// 	}
// }

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
	slowplanqueriesrows := getSlowPlanningJobs(queriesrows, numSlowPlanningJobs)
	if len(slowplanqueriesrows) != 5 {
		t.Errorf("Error")
	}

	numSlowPlanningJobs = 3
	slowplanqueriesrows = getSlowPlanningJobs(queriesrows, numSlowPlanningJobs)
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
	slowexecqueriesrows := getSlowExecJobs(queriesrows, numSlowExecJobs)
	if len(slowexecqueriesrows) != 5 {
		t.Errorf("Error")
	}

	numSlowExecJobs = 3
	slowexecqueriesrows = getSlowExecJobs(queriesrows, numSlowExecJobs)
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
	highcostqueriesrows := getHighCostJobs(queriesrows, numHighQueryCostJobs)
	if len(highcostqueriesrows) != 5 {
		t.Errorf("Error")
	}

	numHighQueryCostJobs = 3
	highcostqueriesrows = getHighCostJobs(queriesrows, numHighQueryCostJobs)
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
	errorqueriesrows := getRecentErrorJobs(queriesrows, numRecentErrorJobs)
	if len(errorqueriesrows) != 3 {
		t.Errorf("Error")
	}

	numRecentErrorJobs = 2
	errorqueriesrows = getRecentErrorJobs(queriesrows, numRecentErrorJobs)
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
