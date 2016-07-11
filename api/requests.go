// Package api provides common utility functions for working with HTTP(S) APIs
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"
	"text/template"
	"time"

	gokitlog "github.com/go-kit/kit/log"
)

// RequestTemplates provides a cache of RequestTemplates
var RequestTemplates map[string]RequestTemplate

// RequestTemplate contains a *template.Template used to build API requests
// If an error occurred parsing the *template.Template this will be accessible through err
type RequestTemplate struct {
	err error
	t   *template.Template
}

// Call defines an API call - for logging
type Call struct {
	OriginID  string
	CalleeID  string
	TimeTaken string
	Req       string
	ReqBody   string
	Res       string
	ResBody   string
	Err       string
}

// Logger provides an interface for logging API calls
type Logger interface {
	Log(apiCall Call)
}

// LoggerFunc provides a type for single function implementations of the Logger interface
type LoggerFunc func(apiCall Call)

// Log defines the single method Logger interface
func (f LoggerFunc) Log(apiCall Call) {
	f(apiCall)
}

// BasicLogger provides a basic console style Logger implementation with the provided io.Writer
func BasicLogger(w io.Writer) Logger {
	var logger LoggerFunc
	logger = func(c Call) {
		m := fmt.Sprintf("OriginID: %s\tDuration: %s ms\tMethod: %s", c.OriginID, c.TimeTaken, c.CalleeID)
		fmt.Fprint(w, "API_CALL\t"+m+fmt.Sprintf("\tRequest: %s\tRequestBody: %s\tResponse %s\tResponseBody: %s\tError: %s\n", c.Req, c.ReqBody, c.Res, c.ResBody, c.Err))
	}
	return logger
}

// StructuredLogger provide a minimal structured Logger implementation with the provided io.Writer
// Uses go-kit log to write in Logfmt
// See https://github.com/go-kit/kit/tree/master/log
func StructuredLogger(w io.Writer) Logger {
	l := gokitlog.NewLogfmtLogger(w)
	var logger LoggerFunc
	logger = func(c Call) {
		l.Log("msg", "calling api", "origin_id", c.OriginID, "duration", c.TimeTaken, "method", c.CalleeID, "request", fmt.Sprintf("%#v", c.Req), "request_body", c.ReqBody, "response", fmt.Sprintf("%#v", c.Res), "response_body", c.ResBody, "error", c.Err)
	}
	return logger
}

// BuildBody returns the body of an API request built from the specified template and data
func BuildBody(templateName string, data interface{}, contentType string) (string, io.Reader, error) {
	rt := RequestTemplates[templateName]
	if rt.err != nil {
		return "", nil, fmt.Errorf("error initialising template %s, %v", templateName, rt.err)
	}
	if rt.t == nil {
		return "", nil, fmt.Errorf("template %s not found", templateName)
	}
	var result bytes.Buffer
	err := rt.t.Execute(&result, data)
	if err != nil {
		return "", nil, fmt.Errorf("error executing template %s, %v", templateName, err)
	}
	if contentType == "application/json" {
		var jsonResult bytes.Buffer
		err = json.Compact(&jsonResult, result.Bytes())
		if err != nil {
			return "", nil, fmt.Errorf("error in json request body generated from template %s, %s, %v", templateName, result.String(), err)
		}
		result = jsonResult
	}
	return result.String(), &result, err

}

// BuildRequest assembles an API request ready for transport
func BuildRequest(userAgent string, contentType string, method string, reqPath string, reqBody io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, reqPath, reqBody)
	if err != nil {
		return nil, fmt.Errorf("error building request %v", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", contentType)

	return req, nil
}

// Do transports a single API request
func Do(client *http.Client, originID string, calleeID string, req *http.Request, reqBody string, logger Logger) (res *http.Response, readBody string, err error) {
	start := time.Now()
	readBody = ""
	res, err = client.Do(req)
	if err != nil {
		if logger != nil {
			timeTaken := strconv.FormatFloat(time.Since(start).Seconds()*1000, 'f', 2, 64)
			logger.Log(Call{OriginID: originID, CalleeID: calleeID, TimeTaken: timeTaken, Req: fmt.Sprintf("%v", req), ReqBody: reqBody, Res: fmt.Sprintf("%v", res), ResBody: readBody, Err: fmt.Sprintf("%v", err)})
		}
		return res, readBody, err
	}
	// Defer closing of underlying connection so it can be re-used
	defer func() {
		if res != nil && res.Body != nil {
			res.Body.Close()
		}
	}()
	var buffer []byte
	buffer, err = ioutil.ReadAll(res.Body)
	readBody = string(buffer)
	if logger != nil {
		timeTaken := strconv.FormatFloat(time.Since(start).Seconds()*1000, 'f', 2, 64)
		logger.Log(Call{OriginID: originID, CalleeID: calleeID, TimeTaken: timeTaken, Req: fmt.Sprintf("%v", req), ReqBody: reqBody, Res: fmt.Sprintf("%v", res), ResBody: readBody, Err: fmt.Sprintf("%v", err)})
	}

	return res, readBody, err
}

type batchedRequest struct {
	Sequence int
	Request  *http.Request
	ReqBody  string
}

// DoBatch transports a sequence of API requests concurrently
func DoBatch(client *http.Client, originID string, calleeID string, reqs []*http.Request, reqBodies []string, logger Logger) (resps []*http.Response, readBodies []string, errs []error) {
	z := len(reqs)

	// Initialise our results
	resps = make([]*http.Response, z)
	readBodies = make([]string, z)
	errs = make([]error, z)

	// Setup a buffered channel to queue up the requests for processing
	batchedRequests := make(chan batchedRequest, z)
	for i := 0; i < z; i++ {
		batchedRequests <- batchedRequest{i, reqs[i], reqBodies[i]}
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
			resps[batchedReq.Sequence], readBodies[batchedReq.Sequence], errs[batchedReq.Sequence] = Do(client, originID, calleeID, batchedReq.Request, batchedReq.ReqBody, logger)
		}()
	}

	wg.Wait()
	return

}
