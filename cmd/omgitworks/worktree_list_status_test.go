package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/daileyo/omgitworks/internal/config"
	"github.com/daileyo/omgitworks/internal/git"
)

// statusFixture is a real repository with three linked worktrees in known
// states, plus a bare origin so ahead/behind counts resolve.
type statusFixture struct {
	Repo  string
	Clean string
	Dirty string
	Ahead string
}

// gitc runs git with a throwaway identity so commits work under a bare HOME.
func gitc(t *testing.T, dir string, args ...string) {
	t.Helper()
	runGitIn(t, dir, append([]string{"-c", "user.name=Test", "-c", "user.email=test@example.com"}, args...)...)
}

// newStatusFixture builds the fixture and saves a config naming it "fixture".
// HOME is redirected first so neither global git hooks nor the real status
// cache are touched; saveConfigForWorktreeTests redirects it again, and the
// cache file lands under that final HOME.
func newStatusFixture(t *testing.T, extra ...config.Worktree) statusFixture {
	t.Helper()
	t.Setenv("HOME", t.TempDir())

	base := t.TempDir()
	repo := filepath.Join(base, "fixture")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	gitc(t, repo, "init", "-b", "main")
	gitc(t, repo, "commit", "--allow-empty", "-m", "init")

	origin := filepath.Join(base, "origin.git")
	runGitIn(t, base, "init", "--bare", origin)
	gitc(t, repo, "remote", "add", "origin", origin)
	gitc(t, repo, "push", "-u", "origin", "main")

	f := statusFixture{Repo: repo}
	wts := map[string]*string{"clean": &f.Clean, "dirty": &f.Dirty, "ahead": &f.Ahead}
	var stored []config.Worktree
	for _, name := range []string{"clean", "dirty", "ahead"} {
		wt := filepath.Join(base, "wt", name)
		gitWorktreeAdd(t, repo, name, wt)
		gitc(t, wt, "push", "-u", "origin", name)
		*wts[name] = wt
		stored = append(stored, config.Worktree{Path: wt, Branch: name, Aligned: name != "dirty"})
	}
	if err := os.WriteFile(filepath.Join(f.Dirty, "untracked.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitc(t, f.Ahead, "commit", "--allow-empty", "-m", "ahead of origin")

	stored = append(stored, extra...)
	saveConfigForWorktreeTests(t, []config.Repository{{Name: "fixture", Path: repo, Worktrees: stored}})
	return f
}

// listWorktrees runs the command and returns its output.
func listWorktrees(t *testing.T, scope worktreeScope, opts worktreeListOptions) string {
	t.Helper()
	var buf bytes.Buffer
	if err := runWorktreeList(scope, opts, &buf); err != nil {
		t.Fatalf("runWorktreeList: %v", err)
	}
	return buf.String()
}

// rowFor returns the output line whose BRANCH cell starts with branch.
func rowFor(t *testing.T, output, branch string) string {
	t.Helper()
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(stripANSI(line))
		if len(fields) >= 2 && fields[0] == "fixture" && fields[1] == branch {
			return line
		}
	}
	t.Fatalf("no row for branch %q in:\n%s", branch, output)
	return ""
}

// iconStart returns the visible column at which the first status icon
// appears in line, or -1 when there is none.
func iconStart(line string) int {
	plain := stripANSI(line)
	idx := strings.IndexAny(plain, "↓↑✓✗")
	if idx < 0 {
		return -1
	}
	return displayWidth(plain[:idx])
}

func TestWorktreeListStatus_Compact(t *testing.T) {
	newStatusFixture(t)
	out := listWorktrees(t, worktreeScope{}, worktreeListOptions{CompactStatus: true})

	if strings.Contains(out, "STATUS") {
		t.Errorf("compact mode must not add a STATUS column:\n%s", out)
	}
	if !strings.Contains(out, "ALIGNED") {
		t.Errorf("header should contain ALIGNED:\n%s", out)
	}
	for branch, icons := range map[string]string{"clean": "✓", "dirty": "✗", "ahead": "↑1 ✓"} {
		row := rowFor(t, out, branch)
		if !strings.Contains(row, branch+"  ") || !strings.Contains(row, icons) {
			t.Errorf("row for %s should carry %q after the branch, got: %s", branch, icons, row)
		}
		if strings.Contains(row, "\x1b[") {
			t.Errorf("row for %s should be plain without color, got: %q", branch, row)
		}
	}
}

func TestWorktreeListStatus_CompactAligned(t *testing.T) {
	newStatusFixture(t)
	out := listWorktrees(t, worktreeScope{}, worktreeListOptions{CompactStatus: true})

	cols := map[string]int{}
	for _, branch := range []string{"clean", "dirty", "ahead"} {
		cols[branch] = iconStart(rowFor(t, out, branch))
	}
	// Icons are right-justified in their sub-column, so the widest icon
	// group starts earliest and the others end at the same column. Compare
	// the end column: start + width.
	ends := map[string]int{}
	for _, branch := range []string{"clean", "dirty", "ahead"} {
		row := stripANSI(rowFor(t, out, branch))
		idx := strings.IndexAny(row, "↓↑✓✗")
		rest := row[idx:]
		iconsEnd := strings.Index(rest, "  ")
		ends[branch] = displayWidth(row[:idx]) + displayWidth(rest[:iconsEnd])
	}
	if ends["clean"] != ends["dirty"] || ends["dirty"] != ends["ahead"] {
		t.Errorf("icon groups should end at the same column, got %v (starts %v):\n%s", ends, cols, out)
	}
	if cols["clean"] < 0 || cols["ahead"] >= cols["clean"] {
		t.Errorf("wider icon group should start earlier: %v", cols)
	}
}

func TestWorktreeListStatus_Color(t *testing.T) {
	f := newStatusFixture(t)

	colored := listWorktrees(t, worktreeScope{}, worktreeListOptions{CompactStatus: true, ColorEnabled: true})
	for branch, path := range map[string]string{"clean": f.Clean, "dirty": f.Dirty, "ahead": f.Ahead} {
		st, err := git.GetStatus(path)
		if err != nil {
			t.Fatalf("GetStatus(%s): %v", branch, err)
		}
		want := formatStatusIcons(st, true)
		if !strings.Contains(rowFor(t, colored, branch), want) {
			t.Errorf("row for %s should contain the colored icons %q, got: %q", branch, want, rowFor(t, colored, branch))
		}
	}

	plain := listWorktrees(t, worktreeScope{}, worktreeListOptions{CompactStatus: true, ColorEnabled: false})
	if strings.Contains(plain, "\x1b[") {
		t.Errorf("--color never output should carry no ANSI codes:\n%q", plain)
	}
}

func TestWorktreeListStatus_MissingWorktree(t *testing.T) {
	gone := filepath.Join(t.TempDir(), "gone")
	newStatusFixture(t, config.Worktree{Path: gone, Branch: "gone", Aligned: false})

	out := listWorktrees(t, worktreeScope{}, worktreeListOptions{CompactStatus: true})

	row := rowFor(t, out, "gone")
	if iconStart(row) >= 0 {
		t.Errorf("a worktree with no readable status should show no icons, got: %s", row)
	}
	if !strings.HasSuffix(strings.TrimRight(row, " "), "  no") {
		t.Errorf("the row should still render its ALIGNED value, got: %s", row)
	}
	if iconStart(rowFor(t, out, "clean")) < 0 {
		t.Errorf("other rows should keep their icons:\n%s", out)
	}
}

func TestWorktreeListStatus_OptIn(t *testing.T) {
	newStatusFixture(t)
	out := listWorktrees(t, worktreeScope{}, worktreeListOptions{})

	if iconStart(out) >= 0 {
		t.Errorf("no status flag should mean no icons:\n%s", out)
	}
	if readCacheFile(t) != nil {
		t.Error("no status flag should leave the status cache untouched")
	}
}

// addDetachedWorktree creates a worktree at a detached HEAD and returns its
// config entry (an empty Branch), ready to pass to newStatusFixture.
func detachedWorktree(t *testing.T) (string, config.Worktree) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "detached")
	return path, config.Worktree{Path: path, Branch: "", Aligned: true}
}

