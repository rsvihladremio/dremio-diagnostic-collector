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
package kubernetes

import (
	"fmt"
	"reflect"
	"testing"
)

type MockCli struct {
	Calls          [][]string
	StoredResponse []string
	StoredErrors   []error
}

func (m *MockCli) Execute(args ...string) (out string, err error) {
	m.Calls = append(m.Calls, args)
	length := len(m.Calls)
	return m.StoredResponse[length-1], m.StoredErrors[length-1]
}

func TestKubectlExec(t *testing.T) {
	namespace := "testns"
	podName := "pod"
	cli := &MockCli{
		StoredResponse: []string{"success"},
		StoredErrors:   []error{nil},
	}
	k := KubectlK8sActions{
		cli: cli,
	}
	out, err := k.PodExecute(podName, namespace, "ls", "-l")
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
	expectedCall := []string{"kubectl", "exec", "-it", "-n", namespace, podName, "ls", "-l"}
	if !reflect.DeepEqual(calls[0], expectedCall) {
		t.Errorf("expected %v call but got %v", expectedCall, calls[0])
	}
}
func TestKubectlSearch(t *testing.T) {

	namespace := "testns"
	labelName := "myPods"
	cli := &MockCli{
		StoredResponse: []string{"pod1\npod2\npod3\n"},
		StoredErrors:   []error{nil},
	}
	k := KubectlK8sActions{
		cli: cli,
	}
	podNames, err := k.PodSearch(labelName, namespace)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	expectedPods := []string{"pod1", "pod2", "pod3"}
	if !reflect.DeepEqual(podNames, expectedPods) {
		t.Errorf("expected %v call but got %v", expectedPods, podNames)
	}
	calls := cli.Calls
	if len(calls) != 1 {
		t.Errorf("expected 1 call but got %v", len(calls))
	}
	expectedCall := []string{"kubectl", "get", "-n", namespace, "-l", labelName, "-o", "name"}
	if !reflect.DeepEqual(calls[0], expectedCall) {
		t.Errorf("expected %v call but got %v", expectedCall, calls[0])
	}
}

func TestKubectCopyFrom(t *testing.T) {
	namespace := "testns"
	podName := "pod"
	source := "/podroot/test.log"
	destination := "/mydir/test.log"
	cli := &MockCli{
		StoredResponse: []string{"success"},
		StoredErrors:   []error{nil},
	}
	k := KubectlK8sActions{
		cli: cli,
	}
	out, err := k.PodCopyFromFile(podName, namespace, source, destination)
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
	expectedCall := []string{"kubectl", "cp", "-n", namespace, fmt.Sprintf("%v:%v", podName, source), destination}
	if !reflect.DeepEqual(calls[0], expectedCall) {
		t.Errorf("expected %v call but got %v", expectedCall, calls[0])
	}
}
