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

// http client library that allows easy http calls without a lot of fuss
package httpclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

var client *http.Client

// init sets all timeouts to 30 seconds and 10 connections for idle
func init() {
	tr := &http.Transport{
		MaxIdleConns:          10,
		IdleConnTimeout:       30 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		TLSHandshakeTimeout:   30 * time.Second,
		ExpectContinueTimeout: 30 * time.Second,
	}
	client = &http.Client{
		Transport: tr,
		Timeout:   30 * time.Second,
	}
}

// Response is just an easy mode wrapper around net/http.Response
type Response struct {
	raw *http.Response
}

// GetBody retrieves the body as a byte array. This is usually what you
// want to use if there is a format besides json to handle (such as xml or yaml)
func (r *Response) GetBody() ([]byte, error) {
	body, err := io.ReadAll(r.raw.Body)
	if err != nil {
		return []byte{}, fmt.Errorf("unable to read body due to error %v", err)
	}
	defer func() {
		err = r.raw.Body.Close()
		if err != nil {
			log.Printf("WARN: unable to close body of http request due to error '%v'", err)
		}
	}()
	return body, nil
}

// GetBodyText retrieves the body as a string
func (r *Response) GetBodyText() (string, error) {
	body, err := r.GetBody()
	return string(body), err
}

// GetJSONObjectFromBody requires you pass in a pointer to an
// object so that it can be filled with values from the json response
// this also means the response must be valid json
func (r *Response) GetJSONObjectFromBody(v any) error {
	body, err := r.GetBody()
	if err != nil {
		return fmt.Errorf("unable to retrieve body from request due to error '%v'", err)
	}
	err = json.Unmarshal(body, v)
	if err != nil {
		return fmt.Errorf("unable to convert body into json due to error '%v'", err)
	}
	return err
}

// GetStatus returns status code and the status text
func (r *Response) GetStatus() (code int, message string) {
	return r.raw.StatusCode, r.raw.Status
}

// GetHeaders returns the header received from the http response
func (r *Response) GetHeaders() (headers map[string][]string) {
	return r.raw.Header
}

// Get submits a http request with a set of headers
func Get(url string, headers map[string][]string) (Response, error) {
	// body is not generally accepted on GET so we are going to skip it
	return do("GET", url, headers, nil)
}

// Put submits a http request with a set of headers and a body, headers and body are optional
func Put(url string, headers map[string][]string, body []byte) (Response, error) {
	// bytes.Reader does not have a close interface so we can just allocate it and let GC clean it up
	reader := bytes.NewReader(body)
	return do("PUT", url, headers, reader)
}

// PutJSON submits a http request with a set of headers and a body as a go object which is then converted into json
func PutJSON(url string, headers map[string][]string, obj any) (Response, error) {
	body, err := json.Marshal(obj)
	if err != nil {
		return Response{}, fmt.Errorf("unable to marshal into json the object %v due to error '%v'", obj, err)
	}
	// bytes.Reader does not have a close interface so we can just allocate it and let GC clean it up
	reader := bytes.NewReader(body)
	return do("PUT", url, headers, reader)
}

// Delete submits a http request with a set of headers
func Delete(url string, headers map[string][]string) (Response, error) {
	// body is not generally accepted on DELETE so we are going to skip it
	return do("DELETE", url, headers, nil)
}

func do(method, url string, headers map[string][]string, body io.Reader) (Response, error) {
	// spawn new request pass in body in any case, if it is nil this is fine
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return Response{}, fmt.Errorf("unable to create %v request '%v' due to error '%v'", method, url, err)
	}
	// support array passing into headers as we can have several values for each key
	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	// execute the request finally, using the global client setup in init()
	resp, err := client.Do(req)
	if err != nil {
		return Response{}, fmt.Errorf("unable to execute %v request '%v' due to error ´%v'", method, url, err)
	}
	// now pass the http response to the Response object so we get an easy mode
	return Response{
		raw: resp,
	}, nil
}

// PostJSON submits a http request with a set of headers and a go object as JSON
func PostJSON(url string, headers map[string][]string, obj any) (Response, error) {
	body, err := json.Marshal(obj)
	if err != nil {
		return Response{}, fmt.Errorf("unable to marshal into json the object %v due to error '%v'", obj, err)
	}
	// bytes.Reader does not have a close interface so we can just allocate it and let GC clean it up
	reader := bytes.NewReader(body)
	return do("POST", url, headers, reader)
}

// Post submits a http request with a set of headers and a body
func Post(url string, headers map[string][]string, body []byte) (Response, error) {
	// bytes.Reader does not have a close interface so we can just allocate it and let GC clean it up
	reader := bytes.NewReader(body)
	return do("POST", url, headers, reader)
}
