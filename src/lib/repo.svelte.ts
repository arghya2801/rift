import * as Go from '../../wailsjs/go/main/App';
import type { main } from '../../wailsjs/go/models';
import { EventsOn } from '../../wailsjs/runtime/runtime';

// One shared, debounced snapshot per worktree. Every view reads from here instead of calling git itself.
export class Repo {
  dir: string;
  snap = $state<main.Snapshot | null>(null);
  // Bumped when HEAD/refs/index change; the log reloads only on this.
  gitRev = $state(0);
  error = $state('');
  #inflight = false;
  #again = false;
  #wantFull = true;

  constructor(dir: string) {
    this.dir = dir;
  }

  get status() {
    return this.snap?.status ?? null;
  }
  get files() {
    return this.snap?.status.files ?? [];
  }
  get refs() {
    return this.snap?.refs ?? [];
  }
  get stashes() {
    return this.snap?.stashes ?? [];
  }
  get op() {
    return this.snap?.op ?? null;
  }
  get branch() {
    return this.snap?.status.branch ?? '';
  }

  async refresh() {
    if (this.#inflight) {
      this.#again = true;
      return;
    }
    this.#inflight = true;
    const full = this.#wantFull || !this.snap;
    this.#wantFull = false;
    try {
      const s = await Go.Snapshot(this.dir, full);
      if (!full && this.snap) {
        s.refs = this.snap.refs;
        s.stashes = this.snap.stashes;
      }
      this.snap = s;
      this.error = s.error ?? '';
    } finally {
      this.#inflight = false;
      if (this.#again) {
        this.#again = false;
        this.refresh();
      }
    }
  }

  // The Go watcher already debounces; overlapping calls coalesce in refresh().
  schedule(git = false) {
    if (git) {
      this.gitRev++;
      this.#wantFull = true;
    }
    this.refresh();
  }

  // Run a mutating call, then refresh state (and the log) once. Throws the git error.
  async run<T>(p: Promise<T>): Promise<T> {
    try {
      return await p;
    } finally {
      this.schedule(true);
    }
  }
}

const repos = new Map<string, Repo>();
let active: Repo | null = null;

export function repo(dir: string): Repo {
  let r = repos.get(dir);
  if (!r) {
    r = new Repo(dir);
    repos.set(dir, r);
    r.refresh();
  }
  return r;
}

// The watched worktree; only one is watched at a time.
export function activate(dir: string): Repo {
  const r = repo(dir);
  if (active !== r) {
    active = r;
    Go.Watch(dir);
  }
  // Refresh even when returning to the same worktree: nothing was watched while away from it.
  r.schedule(true);
  return r;
}

EventsOn('repo://changed', (p: string, git: boolean) => repos.get(p)?.schedule(git));
// Focus refresh catches changes the watcher missed, at most every 3 s.
let lastFocus = 0;
window.addEventListener('focus', () => {
  if (Date.now() - lastFocus < 3000) return;
  lastFocus = Date.now();
  active?.schedule(false);
});
