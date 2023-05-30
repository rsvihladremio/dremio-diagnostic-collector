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

// httpclient library that allows easy http calls without a lot of fuss
package httpclient

import (
	"bufio"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGet(t *testing.T) {
	var method string
	var urlPassed string
	var headersPassed = make(map[string][]string)
	expected := "this comes back"
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		urlPassed = r.URL.String()
		headersPassed = r.Header
		fmt.Fprint(w, expected)
	}))
	defer svr.Close()
	var headers = make(map[string][]string)
	headers["abc"] = []string{"a", "b", "c"}
	response, err := Get(svr.URL, headers)
	assert.Nil(t, err)
	body, err := response.GetBody()
	assert.Nil(t, err)
	assert.Equal(t, body, []byte(expected))
	assert.Equal(t, "GET", method)
	assert.Equal(t, "/", urlPassed)
	//to show that the first letter is capitalized
	assert.Equal(t, headers["abc"], headersPassed["Abc"])
}

func TestGetReponseHeaders(t *testing.T) {
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("test", "1")
		w.Header().Add("test", "2")
		fmt.Fprint(w, "")
	}))
	defer svr.Close()
	response, err := Get(svr.URL, nil)
	assert.Nil(t, err)
	headers := response.GetHeaders()
	expected := make(map[string][]string)
	expected["Test"] = []string{"1", "2"}
	assert.Equal(t, expected["Test"], headers["Test"])

}

func TestGetText(t *testing.T) {
	expected := "this comes back"
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, expected)
	}))
	defer svr.Close()
	response, err := Get(svr.URL, nil)
	assert.Nil(t, err)
	body, err := response.GetBodyText()
	assert.Nil(t, err)
	assert.Equal(t, body, expected)
}

func TestGetJSON(t *testing.T) {
	expected := `{
        "a": 1,
        "b": "abc",
        "c": [1, 2, 3]
    }`
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, expected)
	}))
	defer svr.Close()
	response, err := Get(svr.URL, nil)
	assert.Nil(t, err)
	jsonBody := make(map[string]any)
	err = response.GetJSONObjectFromBody(&jsonBody)
	assert.Nil(t, err)
	expectedJSONBody := make(map[string]any)
	expectedJSONBody["a"] = 1.0
	expectedJSONBody["b"] = "abc"
	expectedJSONBody["c"] = []interface{}{1.0, 2.0, 3.0}
	assert.Equal(t, jsonBody, expectedJSONBody)
}

func TestDelete(t *testing.T) {
	var method string
	var urlPassed string
	var headersPassed = make(map[string][]string)
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		urlPassed = r.URL.String()
		headersPassed = r.Header
		fmt.Fprint(w, "")

	}))
	defer svr.Close()
	var headers = make(map[string][]string)
	headers["abc"] = []string{"a", "b", "c"}
	response, err := Delete(svr.URL, headers)
	assert.Nil(t, err)
	code, status := response.GetStatus()
	assert.Equal(t, 200, code)
	assert.Equal(t, "200 OK", status)
	assert.Equal(t, "DELETE", method)
	assert.Equal(t, "/", urlPassed)
	//to show that the first letter is capitalized
	assert.Equal(t, headers["abc"], headersPassed["Abc"])
}

func TestPost(t *testing.T) {
	var method string
	var urlPassed string
	headersPassed := make(map[string][]string)
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		urlPassed = r.URL.String()
		headersPassed = r.Header
		reader := r.Body
		var builder strings.Builder
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			text := scanner.Text()
			_, err := builder.WriteString(text)
			if err != nil {
				assert.Nil(t, err)
			}
		}
		//echo back out the text
		fmt.Fprint(w, builder.String())
	}))
	defer svr.Close()
	expected := []byte("this is my text")
	headers := make(map[string][]string)
	headers["abc"] = []string{"a", "b", "c"}
	response, err := Post(svr.URL, headers, expected)
	assert.Nil(t, err)
	code, status := response.GetStatus()
	assert.Equal(t, 200, code)
	assert.Equal(t, "200 OK", status)
	body, err := response.GetBody()
	assert.Nil(t, err)
	assert.Equal(t, body, expected)
	assert.Equal(t, "POST", method)
	assert.Equal(t, "/", urlPassed)
	//headers are title cased when going through the http request
	assert.Equal(t, headers["abc"], headersPassed["Abc"])
}

func TestPostJSON(t *testing.T) {
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reader := r.Body
		var builder strings.Builder
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			text := scanner.Text()
			_, err := builder.WriteString(text)
			if err != nil {
				assert.Nil(t, err)
			}
		}
		//echo back out the text
		fmt.Fprint(w, builder.String())
	}))
	defer svr.Close()
	expected := make(map[string]string)
	expected["a"] = "123"
	expected["b"] = "456"
	response, err := PostJSON(svr.URL, nil, expected)
	assert.Nil(t, err)
	code, status := response.GetStatus()
	assert.Equal(t, 200, code)
	assert.Equal(t, "200 OK", status)
	body := make(map[string]string)
	err = response.GetJSONObjectFromBody(&body)
	assert.Nil(t, err)
	assert.Equal(t, body, expected)
}

func TestPut(t *testing.T) {
	var method string
	var urlPassed string
	headersPassed := make(map[string][]string)
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		urlPassed = r.URL.String()
		headersPassed = r.Header
		reader := r.Body
		var builder strings.Builder
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			text := scanner.Text()
			_, err := builder.WriteString(text)
			if err != nil {
				assert.Nil(t, err)
			}
		}
		//echo back out the text
		fmt.Fprint(w, builder.String())
	}))
	defer svr.Close()
	expected := []byte("this is my text")
	headers := make(map[string][]string)
	headers["abc"] = []string{"a", "b", "c"}
	response, err := Put(svr.URL, headers, expected)
	assert.Nil(t, err)
	code, status := response.GetStatus()
	assert.Equal(t, 200, code)
	assert.Equal(t, "200 OK", status)
	body, err := response.GetBody()
	assert.Nil(t, err)
	assert.Equal(t, body, expected)
	assert.Equal(t, "PUT", method)
	assert.Equal(t, "/", urlPassed)
	//headers are title cased when going through the http request
	assert.Equal(t, headers["abc"], headersPassed["Abc"])
}

func TestPutJSON(t *testing.T) {
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reader := r.Body
		var builder strings.Builder
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			text := scanner.Text()
			_, err := builder.WriteString(text)
			if err != nil {
				assert.Nil(t, err)
			}
		}
		//echo back out the text
		fmt.Fprint(w, builder.String())
	}))
	defer svr.Close()
	expected := make(map[string]string)
	expected["a"] = "123"
	expected["b"] = "456"
	response, err := PutJSON(svr.URL, nil, expected)
	assert.Nil(t, err)
	code, status := response.GetStatus()
	assert.Equal(t, 200, code)
	assert.Equal(t, "200 OK", status)
	body := make(map[string]string)
	err = response.GetJSONObjectFromBody(&body)
	assert.Nil(t, err)
	assert.Equal(t, body, expected)
}
