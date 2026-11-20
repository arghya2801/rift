import * as Go from '../../wailsjs/go/main/App';
import type { main, git } from '../../wailsjs/go/models';
import { activate, deactivate, forgetRepo, repo } from '../lib/repo.svelte';
import { ask, sure, toast, type MenuItem } from '../lib/menu.svelte';

import { forgetTabs } from '../lib/tabs.svelte';
import { partials } from '../changes/partial.svelte';

function forget(path: string) {
  forgetRepo(path);
  forgetTabs(path);
  delete partials[path];
  delete app.ui[path];
}

type Ui = { left: 'commit' | 'project' | null; bottom: boolean };

export const app = $state({
  repos: [] as main.Repo[],
  current: null as git.Worktree | null,
  busy: '',
  ui: {} as Record<string, Ui>,
  leftWidth: 340,
  bottomHeight: 0.4,
});

// Read-only: called from $derived, so it must not create state. select() creates the entry.
export const uiOf = (dir: string): Ui | null => app.ui[dir] ?? null;

let inventoryRequest = 0;
export async function refreshRepos() {
  const id = ++inventoryRequest;
  const next = (await Go.Repos()) ?? [];
  if (id !== inventoryRequest) return;
  const byPath = new Map(next.map((r) => [r.path, r]));
  for (const old of app.repos) {
    const current = byPath.get(old.path);
    if (current?.error) continue;
    const live = new Set(current?.worktrees?.map((w) => w.path));
    for (const w of old.worktrees ?? []) {
      if (live.has(w.path)) continue;
      if (app.current?.path === w.path) select(null);
      forget(w.path);
    }
  }
  app.repos = next;
  if (app.current) {
    const fresh = next.flatMap((r) => r.worktrees ?? []).find((w) => w.path === app.current?.path);
    if (fresh && (fresh.branch !== app.current.branch || fresh.detached !== app.current.detached)) app.current = fresh;
  }
}

export async function attempt<T>(p: Promise<T>): Promise<T | undefined> {
  try {
    return await p;
  } catch (e) {
    toast(String(e));
  }
}

export function select(w: git.Worktree | null) {
  if (w?.bare) return;
  if (w) app.ui[w.path] ??= { left: 'commit', bottom: false };
  app.current = w;
  if (w) activate(w.path);
  else deactivate();
}

export async function addRepo() {
  const dir = await Go.PickFolder();
  if (dir) await attempt(Go.AddRepo(dir).then(refreshRepos));
}

export async function removeRepo(path: string) {
  await attempt(Go.RemoveRepo(path).then(refreshRepos));
}

export async function newWorktree(r: main.Repo) {
  const branch = await ask(`New worktree in ${r.name}: branch name`, '', 'Create');
  if (!branch) return;
  const local = (await Go.Refs(r.path).catch(() => [])).some((x) => x.kind === 'local' && x.name === branch);
  const path = await attempt(Go.AddWorktree(r.path, branch, '', !local));
  if (path) {
    await refreshRepos();
    const wt = app.repos.flatMap((x) => x.worktrees ?? []).find((w) => norm(w.path) === norm(path));
    if (wt) select(wt);
  }
}

export const norm = (p: string) => /^(?:[a-z]:[\\/]|\\\\)/i.test(p) ? p.replace(/\\/g, '/').toLowerCase() : p;

export const repoMenu = (r: main.Repo): MenuItem[] => [
  { label: 'New worktree…', run: () => newWorktree(r) },
  { label: 'Prune worktrees', run: () => attempt(Go.PruneWorktrees(r.path).then(refreshRepos)) },
  { label: 'Copy path', run: () => navigator.clipboard.writeText(r.path) },
  null,
  { label: 'Remove from list', danger: true, run: () => removeRepo(r.path) },
];

export const worktreeMenu = (r: main.Repo, w: git.Worktree): MenuItem[] => [
  ...(w.bare ? [] : [{ label: 'Open', run: () => select(w) }]),
  { label: 'Open in ATC', run: () => attempt(Go.OpenInATC(w.path)) },
  { label: 'Open in external editor', run: () => attempt(Go.OpenExternal(w.path, '.', 0)) },
  { label: 'Copy path', run: () => navigator.clipboard.writeText(w.path) },
  ...(w.main
    ? []
    : [
        null,
        {
          label: 'Remove worktree…',
          danger: true,
          run: async () => {
            if (!(await sure('Remove worktree?', w.path, 'Remove'))) return;
            try {
              await Go.RemoveWorktree(r.path, w.path, false);
            } catch (e) {
              if (!(await sure('Force remove worktree?', `${e}\n\nUncommitted changes there are lost.`, 'Force remove'))) return;
              try { await Go.RemoveWorktree(r.path, w.path, true); }
              catch (err) { toast(String(err)); return; }
            }
            if (app.current?.path === w.path) select(null);
            await refreshRepos();
          },
        },
      ]),
];

export function repoOf(w: git.Worktree | null): main.Repo | undefined {
  return w ? app.repos.find((r) => r.worktrees?.some((x) => x.path === w.path)) : undefined;
}

export async function remote(dir: string, label: string, p: () => Promise<string>) {
  if (app.busy) return;
  app.busy = `${label}…`;
  try {
    const out = (await repo(dir).run(p())).trim();
    toast(out.split('\n').pop() || `${label}: done`, 'info');
  } catch (e) {
    toast(String(e));
  } finally {
    app.busy = '';
  }
}
