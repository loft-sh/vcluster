// Package tailnet implements the client side of the platform tailnet used by
// `vcluster platform connect slurm --stdio`. It starts an ephemeral, in-process
// userspace tsnet node, waits for the Slurm login proxy peer to become visible
// on the tailnet, dials its ssh port and pipes the connection between stdin and
// stdout so it can be used as an SSH ProxyCommand.
//
// The tsnet setup mirrors loft-enterprise's pkg/agent/tailscale StartTSServer;
// it cannot import loft-enterprise, so the ~30 lines are replicated here.
package tailnet

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	stdlog "log"
	"net/netip"
	"os"
	"strings"
	"time"

	"tailscale.com/envknob"
	"tailscale.com/ipn/store/mem"
	"tailscale.com/tsnet"
	"tailscale.com/types/logger"
)

// Suffixes of the client and login hostnames on the platform tailnet. They must
// stay in sync with the platform side (loft-enterprise
// pkg/tailscale/networkpeer): the login proxy registers as
// "ssh.<instance>.<project>.slurm" and CLI sessions as
// "<host>-<rand8>.<user>.client.slurm".
const (
	slurmSuffix  = "slurm"
	clientSuffix = "client"
)

// defaultPeerTimeout is how long we wait for the login proxy peer to show up on
// the tailnet before giving up with a clear error.
const defaultPeerTimeout = 30 * time.Second

// Options configures a stdio proxy session.
type Options struct {
	// Host is the platform host (e.g. https://my.loft.host). "/coordinator/" is
	// appended to derive the tsnet control URL.
	Host string
	// AccessKey is the stored platform access key, reused verbatim as the tsnet
	// auth key. The platform appends the slurm-client group to it at creation.
	AccessKey string
	// Hostname is the unique per-session client hostname on the tailnet.
	Hostname string
	// LoginHostname is the login proxy peer to dial, e.g.
	// "ssh.<instance>.<project>.slurm".
	LoginHostname string
	// Timeout bounds how long we wait for the login peer to appear. Zero means
	// defaultPeerTimeout.
	Timeout time.Duration
	// Insecure makes the tsnet control-plane connection tolerate the platform's
	// self-signed certificate.
	Insecure bool
	// Debug emits tailnet/tsnet logs to stderr. When false all tsnet output is
	// discarded so a plain `ssh <alias>` stays quiet.
	Debug bool
}

// BuildClientHostname returns a unique per-session tailnet hostname of the form
// "<sanitized-os-hostname>-<rand8>.<sanitized-user>.client.slurm". A fresh
// random suffix per invocation is mandatory: the platform keys peer
// create-vs-update on hostname alone, so two live sessions sharing a hostname
// would steal each other's peer identity.
func BuildClientHostname(user string) (string, error) {
	osHostname, err := os.Hostname()
	if err != nil || osHostname == "" {
		osHostname = "client"
	}

	suffix, err := randomSuffix()
	if err != nil {
		return "", fmt.Errorf("generate random hostname suffix: %w", err)
	}

	host := sanitize(osHostname)
	if host == "" {
		host = "client"
	}
	userPart := sanitize(user)
	if userPart == "" {
		userPart = "user"
	}

	return fmt.Sprintf("%s-%s.%s.%s.%s", host, suffix, userPart, clientSuffix, slurmSuffix), nil
}

// RunStdio starts an ephemeral tsnet node, waits for the login proxy peer, dials
// its ssh port and pipes the connection between os.Stdin and os.Stdout until one
// side closes or ctx is cancelled. Nothing is ever written to os.Stdout except
// the raw SSH byte stream.
func RunStdio(ctx context.Context, opts Options) error {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = defaultPeerTimeout
	}

	stateDir, err := os.MkdirTemp("", "vcluster-slurm-tsnet-")
	if err != nil {
		return fmt.Errorf("create ephemeral tailnet state dir: %w", err)
	}
	defer os.RemoveAll(stateDir)

	server, err := startServer(opts.Host, opts.AccessKey, opts.Hostname, stateDir, opts.Insecure, opts.Debug)
	if err != nil {
		return err
	}
	defer server.Close()

	// Wait until our own node is up and in the network map, then until the login
	// proxy peer is visible with an IP.
	if err := waitForOnline(ctx, server, timeout); err != nil {
		return err
	}

	ip, err := waitForPeer(ctx, server, opts.LoginHostname, timeout)
	if err != nil {
		return err
	}

	conn, err := server.Dial(ctx, "tcp", netip.AddrPortFrom(ip, 22).String())
	if err != nil {
		return fmt.Errorf("dial login node %s: %w", opts.LoginHostname, err)
	}
	defer conn.Close()

	// Pipe stdin -> conn and conn -> stdout. When either direction finishes the
	// session is over.
	errc := make(chan error, 2)
	go func() {
		_, err := io.Copy(conn, os.Stdin)
		errc <- err
	}()
	go func() {
		_, err := io.Copy(os.Stdout, conn)
		errc <- err
	}()

	select {
	case <-ctx.Done():
		return nil
	case err := <-errc:
		if err != nil && ctx.Err() == nil {
			return fmt.Errorf("proxy connection: %w", err)
		}
		return nil
	}
}

