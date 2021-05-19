package filters

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"fmt"
	"github.com/loft-sh/vcluster/pkg/authorization/delegatingauthorizer"
	"github.com/loft-sh/vcluster/pkg/server/handler"
	requestpkg "github.com/loft-sh/vcluster/pkg/util/request"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"io"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
	"strings"
)

func WithRedirect(h http.Handler, localManager ctrl.Manager, targetNamespace string, resources []delegatingauthorizer.GroupVersionResourceVerb) http.Handler {
	s := serializer.NewCodecFactory(localManager.GetScheme())
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		info, ok := request.RequestInfoFrom(req.Context())
		if !ok {
			requestpkg.FailWithStatus(w, req, http.StatusInternalServerError, fmt.Errorf("request info is missing"))
			return
		}

		if applies(info, resources) {
			originalPath := req.URL.Path

			// authorization was done here already so we will just go forward with the redirect
			req.Header.Del("Authorization")

			// we have to change the request url
			if info.Resource != "nodes" {
				if info.Namespace == "" {
					responsewriters.ErrorNegotiated(kerrors.NewBadRequest("namespace required"), s, corev1.SchemeGroupVersion, w, req)
					return
				}

				splitted := strings.Split(req.URL.Path, "/")
				if len(splitted) < 6 {
					responsewriters.ErrorNegotiated(kerrors.NewBadRequest("unexpected url"), s, corev1.SchemeGroupVersion, w, req)
					return
				}

				// exchange namespace & name
				splitted[4] = targetNamespace

				// make sure we keep the prefix and suffix
				targetName := translate.PhysicalName(splitted[6], info.Namespace)
				if info.Subresource == "proxy" {
					splittedName := strings.Split(splitted[6], ":")
					switch {
					case len(splittedName) == 2:
						targetName = strings.Join([]string{translate.PhysicalName(splittedName[0], info.Namespace), splittedName[1]}, ":")
					case len(splittedName) == 3:
						targetName = strings.Join([]string{splittedName[0], translate.PhysicalName(splittedName[1], info.Namespace), splittedName[2]}, ":")
					}
				}

				splitted[6] = targetName
				req.URL.Path = strings.Join(splitted, "/")

				// we have to add a trailing slash here, because otherwise the
				// host api server would redirect us to a wrong path
				if len(splitted) == 8 {
					req.URL.Path += "/"
				}
			}

			fmt.Println("REDIRECT", req.URL.String(), req.Header)
			req.Header.Del("Authorization")
			transport, err := rest.TransportFor(localManager.GetConfig())
			if err != nil {
				requestpkg.FailWithStatus(w, req, http.StatusInternalServerError, err)
				return
			}

			h, err := handler.Handler("", localManager.GetConfig(), &rewriteTransportWrapper{
				Transport: transport,
				From:      req.URL.Path,
				To:        originalPath,
			})
			if err != nil {
				requestpkg.FailWithStatus(w, req, http.StatusInternalServerError, err)
				return
			}

			h.ServeHTTP(w, req)
			return
		}

		h.ServeHTTP(w, req)
	})
}

func applies(r *request.RequestInfo, resources []delegatingauthorizer.GroupVersionResourceVerb) bool {
	if r.IsResourceRequest == false {
		return false
	}

	for _, gv := range resources {
		if (gv.Group == "*" || gv.Group == r.APIGroup) && (gv.Version == "*" || gv.Version == r.APIVersion) && (gv.Resource == "*" || gv.Resource == r.Resource) && (gv.Verb == "*" || gv.Verb == r.Verb) && (gv.SubResource == "*" || gv.SubResource == r.Subresource) {
			return true
		}
	}

	return false
}

type rewriteTransportWrapper struct {
	Transport http.RoundTripper

	From string
	To   string
}

func (r *rewriteTransportWrapper) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := r.Transport.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	cType := resp.Header.Get("Content-Type")
	cType = strings.TrimSpace(strings.SplitN(cType, ";", 2)[0])
	if cType != "text/html" {
		// Do nothing, simply pass through
		return resp, nil
	}

	return r.rewriteResponse(resp)
}

