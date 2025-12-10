package http

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/vladimirvivien/gexe/vars"
)

// ResourceReader provides types and methods to read content of resources from a server using HTTP
type ResourceReader struct {
	client  *http.Client
	err     error
	url     string
	vars    *vars.Variables
	ctx     context.Context
	data    io.Reader
	headers http.Header
}

// GetWithContextVars uses context ctx and session variables to initiate
// a "GET" operation for the specified resource
func GetWithContextVars(ctx context.Context, url string, variables *vars.Variables) *ResourceReader {
	if variables == nil {
		variables = &vars.Variables{}
	}

	return &ResourceReader{
		ctx:    ctx,
		url:    variables.Eval(url),
		client: &http.Client{},
		vars:   &vars.Variables{},
	}
}

// GetWithVars uses session vars to initiate  a "GET" operation
func GetWithVars(url string, variables *vars.Variables) *ResourceReader {
	return GetWithContextVars(context.Background(), url, variables)
}

// Get initiates a "GET" operation for the specified resource
func Get(url string) *ResourceReader {
	return GetWithContextVars(context.Background(), url, &vars.Variables{})
}

// SetVars sets session variables for ResourceReader
func (r *ResourceReader) SetVars(variables *vars.Variables) *ResourceReader {
	r.vars = variables
	return r
}

// Err returns the last known error
func (r *ResourceReader) Err() error {
	return r.err
}

// WithTimeout sets the HTTP reader's timeout
func (r *ResourceReader) WithTimeout(to time.Duration) *ResourceReader {
	r.client.Timeout = to
	return r
}

// WithContext sets the context for the HTTP request
func (r *ResourceReader) WithContext(ctx context.Context) *ResourceReader {
	r.ctx = ctx
	return r
}

// WithHeaders sets all HTTP headers for GET request
func (r *ResourceReader) WithHeaders(h http.Header) *ResourceReader {
	r.headers = h
	return r
}

// AddHeader convenience method to add request header
func (r *ResourceReader) AddHeader(key, value string) *ResourceReader {
	r.headers.Add(r.vars.Eval(key), r.vars.Eval(value))
	return r
}

// SetHeader convenience method to set a specific header
func (r *ResourceReader) SetHeader(key, value string) *ResourceReader {
	r.headers.Set(r.vars.Eval(key), r.vars.Eval(value))
	return r
}

// RequestString sets GET request data as string
func (r *ResourceReader) String(val string) *ResourceReader {
	r.data = strings.NewReader(r.vars.Eval(val))
	return r
}

// RequestBytes sets GET request data as byte slice
func (r *ResourceReader) Bytes(data []byte) *ResourceReader {
	r.data = bytes.NewReader(data)
	return r
}

// RequestBody sets GET request content as io.Reader
func (r *ResourceReader) Body(body io.Reader) *ResourceReader {
	r.data = body
	return r
}

// Do is a terminal method that actually retrieves the HTTP resource from the server.
// It returns a gexe/http/*Response instance that can be used to access the result.
func (r *ResourceReader) Do() *Response {
	req, err := http.NewRequestWithContext(r.ctx, "GET", r.url, r.data)
	if err != nil {
		return &Response{err: err}
	}

	res, err := r.client.Do(req)
	if err != nil {
		return &Response{err: err}
	}

	return &Response{stat: res.Status, statCode: res.StatusCode, body: res.Body}
}
