package platformauthenticator

import (
	"net/http"
	"sync"

	"k8s.io/apiserver/pkg/authentication/authenticator"
)

var Default = &PlatformAuthenticator{}

var _ authenticator.Request = &PlatformAuthenticator{}

type PlatformAuthenticator struct {
	m sync.RWMutex

	delegate authenticator.Request
}

func (p *PlatformAuthenticator) SetDelegate(delegate authenticator.Request) {
	p.m.Lock()
	defer p.m.Unlock()

	p.delegate = delegate
}

func (p *PlatformAuthenticator) AuthenticateRequest(req *http.Request) (*authenticator.Response, bool, error) {
	p.m.RLock()
	defer p.m.RUnlock()

	if p.delegate == nil {
		return nil, false, nil
	}

	return p.delegate.AuthenticateRequest(req)
}
