// Package sshconfig manages the SSH client configuration for
// `vcluster platform connect slurm`. It maintains a dedicated, fully managed
// drop-in file (~/.ssh/vcluster/config) with one marker-delimited Host block
// per Slurm instance, and idempotently wires it into the user's main
// ~/.ssh/config via a single `Include` line. The Include is placed near the top
// but is careful never to split a leading comment block from the lines it
// documents (see includeInsertIndex). It never edits any other part of the main
// config.
package sshconfig

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// managedFileHeader is written at the top of the managed drop-in file.
	managedFileHeader = "# This file is managed by vcluster (platform connect slurm). Do not edit manually.\n"

	// includeTrailer marks the Include line vcluster adds to the main config.
	includeTrailer = "# managed by vcluster"

	markerPrefix = "vcluster-slurm"
	beginPrefix  = "# BEGIN " + markerPrefix + " "
	endPrefix    = "# END " + markerPrefix + " "

	dirPerm  os.FileMode = 0o700
	filePerm os.FileMode = 0o600
)

// Manager reads and writes the managed SSH configuration.
type Manager struct {
	// sshDir is the on-disk directory that holds the main config, e.g. ~/.ssh.
	sshDir string
	// renderDir is the directory prefix used inside generated config text. For
	// the real user it is "~/.ssh" so blocks read naturally; in tests it is the
	// absolute temp dir so the output is deterministic.
	renderDir string
}

// Block describes a single managed Host block.
type Block struct {
	// PlatformHost, Project and Instance form the block's source identity and
	// are encoded into the BEGIN/END markers. PlatformHost is also pinned into
	// the generated ProxyCommand (--host) so the alias keeps targeting the same
	// platform even after the CLI logs into a different one.
	PlatformHost string
	Project      string
	Instance     string
	// Alias is the SSH host alias the user types (`ssh <alias>`).
	Alias string
	// Executable is the absolute path to the vcluster binary used in the
	// ProxyCommand.
	Executable string
	// Insecure adds --insecure to the generated ProxyCommand so future ssh
	// sessions tolerate the platform's self-signed certificate.
	Insecure bool
}

// New returns a Manager rooted at the current user's ~/.ssh directory.
func New() (*Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home directory: %w", err)
	}
	return &Manager{
		sshDir:    filepath.Join(home, ".ssh"),
		renderDir: "~/.ssh",
	}, nil
}

// NewWithDir returns a Manager rooted at an explicit directory. It is intended
// for tests; generated config text uses absolute paths under dir.
func NewWithDir(dir string) *Manager {
	return &Manager{sshDir: dir, renderDir: dir}
}

func (m *Manager) managedConfigPath() string { return filepath.Join(m.sshDir, "vcluster", "config") }
func (m *Manager) knownHostsPath() string    { return filepath.Join(m.sshDir, "vcluster", "known_hosts") }
func (m *Manager) mainConfigPath() string    { return filepath.Join(m.sshDir, "config") }

func (m *Manager) renderConfigPath() string     { return m.renderDir + "/vcluster/config" }
func (m *Manager) renderKnownHostsPath() string { return m.renderDir + "/vcluster/known_hosts" }

// DefaultAlias returns the default alias for an instance: <instance>.<project>.slurm.
func DefaultAlias(instance, project string) string {
	return fmt.Sprintf("%s.%s.slurm", instance, project)
}

// sourceID is the opaque identity encoded in a block's markers.
func sourceID(platformHost, project, instance string) string {
	return fmt.Sprintf("%s/%s/%s", NormalizeHost(platformHost), project, instance)
}

// NormalizeHost strips the scheme and any trailing slash so the marker stays a
// single clean token.
func NormalizeHost(host string) string {
	h := strings.TrimRight(host, "/")
	if i := strings.Index(h, "://"); i >= 0 {
		h = h[i+3:]
	}
	return h
}

// parsedBlock is an existing managed block read from disk.
type parsedBlock struct {
	source string
	alias  string
	text   string // full block text including markers, no trailing newline
}

