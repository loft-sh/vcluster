package sshconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testBlock() Block {
	return Block{
		PlatformHost: "https://my.loft.host",
		Project:      "myproject",
		Instance:     "demo",
		Executable:   "/usr/local/bin/vcluster",
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func TestAddFreshSetup(t *testing.T) {
	dir := t.TempDir()
	m := NewWithDir(dir)

	if err := m.Add(testBlock()); err != nil {
		t.Fatalf("Add: %v", err)
	}

	// Managed config file exists with the expected block and perms.
	managed := m.managedConfigPath()
	info, err := os.Stat(managed)
	if err != nil {
		t.Fatalf("stat managed config: %v", err)
	}
	if info.Mode().Perm() != filePerm {
		t.Errorf("managed config perm = %o, want %o", info.Mode().Perm(), filePerm)
	}
	dirInfo, err := os.Stat(filepath.Dir(managed))
	if err != nil {
		t.Fatalf("stat managed dir: %v", err)
	}
	if dirInfo.Mode().Perm() != dirPerm {
		t.Errorf("managed dir perm = %o, want %o", dirInfo.Mode().Perm(), dirPerm)
	}

	content := readFile(t, managed)
	for _, want := range []string{
		"# BEGIN vcluster-slurm my.loft.host/myproject/demo",
		"Host demo.myproject.slurm",
		"    User root",
		"    ProxyCommand /usr/local/bin/vcluster platform connect slurm demo --project myproject --host https://my.loft.host --stdio",
		"    UserKnownHostsFile " + filepath.Join(dir, "vcluster", "known_hosts"),
		"    StrictHostKeyChecking accept-new",
		"    ServerAliveInterval 30",
		"# END vcluster-slurm my.loft.host/myproject/demo",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("managed config missing %q\n---\n%s", want, content)
		}
	}

	// Main config gained the Include line as the first line.
	mainContent := readFile(t, m.mainConfigPath())
	firstLine := strings.SplitN(mainContent, "\n", 2)[0]
	wantInclude := "Include " + filepath.Join(dir, "vcluster", "config") + " " + includeTrailer
	if firstLine != wantInclude {
		t.Errorf("first line of main config = %q, want %q", firstLine, wantInclude)
	}
}

func TestAddInsecureProxyCommand(t *testing.T) {
	dir := t.TempDir()
	m := NewWithDir(dir)

	b := testBlock()
	b.Insecure = true
	if err := m.Add(b); err != nil {
		t.Fatalf("Add: %v", err)
	}

	content := readFile(t, m.managedConfigPath())
	want := "    ProxyCommand /usr/local/bin/vcluster platform connect slurm demo --project myproject --host https://my.loft.host --stdio --insecure"
	if !strings.Contains(content, want) {
		t.Errorf("insecure ProxyCommand missing %q\n---\n%s", want, content)
	}
}

func TestAddPinsPlatformHost(t *testing.T) {
	dir := t.TempDir()
	m := NewWithDir(dir)

	b := testBlock()
	b.PlatformHost = "https://other.loft.host/"
	if err := m.Add(b); err != nil {
		t.Fatalf("Add: %v", err)
	}

	content := readFile(t, m.managedConfigPath())
	for _, want := range []string{
		"# BEGIN vcluster-slurm other.loft.host/myproject/demo",
		"    ProxyCommand /usr/local/bin/vcluster platform connect slurm demo --project myproject --host https://other.loft.host/ --stdio",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("managed config missing %q\n---\n%s", want, content)
		}
	}
}

func TestAddIdempotent(t *testing.T) {
	dir := t.TempDir()
	m := NewWithDir(dir)

	if err := m.Add(testBlock()); err != nil {
		t.Fatalf("first Add: %v", err)
	}
	firstManaged := readFile(t, m.managedConfigPath())
	firstMain := readFile(t, m.mainConfigPath())

	if err := m.Add(testBlock()); err != nil {
		t.Fatalf("second Add: %v", err)
	}
	if got := readFile(t, m.managedConfigPath()); got != firstManaged {
		t.Errorf("managed config changed on idempotent re-add\nbefore:\n%s\nafter:\n%s", firstManaged, got)
	}
	if got := readFile(t, m.mainConfigPath()); got != firstMain {
		t.Errorf("main config changed on idempotent re-add\nbefore:\n%s\nafter:\n%s", firstMain, got)
	}

	// Exactly one block and one Include line.
	if n := strings.Count(firstManaged, beginPrefix); n != 1 {
		t.Errorf("expected 1 block, got %d", n)
	}
	if n := strings.Count(firstMain, "Include "); n != 1 {
		t.Errorf("expected 1 Include line, got %d", n)
	}
}

