/*
   Copyright 2022 Ryan SVIHLA

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
//kubernetes package provides access to log collections on k8s
package kubernetes

import (
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/rsvihladremio/dremio-diagnostic-collector/tests"
)

func TestKubectlExec(t *testing.T) {
	namespace := "testns"
	podName := "pod"
	cli := &tests.MockCli{
		StoredResponse: []string{"success"},
		StoredErrors:   []error{nil},
	}
	k := KubectlK8sActions{
		cli:                  cli,
		kubectlPath:          "kubectl",
		coordinatorContainer: "dremio-master-coordinator",
		executorContainer:    "dremio-executor",
	}
	out, err := k.HostExecute(fmt.Sprintf("%v.%v", namespace, podName), true, "ls", "-l")
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	if out != "success" {
		t.Errorf("expected success but got %v", out)
	}
	calls := cli.Calls
	if len(calls) != 1 {
		t.Errorf("expected 1 call but got %v", len(calls))
	}
	expectedCall := []string{"kubectl", "exec", "-n", namespace, "-c", "dremio-master-coordinator", podName, "--", "ls", "-l"}
	if !reflect.DeepEqual(calls[0], expectedCall) {
		t.Errorf("expected %v call but got %v", expectedCall, calls[0])
	}
}

func TestKubectlSearch(t *testing.T) {

	namespace := "testns"
	labelName := "myPods"
	cli := &tests.MockCli{
		StoredResponse: []string{"pod/pod1\npod/pod2\npod/pod3\n"},
		StoredErrors:   []error{nil},
	}
	k := KubectlK8sActions{
		cli:         cli,
		kubectlPath: "kubectl",
	}
	podNames, err := k.FindHosts(fmt.Sprintf("%v:%v", namespace, labelName))
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	//we need to pass the namespace for later commands that may consume this
	// period is handy because it is an illegal character in a kubernetes name and so
	// can act as a separator
	expectedPods := []string{"testns.pod1", "testns.pod2", "testns.pod3"}
	if !reflect.DeepEqual(podNames, expectedPods) {
		t.Errorf("expected %v call but got %v", expectedPods, podNames)
	}
	calls := cli.Calls
	if len(calls) != 1 {
		t.Errorf("expected 1 call but got %v", len(calls))
	}
	expectedCall := []string{"kubectl", "get", "pods", "-n", namespace, "-l", labelName, "-o", "name"}
	if !reflect.DeepEqual(calls[0], expectedCall) {
		t.Errorf("expected %v call but got %v", expectedCall, calls[0])
	}
}

func TestKubectCopyFrom(t *testing.T) {
	namespace := "testns"
	podName := "pod"
	source := "/podroot/test.log"
	destination := "/mydir/test.log"
	cli := &tests.MockCli{
		StoredResponse: []string{"success"},
		StoredErrors:   []error{nil},
	}
	k := KubectlK8sActions{
		cli:                  cli,
		kubectlPath:          "kubectl",
		coordinatorContainer: "dremio-master-coordinator",
		executorContainer:    "dremio-executor",
	}
	out, err := k.CopyFromHost(fmt.Sprintf("%v.%v", namespace, podName), true, source, destination)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	if out != "success" {
		t.Errorf("expected success but got %v", out)
	}
	calls := cli.Calls
	if len(calls) != 1 {
		t.Errorf("expected 1 call but got %v", len(calls))
	}
	expectedCall := []string{"kubectl", "cp", "-n", namespace, "-c", "dremio-master-coordinator", fmt.Sprintf("%v:%v", podName, source), destination}
	if !reflect.DeepEqual(calls[0], expectedCall) {
		t.Errorf("expected %v call but got %v", expectedCall, calls[0])
	}
}

func TestKubectCopyFromWindowsHost(t *testing.T) {
	namespace := "testns"
	podName := "pod"
	source := filepath.Join("podroot", "test.log")
	destination := filepath.Join("C:", "mydir", "test.log")
	cli := &tests.MockCli{
		StoredResponse: []string{"success"},
		StoredErrors:   []error{nil},
	}
	k := KubectlK8sActions{
		cli:                  cli,
		kubectlPath:          "kubectl",
		coordinatorContainer: "dremio-master-coordinator",
		executorContainer:    "dremio-executor",
	}
	out, err := k.CopyFromHost(fmt.Sprintf("%v.%v", namespace, podName), true, source, destination)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	if out != "success" {
		t.Errorf("expected success but got %v", out)
	}
	calls := cli.Calls
	if len(calls) != 1 {
		t.Errorf("expected 1 call but got %v", len(calls))
	}
	//we remove the C: due to issue found in https://github.com/kubernetes/kubernetes/issues/77310"
	expectedDestination := filepath.Join(string(filepath.Separator), "mydir", "test.log")
	expectedCall := []string{"kubectl", "cp", "-n", namespace, "-c", "dremio-master-coordinator", fmt.Sprintf("%v:%v", podName, source), expectedDestination}
	if !reflect.DeepEqual(calls[0], expectedCall) {
		t.Errorf("expected %v call but got %v", expectedCall, calls[0])
	}
}
