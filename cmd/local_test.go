/*
Copyright 2023 Dremio
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/ory/dockertest/v3"
	dc "github.com/ory/dockertest/v3/docker"
)

type AuthResponse struct {
	Token string `json:"token"`
}

type AuthRequest struct {
	Username string `json:"userName"`
	Password string `json:"password"`
}

var dremioTestPort string

// TestMain setups up a docker runtime and we use this to spin up dremio https://github.com/ory/dockertest
func TestMain(m *testing.M) {
	// uses a sensible default on windows (tcp/http) and linux/osx (socket)
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not construct pool: %s", err)
	}

	// uses pool to try to connect to Docker
	err = pool.Client.Ping()
	if err != nil {
		log.Fatalf("Could not connect to Docker: %s", err)
	}

	// pulls an image, creates a container based on it and runs it
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "dremio/dremio-oss",
		Tag:        "24.0",
		//Env:        []string{},
	}, func(config *dc.HostConfig) {
		// set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = dc.RestartPolicy{
			Name: "no",
		}
		pwd, err := os.Getwd()
		if err != nil {
			log.Fatalf("failed to get working directory: %s", err)
		}
		config.Mounts = []dc.HostMount{
			{
				Target: "/opt/dremio/conf/dremio.conf",
				Source: fmt.Sprintf("%s/testfiles/conf/dremio.conf", pwd),
				Type:   "bind",
			},
		}

	})
	dremioTestPort = resource.GetPort("9047/tcp")

	resource.Expire(120)
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	requestURL := fmt.Sprintf("http://localhost:%v", dremioTestPort)
	dremioEndpoint = requestURL

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	if err := pool.Retry(func() error {
		res, err := http.Get(requestURL)
		if err != nil {
			log.Printf("error making http request: %s\n", err)
			return err
		}
		expectedCode := 200
		if res.StatusCode != expectedCode {
			return fmt.Errorf("expected status code %v but instead got %v. Dremio is not ready", expectedCode, res.StatusCode)
		}
		dremioUsername = "dremio"
		authRequest := &AuthRequest{
			Username: "dremio",
			Password: "dremio123",
		}
		body, err := json.Marshal(authRequest)
		if err != nil {
			return fmt.Errorf("Error marshaling JSON: %v", err)
		}
		res, err = http.Post(fmt.Sprintf("http://localhost:%v/apiv2/login", dremioTestPort), "application/json", bytes.NewBuffer(body))
		if err != nil {
			log.Fatalf("error logging in to get token : %s\n", err)
		}
		defer res.Body.Close()
		if res.StatusCode != expectedCode {
			if text, err := io.ReadAll(res.Body); err != nil {
				log.Fatalf("fatal attempt to decode body from dremio auth %v and unable to read body for debugging", err)
			} else {
				log.Printf("body was %s", string(text))
			}
			log.Fatalf("expected status code %v but instead got %v with message %v. Unable to get dremio PAT", expectedCode, res.StatusCode, res.Status)
		}
		var authResponse AuthResponse
		err = json.NewDecoder(res.Body).Decode(&authResponse)
		if err != nil {
			if text, err := io.ReadAll(res.Body); err != nil {
				log.Fatalf("fatal attempt to decode body from dremio auth %v and unable to read body for debugging", err)
			} else {
				log.Printf("body was %s", string(text))
			}
			log.Fatalf("fatal attempt to decode body from dremio auth %v", err)
		}
		dremioPATToken = authResponse.Token
		return nil
	}); err != nil {
		log.Fatalf("Could not connect to dremio: %s", err)
	}

	code := m.Run()

	// You can't defer this because os.Exit doesn't care for defer
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}
	os.Exit(code)
}

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
	jobid := "1bb5803c-5a67-d548-2547-bd180cd2fe00"
	err := downloadJobProfile(jobid)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
}

func TestValidateApiCredentials(t *testing.T) {
	err := validateApiCredentials()
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
}

func TestValidateCollectJobProfiles(t *testing.T) {
	err := collectJobProfiles()
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
}