# Rift

Lightweight git client for Windows, macOS and Linux. Go + Wails v2 host (`main.go`, `app*.go`, `internal/`), Svelte 5 + TS + CodeMirror 6 frontend (`src/`). Git is the `git` CLI, never go-git.

## Commands

- `npm run desktop:dev`: dev app.
- `npm run play`: scratch repo with branches, a worktree and a merge conflict to open in the app.
- `npm test`, `npm run check`: run both before any PR.
- `npm run desktop:build`: release build in `build/bin/`.

## Rules

- Keep it small. 5 lines beats 50. No comments that repeat the code, no features nobody asked for.
- Every exported method on `App` is a command. Regenerate bindings with `wails generate module` after changing them; `wailsjs/go` is committed and CI fails if it drifts.
- `main.go` embeds `dist/`. Run `npm run build` before bare `go build` or `go test`.
- Git calls go through `internal/git` `Run`/`RunEnv`, which set `core.quotepath=off` and `GIT_TERMINAL_PROMPT=0`.
- Diff sides are in working-tree form (`git cat-file --filters`), so CRLF repos diff cleanly.

## Gotchas

- Windows file names are case-insensitive. `Menu.svelte` next to `menu.svelte.ts` breaks imports; keep names distinct.
- Tests must pin `core.autocrlf` in their temp repos, or the machine's global config changes the results.
- On a fresh clone, `wails generate module` needs a `dist/` folder first (a placeholder `dist/index.html` is enough).
- Runes only work in `.svelte` and `.svelte.ts`.
