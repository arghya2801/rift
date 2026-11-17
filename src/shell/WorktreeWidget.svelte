<script lang="ts">
  import Icon from '../lib/Icon.svelte';
  import Popup from './Popup.svelte';
  import { openMenu } from '../lib/menu.svelte';
  import { app, select, addRepo, newWorktree, repoMenu, worktreeMenu, repoOf } from './app.svelte';
  import type { git } from '../../wailsjs/go/models';

  let btn = $state<HTMLButtonElement>();
  let open = $state(false);
  let filter = $state('');

  const leaf = (p: string) => p.split(/[\\/]/).pop() ?? p;
  const label = (w: git.Worktree) => w.branch || (w.detached ? w.head.slice(0, 8) : 'bare');
  const q = $derived(filter.trim().toLowerCase());
  const list = $derived(
    app.repos
      .map((r) => ({ r, wts: (r.worktrees ?? []).filter((w) => !q || `${r.name} ${label(w)} ${w.path}`.toLowerCase().includes(q)) }))
      .filter((x) => x.wts.length || (!q ? true : x.r.name.toLowerCase().includes(q))),
  );
  const cur = $derived(repoOf(app.current));

  function pick(w: git.Worktree) {
    select(w);
    open = false;
  }
</script>

<button class="widget" bind:this={btn} onclick={() => ((open = !open), (filter = ''))} title="Switch repository or worktree">
  <span class="badge">{(cur?.name ?? 'R').slice(0, 2).toUpperCase()}</span>
  <span class="name">{cur?.name ?? 'Open repository'}</span>
  {#if app.current && !app.current.main}<span class="dim">/ {leaf(app.current.path)}</span>{/if}
  <Icon name="chevron-down" size={14} />
</button>

{#if open && btn}
  <Popup anchor={btn} onclose={() => (open = false)} width={400}>
    <div class="head">
      <!-- svelte-ignore a11y_autofocus -->
      <input placeholder="Filter repositories and worktrees" bind:value={filter} autofocus />
    </div>
    <div class="list">
      {#each list as { r, wts } (r.path)}
        <div class="repo" role="group" oncontextmenu={(e) => openMenu(e, repoMenu(r))} title={r.path}>
          <Icon name="folder" size={14} />
          <span class="grow">{r.name}</span>
          <button class="ib" title="New worktree" onclick={() => ((open = false), newWorktree(r))}><Icon name="plus" size={14} /></button>
          <button class="ib" title="More" onclick={(e) => openMenu(e, repoMenu(r))}><Icon name="more" size={14} /></button>
        </div>
        {#if r.error}<div class="err">{r.error}</div>{/if}
        {#each wts as w (w.path)}
          <button class="wt" class:on={app.current?.path === w.path} onclick={() => pick(w)} oncontextmenu={(e) => openMenu(e, worktreeMenu(r, w))} title={w.path}>
            <Icon name="branch" size={14} />
            <span class="grow">{label(w)}</span>
            {#if !w.main}<span class="dim small">{leaf(w.path)}</span>{/if}
            {#if w.locked}<span class="dim small">locked</span>{/if}
          </button>
        {/each}
      {/each}
    </div>
    <div class="foot">
      <button onclick={() => ((open = false), addRepo())}><Icon name="plus" size={14} /> Open repository…</button>
      {#if app.current}<button onclick={() => ((open = false), select(null))}>Welcome screen</button>{/if}
    </div>
  </Popup>
{/if}

<style>
  .widget {
    display: flex;
    align-items: center;
    gap: 6px;
    height: 28px;
    padding: 0 8px;
    font-weight: 600;
    color: var(--fg-bright);
  }
  .badge {
    display: grid;
    place-items: center;
    width: 20px;
    height: 20px;
    border-radius: var(--radius);
    background: color-mix(in srgb, var(--accent) 70%, var(--bg));
    color: var(--on-accent);
    font-size: 10px;
    font-weight: 700;
  }
  .name {
    max-width: 200px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .widget .dim {
    font-weight: 400;
  }
  .head {
    padding: 8px;
  }
  .head input {
    width: 100%;
    height: 28px;
  }
  .list {
    overflow-y: auto;
    padding: 0 6px 6px;
  }
  .repo {
    display: flex;
    align-items: center;
    gap: 6px;
    height: var(--row-h);
    padding: 0 4px 0 8px;
    margin-top: 6px;
    color: var(--fg-dim);
    font-size: var(--fs-small);
    font-weight: 600;
  }
  .ib {
    height: 22px;
    padding: 0 3px;
    visibility: hidden;
  }
  .repo:hover .ib {
    visibility: visible;
  }
  .wt {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    height: var(--row-h);
    padding: 0 8px 0 22px;
    text-align: left;
  }
  .wt.on {
    background: var(--selection);
  }
  .grow {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .small {
    font-size: var(--fs-small);
  }
  .err {
    padding: 2px 22px;
    color: var(--danger);
    font-size: var(--fs-small);
  }
  .foot {
    display: flex;
    gap: 4px;
    padding: 6px;
    border-top: 1px solid var(--line);
  }
  .foot button {
    display: flex;
    gap: 6px;
    align-items: center;
  }
</style>
