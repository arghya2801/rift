# Changelog

Notable changes to Rift. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow [Semantic Versioning](https://semver.org/).

Add a line under **Unreleased** with each change. When cutting a release, rename that section to the version and paste it into the GitHub release body.

## [Unreleased]

## [0.2.0] - 2026-11-20

### Fixed

- Editor saves preserve edits made during a save and detect changes on disk. The conflict resolver keeps drafts and checks that the Git stages still match before applying a resolution.
- Rollback and rename actions respect the selected paths, including filenames with Git metacharacters. Changelist commits preserve unrelated staged content and support index repair after a successful commit.
- Rebase, history and worktree views handle interrupted operations, empty repositories and stale refreshes. The Windows installer leaves unrelated Rift sessions running.

### Changed

- Selecting lines in large diffs, typing in large files and drawing wide commit graphs take less time. Changelist commits update the Git index in batches.
- History loads one page at a time, and long file and rebase lists render fewer rows at once. The dashboard limits how many worktree summaries it fetches at once.

## [0.1.0] - 2026-11-17

First release.

### Added
- Pinned repos with their worktrees, a dashboard of every worktree, and add, remove and prune worktree actions.
- Commit tool window: stage by file, hunk or selected lines, commit, amend, commit and push, and rollback.
- Changelists mode as an alternative to the staging area, set from the Commit tool window.
- Diff tabs, side-by-side or unified, with jump to next change and open file at the cursor line.
- Git log with a commit graph, a branch, tag and stash tree, filters and commit details.
- Branch actions: checkout, create, rename, delete, merge, rebase, cherry-pick, revert, reset, tag, fetch, pull and push.
- Three-way conflict resolver for merge, rebase, cherry-pick and revert.
- Interactive rebase: reorder, pick, reword, squash, fixup, edit and drop. Uncommitted changes are stashed and restored.
- Project file tree and a CodeMirror editor that saves on Ctrl+S and when you leave the tab.
- Command palette with themes, Open in ATC, and Open in external editor through `RIFT_EDITOR`.
- Windows installer and portable `rift.exe`.
