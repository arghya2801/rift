const c = (s: string) => s.charCodeAt(0);

export const MAIN = 'C:/repos/demo';
export const WT = 'C:/repos/demo-feat';

export const worktrees = [
  { path: MAIN, head: 'a'.repeat(40), branch: 'main', detached: false, bare: false, locked: false, main: true },
  { path: WT, head: 'b'.repeat(40), branch: 'feat', detached: false, bare: false, locked: false, main: false },
];

export const repos = [{ path: MAIN, name: 'demo', worktrees }];

export const files = [
  { path: 'src/staged.ts', index: c('M'), work: c('.'), conflict: false },
  { path: 'src/changed.ts', index: c('.'), work: c('M'), conflict: false },
  { path: 'notes.txt', index: c('?'), work: c('?'), conflict: false },
  { path: 'shared.txt', index: c('U'), work: c('U'), conflict: true },
];

export const refs = [
  { name: 'main', full: 'refs/heads/main', kind: 'local', hash: 'a'.repeat(40), upstream: 'origin/main', track: '[ahead 1]', current: true },
  { name: 'feat', full: 'refs/heads/feat', kind: 'local', hash: 'b'.repeat(40), upstream: '', track: '', current: false },
  { name: 'origin/main', full: 'refs/remotes/origin/main', kind: 'remote', hash: 'c'.repeat(40), upstream: '', track: '', current: false },
  { name: 'v1.0', full: 'refs/tags/v1.0', kind: 'tag', hash: 'c'.repeat(40), upstream: '', track: '', current: false },
];

export const stashes = [{ ref: 'stash@{0}', subject: 'WIP on main: tweak', time: 1_700_000_000 }];

export function snapshot(over: { op?: string; files?: typeof files } = {}) {
  const f = over.files ?? files;
  return {
    status: { branch: 'main', upstream: 'origin/main', ahead: 1, behind: 0, files: f },
    refs,
    stashes,
    op: { kind: over.op ?? '', conflicts: f.filter((x) => x.conflict).map((x) => x.path) },
    full: true,
  };
}

export const logRows = [
  { hash: '1'.repeat(40), parents: ['2'.repeat(40), '3'.repeat(40)], author: 'Ada', email: 'ada@x', time: 1_700_000_300, refs: ['HEAD -> main', 'origin/main'], subject: 'Merge feat', col: 0, lines: [{ from: 0, to: 0, bottom: true }, { from: 0, to: 1, bottom: true }] },
  { hash: '3'.repeat(40), parents: ['4'.repeat(40)], author: 'Bob', email: 'bob@x', time: 1_700_000_200, refs: ['feat'], subject: 'Add feature', col: 1, lines: [{ from: 0, to: 0, bottom: false }, { from: 1, to: 1, bottom: false }, { from: 1, to: 0, bottom: true }, { from: 0, to: 0, bottom: true }] },
  { hash: '2'.repeat(40), parents: ['4'.repeat(40)], author: 'Ada', email: 'ada@x', time: 1_700_000_100, refs: [], subject: 'Fix bug', col: 0, lines: [{ from: 0, to: 0, bottom: false }, { from: 0, to: 0, bottom: true }] },
  { hash: '4'.repeat(40), parents: [], author: 'Ada', email: 'ada@x', time: 1_700_000_000, refs: ['tag: v1.0'], subject: 'Initial', col: 0, lines: [{ from: 0, to: 0, bottom: false }, { from: 1, to: 0, bottom: false }] },
];
