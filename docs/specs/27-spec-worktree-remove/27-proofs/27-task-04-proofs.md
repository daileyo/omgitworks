# Task 04 Proofs - Dry run and the confirmation gate

## Task Summary

Nothing is deleted that the user has not seen: one plan structure feeds the dry run, the
confirmation prompt, and the removal itself.

## What This Task Proves

- `--dry-run` previews and changes nothing — not a worktree, not a directory, not a byte
  of `config.json`.
- The preview marks locked entries as skipped, dirty entries as would-fail, and states the
  `--force` posture.
- A multi-worktree run prompts; declining removes nothing and exits zero.
- `--yes` skips the prompt without reading stdin.
- A non-interactive stdin without `--yes` exits non-zero having removed nothing.
- A single-worktree run is not prompted.
- The confirmation listing is **textually identical** to the dry-run listing.

## Evidence Summary

- 11 cases pass, including a byte-comparison of `config.json` across a dry run.
- CLI: the non-interactive refusal exits 1 with nothing removed.

## Artifact: Preview, refusal, and the real run

**What it proves:** The three paths behave as specified against a real binary.

**Result summary:** The dry run lists both worktrees and removes nothing. With stdin not a
terminal and no `--yes`, the command prints the plan and then refuses, exit 1. With
`--yes`, both are removed and `worktree list` is empty.

```text
$ omgw worktree remove -t backend feat-auth --dry-run
Dry run — no changes will be made:

--force is not set: worktrees with uncommitted changes will fail.

Would remove [svc-a] feat-auth
  path: <sandbox>/data/gws/projects/svc-a/feat-auth

Would remove [svc-b] feat-auth
  path: <sandbox>/data/gws/projects/svc-b/feat-auth

Total: 2 worktrees

$ omgw worktree remove -t backend feat-auth   # stdin not a tty
...
Error: confirmation required but stdin is not a terminal; pass --yes to proceed
   exit=1

$ omgw worktree remove -t backend feat-auth --yes
Removed worktree for branch 'feat-auth' from svc-a
Removed worktree for branch 'feat-auth' from svc-b

Removed 2 worktrees
$ omgw worktree list
No worktrees found
```

## Artifact: Dry-run and confirmation suite

**What it proves:** Every Unit 3 requirement, plus preview fidelity.

**Why it matters:** `TestWorktreeRemove_DryRunChangesNothing` compares `config.json` bytes
before and after, so a dry run that quietly rewrote configuration would fail.
`TestWorktreeRemove_PreviewMatchesRun` asserts the paths named in the preview are exactly
what the real run removes — success metric 4.

**Command:**

```bash
go test -v -run 'TestWorktreeRemove_' ./cmd/omgitworks/
```

**Result summary:** All pass.

```text
--- PASS: TestWorktreeRemove_DryRunChangesNothing (0.04s)
--- PASS: TestWorktreeRemove_DryRunAnnotations (0.04s)
--- PASS: TestWorktreeRemove_ConfirmationGate (0.09s)
    --- PASS: TestWorktreeRemove_ConfirmationGate/declining_removes_nothing_and_returns_nil (0.04s)
    --- PASS: TestWorktreeRemove_ConfirmationGate/accepting_proceeds (0.05s)
--- PASS: TestWorktreeRemove_YesSkipsPrompt (0.05s)
--- PASS: TestWorktreeRemove_NonInteractiveRefuses (0.04s)
--- PASS: TestWorktreeRemove_SingleNotPrompted (0.02s)
--- PASS: TestWorktreeRemove_DryRunWithYes (0.04s)
--- PASS: TestWorktreeRemove_PreviewMatchesRun (0.06s)
--- PASS: TestWorktreeRemove_ConfirmListingMatchesDryRun (0.05s)
ok  	github.com/daileyo/omgitworks/cmd/omgitworks	0.429s
```

## Deviation found and corrected during implementation

The first working version rendered the confirmation listing with "Removing" while
`--dry-run` rendered "Would remove". The spec's Design Considerations require the two
listings to be textually the same, so the preview a user approves matches what
`--dry-run` shows.

Both are previews of work not yet done, so both now use "Would remove".
`TestWorktreeRemove_ConfirmListingMatchesDryRun` compares the two listings directly and
would fail if they diverged again.

## Defects found in validation and fixed

Validation found two defects no implementation-phase test covered:

- **Preview fidelity.** The dry run omitted repositories that would be skipped for lacking
  the branch, and counted locked worktrees as removals while the real run counted them as
  skips. The preview now shows `Would skip — ...` lines and a `Total: N to remove, M
  skipped` footer matching the real summary. Guard: `TestWorktreeRemove_PreviewShowsSkips`.
- **Cleanup containment** (task 2.0's code). See `27-validation-worktree-remove.md`.

## Reviewer Conclusion

The destructive path cannot run unseen: preview and confirmation share one rendering, a
dry run provably changes nothing, and a non-interactive stream is refused rather than
consumed as consent.