// pathField returns the field of a (stripped) row that should hold the path.
func pathField(t *testing.T, row string, index int) string {
	t.Helper()
	fields := strings.Fields(stripANSI(row))
	if len(fields) <= index {
		t.Fatalf("row has only %d fields: %q", len(fields), row)
	}
	return fields[index]
}

func TestWorktreeListStatus_Full(t *testing.T) {
	newStatusFixture(t)
	out := listWorktrees(t, worktreeScope{}, worktreeListOptions{ShowStatus: true})

	header := strings.Split(out, "\n")[0]
	if !strings.HasSuffix(strings.TrimRight(header, " "), "ALIGNED  STATUS") {
		t.Errorf("header should end with ALIGNED then STATUS, got: %q", header)
	}
	for branch, icons := range map[string]string{"clean": "✓", "dirty": "✗", "ahead": "↑1 ✓"} {
		row := rowFor(t, out, branch)
		if !strings.HasSuffix(strings.TrimRight(row, " "), icons) {
			t.Errorf("row for %s should end with its icons %q, got: %q", branch, icons, row)
		}
		// BRANCH stays plain: the field right after the branch is the path.
		if p := pathField(t, row, 2); !strings.HasPrefix(p, "/") {
			t.Errorf("BRANCH cell for %s should be plain, but the next field is %q", branch, p)
		}
	}
}