// Add writes or replaces the managed block for b and ensures the main config
// includes the managed file. It is idempotent: re-running with the same inputs
// produces no change. An alias already used by a differently-sourced block is a
// collision and returns an error suggesting --alias.
func (m *Manager) Add(b Block) error {
	if b.Alias == "" {
		b.Alias = DefaultAlias(b.Instance, b.Project)
	}

	blocks, err := m.readBlocks()
	if err != nil {
		return err
	}

	src := sourceID(b.PlatformHost, b.Project, b.Instance)
	for _, existing := range blocks {
		if existing.alias == b.Alias && existing.source != src {
			return fmt.Errorf("alias %q is already used by another connection (%s); choose a different alias with --alias", b.Alias, existing.source)
		}
	}

	newText := m.renderBlock(b, src)

	replaced := false
	out := make([]parsedBlock, 0, len(blocks)+1)
	for _, existing := range blocks {
		if existing.source == src {
			out = append(out, parsedBlock{source: src, alias: b.Alias, text: newText})
			replaced = true
			continue
		}
		out = append(out, existing)
	}
	if !replaced {
		out = append(out, parsedBlock{source: src, alias: b.Alias, text: newText})
	}

	if err := m.writeBlocks(out); err != nil {
		return err
	}
	return m.ensureInclude()
}

// Remove deletes the managed block identified by platformHost/project/instance
// and cleans up its known_hosts entries. It reports whether a block was
// removed.
func (m *Manager) Remove(platformHost, project, instance string) (bool, error) {
	blocks, err := m.readBlocks()
	if err != nil {
		return false, err
	}

	src := sourceID(platformHost, project, instance)
	out := make([]parsedBlock, 0, len(blocks))
	var removedAlias string
	for _, existing := range blocks {
		if existing.source == src {
			removedAlias = existing.alias
			continue
		}
		out = append(out, existing)
	}
	if removedAlias == "" {
		return false, nil
	}

	if err := m.writeBlocks(out); err != nil {
		return false, err
	}
	if err := m.removeKnownHosts(removedAlias); err != nil {
		return true, err
	}
	return true, nil
}

// List returns the aliases of all managed blocks.
func (m *Manager) List() ([]string, error) {
	blocks, err := m.readBlocks()
	if err != nil {
		return nil, err
	}
	aliases := make([]string, 0, len(blocks))
	for _, b := range blocks {
		aliases = append(aliases, b.alias)
	}
	return aliases, nil
}

// renderBlock builds the marker-delimited Host block text for b.
func (m *Manager) renderBlock(b Block, src string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "%s%s\n", beginPrefix, src)
	fmt.Fprintf(&sb, "Host %s\n", b.Alias)
	fmt.Fprintf(&sb, "    User root\n")
	insecureFlag := ""
	if b.Insecure {
		insecureFlag = " --insecure"
	}
	fmt.Fprintf(&sb, "    ProxyCommand %s platform connect slurm %s --project %s --host %s --stdio%s\n", b.Executable, b.Instance, b.Project, b.PlatformHost, insecureFlag)
	fmt.Fprintf(&sb, "    UserKnownHostsFile %s\n", m.renderKnownHostsPath())
	fmt.Fprintf(&sb, "    StrictHostKeyChecking accept-new\n")
	fmt.Fprintf(&sb, "    ServerAliveInterval 30\n")
	fmt.Fprintf(&sb, "%s%s", endPrefix, src)
	return sb.String()
}

// readBlocks parses the managed drop-in file into blocks. A missing file yields
// no blocks and no error.
func (m *Manager) readBlocks() ([]parsedBlock, error) {
	data, err := os.ReadFile(m.managedConfigPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read managed ssh config: %w", err)
	}

	var blocks []parsedBlock
	var current *parsedBlock
	var currentLines []string

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		switch {
		case strings.HasPrefix(trimmed, beginPrefix):
			current = &parsedBlock{source: strings.TrimSpace(strings.TrimPrefix(trimmed, beginPrefix))}
			currentLines = []string{line}
		case current != nil && strings.HasPrefix(trimmed, endPrefix):
			currentLines = append(currentLines, line)
			current.text = strings.Join(currentLines, "\n")
			blocks = append(blocks, *current)
			current = nil
			currentLines = nil
		case current != nil:
			currentLines = append(currentLines, line)
			if alias, ok := parseHostAlias(trimmed); ok && current.alias == "" {
				current.alias = alias
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan managed ssh config: %w", err)
	}
	return blocks, nil
}

// writeBlocks rewrites the managed drop-in file from blocks.
func (m *Manager) writeBlocks(blocks []parsedBlock) error {
	dir := filepath.Dir(m.managedConfigPath())
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return fmt.Errorf("create ssh config dir: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(managedFileHeader)
	for _, b := range blocks {
		sb.WriteString("\n")
		sb.WriteString(b.text)
		sb.WriteString("\n")
	}

	if err := os.WriteFile(m.managedConfigPath(), []byte(sb.String()), filePerm); err != nil {
		return fmt.Errorf("write managed ssh config: %w", err)
	}
	// Enforce perms even if the file already existed with looser bits.
	if err := os.Chmod(m.managedConfigPath(), filePerm); err != nil {
		return fmt.Errorf("chmod managed ssh config: %w", err)
	}
	return nil
}

// ensureInclude makes sure `Include <managed>` is present near the top of the
// main ~/.ssh/config. It is inserted at most once at the position chosen by
// includeInsertIndex and nothing else in the file is touched.
func (m *Manager) ensureInclude() error {
	includeLine := fmt.Sprintf("Include %s %s", m.renderConfigPath(), includeTrailer)
	main := m.mainConfigPath()

	data, err := os.ReadFile(main)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("read main ssh config: %w", err)
		}
		// Fresh main config: just the Include line.
		if err := os.MkdirAll(m.sshDir, dirPerm); err != nil {
			return fmt.Errorf("create ssh dir: %w", err)
		}
		if err := os.WriteFile(main, []byte(includeLine+"\n"), filePerm); err != nil {
			return fmt.Errorf("write main ssh config: %w", err)
		}
		return nil
	}

	lines := strings.Split(string(data), "\n")
	if includeIndex(lines, m.renderConfigPath()) >= 0 {
		return nil // already included
	}

	insertAt := includeInsertIndex(lines)
	updated := make([]string, 0, len(lines)+1)
	updated = append(updated, lines[:insertAt]...)
	updated = append(updated, includeLine)
	updated = append(updated, lines[insertAt:]...)

	if err := os.WriteFile(main, []byte(strings.Join(updated, "\n")), filePerm); err != nil {
		return fmt.Errorf("write main ssh config: %w", err)
	}
	return nil
}

