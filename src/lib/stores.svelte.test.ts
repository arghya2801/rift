import { describe, it, expect, vi, beforeEach } from 'vitest';
import { flushSync } from 'svelte';
import { openTab, closeTab, focusTab, tabsOf, activeTab, diffTab, fileTab } from './tabs.svelte';
import { ask, sure, dialog, toast, toasts } from './menu.svelte';
import { repo, activate } from './repo.svelte';
import { app, uiOf, select, refreshRepos } from '../shell/app.svelte';
import { partials } from '../changes/partial.svelte';
import { fuzzyScore } from './fuzzy';
import { findTheme } from './themes';
import { applyPalette } from './theme';
import { Go, emit } from '../test/wails';
import * as fx from '../test/fixtures';

const tick = () => new Promise((r) => setTimeout(r, 0));

describe('tabs', () => {
  it('readers never create a group (safe inside $derived)', () => {
    let seen: unknown;
    const stop = $effect.root(() => {
      const d = $derived(tabsOf('/never') && activeTab('/never'));
      $effect(() => {
        seen = d;
      });
    });
    flushSync();
    expect(seen).toBe(null);
    stop();
  });

  it('dedupes, replaces, focuses and closes to the neighbour', () => {
    const d = '/tabs';
    openTab(d, diffTab('a.ts', 'unstaged'));
    openTab(d, fileTab('b.ts'));
    openTab(d, diffTab('a.ts', 'unstaged'));
    expect(tabsOf(d).map((t) => t.id)).toEqual(['diff:unstaged:a.ts', 'file:b.ts']);
    expect(activeTab(d)?.id).toBe('diff:unstaged:a.ts');
    openTab(d, fileTab('c.ts'));
    focusTab(d, 'file:b.ts');
    closeTab(d, 'file:b.ts');
    expect(activeTab(d)?.id).toBe('file:c.ts');
    closeTab(d, 'file:c.ts');
    closeTab(d, 'diff:unstaged:a.ts');
    expect(activeTab(d)).toBe(null);
  });

  it('a preview tab is replaced by the next preview, and pinned by a normal open', () => {
    const d = '/preview';
    openTab(d, diffTab('a.ts', 'unstaged'), true);
    openTab(d, diffTab('b.ts', 'unstaged'), true);
    expect(tabsOf(d).map((t) => t.id)).toEqual(['diff:unstaged:b.ts']);
    openTab(d, diffTab('b.ts', 'unstaged'));
    openTab(d, diffTab('c.ts', 'unstaged'), true);
    openTab(d, diffTab('b.ts', 'unstaged'), true);
    openTab(d, diffTab('d.ts', 'unstaged'), true);
    expect(tabsOf(d).map((t) => t.id)).toEqual(['diff:unstaged:b.ts', 'diff:unstaged:d.ts']);
  });

  it('stays reactive from the very first open of a worktree', () => {
    const d = '/fresh';
    const titles: string[][] = [];
    const stop = $effect.root(() => {
      $effect(() => {
        titles.push(tabsOf(d).map((t) => t.title));
      });
    });
    flushSync();
    openTab(d, fileTab('x/one.ts'));
    flushSync();
    openTab(d, fileTab('x/two.ts'));
    flushSync();
    expect(titles.at(-1)).toEqual(['one.ts', 'two.ts']);
    stop();
  });
});

describe('dialogs and toasts', () => {
  it('ask resolves the trimmed text, or empty on cancel', async () => {
    const p = ask('Name?', ' x ');
    dialog.req!.resolve(' feat ');
    expect(await p).toBe('feat');
    const q = ask('Name?');
    dialog.req!.resolve(null);
    expect(await q).toBe('');
  });

  it('sure is true on OK and false on cancel; a second dialog cancels the first', async () => {
    const first = sure('Delete?');
    const second = sure('Really?');
    expect(await first).toBe(false);
    dialog.req!.resolve('');
    expect(await second).toBe(true);
  });

  it('toasts dismiss themselves', () => {
    vi.useFakeTimers();
    toast('boom');
    toast('ok', 'info');
    expect(toasts.map((t) => t.text)).toEqual(['boom', 'ok']);
    vi.advanceTimersByTime(4001);
    expect(toasts.map((t) => t.text)).toEqual(['boom']);
    vi.advanceTimersByTime(4000);
    expect(toasts).toHaveLength(0);
    vi.useRealTimers();
  });
});