func TestWorktreeListStatus_ShowStatusWins(t *testing.T) {
	newStatusFixture(t)
	out := listWorktrees(t, worktreeScope{}, worktreeListOptions{CompactStatus: true, ShowStatus: true})

	if !strings.Contains(strings.Split(out, "\n")[0], "STATUS") {
		t.Errorf("-s with -S should still show the STATUS column:\n%s", out)
	}
	row := rowFor(t, out, "ahead")
	if p := pathField(t, row, 2); !strings.HasPrefix(p, "/") {
		t.Errorf("-S should keep BRANCH plain, but the next field is %q", p)
	}
	if !strings.HasSuffix(strings.TrimRight(row, " "), "↑1 ✓") {
		t.Errorf("icons should be in the STATUS column, got: %q", row)
	}
}

func TestWorktreeListStatus_Detached(t *testing.T) {
	path, entry := detachedWorktree(t)
	f := newStatusFixture(t, entry)
	runGitIn(t, f.Repo, "worktree", "add", "--detach", path)

	full := listWorktrees(t, worktreeScope{}, worktreeListOptions{ShowStatus: true})
	row := rowFor(t, full, "(detached)")
	if !strings.HasSuffix(strings.TrimRight(row, " "), "✓") {
		t.Errorf("detached worktree should show its icon in STATUS, got: %q", row)
	}
	if p := pathField(t, row, 2); !strings.HasPrefix(p, "/") {
		t.Errorf("BRANCH should read (detached) with the path next, got: %q", row)
	}

	compact := listWorktrees(t, worktreeScope{}, worktreeListOptions{CompactStatus: true})
	row = rowFor(t, compact, "(detached)")
	if p := pathField(t, row, 2); p != "✓" {
		t.Errorf("compact mode should append the icon after (detached), got: %q", row)
	}
}

func TestWorktreeListHelp_StatusFlags(t *testing.T) {
	listed := helpListedFlags(t, worktreeListCmd)
	for _, name := range []string{"status", "show-status", "color"} {
		if !listed[name] {
			t.Errorf("help should list --%s", name)
		}
	}
	for _, want := range []string{"clean", "dirty", "ahead", "behind", "-s=dirty", "-S wins"} {
		if !strings.Contains(worktreeListCmd.Long, want) {
			t.Errorf("help text should mention %q", want)
		}
	}
}

// addDirtyRepo adds a second tracked repository with one dirty worktree to
// the saved config, for scope-then-filter tests.
func addDirtyRepo(t *testing.T, name string) {
	t.Helper()
	base := t.TempDir()
	repo := filepath.Join(base, name)
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	gitc(t, repo, "init", "-b", "main")
	gitc(t, repo, "commit", "--allow-empty", "-m", "init")
	wt := filepath.Join(base, "wt", "wip")
	gitWorktreeAdd(t, repo, "wip", wt)
	if err := os.WriteFile(filepath.Join(wt, "untracked.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Repositories = append(cfg.Repositories, config.Repository{
		Name: name, Path: repo, Worktrees: []config.Worktree{{Path: wt, Branch: "wip"}},
	})
	if err := config.Save(cfg); err != nil {
		t.Fatal(err)
	}
}