func TestAddReplacesBlockOnAliasChange(t *testing.T) {
	dir := t.TempDir()
	m := NewWithDir(dir)

	if err := m.Add(testBlock()); err != nil {
		t.Fatalf("Add: %v", err)
	}

	b := testBlock()
	b.Alias = "mycluster"
	if err := m.Add(b); err != nil {
		t.Fatalf("Add with alias: %v", err)
	}

	content := readFile(t, m.managedConfigPath())
	if n := strings.Count(content, beginPrefix); n != 1 {
		t.Fatalf("expected 1 block after replacement, got %d\n%s", n, content)
	}
	if strings.Contains(content, "Host demo.myproject.slurm") {
		t.Errorf("old default alias should be gone\n%s", content)
	}
	if !strings.Contains(content, "Host mycluster") {
		t.Errorf("new alias missing\n%s", content)
	}
}

func TestAddSecondInstanceCoexists(t *testing.T) {
	dir := t.TempDir()
	m := NewWithDir(dir)

	if err := m.Add(testBlock()); err != nil {
		t.Fatalf("Add first: %v", err)
	}
	b2 := testBlock()
	b2.Instance = "other"
	if err := m.Add(b2); err != nil {
		t.Fatalf("Add second: %v", err)
	}

	content := readFile(t, m.managedConfigPath())
	if n := strings.Count(content, beginPrefix); n != 2 {
		t.Errorf("expected 2 blocks, got %d\n%s", n, content)
	}
	if !strings.Contains(content, "Host demo.myproject.slurm") || !strings.Contains(content, "Host other.myproject.slurm") {
		t.Errorf("both aliases should be present\n%s", content)
	}
}

func TestAddAliasCollisionErrors(t *testing.T) {
	dir := t.TempDir()
	m := NewWithDir(dir)

	b1 := testBlock()
	b1.Alias = "shared"
	if err := m.Add(b1); err != nil {
		t.Fatalf("Add first: %v", err)
	}

	// Different instance, same alias -> collision.
	b2 := testBlock()
	b2.Instance = "other"
	b2.Alias = "shared"
	err := m.Add(b2)
	if err == nil {
		t.Fatal("expected alias collision error, got nil")
	}
	if !strings.Contains(err.Error(), "--alias") {
		t.Errorf("collision error should suggest --alias, got: %v", err)
	}
}

func TestRemove(t *testing.T) {
	dir := t.TempDir()
	m := NewWithDir(dir)

	if err := m.Add(testBlock()); err != nil {
		t.Fatalf("Add: %v", err)
	}

	// Seed known_hosts with an entry for the alias plus an unrelated one.
	khPath := m.knownHostsPath()
	seed := "demo.myproject.slurm ssh-ed25519 AAAAKEY\nother-host ssh-rsa BBBBKEY\n"
	if err := os.WriteFile(khPath, []byte(seed), filePerm); err != nil {
		t.Fatalf("seed known_hosts: %v", err)
	}

	removed, err := m.Remove("https://my.loft.host", "myproject", "demo")
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if !removed {
		t.Fatal("expected removed = true")
	}

	content := readFile(t, m.managedConfigPath())
	if strings.Contains(content, beginPrefix) {
		t.Errorf("block should be gone\n%s", content)
	}

	kh := readFile(t, khPath)
	if strings.Contains(kh, "demo.myproject.slurm") {
		t.Errorf("known_hosts entry for alias not cleaned\n%s", kh)
	}
	if !strings.Contains(kh, "other-host") {
		t.Errorf("unrelated known_hosts entry was wrongly removed\n%s", kh)
	}
}

func TestRemoveMissing(t *testing.T) {
	dir := t.TempDir()
	m := NewWithDir(dir)

	removed, err := m.Remove("https://my.loft.host", "myproject", "nope")
	if err != nil {
		t.Fatalf("Remove missing: %v", err)
	}
	if removed {
		t.Error("expected removed = false for non-existent block")
	}
}

func TestEnsureIncludeAlreadyPresent(t *testing.T) {
	dir := t.TempDir()
	m := NewWithDir(dir)

	// Pre-existing main config that already includes the managed file plus user
	// content that must be preserved untouched.
	existing := "Include " + filepath.Join(dir, "vcluster", "config") + "\n\nHost myserver\n    HostName 10.0.0.1\n"
	if err := os.WriteFile(m.mainConfigPath(), []byte(existing), filePerm); err != nil {
		t.Fatalf("write main: %v", err)
	}

	if err := m.Add(testBlock()); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got := readFile(t, m.mainConfigPath())
	if got != existing {
		t.Errorf("main config was modified despite existing Include\nbefore:\n%s\nafter:\n%s", existing, got)
	}
}

