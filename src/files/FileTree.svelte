<script lang="ts">
  import { untrack } from 'svelte';
  import * as Go from '../../wailsjs/go/main/App';
  import type { main } from '../../wailsjs/go/models';
  import { repo } from '../lib/repo.svelte';
  import { openTab, fileTab, diffTab, mergeTab } from '../lib/tabs.svelte';
  import { openMenu, toast } from '../lib/menu.svelte';
  import PagedRows from '../lib/PagedRows.svelte';
  import Icon from '../lib/Icon.svelte';

  let { dir, collapse = 0, reloadKey = 0 }: { dir: string; collapse?: number; reloadKey?: number } = $props();

  let children = $state<Record<string, main.Entry[]>>({});
  let open = $state<Record<string, boolean>>({});
  let selected = $state('');

  async function load(rel: string) {
    try {
      children[rel] = await Go.ListDir(dir, rel);
    } catch (e) {
      toast(String(e));
    }
  }

  function toggle(rel: string) {
    open[rel] = !open[rel];
    if (open[rel] && !children[rel]) load(rel);
  }

  const status = $derived.by(() => {
    const m = new Map<string, string>();
    for (const f of repo(dir).files) {
      const w = String.fromCharCode(f.work);
      const c = f.conflict ? 'U' : w !== '.' ? w : String.fromCharCode(f.index);
      m.set(f.path, c === '?' ? 'q' : c);
    }
    return m;
  });

  const dirty = $derived.by(() => {
    const s = new Set<string>();
    for (const p of status.keys()) {
      let i = p.lastIndexOf('/');
      while (i > 0) {
        s.add(p.slice(0, i));
        i = p.lastIndexOf('/', i - 1);
      }
    }
    return s;
  });

  function menu(e: MouseEvent, rel: string, isDir: boolean) {
    selected = rel;
    const change = !isDir && repo(dir).files.find((f) => f.path === rel);
    openMenu(e, [
      ...(isDir ? [] : [{ label: 'Open', run: () => openTab(dir, fileTab(rel)) }]),
      ...(change ? [{
        label: change.conflict ? 'Resolve Conflict' : 'Show Diff',
        run: () => openTab(dir, change.conflict ? mergeTab(rel) : diffTab(rel, String.fromCharCode(change.work) === '.' ? 'staged' : 'unstaged', change.origPath)),
      }] : []),
      { label: 'Open in External Editor', run: () => Go.OpenExternal(dir, rel, 1).catch((x) => toast(String(x))) },
      null,
      { label: 'Copy Path', run: () => navigator.clipboard.writeText(`${dir}/${rel}`) },
      { label: 'Copy Relative Path', run: () => navigator.clipboard.writeText(rel) },
    ]);
  }

  function activate(rel: string, isDir: boolean) {
    selected = rel;
    if (isDir) toggle(rel);
    else openTab(dir, fileTab(rel), true);
  }

  const entries = $derived.by(() => {
    const rows: { e: main.Entry; rel: string; depth: number }[] = [];
    function append(base: string, depth: number) {
      for (const e of children[base] ?? []) {
        const rel = join(base, e.name);
        rows.push({ e, rel, depth });
        if (e.isDir && open[rel]) append(rel, depth + 1);
      }
    }
    append('', 0);
    return rows;
  });
  const join = (base: string, name: string) => (base ? `${base}/${name}` : name);

  $effect(() => {
    dir;
    untrack(() => {
      children = {};
      open = {};
      load('');
    });
  });

  $effect(() => {
    if (collapse) untrack(() => (open = {}));
  });

  $effect(() => {
    if (!reloadKey) return;
    untrack(() => {
      for (const rel of Object.keys(children)) {
        if (rel === '' || open[rel]) load(rel);
        else delete children[rel];
      }
    });
  });
</script>

<div class="tree" role="tree">
  <PagedRows items={entries} keyFor={(item) => item.rel}>
    {#snippet children({ e, rel, depth })}
      <div
        class="item"
        class:sel={selected === rel}
        style="padding-left:{4 + depth * 16}px"
        role="treeitem"
        aria-level={depth + 1}
        aria-selected={selected === rel}
        aria-expanded={e.isDir ? !!open[rel] : undefined}
        tabindex="0"
        title={rel}
        onclick={() => activate(rel, e.isDir)}
        ondblclick={() => !e.isDir && openTab(dir, fileTab(rel))}
        onkeydown={(k) => k.key === 'Enter' && activate(rel, e.isDir)}
        oncontextmenu={(m) => menu(m, rel, e.isDir)}
      >
        <span class="chev">{#if e.isDir}<Icon name={open[rel] ? 'chevron-down' : 'chevron-right'} size={14} />{/if}</span>
        <span class="ico" class:folder={e.isDir}><Icon name={e.isDir ? (open[rel] ? 'folder-open' : 'folder') : 'file'} size={15} /></span>
        <span class="name {e.isDir ? (dirty.has(rel) ? 'vcs-M' : '') : status.has(rel) ? `vcs-${status.get(rel)}` : ''}">{e.name}</span>
      </div>
    {/snippet}
  </PagedRows>
</div>

<style>
  .tree {
    overflow: auto;
    height: 100%;
    padding: 2px 0 8px;
  }
  .item {
    display: flex;
    align-items: center;
    gap: 4px;
    height: var(--row-h);
    padding-right: 8px;
    white-space: nowrap;
    cursor: default;
  }
  .item:hover {
    background: var(--bg-hover);
  }
  .item.sel {
    background: var(--selection);
  }
  .chev {
    width: 14px;
    display: flex;
    color: var(--fg-dim);
  }
  .ico {
    display: flex;
    color: var(--fg-dim);
  }
  .ico.folder {
    color: #87939a;
  }
  .name {
    overflow: hidden;
    text-overflow: ellipsis;
  }
</style>
