// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

package controlhttp

import (
	"net/netip"

	"tailscale.com/control/controlbase"
	"tailscale.com/feature"
	"tailscale.com/net/netx"
)

// ClientConn is a Tailscale control client as returned by the Dialer.
//
// It's effectively just a *controlbase.Conn (which it embeds) with
// optional metadata.
type ClientConn struct {
	// Conn is the noise connection.
	*controlbase.Conn
}

// HookMakeACEDialer, if set, is used by the HTTPS Dial path to build a dialer
// for connecting via an Anycast Client Endpoint (ACE) host. Kept here (not in
// client.go) so it's compiled on platforms where client.go is excluded by
// build tag but the feature registry still needs to reference it.
var HookMakeACEDialer feature.Hook[func(dialer netx.DialFunc, aceHost string, optIP netip.Addr) netx.DialFunc]