// resetWorktreeListFlags clears the command's flag state so ParseFlags can be
// called repeatedly within one test and leaves it clean afterwards.
func resetWorktreeListFlags(t *testing.T) {
	t.Helper()
	reset := func() {
		worktreeListCmd.Flags().VisitAll(func(f *pflag.Flag) { f.Changed = false })
		flagWTStatus, flagWTShowStatus, flagWTColor, flagWTWorkers = "", "", "auto", 0
	}
	reset()
	t.Cleanup(reset)
}

func TestWorktreeListStatus_FilterDirty(t *testing.T) {
	newStatusFixture(t)
	out := listWorktrees(t, worktreeScope{}, worktreeListOptions{CompactStatus: true, FilterStatus: "dirty"})

	rowFor(t, out, "dirty")
	for _, absent := range []string{"clean", "ahead"} {
		if strings.Contains(out, "fixture  "+absent) {
			t.Errorf("filter dirty should drop the %s worktree:\n%s", absent, out)
		}
	}
}

func TestWorktreeListStatus_ShowAndFilterAhead(t *testing.T) {
	newStatusFixture(t)
	out := listWorktrees(t, worktreeScope{}, worktreeListOptions{ShowStatus: true, FilterStatus: "ahead"})

	if !strings.Contains(strings.Split(out, "\n")[0], "STATUS") {
		t.Errorf("-S=ahead should still show the STATUS column:\n%s", out)
	}
	rowFor(t, out, "ahead")
	for _, absent := range []string{"clean", "dirty"} {
		if strings.Contains(out, "fixture  "+absent) {
			t.Errorf("filter ahead should drop the %s worktree:\n%s", absent, out)
		}
	}
}

func TestWorktreeListFlags_PositionalIsRepo(t *testing.T) {
	resetWorktreeListFlags(t)
	cases := []struct {
		args   []string
		filter string
		rest   []string
	}{
		{[]string{"-s", "fixture"}, "", []string{"fixture"}},
		{[]string{"-s=dirty"}, "dirty", nil},
		{[]string{"--status=dirty"}, "dirty", nil},
		{[]string{"fixture", "-S=ahead"}, "ahead", []string{"fixture"}},
	}
	for _, c := range cases {
		resetWorktreeListFlags(t)
		if err := worktreeListCmd.ParseFlags(c.args); err != nil {
			t.Fatalf("%v: unexpected parse error: %v", c.args, err)
		}
		opts := parseWorktreeListFlags(worktreeListCmd)
		if opts.FilterStatus != c.filter {
			t.Errorf("%v: FilterStatus = %q, want %q", c.args, opts.FilterStatus, c.filter)
		}
		if !opts.CompactStatus && !opts.ShowStatus {
			t.Errorf("%v: a status flag should turn status display on", c.args)
		}
		if got := worktreeListCmd.Flags().Args(); strings.Join(got, ",") != strings.Join(c.rest, ",") {
			t.Errorf("%v: positional args = %v, want %v", c.args, got, c.rest)
		}
	}

	// The joined short form is rejected, not silently misread, so the help
	// text and docs can name only the = forms.
	resetWorktreeListFlags(t)
	if err := worktreeListCmd.ParseFlags([]string{"-sdirty"}); err == nil {
		t.Error("-sdirty should be rejected: pflag reads the trailing letters as more short flags")
	}
}

func TestWorktreeListStatus_ScopeThenFilter(t *testing.T) {
	newStatusFixture(t)
	addDirtyRepo(t, "other")

	scoped := listWorktrees(t, worktreeScope{NamePattern: "fixture", Label: "fixture"},
		worktreeListOptions{CompactStatus: true, FilterStatus: "dirty"})
	rowFor(t, scoped, "dirty")
	if strings.Contains(scoped, "other") {
		t.Errorf("scope fixture should exclude the other repository:\n%s", scoped)
	}

	all := listWorktrees(t, worktreeScope{}, worktreeListOptions{CompactStatus: true, FilterStatus: "dirty"})
	if !strings.Contains(all, "other") || !strings.Contains(all, "wip") {
		t.Errorf("unscoped filter should include the other repository's dirty worktree:\n%s", all)
	}
	if strings.Contains(all, "fixture  clean") {
		t.Errorf("unscoped filter should still drop clean worktrees:\n%s", all)
	}
}