// rewriteResponse modifies an HTML response by updating absolute links referring
// to the original host to instead refer to the proxy transport.
func (r *rewriteTransportWrapper) rewriteResponse(resp *http.Response) (*http.Response, error) {
	origBody := resp.Body
	defer origBody.Close()

	newContent := &bytes.Buffer{}
	var reader io.Reader = origBody
	var writer io.Writer = newContent
	encoding := resp.Header.Get("Content-Encoding")
	switch encoding {
	case "gzip":
		var err error
		reader, err = gzip.NewReader(reader)
		if err != nil {
			return nil, fmt.Errorf("errorf making gzip reader: %v", err)
		}
		gzw := gzip.NewWriter(writer)
		defer gzw.Close()
		writer = gzw
	case "deflate":
		var err error
		reader = flate.NewReader(reader)
		flw, err := flate.NewWriter(writer, flate.BestCompression)
		if err != nil {
			return nil, fmt.Errorf("errorf making flate writer: %v", err)
		}
		defer func() {
			flw.Close()
			flw.Flush()
		}()
		writer = flw
	case "":
		// This is fine
	default:
		// Some encoding we don't understand-- don't try to parse this
		klog.Errorf("Proxy encountered encoding %v for text/html; can't understand this so not fixing links.", encoding)
		return resp, nil
	}

	err := rewriteURL(reader, writer, r.From, r.To)
	if err != nil {
		klog.Errorf("Failed to rewrite URLs: %v", err)
		return resp, err
	}

	resp.Body = ioutil.NopCloser(newContent)
	// Update header node with new content-length
	// TODO: Remove any hash/signature headers here?
	resp.Header.Del("Content-Length")
	resp.ContentLength = int64(newContent.Len())

	return resp, err
}

// atomsToAttrs states which attributes of which tags require URL substitution.
// Sources: http://www.w3.org/TR/REC-html40/index/attributes.html
//          http://www.w3.org/html/wg/drafts/html/master/index.html#attributes-1
var atomsToAttrs = map[atom.Atom]sets.String{
	atom.A:          sets.NewString("href"),
	atom.Applet:     sets.NewString("codebase"),
	atom.Area:       sets.NewString("href"),
	atom.Audio:      sets.NewString("src"),
	atom.Base:       sets.NewString("href"),
	atom.Blockquote: sets.NewString("cite"),
	atom.Body:       sets.NewString("background"),
	atom.Button:     sets.NewString("formaction"),
	atom.Command:    sets.NewString("icon"),
	atom.Del:        sets.NewString("cite"),
	atom.Embed:      sets.NewString("src"),
	atom.Form:       sets.NewString("action"),
	atom.Frame:      sets.NewString("longdesc", "src"),
	atom.Head:       sets.NewString("profile"),
	atom.Html:       sets.NewString("manifest"),
	atom.Iframe:     sets.NewString("longdesc", "src"),
	atom.Img:        sets.NewString("longdesc", "src", "usemap"),
	atom.Input:      sets.NewString("src", "usemap", "formaction"),
	atom.Ins:        sets.NewString("cite"),
	atom.Link:       sets.NewString("href"),
	atom.Object:     sets.NewString("classid", "codebase", "data", "usemap"),
	atom.Q:          sets.NewString("cite"),
	atom.Script:     sets.NewString("src"),
	atom.Source:     sets.NewString("src"),
	atom.Video:      sets.NewString("poster", "src"),

	// TODO: css URLs hidden in style elements.
}

func rewriteURL(reader io.Reader, writer io.Writer, from, to string) error {
	// Note: This assumes the content is UTF-8.
	tokenizer := html.NewTokenizer(reader)

	var err error
	for err == nil {
		tokenType := tokenizer.Next()
		switch tokenType {
		case html.ErrorToken:
			err = tokenizer.Err()
		case html.StartTagToken, html.SelfClosingTagToken:
			token := tokenizer.Token()
			if urlAttrs, ok := atomsToAttrs[token.DataAtom]; ok {
				for i, attr := range token.Attr {
					if urlAttrs.Has(attr.Key) {
						token.Attr[i].Val = strings.Replace(attr.Val, from, to, 1)
					}
				}
			}
			_, err = writer.Write([]byte(token.String()))
		default:
			_, err = writer.Write(tokenizer.Raw())
		}
	}
	if err != io.EOF {
		return err
	}
	return nil
}
