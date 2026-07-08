package ssdp

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	ssdpDiscover   = `"ssdp:discover"`
	ntsAlive       = `ssdp:alive`
	ntsByebye      = `ssdp:byebye`
	ntsUpdate      = `ssdp:update`
	ssdpUDP4Addr   = "239.255.255.250:1900"
	ssdpSearchPort = 1900
	methodSearch   = "M-SEARCH"
	methodNotify   = "NOTIFY"

	// SSDPAll is a value for searchTarget that searches for all devices and services.
	SSDPAll = "ssdp:all"
	// UPNPRootDevice is a value for searchTarget that searches for all root devices.
	UPNPRootDevice = "upnp:rootdevice"
)

// HTTPUClient is the interface required to perform HTTP-over-UDP requests.
type HTTPUClient interface {
	Do(
		req *http.Request,
		timeout time.Duration,
		numSends int,
	) ([]*http.Response, error)
}

// HTTPUClientCtx is an optional interface that will be used to perform
// HTTP-over-UDP requests if the client implements it.
type HTTPUClientCtx interface {
	DoWithContext(
		req *http.Request,
		numSends int,
	) ([]*http.Response, error)
}

// SSDPRawSearchCtx performs a fairly raw SSDP search request, and returns the
// unique response(s) that it receives. Each response has the requested
// searchTarget, a USN, and a valid location. maxWaitSeconds states how long to
// wait for responses in seconds, and must be a minimum of 1 (the
// implementation waits an additional 100ms for responses to arrive), 2 is a
// reasonable value for this. numSends is the number of requests to send - 3 is
// a reasonable value for this.
func SSDPRawSearchCtx(
	ctx context.Context,
	httpu HTTPUClient,
	searchTarget string,
	maxWaitSeconds int,
	numSends int,
) ([]*http.Response, error) {
	req, err := prepareRequest(ctx, searchTarget, maxWaitSeconds)
	if err != nil {
		return nil, err
	}

	allResponses, err := httpu.Do(req, time.Duration(maxWaitSeconds)*time.Second+100*time.Millisecond, numSends)
	if err != nil {
		return nil, err
	}
	return processSSDPResponses(searchTarget, allResponses)
}

// RawSearch performs a fairly raw SSDP search request, and returns the
// unique response(s) that it receives. Each response has the requested
// searchTarget, a USN, and a valid location. If the provided context times out
// or is canceled, the search will be aborted. numSends is the number of
// requests to send - 3 is a reasonable value for this.
//
// The provided context should have a deadline, since the SSDP protocol
// requires the max wait time be included in search requests. If the context
// has no deadline, then a default deadline of 3 seconds will be applied.
func RawSearch(
	ctx context.Context,
	httpu HTTPUClientCtx,
	searchTarget string,
	numSends int,
) ([]*http.Response, error) {
	// We need a timeout value to include in the SSDP request; get it by
	// checking the deadline on the context.
	var maxWaitSeconds int
	if deadline, ok := ctx.Deadline(); ok {
		maxWaitSeconds = int(deadline.Sub(time.Now()) / time.Second)
	} else {
		// Pick a default timeout of 3 seconds if none was provided.
		maxWaitSeconds = 3

		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, time.Duration(maxWaitSeconds)*time.Second)
		defer cancel()
	}

	req, err := prepareRequest(ctx, searchTarget, maxWaitSeconds)
	if err != nil {
		return nil, err
	}

	allResponses, err := httpu.DoWithContext(req, numSends)
	if err != nil {
		return nil, err
	}
	return processSSDPResponses(searchTarget, allResponses)
}

// prepareRequest checks the provided parameters and constructs a SSDP search
// request to be sent.
func prepareRequest(ctx context.Context, searchTarget string, maxWaitSeconds int) (*http.Request, error) {
	if maxWaitSeconds < 1 {
		return nil, errors.New("ssdp: request timeout must be at least 1s")
	}

	req := (&http.Request{
		Method: methodSearch,
		// TODO: Support both IPv4 and IPv6.
		Host: ssdpUDP4Addr,
		URL:  &url.URL{Opaque: "*"},
		Header: http.Header{
			// Putting headers in here avoids them being title-cased.
			// (The UPnP discovery protocol uses case-sensitive headers)
			"HOST": []string{ssdpUDP4Addr},
			"MX":   []string{strconv.FormatInt(int64(maxWaitSeconds), 10)},
			"MAN":  []string{ssdpDiscover},
			"ST":   []string{searchTarget},
		},
	}).WithContext(ctx)
	return req, nil
}

func processSSDPResponses(
	searchTarget string,
	allResponses []*http.Response,
) ([]*http.Response, error) {
	isExactSearch := searchTarget != SSDPAll && searchTarget != UPNPRootDevice

	seenIDs := make(map[string]bool)
	var responses []*http.Response
	for _, response := range allResponses {
		if response.StatusCode != 200 {
			log.Printf("ssdp: got response status code %q in search response", response.Status)
			continue
		}
		if st := response.Header.Get("ST"); isExactSearch && st != searchTarget {
			continue
		}
		usn := response.Header.Get("USN")
		loc, err := response.Location()
		if err != nil {
			// No usable location in search response - discard.
			continue
		}
		id := loc.String() + "\x00" + usn
		if _, alreadySeen := seenIDs[id]; !alreadySeen {
			seenIDs[id] = true
			responses = append(responses, response)
		}
	}

	return responses, nil
}

// SSDPRawSearch is the legacy version of SSDPRawSearchCtx, but uses
// context.Background() as the context.
func SSDPRawSearch(httpu HTTPUClient, searchTarget string, maxWaitSeconds int, numSends int) ([]*http.Response, error) {
	return SSDPRawSearchCtx(context.Background(), httpu, searchTarget, maxWaitSeconds, numSends)
}
