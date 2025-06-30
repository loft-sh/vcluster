package registry

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/spf13/cobra"

	"k8s.io/client-go/rest"
)

type ProxyOptions struct {
	*flags.GlobalFlags

	Port int

	Log log.Logger
}

func NewProxyCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	o := &ProxyOptions{
		GlobalFlags: globalFlags,

		Log: log.GetInstance(),
	}

	cmd := &cobra.Command{
		Use:   "proxy",
		Short: "Proxy the vCluster registry to use it with docker and other tools",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().IntVar(&o.Port, "port", 15000, "The local port to proxy the registry to")

	return cmd
}

func (o *ProxyOptions) Run(ctx context.Context) error {
	// get the client config
	restConfig, err := getConfig(ctx, o.GlobalFlags)
	if err != nil {
		return fmt.Errorf("failed to get client config: %w", err)
	}

	// create the proxy server
	if err := startReverseProxy(restConfig, o.Port, o.Log); err != nil {
		return fmt.Errorf("failed to start reverse proxy: %w", err)
	}

	// serve the proxy server
	o.Log.Infof("Serving registry on http://localhost:%d...", o.Port)
	o.Log.Infof("You can now push images to the registry by running: \n\n# Tag an image\ndocker tag nginx localhost:%d/nginx\n\n#Push the image\ndocker push localhost:%d/nginx", o.Port, o.Port)
	<-ctx.Done()
	return nil
}

func startReverseProxy(restConfig *rest.Config, port int, log log.Logger) error {
	// get the transport
	transport, err := rest.TransportFor(restConfig)
	if err != nil {
		return fmt.Errorf("failed to get transport: %w", err)
	}

	// parse the host
	host := restConfig.Host
	if !strings.HasSuffix(host, "/") {
		host = host + "/"
	}
	target, err := url.Parse(host)
	if err != nil {
		return err
	}

	// create the new host
	newHost := fmt.Sprintf("127.0.0.1:%d", port)

	// create the proxy
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = &rewriteHeaderTransport{
		RoundTripper: transport,
		replaceHost: func(host string) string {
			// we need to replace the target host with the new host
			host = strings.Replace(host, target.String(), "http://"+newHost, -1)
			// this is for proxies where the target host is not the same as the host
			host = strings.Replace(host, "https://localhost:8443", "http://"+newHost, -1)
			return host
		},
	}
	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.Host = target.Host
		req.URL.Path, req.URL.RawPath = joinURLPath(target, req.URL)

		// some clients try to set authorization header, we need to remove it
		delete(req.Header, "Authorization")
	}

	// Plain net/http server.
	server := &http.Server{
		Addr:    newHost,
		Handler: proxy,
	}

	// start the server
	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Errorf("Failed to serve proxy server: %v", err)
		}
	}()

	// wait for the server to be ready
	now := time.Now()
	for {
		resp, err := http.Get("http://" + newHost + "/v2/")
		if err != nil {
			if time.Since(now) > 10*time.Second {
				log.Errorf("Failed to get registry: %v", err)
			}
			time.Sleep(time.Second)
			continue
		} else if resp.StatusCode != http.StatusOK {
			if time.Since(now) > 10*time.Second {
				log.Errorf("Failed to get registry: %v", resp.StatusCode)
			}
			time.Sleep(time.Second)
			continue
		}
		resp.Body.Close()
		break
	}

	return nil
}

func joinURLPath(a, b *url.URL) (path, rawpath string) {
	if a.RawPath == "" && b.RawPath == "" {
		return singleJoiningSlash(a.Path, b.Path), ""
	}
	// Same as singleJoiningSlash, but uses EscapedPath to determine
	// whether a slash should be added
	apath := a.EscapedPath()
	bpath := b.EscapedPath()

	aslash := strings.HasSuffix(apath, "/")
	bslash := strings.HasPrefix(bpath, "/")

	switch {
	case aslash && bslash:
		return a.Path + b.Path[1:], apath + bpath[1:]
	case !aslash && !bslash:
		return a.Path + "/" + b.Path, apath + "/" + bpath
	}
	return a.Path + b.Path, apath + bpath
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

type rewriteHeaderTransport struct {
	http.RoundTripper

	replaceHost func(string) string
}

func (t *rewriteHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.RoundTripper.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	for k := range resp.Header {
		for i := range resp.Header[k] {
			resp.Header[k][i] = t.replaceHost(resp.Header[k][i])
		}
	}

	return resp, nil
}