// startServer builds and starts an ephemeral in-memory tsnet node. This mirrors
// loft-enterprise pkg/agent/tailscale.StartTSServer. stateDir is used only as a
// throwaway var root: the node's state lives in the in-memory store, but tsnet
// still writes its log config there and defaults to a persistent location under
// os.UserConfigDir when Dir is empty, so we point it at a temp dir the caller
// removes on close.
func startServer(host, accessKey, hostname, stateDir string, insecure, debug bool) (*tsnet.Server, error) {
	if host == "" || accessKey == "" {
		return nil, fmt.Errorf("platform host and access key are required; run `vcluster platform login` first")
	}

	if insecure {
		// Same knob the platform agent relies on: tailscale's tlsdial control
		// dialer skips verification when this is set, which lets us reach a
		// coordinator behind a self-signed certificate.
		envknob.Setenv("TS_DEBUG_TLS_DIAL_INSECURE_SKIP_VERIFY", "true")
	}

	// Some fork packages (derp/derphttp, net/tlsdial) log via the stdlib global
	// logger and offer no injection point, so tsnet's Logf cannot capture them;
	// silence the global logger or they leak onto every plain `ssh` invocation.
	if !debug {
		stdlog.SetOutput(io.Discard)
	}

	lf := logf(debug)
	store, _ := mem.New(lf, "")
	server := &tsnet.Server{
		Hostname:   hostname,
		Logf:       lf,
		UserLogf:   lf,
		ControlURL: controlURL(host),
		AuthKey:    accessKey,
		Dir:        stateDir,
		Ephemeral:  true,
		Store:      store,
	}

	if err := server.Start(); err != nil {
		return nil, fmt.Errorf("start tailnet node: %w", err)
	}
	return server, nil
}

// waitForOnline blocks until the local node reports Online and InNetworkMap.
func waitForOnline(ctx context.Context, server *tsnet.Server, timeout time.Duration) error {
	lc, err := server.LocalClient()
	if err != nil {
		return fmt.Errorf("get local tailnet client: %w", err)
	}

	deadline := time.Now().Add(timeout)
	for {
		status, err := lc.Status(ctx)
		if err == nil && status.Self != nil && status.Self.Online && status.Self.InNetworkMap {
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for the tailnet node to come online after %s; check that the platform coordinator is reachable", timeout)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
}

// waitForPeer polls the netmap for the login proxy peer identified by
// loginHostname and returns its first IPv4 tailnet address.
func waitForPeer(ctx context.Context, server *tsnet.Server, loginHostname string, timeout time.Duration) (netip.Addr, error) {
	lc, err := server.LocalClient()
	if err != nil {
		return netip.Addr{}, fmt.Errorf("get local tailnet client: %w", err)
	}

	deadline := time.Now().Add(timeout)
	for {
		status, err := lc.Status(ctx)
		if err == nil {
			for _, peer := range status.Peer {
				if !peerMatches(peer.HostName, peer.DNSName, loginHostname) {
					continue
				}
				for _, ip := range peer.TailscaleIPs {
					if ip.Is4() {
						return ip, nil
					}
				}
			}
		}

		if time.Now().After(deadline) {
			return netip.Addr{}, fmt.Errorf("login node proxy %q not connected after %s; check the SlurmInstance status and that it is shared with you", loginHostname, timeout)
		}
		select {
		case <-ctx.Done():
			return netip.Addr{}, ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
}

// peerMatches reports whether a peer's HostName or DNSName identifies the target
// login hostname. DNSName is an FQDN ending in a dot with the MagicDNS suffix
// appended, so we match on the leading label.
func peerMatches(hostName, dnsName, target string) bool {
	if hostName == target {
		return true
	}
	trimmed := strings.TrimSuffix(dnsName, ".")
	return trimmed == target || strings.HasPrefix(trimmed, target+".")
}

// controlURL derives the tsnet control URL from the platform host.
func controlURL(host string) string {
	return strings.TrimRight(host, "/") + "/coordinator/"
}

// logf routes tsnet logging to stderr when debug is set, and discards it
// otherwise. It is wired into both tsnet.Server.Logf and UserLogf; UserLogf
// carries the login/status lines that otherwise default to the standard
// library logger and leak onto the terminal. It must never write to stdout,
// which carries the raw SSH byte stream.
func logf(debug bool) logger.Logf {
	if debug {
		return func(format string, args ...any) {
			fmt.Fprintf(os.Stderr, format+"\n", args...)
		}
	}
	return logger.Discard
}

// sanitize reduces s to the tailnet-safe label charset [a-z0-9-], collapsing
// runs of other characters to a single dash and trimming leading/trailing
// dashes.
func sanitize(s string) string {
	var b strings.Builder
	lastDash := true // avoid a leading dash
	for _, r := range strings.ToLower(s) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteRune('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

// randomSuffix returns 8 hex characters of cryptographic randomness.
func randomSuffix() (string, error) {
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
