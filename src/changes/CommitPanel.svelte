<script lang="ts">
  import { untrack } from 'svelte';
  import * as Go from '../../wailsjs/go/main/App';
  import { git } from '../../wailsjs/go/models';
  import Icon from '../lib/Icon.svelte';
  import { repo } from '../lib/repo.svelte';
  import { openTab, diffTab, fileTab, mergeTab } from '../lib/tabs.svelte';
  import { openMenu, ask, sure, toast, type MenuItem } from '../lib/menu.svelte';
  import { partials, commitDraft } from './partial.svelte';

  let { dir }: { dir: string } = $props();

  type Source = 'unstaged' | 'staged' | 'head';
  type Item = { f: git.FileStatus; letter: string; source: Source };
  type Group = { key: string; name: string; kind: 'staged' | 'changes' | 'new' | 'list'; items: Item[] };
  type Row =
    | { key: string; kind: 'group'; depth: 0; group: Group }
    | { key: string; kind: 'folder'; depth: 1; name: string; group: Group; items: Item[] }
    | { key: string; kind: 'file'; depth: number; group: Group; item: Item };

  const DEFAULT = 'Changes';
  const r = $derived(repo(dir));
  let mode = $state<'staging' | 'changelists'>('staging');
  let lists = $state<Record<string, string[]>>({});
  const draft = untrack(() => commitDraft(dir));
  let collapsed = $state<Record<string, boolean>>({});
  let byDir = $state(false);
  let selected = $state<string | null>(null);
  let busy = $state(false);
  let commitH = $state(170);
  let treeEl: HTMLDivElement;

  const ch = (c: number) => String.fromCharCode(c);
  const untracked = (f: git.FileStatus) => ch(f.index) === '?';
  const leaf = (p: string) => p.slice(p.lastIndexOf('/') + 1);
  const parent = (p: string) => (p.includes('/') ? p.slice(0, p.lastIndexOf('/')) : '');
  const cls = (l: string) => `vcs-${l === '?' ? 'q' : l}`;
  const workLetter = (f: git.FileStatus) => (f.conflict ? 'U' : ch(f.work) !== '.' ? ch(f.work) : ch(f.index));

  $effect(() => {
    const d = dir;
    Go.GetMode().then((m) => (mode = m === 'changelists' ? 'changelists' : 'staging'));
    Go.Changelists(d).then((l) => (lists = l ?? {}));
  });

  const groups = $derived.by<Group[]>(() => {
    const files = r.files;
    const fresh = files.filter(untracked).map((f) => ({ f, letter: '?', source: (mode === 'staging' ? 'unstaged' : 'head') as Source }));
    const out: Group[] = [];
    if (mode === 'staging') {
      out.push({
        key: 'g:staged',
        name: 'Staged',
        kind: 'staged',
        items: files.filter((f) => !f.conflict && !untracked(f) && ch(f.index) !== '.').map((f) => ({ f, letter: ch(f.index), source: 'staged' })),
      });
      out.push({
        key: 'g:changes',
        name: 'Changes',
        kind: 'changes',
        items: files.filter((f) => f.conflict || (!untracked(f) && ch(f.work) !== '.')).map((f) => ({ f, letter: f.conflict ? 'U' : ch(f.work), source: 'unstaged' })),
      });
    } else {
      const fallback: Group = { key: 'l:' + DEFAULT, name: DEFAULT, kind: 'list', items: [] };
      out.push(fallback);
      const memberships = new Map<string, Group[]>();
      for (const [name, paths] of Object.entries(lists)) {
        if (name === DEFAULT) continue;
        const g: Group = { key: 'l:' + name, name, kind: 'list', items: [] };
        out.push(g);
        for (const path of new Set(paths)) {
          const groups = memberships.get(path);
          if (groups) groups.push(g);
          else memberships.set(path, [g]);
        }
      }
      for (const f of files) {
        if (untracked(f)) continue;
        const item: Item = { f, letter: workLetter(f), source: 'head' };
        const groups = memberships.get(f.path);
        if (groups) for (const g of groups) g.items.push(item);
        else fallback.items.push(item);
      }
    }
    if (fresh.length) out.push({ key: 'g:new', name: 'Unversioned Files', kind: 'new', items: fresh });
    return out;
  });

  // ponytail: one folder level per distinct directory; IntelliJ's compacted nested tree if long paths get noisy.
  const rows = $derived.by<Row[]>(() => {
    const out: Row[] = [];
    for (const g of groups) {
      out.push({ key: g.key, kind: 'group', depth: 0, group: g });
      if (collapsed[g.key]) continue;
      if (!byDir) {
        for (const item of g.items) out.push({ key: `${g.key}|${item.f.path}`, kind: 'file', depth: 1, group: g, item });
        continue;
      }
      const dirs = new Map<string, Item[]>();
      for (const item of g.items) {
        const d = parent(item.f.path);
        const bucket = dirs.get(d);
        if (bucket) bucket.push(item);
        else dirs.set(d, [item]);
      }
      for (const [d, items] of [...dirs].sort(([a], [b]) => a.localeCompare(b))) {
        const fk = `${g.key}/${d}`;
        if (d) out.push({ key: fk, kind: 'folder', depth: 1, name: d, group: g, items });
        if (d && collapsed[fk]) continue;
        for (const item of items) out.push({ key: `${g.key}|${item.f.path}`, kind: 'file', depth: d ? 2 : 1, group: g, item });
      }
    }
    return out;
  });

  // Prune changelist entries and partial texts for files that are no longer changed.
  $effect(() => {
    if (!r.snap || r.error) return;
    const live = new Set(r.files.map((f) => f.path));
    let dirty = false;
    const next: Record<string, string[]> = {};
    for (const [name, paths] of Object.entries(lists)) {
      next[name] = paths.filter((p) => live.has(p));
      if (next[name].length !== paths.length) dirty = true;
    }
    const part = partials[dir];
    if (part) for (const p of Object.keys(part)) if (!live.has(p)) delete part[p];
    if (dirty) saveLists(next);
  });

  function saveLists(next: Record<string, string[]>) {
    lists = next;
    Go.SetChangelists(dir, next).catch((e) => toast(String(e)));
  }

  const on = (item: Item, g: Group) => {
    if (mode === 'staging') return g.kind === 'staged';
    return draft.pick[item.f.path] ?? g.kind !== 'new';
  };
  const commitItems = $derived(mode === 'staging' ? (groups.find((g) => g.kind === 'staged')?.items ?? []) : groups.flatMap((g) => g.items.filter((i) => on(i, g))));

  async function act(p: Promise<unknown>) {
    try {
      await r.run(p, false);
    } catch (e) {
      toast(String(e));
    }
  }

  function setOn(items: Item[], g: Group, value: boolean) {
    if (r.error) return;
    if (mode === 'staging') {
      const paths = [...new Set(items.flatMap(({ f }) =>
        !value && ch(f.index) === 'R' && f.origPath ? [f.path, f.origPath] : [f.path],
      ))];
      if (!paths.length) return;
      act(value ? Go.Stage(dir, paths) : Go.Unstage(dir, paths));
      return;
    }
    for (const i of items) draft.pick[i.f.path] = value;
  }

  const allOn = (items: Item[], g: Group) => items.length > 0 && items.every((i) => on(i, g));
  const someOn = (items: Item[], g: Group) => items.some((i) => on(i, g)) && !allOn(items, g);

  function show(item: Item) {
    if (item.f.conflict) openTab(dir, mergeTab(item.f.path));
    else openTab(dir, diffTab(item.f.path, item.source, item.f.origPath), true);
  }

  function select(row: Row) {
    selected = row.key;
    if (row.kind === 'file') show(row.item);
  }

  function rowItems(row: Row): Item[] {
    return row.kind === 'file' ? [row.item] : row.kind === 'folder' ? row.items : row.group.items;
  }

  async function rollback(items: Item[], folder?: string) {
    if (r.error || !items.length) return;
    const paths = [...new Set(items.map((i) => i.f.path))];
    const what = folder || (paths.length === 1 ? paths[0] : `${paths.length} files`);
    if (!(await sure(`Rollback ${what}?`, 'Local changes are discarded. Unversioned files are deleted.', 'Rollback'))) return;
    act(Go.Discard(dir, paths));
  }

  function moveTo(paths: string[], name: string) {
    const next: Record<string, string[]> = {};
    const moved = new Set(paths);
    for (const [n, ps] of Object.entries(lists)) next[n] = ps.filter((p) => !moved.has(p));
    if (name !== DEFAULT) next[name] = [...(next[name] ?? []), ...paths];
    saveLists(next);
  }

  async function newList(paths: string[] = []) {
    const n = await ask('New changelist', '', 'Create');
    if (!n) return;
    if (!lists[n]) lists[n] = [];
    moveTo(paths, n);
  }

  function fileMenu(e: MouseEvent, item: Item, g: Group) {
    const p = item.f.path;
    const items: MenuItem[] = [
      { label: 'Show Diff', run: () => show(item) },
      { label: 'Open File', hint: 'F4', run: () => openTab(dir, fileTab(p)) },
      null,
      { label: 'Rollback…', danger: true, run: () => rollback([item]) },
    ];
    if (mode === 'staging') {
      items.push(g.kind === 'staged' ? { label: 'Unstage', run: () => setOn([item], g, false) } : { label: 'Stage', run: () => setOn([item], g, true) });
    } else if (g.kind === 'list') {
      items.push(null);
      for (const n of [DEFAULT, ...Object.keys(lists).filter((n) => n !== DEFAULT)]) if (n !== g.name) items.push({ label: `Move to ${n}`, run: () => moveTo([p], n) });
      items.push({ label: 'New Changelist…', run: () => newList([p]) });
      if (partials[dir]?.[p] !== undefined) items.push({ label: 'Include Whole File', run: () => delete partials[dir][p] });
    }
    if (untracked(item.f)) {
      items.push(null, { label: 'Delete', danger: true, run: () => rollback([item]) }, { label: 'Add to .gitignore', run: () => act(Go.AddToGitignore(dir, p)) });
    }
    items.push(null, { label: 'Copy Path', run: () => navigator.clipboard.writeText(p) }, { label: 'Open in External Editor', run: () => Go.OpenExternal(dir, p, 0).catch((e) => toast(String(e))) });
    openMenu(e, items);
  }

  function groupMenu(e: MouseEvent, row: Extract<Row, { kind: 'group' | 'folder' }>) {
    const g = row.group;
    const items: MenuItem[] = [{ label: 'Rollback All…', danger: true, run: () => rollback(rowItems(row), row.kind === 'folder' ? row.name : undefined) }];
    if (mode === 'changelists' && row.kind === 'group') {
      items.push(null, { label: 'New Changelist…', run: () => newList() });
      if (g.kind === 'list' && g.name !== DEFAULT) {
        items.push(
          {
            label: 'Rename…',
            run: async () => {
              const n = await ask('Rename changelist', g.name, 'Rename');
              if (!n || n === g.name || lists[n]) return;
              const next: Record<string, string[]> = {};
              for (const [k, v] of Object.entries(lists)) next[k === g.name ? n : k] = v;
              saveLists(next);
            },
          },
          {
            label: 'Delete Changelist',
            danger: true,
            run: async () => {
              if (!(await sure(`Delete changelist ${g.name}?`, `Its files move to ${DEFAULT}.`, 'Delete'))) return;
              const { [g.name]: _, ...rest } = lists;
              saveLists(rest);
            },
          },
        );
      }
    }
    openMenu(e, items);
  }

  function selectedItems(): Item[] {
    const row = rows.find((x) => x.key === selected);
    return row ? rowItems(row) : [];
  }

  const rowHeight = 24;
  let treeTop = $state(0), treeHeight = $state(400);
  const start = $derived(Math.max(0, Math.min(Math.floor(treeTop / rowHeight), rows.length - Math.ceil(treeHeight / rowHeight)) - 8));
  const end = $derived(Math.min(rows.length, start + Math.ceil(treeHeight / rowHeight) + 16));

  function onKey(e: KeyboardEvent) {
    const i = rows.findIndex((x) => x.key === selected);
    const row = rows[i];
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
      e.preventDefault();
      const index = Math.max(0, Math.min(rows.length - 1, i + (e.key === 'ArrowDown' ? 1 : -1)));
      const next = rows[index];
      if (next) {
        select(next);
        const top = index * rowHeight;
        if (top < treeEl.scrollTop) treeEl.scrollTop = top;
        else if (top + rowHeight > treeEl.scrollTop + treeHeight) treeEl.scrollTop = top + rowHeight - treeHeight;
      }
      return;
    }
    if (!row) return;
    if (e.key === 'ArrowLeft' && row.kind !== 'file') collapsed[row.key] = true;
    else if (e.key === 'ArrowRight' && row.kind !== 'file') collapsed[row.key] = false;
    else if (e.key === ' ') {
      e.preventDefault();
      const items = rowItems(row);
      setOn(items, row.group, !allOn(items, row.group));
    } else if ((e.key === 'Enter' || e.key === 'F4') && row.kind === 'file') openTab(dir, fileTab(row.item.f.path));
    else if (e.key === 'Delete') rollback(rowItems(row));
  }

  async function toggleAmend() {
    draft.amend = !draft.amend;
    if (draft.amend && !draft.message.trim()) draft.message = (await Go.LastMessage(dir)).trim();
  }

  async function setMode(m: string) {
    try {
      await Go.SetMode(m);
      mode = m === 'changelists' ? 'changelists' : 'staging';
    } catch (e) {
      toast(String(e));
    }
  }

  const canCommit = $derived(!busy && !draft.repair && !r.error && (mode === 'staging' || !r.files.some((f) => f.conflict)) && !!draft.message.trim() && (commitItems.length > 0 || draft.amend));

  async function commit(push = false) {
    if (!canCommit) return;
    busy = true;
    try {
      if (mode === 'staging') {
        await r.run(Go.Commit(dir, draft.message, draft.amend));
      } else {
        const part = partials[dir] ?? {};
        const out: git.PartialFile[] = [];
        for (const { f } of commitItems) {
          out.push(git.PartialFile.createFrom({ path: f.path, text: part[f.path]?.text, expectedBase: part[f.path]?.base, delete: false }));
          if (f.origPath && (ch(f.index) === 'R' || ch(f.work) === 'R')) out.push(git.PartialFile.createFrom({ path: f.origPath, delete: true }));
        }
        draft.repair = (await r.run(Go.CommitPartial(dir, draft.message, draft.amend, out))) ?? undefined;
        for (const { f } of commitItems) {
          delete part[f.path];
          delete draft.pick[f.path];
        }
      }
      draft.message = '';
      draft.amend = false;
      if (push && !draft.repair) {
        const out = await r.run(Go.Push(dir, false));
        toast(out.trim().split('\n').pop() || 'Pushed', 'info');
      }
    } catch (e) {
      toast(String(e));
    } finally {
      busy = false;
    }
  }

  async function repairIndex() {
    if (!draft.repair || busy) return;
    busy = true;
    try {
      await r.run(Go.RepairPartialIndex(dir, draft.repair));
      delete draft.repair;
    } catch (e) { toast(String(e)); }
    finally { busy = false; }
  }

  function drag(e: MouseEvent) {
    const startY = e.clientY;
    const startH = commitH;
    const move = (m: MouseEvent) => (commitH = Math.max(90, Math.min(window.innerHeight - 200, startH + startY - m.clientY)));
    const up = () => {
      window.removeEventListener('mousemove', move);
      window.removeEventListener('mouseup', up);
    };
    window.addEventListener('mousemove', move);
    window.addEventListener('mouseup', up);
  }