func TestEnsureIncludeInsertedAfterComments(t *testing.T) {
	dir := t.TempDir()
	m := NewWithDir(dir)

	existing := "# my ssh config\n# second comment\n\nHost myserver\n    HostName 10.0.0.1\n"
	if err := os.WriteFile(m.mainConfigPath(), []byte(existing), filePerm); err != nil {
		t.Fatalf("write main: %v", err)
	}

	if err := m.Add(testBlock()); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got := readFile(t, m.mainConfigPath())
	lines := strings.Split(got, "\n")
	// Comments preserved at the top; Include inserted before the first
	// non-comment line ("Host myserver"), which follows the blank line.
	if lines[0] != "# my ssh config" || lines[1] != "# second comment" {
		t.Errorf("leading comments not preserved: %q", lines[:2])
	}
	wantInclude := "Include " + filepath.Join(dir, "vcluster", "config") + " " + includeTrailer
	foundIncludeIdx, foundHostIdx := -1, -1
	for i, l := range lines {
		if l == wantInclude {
			foundIncludeIdx = i
		}
		if l == "Host myserver" {
			foundHostIdx = i
		}
	}
	if foundIncludeIdx < 0 || foundHostIdx < 0 || foundIncludeIdx > foundHostIdx {
		t.Errorf("Include (%d) should precede Host (%d)\n%s", foundIncludeIdx, foundHostIdx, got)
	}
	if !strings.Contains(got, "HostName 10.0.0.1") {
		t.Errorf("user content lost\n%s", got)
	}
}

// TestEnsureIncludeAfterOrbStackHeader reproduces the real OrbStack header
// layout: a leading comment block, its own documented Include line, then a blank
// line before the first Host block. Our Include must land after that blank line
// (after the whole OrbStack stanza) and never between the comments and the
// Include they document.
func TestEnsureIncludeAfterOrbStackHeader(t *testing.T) {
	dir := t.TempDir()
	m := NewWithDir(dir)

	stanza := []string{
		"# Added by OrbStack: 'orb' SSH host for Linux machines",
		"# This only works if it's at the top of ssh_config (before any Host blocks).",
		"# This won't be added again if you remove it.",
		"Include ~/.orbstack/ssh/config",
		"",
	}
	existing := strings.Join(append(append([]string{}, stanza...), "Host firmus-jumphost", "    HostName 10.0.0.1", ""), "\n")
	if err := os.WriteFile(m.mainConfigPath(), []byte(existing), filePerm); err != nil {
		t.Fatalf("write main: %v", err)
	}

	if err := m.Add(testBlock()); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got := readFile(t, m.mainConfigPath())
	lines := strings.Split(got, "\n")

	for i, want := range stanza {
		if lines[i] != want {
			t.Fatalf("OrbStack stanza altered at line %d: got %q want %q\n%s", i, lines[i], want, got)
		}
	}

	wantInclude := "Include " + filepath.Join(dir, "vcluster", "config") + " " + includeTrailer
	if lines[len(stanza)] != wantInclude {
		t.Errorf("our Include should follow the OrbStack stanza, got %q\n%s", lines[len(stanza)], got)
	}
	if lines[len(stanza)+1] != "Host firmus-jumphost" {
		t.Errorf("Host block should follow our Include, got %q\n%s", lines[len(stanza)+1], got)
	}
}

// TestEnsureIncludeCommentGluedToHostInsertsAtTop covers a leading comment block
// that directly precedes a Host block with no blank line between them. To avoid
// splitting the comments from the Host they document, our Include is placed at
// the very top instead.
func TestEnsureIncludeCommentGluedToHostInsertsAtTop(t *testing.T) {
	dir := t.TempDir()
	m := NewWithDir(dir)

	block := []string{
		"# documents the host below",
		"# keep me attached",
		"Host myserver",
	}
	existing := strings.Join(append(append([]string{}, block...), "    HostName 10.0.0.1", ""), "\n")
	if err := os.WriteFile(m.mainConfigPath(), []byte(existing), filePerm); err != nil {
		t.Fatalf("write main: %v", err)
	}

	if err := m.Add(testBlock()); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got := readFile(t, m.mainConfigPath())
	lines := strings.Split(got, "\n")

	wantInclude := "Include " + filepath.Join(dir, "vcluster", "config") + " " + includeTrailer
	if lines[0] != wantInclude {
		t.Fatalf("expected our Include at the very top, got %q\n%s", lines[0], got)
	}
	for i, want := range block {
		if lines[i+1] != want {
			t.Errorf("comment/host block split at line %d: got %q want %q\n%s", i+1, lines[i+1], want, got)
		}
	}
}

func TestListReturnsAliases(t *testing.T) {
	dir := t.TempDir()
	m := NewWithDir(dir)

	if err := m.Add(testBlock()); err != nil {
		t.Fatalf("Add: %v", err)
	}
	b2 := testBlock()
	b2.Instance = "other"
	b2.Alias = "custom"
	if err := m.Add(b2); err != nil {
		t.Fatalf("Add: %v", err)
	}

	aliases, err := m.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(aliases) != 2 {
		t.Fatalf("expected 2 aliases, got %d: %v", len(aliases), aliases)
	}
	joined := strings.Join(aliases, ",")
	if !strings.Contains(joined, "demo.myproject.slurm") || !strings.Contains(joined, "custom") {
		t.Errorf("unexpected aliases: %v", aliases)
	}
}
