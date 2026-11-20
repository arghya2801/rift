<script lang="ts">
  import * as Go from '../../wailsjs/go/main/App';
  import type { git } from '../../wailsjs/go/models';
  import { repo } from '../lib/repo.svelte';
  import { openTab, rebaseTab } from '../lib/tabs.svelte';
  import { openMenu, ask, sure, toast } from '../lib/menu.svelte';
  import Icon from '../lib/Icon.svelte';
  import Graph, { LANE, ROW } from './Graph.svelte';
  import Branches from './Branches.svelte';
  import Details from './Details.svelte';

  let { dir }: { dir: string } = $props();

  const r = $derived(repo(dir));
  let rows = $state.raw<git.Row[]>([]);
  let more = $state(false);
  let laneState: string[] = [];
  let generation = 0;
  let loading = $state(false);
  let filter = $state('');
  let branch = $state('');
  let selected = $state<git.Row | null>(null);
  let scrollTop = $state(0);
  let height = $state(400);
  let list: HTMLDivElement;
  let leftW = $state(220);
  let rightW = $state(340);

  async function load(append = false) {
    const id = ++generation;
    loading = true;
    try {
      const offset = append ? rows.length : 0;
      const page = await Go.LogPage(dir, offset, append ? 3000 : 2000, offset ? rows[offset - 1].hash : '', append ? laneState : []);
      if (id !== generation) return;
      if (page.restart) { await load(); return; }
      rows = append ? rows.concat(page.rows) : page.rows;
      laneState = page.lanes ?? [];
      more = page.more;
      if (selected) selected = rows.find((x) => x.hash === selected!.hash) ?? null;
    } catch (e) {
      if (id === generation) toast(String(e));
    } finally {
      if (id === generation) loading = false;
    }
  }

  $effect(() => {
    r.gitRev;
    const t = setTimeout(() => load(), 150);
    return () => { generation++; clearTimeout(t); };
  });

  const act = (p: Promise<unknown>) => r.run(p).catch((e) => toast(String(e)));

  const byHash = $derived(new Map(rows.map((x) => [x.hash, x])));
  const lanes = $derived.by(() => {
    let max = 0;
    for (const x of rows) {
      if (x.col > max) max = x.col;
      for (const l of x.lines) max = Math.max(max, l.from, l.to);
    }
    return max + 1;
  });

  const reachable = $derived.by(() => {
    if (!branch) return null;
    const start = r.refs.find((x) => x.name === branch)?.hash;
    const head = start && byHash.has(start) ? start : rows.find((x) => x.refs.some((ref) => ref.replace(/^(HEAD -> |tag: )/, '') === branch))?.hash;
    const seen = new Set<string>();
    const stack = head ? [head] : [];
    while (stack.length) {
      const h = stack.pop()!;
      if (seen.has(h)) continue;
      seen.add(h);
      for (const p of byHash.get(h)?.parents ?? []) stack.push(p);
    }
    return seen;
  });

  const q = $derived(filter.trim().toLowerCase());
  const shown = $derived(
    rows.filter(
      (x) =>
        (!reachable || reachable.has(x.hash)) &&
        (!q || x.subject.toLowerCase().includes(q) || x.author.toLowerCase().includes(q) || x.hash.startsWith(q) || x.refs.some((ref) => ref.toLowerCase().includes(q))),
    ),
  );
  const plain = $derived(!!q || !!branch);
  const start = $derived(Math.max(0, Math.floor(scrollTop / ROW) - 10));
  const visible = $derived(shown.slice(start, start + Math.ceil(height / ROW) + 20));
  const visibleToday = $derived.by(() => { visible; return new Date().toDateString(); });
  const timeFormat = new Intl.DateTimeFormat(undefined, { timeStyle: 'short' });
  const dateFormat = new Intl.DateTimeFormat(undefined, { dateStyle: 'short', timeStyle: 'short' });
  const remoteNames = $derived(new Set(r.refs.filter((x) => x.kind === 'remote').map((x) => x.name)));
  const date = (t: number, today: string) => {
    const d = new Date(t * 1000);
    return d.toDateString() === today ? `Today ${timeFormat.format(d)}` : dateFormat.format(d);
  };

  function refClass(ref: string) {
    if (ref.startsWith('HEAD')) return 'head';
    if (ref.startsWith('tag: ')) return 'tag';
    return remoteNames.has(ref) ? 'remote' : 'local';
  }

  function select(x: git.Row | undefined, scroll = false) {
    if (!x) return;
    selected = x;
    if (!scroll) return;
    const i = shown.indexOf(x);
    if (i < 0) return;
    const top = i * ROW;
    if (top < list.scrollTop) list.scrollTop = top;
    else if (top + ROW > list.scrollTop + height) list.scrollTop = top + ROW - height;
  }

  function keys(e: KeyboardEvent) {
    if (e.key !== 'ArrowDown' && e.key !== 'ArrowUp') return;
    e.preventDefault();
    const i = selected ? shown.indexOf(selected) : -1;
    select(shown[Math.min(shown.length - 1, Math.max(0, i + (e.key === 'ArrowDown' ? 1 : -1)))], true);
  }

  function commitMenu(e: MouseEvent, x: git.Row) {
    selected = x;
    const short = x.hash.slice(0, 8);
    const cur = r.branch || 'HEAD';
    openMenu(e, [
      { label: 'Copy Revision Number', run: () => navigator.clipboard.writeText(x.hash) },
      null,
      { label: 'Checkout Revision', run: () => act(Go.Checkout(dir, x.hash)) },
      {
        label: 'New Branch…',
        run: async () => {
          const n = await ask('New branch name', '', 'Create');
          if (n) act(Go.CreateBranch(dir, n, x.hash, true));
        },
      },
      {
        label: 'New Tag…',
        run: async () => {
          const n = await ask('New tag name', '', 'Create');
          if (n) act(Go.CreateTag(dir, n, x.hash));
        },
      },
      null,
      { label: 'Cherry-Pick', run: () => act(Go.CherryPick(dir, x.hash)) },
      { label: 'Revert Commit', run: () => act(Go.Revert(dir, x.hash)) },
      ...(x.parents.length ? [{ label: 'Interactively Rebase from Here…', run: () => openTab(dir, rebaseTab(x.parents[0])) }] : []),
      null,
      { label: `Reset '${cur}' to Here (Soft)`, run: () => act(Go.Reset(dir, 'soft', x.hash)) },
      { label: `Reset '${cur}' to Here (Mixed)`, run: () => act(Go.Reset(dir, 'mixed', x.hash)) },
      {
        label: `Reset '${cur}' to Here (Hard)`,
        danger: true,
        run: async () => (await sure(`Hard reset '${cur}' to ${short}?`, 'Uncommitted changes will be lost.', 'Reset')) && act(Go.Reset(dir, 'hard', x.hash)),
      },
    ]);
  }

  function drag(e: PointerEvent, side: 'left' | 'right') {
    const startX = e.clientX;
    const w0 = side === 'left' ? leftW : rightW;
    const move = (m: PointerEvent) => {
      const d = m.clientX - startX;
      if (side === 'left') leftW = Math.min(500, Math.max(140, w0 + d));
      else rightW = Math.min(700, Math.max(200, w0 - d));
    };
    const up = () => {
      window.removeEventListener('pointermove', move);
      window.removeEventListener('pointerup', up);
    };
    window.addEventListener('pointermove', move);
    window.addEventListener('pointerup', up);
  }