</script>

<div class="panel">
  <header>
    <span class="title">Commit</span>
    <span class="grow"></span>
    <button class="icon" title="Refresh" onclick={() => r.schedule(true)}><Icon name="refresh" /></button>
    <button class="icon" title="Rollback" disabled={!selected} onclick={() => rollback(selectedItems())}><Icon name="undo" /></button>
    <button class="icon" class:on={byDir} title="Group by directory" onclick={() => (byDir = !byDir)}><Icon name="folder" /></button>
    <button class="icon" title="Expand all" onclick={() => (collapsed = {})}><Icon name="expand-all" /></button>
    <button class="icon" title="Collapse all" onclick={() => (collapsed = Object.fromEntries(rows.filter((x) => x.kind !== 'file').map((x) => [x.key, true])))}><Icon name="collapse-all" /></button>
    <select title="Commit model" value={mode} onchange={(e) => setMode(e.currentTarget.value)}>
      <option value="staging">Staging area</option>
      <option value="changelists">Changelists</option>
    </select>
  </header>

  {#if draft.repair}
    <div role="alert">{draft.repair.message}
      <button disabled={busy} onclick={repairIndex}>Retry index update</button>
      <button disabled={busy} onclick={() => delete draft.repair}>I reviewed staging</button>
    </div>
  {/if}
  {#if r.error}<div class="error" role="alert">{r.error}</div>{/if}
  <div class="tree" role="tree" tabindex="0" bind:this={treeEl} bind:clientHeight={treeHeight} onscroll={(e) => (treeTop = e.currentTarget.scrollTop)} onkeydown={onKey}>
    <div aria-hidden="true" style="height:{start * rowHeight}px"></div>
    {#each rows.slice(start, end) as row (row.key)}
      {@const items = rowItems(row)}
      <div
        class="row {row.kind}"
        class:sel={selected === row.key}
        style="padding-left:{6 + row.depth * 16}px"
        data-key={row.key}
        role="treeitem"
        aria-level={row.depth + 1}
        aria-selected={selected === row.key}
        tabindex="-1"
        onclick={() => select(row)}
        ondblclick={() => row.kind === 'file' ? openTab(dir, fileTab(row.item.f.path)) : (collapsed[row.key] = !collapsed[row.key])}
        oncontextmenu={(e) => { selected = row.key; row.kind === 'file' ? fileMenu(e, row.item, row.group) : groupMenu(e, row); }}
        onkeydown={() => {}}
      >
        {#if row.kind !== 'file'}
          <button class="chev" tabindex="-1" onclick={(e) => { e.stopPropagation(); collapsed[row.key] = !collapsed[row.key]; }}>
            <Icon name={collapsed[row.key] ? 'chevron-right' : 'chevron-down'} size={14} />
          </button>
        {/if}
        <input
          type="checkbox"
          tabindex="-1"
          checked={allOn(items, row.group)}
          indeterminate={someOn(items, row.group)}
          disabled={!items.length}
          onclick={(e) => e.stopPropagation()}
          onchange={(e) => setOn(items, row.group, e.currentTarget.checked)}
        />
        {#if row.kind === 'group'}
          <span class="gname">{row.group.name}</span>
          <span class="dim small">{row.group.items.length} {row.group.items.length === 1 ? 'file' : 'files'}</span>
        {:else if row.kind === 'folder'}
          <span class="ico"><Icon name={collapsed[row.key] ? 'folder' : 'folder-open'} size={14} /></span>
          <span class="name">{row.name}</span>
          <span class="dim small">{row.items.length}</span>
        {:else}
          {@const f = row.item.f}
          <span class="ico {cls(row.item.letter)}"><Icon name="file" size={14} /></span>
          <span class="name {cls(row.item.letter)}" title={f.origPath ? `${f.origPath} → ${f.path}` : f.path}>{leaf(f.path)}</span>
          {#if !byDir && parent(f.path)}<span class="dim small path">{parent(f.path)}</span>{/if}
          {#if f.origPath}<span class="dim small path">from {leaf(f.origPath)}</span>{/if}
          {#if mode === 'changelists' && partials[dir]?.[f.path] !== undefined}<span class="tag" title="Only included hunks will be committed">partial</span>{/if}
        {/if}
      </div>
    {:else}
      {#if !r.error}<div class="empty dim">No changes</div>{/if}
    {/each}
    <div aria-hidden="true" style="height:{(rows.length - end) * rowHeight}px"></div>
  </div>

  <button class="splitter" aria-label="Resize commit message" tabindex="-1" onmousedown={drag}></button>

  <div class="commit" style="height:{commitH}px">
    <textarea
      bind:value={draft.message}
      placeholder="Commit message"
      onkeydown={(e) => { if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) { e.preventDefault(); commit(); } }}
    ></textarea>
    <div class="actions">
      <label class="amend"><input type="checkbox" checked={draft.amend} onchange={toggleAmend} /> Amend</label>
      <span class="grow"></span>
      <span class="dim small">{commitItems.length} {commitItems.length === 1 ? 'file' : 'files'}</span>
      <button class="primary" disabled={!canCommit} onclick={() => commit()} title="Ctrl+Enter">Commit</button>
      <button class="outline" disabled={!canCommit} onclick={() => commit(true)}>Commit and Push…</button>
    </div>
  </div>
</div>

<style>
  .panel {
    height: 100%;
    display: flex;
    flex-direction: column;
    background: var(--bg-panel);
    min-height: 0;
  }
  header {
    height: 32px;
    flex-shrink: 0;
    display: flex;
    align-items: center;
    gap: 2px;
    padding: 0 6px 0 10px;
  }
  .title {
    font-weight: 600;
  }
  .grow {
    flex: 1;
  }
  .icon {
    width: 26px;
    padding: 0;
    display: grid;
    place-items: center;
    color: var(--fg-dim);
  }
  .icon:hover:not(:disabled),
  .icon.on {
    color: var(--fg);
  }
  .icon.on {
    background: var(--bg-hover);
  }
  header select {
    height: 24px;
    padding: 0 4px;
    margin-left: 4px;
    font-size: var(--fs-small);
    background: transparent;
  }
  .tree {
    flex: 1;
    min-height: 0;
    overflow: auto;
    outline: none;
    padding-bottom: 4px;
  }
  .row {
    height: var(--row-h);
    display: flex;
    align-items: center;
    gap: 5px;
    padding-right: 8px;
    white-space: nowrap;
    cursor: default;
  }
  .row:hover {
    background: var(--bg-hover);
  }
  .row.sel {
    background: color-mix(in srgb, var(--selection) 60%, transparent);
  }
  .tree:focus .row.sel {
    background: var(--selection);
  }
  .chev {
    width: 16px;
    height: 16px;
    padding: 0;
    display: grid;
    place-items: center;
    color: var(--fg-dim);
  }
  input[type='checkbox'] {
    margin: 0;
    width: 14px;
    height: 14px;
    accent-color: var(--accent);
    flex-shrink: 0;
  }
  .row.file input[type='checkbox'] {
    margin-left: 21px;
  }
  .gname {
    font-weight: 600;
  }
  .ico {
    display: grid;
    place-items: center;
    color: var(--fg-dim);
  }
  .name {
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .path {
    overflow: hidden;
    text-overflow: ellipsis;
    min-width: 0;
  }
  .small {
    font-size: var(--fs-small);
  }
  .tag {
    font-size: 11px;
    padding: 0 5px;
    border-radius: var(--radius);
    background: var(--bg-surface);
    color: var(--fg-dim);
  }
  .empty {
    padding: 12px 16px;
  }
  .splitter {
    height: 5px;
    width: 100%;
    padding: 0;
    border: none;
    border-radius: 0;
    flex-shrink: 0;
    cursor: row-resize;
    border-top: 1px solid var(--line);
  }
  .splitter:hover {
    background: color-mix(in srgb, var(--accent) 40%, transparent);
  }
  .commit {
    flex-shrink: 0;
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding: 0 8px 8px;
  }
  textarea {
    flex: 1;
    min-height: 40px;
    resize: none;
    background: var(--bg-editor);
    font-family: var(--font-ui);
  }
  .actions {
    display: flex;
    align-items: center;
    gap: 6px;
  }
  .amend {
    display: flex;
    align-items: center;
    gap: 5px;
  }
</style>
