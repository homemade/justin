// Package api provides common utility functions for working with HTTP(S) APIs
package api

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
	"text/template"
)

// RequestTemplates provides a cache of RequestTemplates
var RequestTemplates map[string]RequestTemplate

// RequestTemplate contains a *template.Template used to build API requests
type RequestTemplate struct {
	err error
	t   *template.Template
}

// Logger provides a simple interface for logging API calls
type Logger interface {
	Log(m string) (err error)
}

// LoggerFunc provides a simple type for single function implementations of the Logger interface
type LoggerFunc func(m string) (err error)

// Log defines the single method Logger interface
func (f LoggerFunc) Log(m string) (err error) {
	return f(m)
}

// BuildBody returns the body of an API request built from the specified template and data
func BuildBody(templateName string, data interface{}) (io.Reader, error) {
	rt := RequestTemplates[templateName]
	if rt.err != nil {
		return nil, fmt.Errorf("error initialising template %s, %v", templateName, rt.err)
	}
	if rt.t == nil {
		return nil, fmt.Errorf("template %s not found", templateName)
	}
	var result bytes.Buffer
	err := rt.t.Execute(&result, data)
	if err != nil {
		return nil, fmt.Errorf("error executing template %s, %v", templateName, err)
	}
	return &result, err
}

// BuildRequest assembles an API request ready for transport
func BuildRequest(userAgent string, method string, reqPath string, reqBody io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, reqPath, reqBody)
	if err != nil {
		return nil, fmt.Errorf("error building request %v", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

// Do transports a single API request
func Do(client *http.Client, req *http.Request, logger Logger) (res *http.Response, readBody string, err error) {
	readBody = ""
	res, err = client.Do(req)
	if err != nil {
		if logger != nil {
			logger.Log(fmt.Sprintf("logging api call\nhttp request=%v\nhttp response=%v\n", req, res))
		}
		return res, readBody, err
	}
	defer res.Body.Close()
	buffer, err := ioutil.ReadAll(res.Body)
	readBody = string(buffer)
	if logger != nil {
		logger.Log(fmt.Sprintf("logging api call\nhttp request=%v\nhttp response=%v\nhttp response body=%s\n", req, res, readBody))
	}

	return res, readBody, err
}

type batchedRequest struct {
	Sequence int
	Request  *http.Request
}

// DoBatch transports a sequence of API requests concurrently
func DoBatch(client *http.Client, reqs []*http.Request, logger Logger) (resps []*http.Response, readBodies []string, errs []error) {
	z := len(reqs)

	// Initialise our results
	resps = make([]*http.Response, z)
	readBodies = make([]string, z)
	errs = make([]error, z)

	// Setup a buffered channel to queue up the requests for processing
	batchedRequests := make(chan batchedRequest, z)
	for i := 0; i < z; i++ {
		batchedRequests <- batchedRequest{i, reqs[i]}
	}
	// Close the channel - nothing else is sent to it
	close(batchedRequests)

	// Setup a wait group so we know when all the batchedRequests have been processed
	var wg sync.WaitGroup
	wg.Add(z)

	// Start our individual goroutines to process each batchedRequest
	for i := 0; i < z; i++ {
		go func() {
			defer wg.Done()
			// Process a request
			batchedReq := <-batchedRequests
			resps[batchedReq.Sequence], readBodies[batchedReq.Sequence], errs[batchedReq.Sequence] = Do(client, batchedReq.Request, logger)
		}()
	}

	wg.Wait()
	return

}
