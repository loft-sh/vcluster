package http

import "io"

// Response stores high level metadata and responses from HTTP request results
type Response struct {
	stat     string
	statCode int
	body     io.ReadCloser
	err      error
}

// Status returns the standard lib http.Response.Status value from the server
func (res *Response) Status() string {
	return res.stat
}

// StatusCode returns the standard lib http.Response.StatusCode value from the server
func (res *Response) StatusCode() int {
	return res.statCode
}

// Body is io.ReadCloser stream to the content from serve.
// NOTE: ensure to call Close() if used directly.
func (res *Response) Body() io.ReadCloser {
	return res.body
}

// Err returns the response known error
func (r *Response) Err() error {
	return r.err
}

// Bytes returns the server response as a []byte
func (r *Response) Bytes() []byte {
	return r.read()
}

// String returns the server response as a string
func (r *Response) String() string {
	return string(r.read())
}

// read reads the content of the response body and returns as []byte
func (r *Response) read() []byte {
	if r.body == nil {
		return nil
	}

	data, err := io.ReadAll(r.body)
	defer r.body.Close()
	if err != nil {
		r.err = err
		return nil
	}
	return data
}
