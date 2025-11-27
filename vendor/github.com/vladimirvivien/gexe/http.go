package gexe

import (
	"context"
	"strings"

	"github.com/vladimirvivien/gexe/http"
)

// HttpGetWithContext uses context ctx to start an HTTP GET operation to retrieve server resource from given URL/paths.
func (e *Echo) HttpGetWithContext(ctx context.Context, url string, paths ...string) *http.ResourceReader {
	var exapandedUrl strings.Builder
	exapandedUrl.WriteString(e.vars.Eval(url))
	for _, path := range paths {
		exapandedUrl.WriteString(e.vars.Eval(path))
	}
	return http.GetWithContextVars(ctx, exapandedUrl.String(), e.vars)
}

// HttpGetWithContext starts an HTTP GET operation to retrieve server resource from given URL/paths.
func (e *Echo) HttpGet(url string, paths ...string) *http.ResourceReader {
	return e.HttpGetWithContext(context.Background(), url, paths...)
}

// HttpPostWithContext uses context ctx to start an HTTP POST operation to post resource to a server at given URL/path.
func (e *Echo) HttpPostWithContext(ctx context.Context, url string, paths ...string) *http.ResourceWriter {
	var exapandedUrl strings.Builder
	exapandedUrl.WriteString(e.vars.Eval(url))
	for _, path := range paths {
		exapandedUrl.WriteString(e.vars.Eval(path))
	}
	return http.PostWithContextVars(ctx, exapandedUrl.String(), e.vars)
}

// HttpPost starts an HTTP POST operation to post resource to a server at given URL/path.
func (e *Echo) HttpPost(url string, paths ...string) *http.ResourceWriter {
	return e.HttpPostWithContext(context.Background(), url, paths...)
}

// Get is convenient alias for HttpGet to retrieve a resource at given URL/path
func (e *Echo) Get(url string, paths ...string) *http.Response {
	return e.HttpGet(url, paths...).Do()
}

// Post is a convenient alias for HttpPost to post the specified data to given URL/path
func (e *Echo) Post(data []byte, url string, paths ...string) *http.Response {
	return e.HttpPost(url, paths...).Bytes(data).Do()
}
