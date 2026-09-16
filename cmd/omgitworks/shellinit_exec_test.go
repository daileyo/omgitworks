package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// These tests source the generated shell integration in a real shell and drive
// the omgw function against a stub binary, so routing is verified by behavior
// rather than by string matching on the templates.

const (
	stubHelpText    = "Manage git worktrees across your workspace."
	stubVersionText = "omgitworks version stub"
)

// stubBinary mimics Cobra's output conventions: help and version go to stdout,
// and navigation prints the destination path. Every non-completion invocation
// is appended to $OMGW_STUB_LOG.
const stubBinary = `#!/bin/sh
if [ "$1" = completion ]; then
  echo '# stub completion'
  exit 0
fi
printf '%s\n' "$*" >> "$OMGW_STUB_LOG"
for a in "$@"; do
  case "$a" in
    -h|--help)
      printf '` + stubHelpText + `\n\nUsage:\n  omgitworks worktree [branch-pattern] [flags]\n'
      exit 0 ;;
    --version)
      echo '` + stubVersionText + `'
      exit 0 ;;
  esac
done
case "$1 $2" in
  "worktree list"|"worktree align"|"worktree add"|"worktree remove"|"worktree rm")
    echo "stub $2"
    exit 0 ;;
esac
printf '%s\n' "$OMGW_STUB_DEST"
`

type wrapperResult struct {
	stdout, stderr, cwd, invocations string
}

type wrapperEnv struct {
	dir, bin, start, dest string
}

func newWrapperEnv(t *testing.T) wrapperEnv {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("stub binary is a POSIX shell script")
	}
	dir := t.TempDir()
	// Resolve symlinks (e.g. macOS /var -> /private/var) so cwd comparisons hold.
	dir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	env := wrapperEnv{
		dir:   dir,
		bin:   filepath.Join(dir, "omgitworks-stub"),
		start: filepath.Join(dir, "start"),
		dest:  filepath.Join(dir, "dest"),
	}
	for _, d := range []string{env.start, env.dest} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(env.bin, []byte(stubBinary), 0o755); err != nil {
		t.Fatal(err)
	}
	return env
}

func (e wrapperEnv) file(name string) string { return filepath.Join(e.dir, name) }

func (e wrapperEnv) read(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(e.file(name))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return string(b)
}