func TestWorktreeListStatus_NoStatusExcluded(t *testing.T) {
	gone := filepath.Join(t.TempDir(), "gone")
	newStatusFixture(t, config.Worktree{Path: gone, Branch: "gone", Aligned: false})

	out := listWorktrees(t, worktreeScope{}, worktreeListOptions{CompactStatus: true, FilterStatus: "clean"})
	rowFor(t, out, "clean")
	if strings.Contains(out, "gone") {
		t.Errorf("a worktree with no status must not match any filter:\n%s", out)
	}
}

func TestWorktreeListStatus_FilterEmpty(t *testing.T) {
	newStatusFixture(t)

	scoped := listWorktrees(t, worktreeScope{NamePattern: "fixture", Label: "fixture"},
		worktreeListOptions{CompactStatus: true, FilterStatus: "behind"})
	if strings.TrimSpace(scoped) != "No worktrees found for 'fixture'" {
		t.Errorf("scoped empty result should name the scope, got: %q", scoped)
	}

	all := listWorktrees(t, worktreeScope{}, worktreeListOptions{CompactStatus: true, FilterStatus: "behind"})
	if strings.TrimSpace(all) != "No worktrees found" {
		t.Errorf("unscoped empty result should be the plain message, got: %q", all)
	}
}

func TestWorktreeListStatus_CacheWritten(t *testing.T) {
	f := newStatusFixture(t)
	listWorktrees(t, worktreeScope{}, worktreeListOptions{CompactStatus: true})

	cache := readCacheFile(t)
	if cache == nil {
		t.Fatal("expected the shared status cache file to be written")
	}
	for name, path := range map[string]string{"clean": f.Clean, "dirty": f.Dirty, "ahead": f.Ahead} {
		entry := cache[path]
		if entry == nil {
			t.Errorf("cache should hold an entry keyed by the %s worktree path", name)
			continue
		}
		if entry.LastChecked.IsZero() {
			t.Errorf("cache entry for %s should carry a LastChecked time", name)
		}
		if entry.Branch != name {
			t.Errorf("cache entry for %s should record its branch, got %q", name, entry.Branch)
		}
	}
}

func TestWorktreeListStatus_CacheHit(t *testing.T) {
	newStatusFixture(t)
	first := listWorktrees(t, worktreeScope{}, worktreeListOptions{CompactStatus: true})

	// With no git binary reachable, only a cache hit can produce icons; a
	// miss would render a blank icon area.
	t.Setenv("PATH", t.TempDir())
	second := listWorktrees(t, worktreeScope{}, worktreeListOptions{CompactStatus: true})

	if second != first {
		t.Errorf("a cache hit should reproduce the first output without git\nfirst:\n%s\nsecond:\n%s", first, second)
	}
	for _, branch := range []string{"clean", "dirty", "ahead"} {
		if iconStart(rowFor(t, second, branch)) < 0 {
			t.Errorf("row for %s lost its icons on the cached run", branch)
		}
	}
}

func TestWorktreeListWorkers_Resolution(t *testing.T) {
	withPref := &config.Config{Preferences: &config.Preferences{StatusWorkers: 5}}
	noPref := &config.Config{}

	cases := []struct {
		name string
		flag int
		cfg  *config.Config
		want int
	}{
		{"flag wins over preference", 3, withPref, 3},
		{"preference wins over default", 0, withPref, 5},
		{"default is 8", 0, noPref, 8},
	}
	for _, c := range cases {
		if got := resolveWorkers(c.flag, c.cfg); got != c.want {
			t.Errorf("%s: resolveWorkers(%d) = %d, want %d", c.name, c.flag, got, c.want)
		}
	}

	resetWorktreeListFlags(t)
	if err := worktreeListCmd.ParseFlags([]string{"--workers", "3", "-s"}); err != nil {
		t.Fatal(err)
	}
	if opts := parseWorktreeListFlags(worktreeListCmd); opts.Workers != 3 {
		t.Errorf("--workers 3 should reach the options, got %d", opts.Workers)
	}
	if !helpListedFlags(t, worktreeListCmd)["workers"] {
		t.Error("help should list --workers")
	}
}
