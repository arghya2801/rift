<script lang="ts">
  import * as Go from '../../wailsjs/go/main/App';
  import type { main, git } from '../../wailsjs/go/models';
  import Icon from '../lib/Icon.svelte';
  import { openMenu } from '../lib/menu.svelte';
  import { app, select, addRepo, newWorktree, repoMenu, worktreeMenu } from '../shell/app.svelte';

  let cards = $state<main.Overview[]>([]);
  let loading = $state(false);
  let filter = $state('');

  let request = 0;
  async function load(repos: main.Repo[]) {
    const id = ++request;
    loading = true;
    const next = repos.length ? (await Go.Overview(repos).catch(() => [])) ?? [] : [];
    if (id !== request) return;
    cards = next;
    loading = false;
  }

  const q = $derived(filter.trim().toLowerCase());
  const shown = $derived(cards.filter((c) => !q || `${c.repo} ${c.status.branch} ${c.wt.branch} ${c.wt.path}`.toLowerCase().includes(q)));
  const label = (w: git.Worktree) => w.branch || (w.detached ? w.head.slice(0, 8) : 'bare');
  const leaf = (p: string) => p.split(/[\\/]/).pop() ?? p;
  $effect(() => { load(app.repos); });
</script>

<div class="welcome">
  <aside>
    <div class="brand">
      <span class="logo">R</span>
      <div>
        <div class="title">Rift</div>
        <div class="dim small">Lightweight git client</div>
      </div>
    </div>
    <input placeholder="Search repositories" bind:value={filter} />
    <div class="repos">
      {#each app.repos as r (r.path)}
        <div class="repo" role="group" oncontextmenu={(e) => openMenu(e, repoMenu(r))} title={r.path}>
          <span class="ricon">{r.name.slice(0, 2).toUpperCase()}</span>
          <span class="grow">
            <span class="rname">{r.name}</span>
            <span class="dim small path">{r.path}</span>
          </span>
          <button class="ib" title="New worktree" onclick={() => newWorktree(r)}><Icon name="plus" size={14} /></button>
        </div>
        {#each r.worktrees ?? [] as w (w.path)}
          <button class="wt" onclick={() => select(w)} oncontextmenu={(e) => openMenu(e, worktreeMenu(r, w))} title={w.path}>
            <Icon name="branch" size={14} />
            <span class="grow">{label(w)}</span>
            {#if !w.main}<span class="dim small">{leaf(w.path)}</span>{/if}
          </button>
        {/each}
      {:else}
        <p class="dim small">No repositories yet.</p>
      {/each}
    </div>
  </aside>
  <section>
    <div class="top">
      <h1>Worktrees</h1>
      <span class="grow"></span>
      <button class="outline" onclick={() => load(app.repos)} disabled={loading}><Icon name="refresh" size={14} /> {loading ? 'Loading…' : 'Refresh'}</button>
      <button class="primary" onclick={addRepo}>Open…</button>
    </div>
    {#if !app.repos.length}
      <div class="hint">
        <p>Open a git repository to get started. Its linked worktrees show up automatically.</p>
        <button class="primary" onclick={addRepo}>Open repository…</button>
      </div>
    {/if}
    <div class="grid">
      {#each shown as c (c.wt.path)}
        <button class="card" onclick={() => select(c.wt)} title={c.wt.path}>
          <div class="row">
            <span class="dim small">{c.repo}</span>
            <span class="grow"></span>
            {#if c.op}<span class="pill warn">{c.op}</span>{/if}
            {#if c.conflicts}<span class="pill bad">{c.conflicts} conflicts</span>{/if}
          </div>
          <div class="row branch"><Icon name="branch" size={14} /> <span class="grow">{c.status.branch || label(c.wt)}</span></div>
          {#if c.error}
            <div class="bad small">{c.error}</div>
          {:else}
            <div class="row small">
              <span class:dim={!c.changed}>{c.changed} changed</span>
              {#if c.staged}<span>· {c.staged} staged</span>{/if}
              <span class="grow"></span>
              {#if c.status.upstream}<span class="dim">↑{c.status.ahead} ↓{c.status.behind}</span>{/if}
            </div>
            <div class="dim small last">{c.last}</div>
          {/if}
        </button>
      {/each}
    </div>
  </section>
</div>

<style>
  .welcome {
    flex: 1;
    min-height: 0;
    display: flex;
  }
  aside {
    width: 300px;
    flex-shrink: 0;
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding: 16px 10px;
    background: var(--bg-panel);
    border-right: 1px solid var(--line);
  }
  .brand {
    display: flex;
    gap: 10px;
    align-items: center;
    padding: 0 6px 8px;
  }
  .logo {
    display: grid;
    place-items: center;
    width: 36px;
    height: 36px;
    border-radius: var(--radius-lg);
    background: var(--accent);
    color: var(--on-accent);
    font-weight: 700;
    font-size: 18px;
  }
  .title {
    font-weight: 600;
    color: var(--fg-bright);
  }
  aside input {
    margin: 0 6px;
    height: 28px;
  }
  .repos {
    overflow-y: auto;
    flex: 1;
  }
  .repo {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px;
    margin-top: 4px;
  }
  .ricon {
    display: grid;
    place-items: center;
    width: 24px;
    height: 24px;
    flex-shrink: 0;
    border-radius: var(--radius);
    background: color-mix(in srgb, var(--accent) 45%, var(--bg));
    color: var(--fg-bright);
    font-size: 10px;
    font-weight: 700;
  }
  .rname {
    display: block;
    color: var(--fg-bright);
  }
  .path {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
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
    padding: 0 8px 0 38px;
    text-align: left;
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
  section {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    background: var(--bg-editor);
  }
  .top {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 16px 20px 8px;
  }
  .top button {
    display: flex;
    align-items: center;
    gap: 6px;
  }
  h1 {
    margin: 0;
    font-size: 15px;
    font-weight: 600;
    color: var(--fg-bright);
  }
  .hint {
    padding: 24px 20px;
    color: var(--fg-dim);
  }
  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
    gap: 10px;
    padding: 8px 20px 20px;
    overflow-y: auto;
    align-content: start;
  }
  .card {
    height: auto;
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding: 12px;
    text-align: left;
    background: var(--bg-panel);
    border-radius: var(--radius-lg);
  }
  .card:hover {
    background: var(--bg-hover);
  }
  .row {
    display: flex;
    align-items: center;
    gap: 6px;
  }
  .branch {
    color: var(--fg-bright);
    font-weight: 600;
  }
  .pill {
    font-size: 11px;
    padding: 0 6px;
    border-radius: 9px;
  }
  .warn {
    color: var(--warn);
    background: color-mix(in srgb, var(--warn) 15%, transparent);
  }
  .bad {
    color: var(--danger);
  }
  .pill.bad {
    background: color-mix(in srgb, var(--danger) 15%, transparent);
  }
  .last {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
</style>