</script>

<div class="log">
  <div class="col" style="width:{leftW}px">
    <Branches {dir} selected={branch} onpick={(b) => (branch = b)} />
  </div>
  <div class="split" role="separator" aria-orientation="vertical" onpointerdown={(e) => drag(e, 'left')}></div>
  <div class="center">
    <div class="toolbar">
      <div class="search">
        <Icon name="search" size={14} />
        <input placeholder="Text or hash" bind:value={filter} />
      </div>
      <label class="branch">
        <Icon name="branch" size={14} />
        <select bind:value={branch}>
          <option value="">All branches</option>
          {#each r.refs.filter((x) => x.kind !== 'tag') as x (x.full)}<option value={x.name}>{x.name}</option>{/each}
        </select>
      </label>
      <span class="grow"></span>
      <span class="dim small">{shown.length} commits</span>
      <button class="icon" title="Refresh" onclick={() => load()} disabled={loading}><Icon name="refresh" /></button>
    </div>
    <div class="thead">
      <span class="c-subject" style="padding-left:{plain ? 8 : lanes * LANE + 12}px">Subject</span>
      <span class="c-author">Author</span>
      <span class="c-date">Date</span>
    </div>
    <div class="list" tabindex="0" role="grid" bind:this={list} bind:clientHeight={height} onscroll={(e) => (scrollTop = e.currentTarget.scrollTop)} onkeydown={keys}>
      <div style="height:{(shown.length + (more ? 1 : 0)) * ROW}px; position:relative">
        {#each visible as x, i (x.hash)}
          <div
            class="row"
            class:sel={selected?.hash === x.hash}
            class:head={x.refs.some((ref) => ref.startsWith('HEAD'))}
            style="top:{(start + i) * ROW}px"
            role="row"
            tabindex="-1"
            aria-selected={selected?.hash === x.hash}
            onclick={() => select(x)}
            oncontextmenu={(e) => commitMenu(e, x)}
            onkeydown={() => {}}
          >
            {#if !plain}<Graph row={x} width={lanes * LANE + 4} />{/if}
            <span class="c-subject">
              {#each x.refs as ref}<span class="ref {refClass(ref)}">{ref.replace(/^tag: /, '')}</span>{/each}
              <span class="text">{x.subject}</span>
            </span>
            <span class="c-author">{x.author}</span>
            <span class="c-date">{date(x.time, visibleToday)}</span>
          </div>
        {/each}
        {#if more}
          <button class="more" style="top:{shown.length * ROW}px" onclick={() => load(true)} disabled={loading}>{loading ? 'Loading…' : `Load more commits (showing ${rows.length})`}</button>
        {/if}
      </div>
    </div>
  </div>
  <div class="split" role="separator" aria-orientation="vertical" onpointerdown={(e) => drag(e, 'right')}></div>
  <div class="col" style="width:{rightW}px">
    {#if selected}
      <Details {dir} row={selected} onparent={(h) => select(byHash.get(h), true)} />
    {:else}
      <div class="empty dim">Select a commit to view changes</div>
    {/if}
  </div>
</div>

<style>
  .log {
    display: flex;
    flex: 1;
    height: 100%;
    min-height: 0;
    background: var(--bg-panel);
  }
  .col {
    flex-shrink: 0;
    min-width: 0;
    display: flex;
    flex-direction: column;
  }
  .split {
    width: 1px;
    flex-shrink: 0;
    background: var(--line);
    cursor: col-resize;
    position: relative;
  }
  .split::after {
    content: '';
    position: absolute;
    inset: 0 -3px;
  }
  .center {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
  }
  .toolbar {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 4px 8px;
    border-bottom: 1px solid var(--line);
  }
  .search,
  .branch {
    display: flex;
    align-items: center;
    gap: 5px;
    color: var(--fg-dim);
  }
  .search input {
    width: 220px;
    height: 24px;
  }
  .branch select {
    height: 24px;
    max-width: 200px;
  }
  .grow {
    flex: 1;
  }
  .small {
    font-size: var(--fs-small);
  }
  .icon {
    width: 26px;
    padding: 0;
    display: grid;
    place-items: center;
    color: var(--fg-dim);
  }
  .thead {
    display: flex;
    height: 24px;
    align-items: center;
    color: var(--fg-dim);
    font-size: var(--fs-small);
    border-bottom: 1px solid var(--line);
    padding-right: 10px;
  }
  .list {
    flex: 1;
    overflow-y: auto;
    min-height: 0;
    outline: none;
  }
  .row {
    position: absolute;
    left: 0;
    right: 0;
    height: 24px;
    display: flex;
    align-items: center;
    padding: 0 10px 0 4px;
    white-space: nowrap;
    cursor: default;
  }
  .row:hover {
    background: var(--bg-hover);
  }
  .row.sel {
    background: var(--selection);
  }
  .row.head .text {
    font-weight: 600;
  }
  .c-subject {
    flex: 1;
    min-width: 140px;
    display: flex;
    align-items: center;
    gap: 4px;
    overflow: hidden;
  }
  .text {
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .c-author {
    width: 150px;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    color: var(--fg-dim);
  }
  .c-date {
    width: 130px;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    color: var(--fg-dim);
  }
  .thead .c-author,
  .thead .c-date,
  .thead .c-subject {
    color: var(--fg-dim);
  }
  .ref {
    flex-shrink: 0;
    border-radius: 3px;
    padding: 0 5px;
    line-height: 16px;
    font-size: 11px;
    background: var(--bg-surface);
  }
  .ref.head {
    background: color-mix(in srgb, var(--warn) 25%, transparent);
    color: var(--warn);
  }
  .ref.local {
    background: color-mix(in srgb, var(--ok) 20%, transparent);
    color: var(--ok);
  }
  .ref.remote {
    background: color-mix(in srgb, var(--info) 20%, transparent);
    color: var(--info);
  }
  .ref.tag {
    background: color-mix(in srgb, var(--fg) 12%, transparent);
  }
  .more {
    position: absolute;
    left: 0;
    right: 0;
    height: 24px;
    border-radius: 0;
    color: var(--accent);
  }
  .empty {
    margin: auto;
    padding: 20px;
    text-align: center;
  }
</style>
