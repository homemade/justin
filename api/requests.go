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

var RequestTemplates map[string]RequestTemplate

type RequestTemplate struct {
	err error
	t   *template.Template
}

// Logger provides a simple interface for logging API calls made using frazzle
type Logger interface {
	Log(m string) (err error)
}

// LoggerFunc provides a simple type for single function implementations of the Logger interface
type LoggerFunc func(m string) (err error)

// Log defines the single method Logger interface
func (f LoggerFunc) Log(m string) (err error) {
	return f(m)
}

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

func BuildRequest(userAgent string, method string, reqPath string, reqBody io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, reqPath, reqBody)
	if err != nil {
		return nil, fmt.Errorf("error building request %v", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

func Do(client *http.Client, req *http.Request, logger Logger) (res *http.Response, readBody string, err error) {
	readBody = ""
	if logger != nil {
		logger.Log(fmt.Sprintf("http request=%v\n", req))
	}
	res, err = client.Do(req)
	if logger != nil {
		logger.Log(fmt.Sprintf("http response=%v\n", res))
	}
	if err != nil {
		return res, readBody, err
	}
	defer res.Body.Close()
	buffer, err := ioutil.ReadAll(res.Body)
	readBody = string(buffer)
	if logger != nil {
		logger.Log(fmt.Sprintf("http response body=%s\n", readBody))
	}
	return res, readBody, err
}

type batchedRequest struct {
	Sequence int
	Request  *http.Request
}

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
