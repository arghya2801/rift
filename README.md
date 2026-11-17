# Rift

A small desktop git client built with Go, Wails and Svelte. It covers the git tooling of an IDE (log graph, partial staging, conflict resolution, interactive rebase) without the IDE, plus a light editor for quick fixes. It is built for moving between many repos and worktrees.

## Features

- Workspaces: pinned repos with their worktrees, a dashboard of every worktree's branch, changes and ahead/behind, and add/remove/prune worktree actions
- Staging by file, hunk or line, with commit and amend
- Changelists mode as an alternative to the staging area
- Log with a commit graph, branch and tag tree, filters and commit details
- Branch actions: checkout, create, rename, delete, merge, rebase, cherry-pick, revert, reset, tag, fetch, pull, push, stash
- Three-way conflict resolver for merge, rebase, cherry-pick and revert
- Interactive rebase: reorder, pick, reword, squash, fixup, edit, drop
- File tree and CodeMirror editor
- Open a worktree in ATC (`atc --open <dir>`), or a file in an external editor. The `RIFT_EDITOR` template sets the editor, default `code -g {path}:{line}`

## Requirements

- git 2.30 or newer on PATH
- Go, Node and the Wails CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@v2.16.0`

## Install

Download `Rift_<version>_x64-setup.exe` (per-user install, no admin) or the portable `rift.exe` from the GitHub releases page. The binaries are unsigned, so SmartScreen warns on first run.

## Releasing

Set the version in `package.json` and `wails.json`, move the **Unreleased** notes in `CHANGELOG.md` under it, then push a `v<version>` tag. The Release workflow builds the exe and installer, waits for CI, and leaves a draft release to publish.

## Development

```
npm install
npm run desktop:dev     # dev app with hot reload
npm run desktop:build   # build/bin/rift(.exe)
npm test                # vitest and go test
npm run test:e2e        # drives the dev app in headless Edge/Chrome and checks git state on disk
npm run play            # scratch repo with branches, a worktree and a merge conflict
```