// removeKnownHosts drops any known_hosts lines whose first field references the
// given alias. A missing file is not an error.
func (m *Manager) removeKnownHosts(alias string) error {
	path := m.knownHostsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read known_hosts: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	kept := make([]string, 0, len(lines))
	changed := false
	for _, line := range lines {
		if knownHostsLineMatches(line, alias) {
			changed = true
			continue
		}
		kept = append(kept, line)
	}
	if !changed {
		return nil
	}

	if err := os.WriteFile(path, []byte(strings.Join(kept, "\n")), filePerm); err != nil {
		return fmt.Errorf("write known_hosts: %w", err)
	}
	return nil
}

// parseHostAlias extracts the alias from a `Host <alias>` line. Only the first
// alias on the line is returned.
func parseHostAlias(trimmed string) (string, bool) {
	fields := strings.Fields(trimmed)
	if len(fields) >= 2 && strings.EqualFold(fields[0], "Host") {
		return fields[1], true
	}
	return "", false
}

// includeIndex returns the index of an existing Include line that references
// renderPath, or -1.
func includeIndex(lines []string, renderPath string) int {
	for i, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 && strings.EqualFold(fields[0], "Include") {
			for _, f := range fields[1:] {
				if f == renderPath {
					return i
				}
			}
		}
	}
	return -1
}

// includeInsertIndex chooses where to insert our Include line in the main
// config. The overriding rule is to never split a leading comment block from
// the lines it documents (e.g. OrbStack writes a comment stanza that must stay
// directly above its own Include line).
//
//   - No leading comment: insert at the very top.
//   - Leading comment block followed directly (no blank line) by a Host/Match
//     block: the comments document that block, so insert at the very top rather
//     than between them.
//   - Otherwise: insert after the first blank line following the leading comment
//     block, i.e. after the whole leading stanza (comments plus any lines they
//     document). Inserting right after a blank line can never split a comment
//     from the block it documents. If no blank line follows, append at the end.
func includeInsertIndex(lines []string) int {
	if len(lines) == 0 || !isCommentLine(lines[0]) {
		return 0
	}

	i := 0
	for i < len(lines) && isCommentLine(lines[i]) {
		i++
	}
	if i >= len(lines) {
		return len(lines)
	}
	if isHostOrMatch(lines[i]) {
		return 0
	}
	for j := i; j < len(lines); j++ {
		if strings.TrimSpace(lines[j]) == "" {
			return j + 1
		}
	}
	return len(lines)
}

func isCommentLine(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "#")
}

// isHostOrMatch reports whether a line opens a Host or Match block.
func isHostOrMatch(line string) bool {
	fields := strings.Fields(strings.TrimSpace(line))
	if len(fields) == 0 {
		return false
	}
	return strings.EqualFold(fields[0], "Host") || strings.EqualFold(fields[0], "Match")
}

// knownHostsLineMatches reports whether a known_hosts line's host field
// includes alias. Host fields may be comma-separated lists.
func knownHostsLineMatches(line, alias string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return false
	}
	fields := strings.Fields(trimmed)
	if len(fields) == 0 {
		return false
	}
	for _, host := range strings.Split(fields[0], ",") {
		if host == alias {
			return true
		}
	}
	return false
}