func (e wrapperEnv) writeInit(t *testing.T, shell string) string {
	t.Helper()
	script, err := renderShellInit(shell, e.bin)
	if err != nil {
		t.Fatal(err)
	}
	ext := shell
	if shell == "powershell" {
		ext = "ps1" // PowerShell only dot-sources files with a .ps1 extension
	}
	path := e.file("init." + ext)
	if err := os.WriteFile(path, []byte(script), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func (e wrapperEnv) command(t *testing.T, name string, args ...string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = e.dir
	cmd.Env = append(os.Environ(),
		"HOME="+e.dir,
		"ZDOTDIR="+e.dir,
		"OMGW_STUB_LOG="+e.file("invocations"),
		"OMGW_STUB_DEST="+e.dest,
	)
	return cmd
}

func (e wrapperEnv) result(t *testing.T) wrapperResult {
	t.Helper()
	return wrapperResult{
		stdout:      e.read(t, "stdout"),
		stderr:      e.read(t, "stderr"),
		cwd:         strings.TrimSpace(e.read(t, "cwd")),
		invocations: strings.TrimSpace(e.read(t, "invocations")),
	}
}

// runPosixWrapper sources the bash or zsh integration and runs `omgw <line>`.
// Navigation redirects through /dev/tty, so needTTY runs the shell under a pty.
func runPosixWrapper(t *testing.T, shell, line string, needTTY bool) (wrapperEnv, wrapperResult) {
	t.Helper()
	shellPath, err := exec.LookPath(shell)
	if err != nil {
		t.Skipf("%s not installed", shell)
	}
	e := newWrapperEnv(t)
	initPath := e.writeInit(t, shell)
	driver := e.file("driver.sh")
	body := strings.Join([]string{
		"source " + initPath,
		"cd " + e.start,
		"omgw " + line + " >" + e.file("stdout") + " 2>" + e.file("stderr"),
		"pwd >" + e.file("cwd"),
	}, "\n") + "\n"
	if err := os.WriteFile(driver, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	flags := "--norc --noprofile"
	if shell == "zsh" {
		flags = "-f"
	}
	var cmd *exec.Cmd
	if needTTY {
		scriptPath, err := exec.LookPath("script")
		if runtime.GOOS != "linux" || err != nil {
			t.Skip("navigation needs util-linux script(1) to provide a tty")
		}
		cmd = e.command(t, scriptPath, "-qec", shellPath+" "+flags+" "+driver, "/dev/null")
	} else {
		cmd = e.command(t, shellPath, append(strings.Fields(flags), driver)...)
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s driver failed: %v\n%s", shell, err, out)
	}
	return e, e.result(t)
}

func runPowerShellWrapper(t *testing.T, line string) (wrapperEnv, wrapperResult) {
	t.Helper()
	pwsh, err := exec.LookPath("pwsh")
	if err != nil {
		t.Skip("pwsh not installed")
	}
	e := newWrapperEnv(t)
	initPath := e.writeInit(t, "powershell")
	driver := e.file("driver.ps1")
	body := strings.Join([]string{
		". '" + initPath + "'",
		"Set-Location '" + e.start + "'",
		"omgw " + line + " >'" + e.file("stdout") + "' 2>'" + e.file("stderr") + "'",
		"(Get-Location).Path | Set-Content '" + e.file("cwd") + "'",
	}, "\n") + "\n"
	if err := os.WriteFile(driver, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := e.command(t, pwsh, "-NoProfile", "-NonInteractive", "-File", driver)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("pwsh driver failed: %v\n%s", err, out)
	}
	return e, e.result(t)
}

func runWrapper(t *testing.T, shell, line string, needTTY bool) (wrapperEnv, wrapperResult) {
	t.Helper()
	if shell == "powershell" {
		return runPowerShellWrapper(t, line)
	}
	return runPosixWrapper(t, shell, line, needTTY)
}

var wrapperShells = []string{"zsh", "bash", "powershell"}

func TestShellWrapperWorktreePassthrough(t *testing.T) {
	cases := []struct {
		line string
		want string // expected in stdout
	}{
		{"worktree -h", stubHelpText},
		{"worktree --help", stubHelpText},
		{"worktree --version", stubVersionText},
		{"worktree navigate --help", stubHelpText},
		{"worktree feat-auth -h", stubHelpText},
		{"worktree list", "stub list"},
		{"worktree align", "stub align"},
		{"worktree add my-repo feat-auth", "stub add"},
		{"worktree remove my-repo feat-auth", "stub remove"},
		{"worktree rm feat-auth", "stub rm"},
		{"worktree remove -t backend feat-auth --dry-run", "stub remove"},
	}
	for _, shell := range wrapperShells {
		for _, tc := range cases {
			t.Run(shell+"/"+tc.line, func(t *testing.T) {
				e, res := runWrapper(t, shell, tc.line, false)
				if !strings.Contains(res.stdout, tc.want) {
					t.Errorf("stdout = %q, want it to contain %q", res.stdout, tc.want)
				}
				if res.stderr != "" {
					t.Errorf("unexpected stderr: %q", res.stderr)
				}
				if res.cwd != e.start {
					t.Errorf("cwd = %q, want unchanged %q", res.cwd, e.start)
				}
				if res.invocations != tc.line {
					t.Errorf("binary invoked with %q, want %q (no -q)", res.invocations, tc.line)
				}
			})
		}
	}
}

func TestShellWrapperWorktreeNavigation(t *testing.T) {
	for _, shell := range wrapperShells {
		t.Run(shell, func(t *testing.T) {
			e, res := runWrapper(t, shell, "worktree feat-auth", true)
			if res.cwd != e.dest {
				t.Errorf("cwd = %q, want %q", res.cwd, e.dest)
			}
			if res.invocations != "worktree feat-auth -q" {
				t.Errorf("binary invoked with %q, want %q", res.invocations, "worktree feat-auth -q")
			}
		})
	}
}
