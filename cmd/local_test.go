package cmd

import (
	"testing"
)

func TestCreateAllDirs(t *testing.T) {

	err := createAllDirs()
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
}

func TestCollectWlm(t *testing.T) {

	err := collectWlm()
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
}

func TestCollectKVReport(t *testing.T) {

	err := collectKvReport()
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
}

func TestCollectDremioSystemTables(t *testing.T) {
	err := collectDremioSystemTables()
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
}

func TestDownloadJobProfile(t *testing.T) {
	// Requires
	jobid := "1bb5803c-5a67-d548-2547-bd180cd2fe00"
	err := downloadJobProfile(jobid)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
}

func TestValidateApiCredentials(t *testing.T) {
	// dremioendpoint := "https://hc.fieldnarwhal.com/"
	// pat := "CUO+KPLhQQ+FYjlFYxi2sp+CzzX2LFxNQeR/w42RmzxaIqK4T/TdncthpKX39w=="
	err := validateApiCredentials()
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
}
