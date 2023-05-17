//  Copyright 2023 Dremio Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// package assert has a simple assertion library
package assert

import (
	"reflect"
	"strings"
	"testing"
)

// Equal checks if two values are equal
func Equal(t *testing.T, a interface{}, b interface{}) {
	if !reflect.DeepEqual(a, b) {
		t.Errorf("Received %v (type %v), expected %v (type %v)", a, reflect.TypeOf(a), b, reflect.TypeOf(b))
	}
}

// NotEqual checks if two values are not equal
func NotEqual(t *testing.T, a interface{}, b interface{}) {
	if reflect.DeepEqual(a, b) {
		t.Errorf("Received %v (type %v), expected %v (type %v) to not be equal but they were", a, reflect.TypeOf(a), b, reflect.TypeOf(b))
	}
}

// True checks if a value is true
func True(t *testing.T, a interface{}) {
	if a != true {
		t.Errorf("Received %v (type %v), expected true", a, reflect.TypeOf(a))
	}
}

// Nil checks if a value is nil
func Nil(t *testing.T, a interface{}) {
	if a != nil {
		t.Errorf("Received %v (type %v), expected nil", a, reflect.TypeOf(a))
	}
}

// Error checks if an expected error occurred
func Error(t *testing.T, err error) {
	if err == nil {
		t.Error("expected an error, but got nil")
	}
}

// NoError checks that no error occurred
func NoError(t *testing.T, err error) {
	if err != nil {
		t.Errorf("expected no error, but got %v", err)
	}
}

// Contains checks if a string is contained within another string.
func Contains(t *testing.T, haystack, needle string) {
	if !strings.Contains(haystack, needle) {
		t.Errorf("expected %q to contain %q", haystack, needle)
	}
}