describe('repo store', () => {
  beforeEach(() => Go.Snapshot.mockClear());

  it('coalesces overlapping refreshes into at most two Snapshot calls', async () => {
    const r = repo('/coalesce');
    await tick();
    Go.Snapshot.mockClear();
    for (let i = 0; i < 10; i++) r.schedule(false);
    await tick();
    await tick();
    expect(Go.Snapshot.mock.calls.length).toBeLessThanOrEqual(2);
  });

  it('keeps refs on a status-only refresh and bumps gitRev only for git changes', async () => {
    const r = repo('/refs');
    await tick();
    expect(r.refs).toHaveLength(fx.refs.length);
    const rev = r.gitRev;
    Go.Snapshot.mockResolvedValueOnce({ ...fx.snapshot(), refs: [], full: false });
    r.schedule(false);
    await tick();
    expect(Go.Snapshot).toHaveBeenLastCalledWith('/refs', false);
    expect(r.refs).toHaveLength(fx.refs.length);
    expect(r.gitRev).toBe(rev);
    r.schedule(true);
    await tick();
    expect(r.gitRev).toBe(rev);
    expect(Go.Snapshot).toHaveBeenLastCalledWith('/refs', true);
  });

  it('coalesces staging with its watcher event and reloads history only when a ref moves', async () => {
    const r = repo('/stage-count');
    await tick();
    Go.Snapshot.mockClear();
    const rev = r.gitRev;
    await r.run(Promise.resolve(), false);
    emit('repo://changed', '/stage-count', false, true, []);
    await new Promise((resolve) => setTimeout(resolve, 350));
    expect(Go.Snapshot.mock.calls).toEqual([['/stage-count', false]]);
    expect(r.gitRev).toBe(rev);
    Go.Snapshot.mockResolvedValueOnce({ ...fx.snapshot(), refs: fx.refs.map((ref) => ({ ...ref, hash: 'f'.repeat(40) })) });
    r.schedule(true);
    await tick();
    expect(r.gitRev).toBe(rev + 1);
  });

  it('keeps the last good snapshot across a failed status refresh', async () => {
    const r = repo('/failed-status');
    await tick();
    const snapshot = r.snap;
    Go.Snapshot.mockResolvedValueOnce(Object.assign(fx.snapshot({ files: [] }), { error: 'status failed' }));
    await r.refresh();
    expect(r.error).toBe('status failed');
    expect(r.snap).toBe(snapshot);
    await r.refresh();
    expect(Go.Snapshot).toHaveBeenLastCalledWith('/failed-status', true);
    expect(r.error).toBe('');
  });

  it('routes repo://changed to the matching repo only', async () => {
    const r = repo('/routed');
    await tick();
    Go.Snapshot.mockClear();
    emit('repo://changed', '/unknown', true);
    await tick();
    expect(Go.Snapshot).not.toHaveBeenCalled();
    const rev = r.gitRev;
    emit('repo://changed', '/routed', true);
    await tick();
    expect(r.gitRev).toBe(rev);
  });

  it('loads a new worktree once and watches it once', async () => {
    Go.Snapshot.mockClear();
    Go.Watch.mockClear();
    activate('/w1');
    await tick();
    expect(Go.Snapshot).toHaveBeenCalledTimes(1);
    activate('/w1');
    expect(Go.Watch).toHaveBeenCalledTimes(1);
  });
});

describe('shell state', () => {
  it('uiOf is read-only and select creates the entry', () => {
    expect(uiOf(fx.WT)).toBe(null);
    expect(app.ui[fx.WT]).toBeUndefined();
    select(fx.worktrees[1]);
    expect(uiOf(fx.WT)).toEqual({ left: 'commit', bottom: false });
    select(null);
    expect(app.current).toBe(null);
  });

  it('drops removed worktree state on inventory refresh', async () => {
    Go.Repos.mockResolvedValueOnce(fx.repos);
    await refreshRepos();
    select(fx.worktrees[1]);
    openTab(fx.WT, fileTab('old.txt'));
    partials[fx.WT] = { 'old.txt': { text: 'partial', base: 'old' } };
    Go.Repos.mockResolvedValueOnce([]);
    await refreshRepos();
    expect(app.current).toBe(null);
    expect(tabsOf(fx.WT)).toHaveLength(0);
    expect(partials[fx.WT]).toBeUndefined();
    expect(app.ui[fx.WT]).toBeUndefined();
  });

  it('partials is a plain reactive record', () => {
    partials['/p'] = { 'a.ts': { text: 'x', base: '' } };
    expect(partials['/p']['a.ts'].text).toBe('x');
  });
});

describe('fuzzy and theme', () => {
  it('scores matches and rejects non-matches', () => {
    expect(fuzzyScore('Git: Fetch', 'gf')).not.toBe(null);
    expect(fuzzyScore('Git: Fetch', 'zz')).toBe(null);
  });

  it('applyPalette writes CSS variables', async () => {
    const el = document.createElement('div');
    applyPalette(await findTheme('Dracula'), el);
    expect(el.style.getPropertyValue('--bg')).not.toBe('');
  });
});
