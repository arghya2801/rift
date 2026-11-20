import { vi } from 'vitest';
import * as fx from './fixtures';

// Mock of every Wails binding. Tests override per call with Go.X.mockResolvedValueOnce(...).
export const Go = {
  AcceptSide: vi.fn(async () => {}),
  AddRepo: vi.fn(async (d: string) => d),
  AddToGitignore: vi.fn(async () => {}),
  AddWorktree: vi.fn(async () => fx.WT),
  Changelists: vi.fn(async () => ({})),
  Checkout: vi.fn(async () => {}),
  CheckoutRemote: vi.fn(async () => {}),
  CherryPick: vi.fn(async () => {}),
  Commit: vi.fn(async () => {}),
  CommitFiles: vi.fn(async () => [{ status: 'M', path: 'src/changed.ts' }]),
  CommitMessage: vi.fn(async () => 'Merge feat\n\nbody'),
  CommitPartial: vi.fn(async () => null),
  RepairPartialIndex: vi.fn(async () => {}),
  CommitSides: vi.fn(async () => ({ old: 'a\n', new: 'b\n', binary: false })),
  ConflictSides: vi.fn(async () => ({ stages: 'original stages', markerSize: 7, base: 'x\n', ours: 'o\n', theirs: 't\n', merged: '<<<<<<< ours\no\n||||||| base\nx\n=======\nt\n>>>>>>> theirs\n' })),
  CreateBranch: vi.fn(async () => {}),
  CreateTag: vi.fn(async () => {}),
  DeleteBranch: vi.fn(async () => {}),
  Discard: vi.fn(async () => {}),
  Fetch: vi.fn(async () => 'fetched'),
  FileSides: vi.fn(async () => ({ old: 'one\ntwo\n', new: 'one\nTWO\n', binary: false })),
  GetMode: vi.fn(async () => 'staging'),
  HeadSides: vi.fn(async () => ({ old: 'one\n', new: 'two\n', binary: false })),
  Initial: vi.fn(async () => ''),
  LastMessage: vi.fn(async () => 'last message\n'),
  ListDir: vi.fn(async (_d: string, rel: string) => (rel ? [{ name: 'changed.ts', isDir: false }] : [{ name: 'src', isDir: true }, { name: 'notes.txt', isDir: false }])),
  Log: vi.fn(async () => fx.logRows),
  Merge: vi.fn(async () => {}),
  OpAbort: vi.fn(async () => {}),
  OpContinue: vi.fn(async () => {}),
  OpSkip: vi.fn(async () => {}),
  OpState: vi.fn(async () => ({ kind: '', conflicts: [] })),
  OpenExternal: vi.fn(async () => {}),
  OpenInATC: vi.fn(async () => {}),
  Overview: vi.fn(async () => []),
  PickFolder: vi.fn(async () => ''),
  PruneWorktrees: vi.fn(async () => {}),
  Pull: vi.fn(async () => 'pulled'),
  Push: vi.fn(async () => 'pushed'),
  ReadFile: vi.fn(async () => 'file text\n'),
  Rebase: vi.fn(async () => {}),
  RebaseTodo: vi.fn(async () => []),
  Refs: vi.fn(async () => fx.refs),
  RemoveRepo: vi.fn(async () => {}),
  RemoveWorktree: vi.fn(async () => {}),
  RenameBranch: vi.fn(async () => {}),
  Repos: vi.fn(async () => fx.repos),
  Reset: vi.fn(async () => {}),
  ResolveFile: vi.fn(async () => {}),
  Revert: vi.fn(async () => {}),
  RunRebase: vi.fn(async () => {}),
  SetChangelists: vi.fn(async () => {}),
  SetMode: vi.fn(async () => {}),
  Snapshot: vi.fn(async (_d: string, _full: boolean) => fx.snapshot()),
  Stage: vi.fn(async () => {}),
  StageContent: vi.fn(async () => {}),
  StashAction: vi.fn(async () => {}),
  StashPush: vi.fn(async () => {}),
  Stashes: vi.fn(async () => fx.stashes),
  Status: vi.fn(async () => fx.snapshot().status),
  Unstage: vi.fn(async () => {}),
  Watch: vi.fn(async () => {}),
  SaveFile: vi.fn(async (_dir: string, _path: string, _text: string, _expected: string) => {}),
  WriteFile: vi.fn(async () => {}),
};

type Handler = (...args: unknown[]) => void;
export const handlers = new Map<string, Set<Handler>>();

export const runtime = {
  EventsOn: vi.fn((name: string, fn: Handler) => {
    if (!handlers.has(name)) handlers.set(name, new Set());
    handlers.get(name)!.add(fn);
    return () => handlers.get(name)?.delete(fn);
  }),
  EventsEmit: vi.fn(),
  EventsOff: vi.fn(),
};

export function emit(name: string, ...args: unknown[]) {
  for (const fn of handlers.get(name) ?? []) fn(...args);
}
