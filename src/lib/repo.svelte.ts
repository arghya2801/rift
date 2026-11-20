import * as Go from '../../wailsjs/go/main/App';
import type { main } from '../../wailsjs/go/models';
import { EventsOn } from '../../wailsjs/runtime/runtime';

// One shared, debounced snapshot per worktree. Every view reads from here instead of calling git itself.
export class Repo {
  dir: string;
  snap = $state<main.Snapshot | null>(null);
  // Only a changed history tip or decoration invalidates the log.
  gitRev = $state(0);
  error = $state('');
  #disposed = false;
  #inflight = false;
  #again = false;
  #wantFull = true;
  #pending?: ReturnType<typeof setTimeout>;
  #pendingFull = false;
  #listeners = new Set<(refs: boolean, index: boolean, paths?: string[] | null) => void>();

  onFiles(listener: (refs: boolean, index: boolean, paths?: string[] | null) => void) {
    this.#listeners.add(listener);
    return () => { this.#listeners.delete(listener); };
  }

  dispose() {
    this.#disposed = true;
    clearTimeout(this.#pending);
    this.#listeners.clear();
  }

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
    if (this.#disposed) return;
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
      if (this.snap && !s.error) {
        const previous = this.snap;
        const moved = previous.status.head !== s.status.head || previous.status.branch !== s.status.branch;
        const refsChanged = full && (s.refs.length !== previous.refs.length || s.refs.some((ref, i) =>
          ref.full !== previous.refs[i].full || ref.hash !== previous.refs[i].hash));
        if (moved || refsChanged) this.gitRev++;
      }
      this.error = s.error ?? '';
      if (this.error) this.#wantFull = true;
      else this.snap = s;
    } catch (e) {
      this.error = String(e);
      this.#wantFull = true;
    } finally {
      this.#inflight = false;
      if (this.#again) {
        this.#again = false;
        this.refresh();
      }
    }
  }

  // The Go watcher already debounces; overlapping calls coalesce in refresh().
  schedule(refs = false, index = true, paths?: string[] | null) {
    // A watcher event also satisfies a mutation's pending status refresh.
    if (this.#pending !== undefined) {
      clearTimeout(this.#pending);
      this.#pending = undefined;
      refs ||= this.#pendingFull;
      index = true;
      this.#pendingFull = false;
    }
    if (refs) this.#wantFull = true;
    for (const listener of this.#listeners) listener(refs, index, paths);
    this.refresh();
  }

  async run<T>(p: Promise<T>, refs = true): Promise<T> {
    try {
      return await p;
    } finally {
      this.#pendingFull ||= refs;
      clearTimeout(this.#pending);
      if (!this.#disposed) this.#pending = setTimeout(() => this.schedule(this.#pendingFull), 320);
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
  const cached = repos.has(dir);
  const r = repo(dir);
  if (active === r) return r;
  active = r;
  Go.Watch(dir);
  // A cached worktree may have changed while another worktree was watched.
  if (cached) r.schedule(true);
  return r;
}

export function deactivate() {
  if (active) Go.Watch('');
  active = null;
}

export function forgetRepo(dir: string) {
  const r = repos.get(dir);
  if (r && r === active) deactivate();
  r?.dispose();
  repos.delete(dir);
}

export function touches(path: string, paths?: string[] | null) {
  return !paths || paths.some((p) => p === '.' || p === path || path.startsWith(p + '/'));
}

EventsOn('repo://changed', (p: string, refs: boolean, index: boolean, paths?: string[]) => repos.get(p)?.schedule(refs, index, paths));
// Focus refresh catches changes the watcher missed, at most every 3 s.
let lastFocus = 0;
window.addEventListener('focus', () => {
  if (Date.now() - lastFocus < 3000) return;
  lastFocus = Date.now();
  active?.schedule(true);
});
