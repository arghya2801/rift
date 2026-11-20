<script lang="ts">
  import * as Go from '../../wailsjs/go/main/App';
  import type { git } from '../../wailsjs/go/models';
  import { openTab, commitDiffTab } from '../lib/tabs.svelte';
  import { toast } from '../lib/menu.svelte';
  import Icon from '../lib/Icon.svelte';

  let { dir, row, onparent }: { dir: string; row: git.Row; onparent: (hash: string) => void } = $props();

  let message = $state('');
  let files = $state<git.ChangedFile[]>([]);

  $effect(() => {
    const h = row.hash;
    files = [];
    message = '';
    let dead = false;
    Promise.all([Go.CommitMessage(dir, h), Go.CommitFiles(dir, h)]).then(
      ([m, f]) => {
        if (dead) return;
        message = m;
        files = f;
      },
      (e) => toast(String(e)),
    );
    return () => (dead = true);
  });

  const groups = $derived.by(() => {
    const m = new Map<string, git.ChangedFile[]>();
    for (const f of [...files].sort((a, b) => a.path.localeCompare(b.path))) {
      const i = f.path.lastIndexOf('/');
      const d = i < 0 ? '' : f.path.slice(0, i);
      if (!m.has(d)) m.set(d, []);
      m.get(d)!.push(f);
    }
    return [...m];
  });

  const leaf = (p: string) => p.slice(p.lastIndexOf('/') + 1);
  const date = (t: number) => new Date(t * 1000).toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' });
  const body = $derived(message.split('\n').slice(1).join('\n').trim());
</script>

<div class="details">
  <div class="files">
    <div class="caption dim">{files.length} file{files.length === 1 ? '' : 's'} changed</div>
    {#each groups as [d, list] (d)}
      {#if d}
        <div class="row dir"><Icon name="folder" size={14} /><span class="name">{d}</span><span class="dim">{list.length}</span></div>
      {/if}
      {#each list as f (f.path)}
        <button class="row file" style="padding-left:{d ? 28 : 10}px" title={f.oldPath ? `${f.oldPath} → ${f.path}` : f.path} onclick={() => openTab(dir, commitDiffTab(row.hash, f.path, f.oldPath), true)}>
          <Icon name="file" size={14} />
          <span class="name vcs-{f.status}">{leaf(f.path)}</span>
          {#if f.oldPath}<span class="dim small">from {leaf(f.oldPath)}</span>{/if}
        </button>
      {/each}
    {/each}
  </div>
  <div class="meta">
    <div class="subject">{row.subject}</div>
    {#if body}<pre class="body">{body}</pre>{/if}
    <div class="line">
      <button class="hash mono" title="Copy hash" onclick={() => navigator.clipboard.writeText(row.hash).then(() => toast('Hash copied', 'info'))}>{row.hash.slice(0, 10)}</button>
      <span class="dim">by <span class="fg">{row.author}</span> &lt;{row.email}&gt;</span>
    </div>
    <div class="dim small">{date(row.time)}</div>
    {#if row.parents.length}
      <div class="line small">
        <span class="dim">{row.parents.length > 1 ? 'Parents' : 'Parent'}</span>
        {#each row.parents as p}<button class="link mono" onclick={() => onparent(p)}>{p.slice(0, 8)}</button>{/each}
      </div>
    {/if}
    {#if row.refs.length}
      <div class="line small">{#each row.refs as ref}<span class="pill">{ref.replace('tag: ', '')}</span>{/each}</div>
    {/if}
  </div>
</div>

<style>
  .details {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
  }
  .files {
    flex: 1;
    overflow: auto;
    min-height: 60px;
    padding-bottom: 4px;
  }
  .caption {
    padding: 6px 10px 2px;
    font-size: var(--fs-small);
  }
  .row {
    display: flex;
    align-items: center;
    gap: 6px;
    width: 100%;
    height: var(--row-h);
    padding: 0 10px;
    border: none;
    border-radius: 0;
    white-space: nowrap;
    text-align: left;
    color: var(--fg);
  }
  .row :global(svg) {
    color: var(--fg-dim);
  }
  .file:hover {
    background: var(--bg-hover);
  }
  .name {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .dir .name {
    flex: 1;
  }
  .meta {
    border-top: 1px solid var(--line);
    padding: 10px 12px;
    max-height: 45%;
    overflow: auto;
    user-select: text;
  }
  .subject {
    color: var(--fg-bright);
    font-weight: 600;
    margin-bottom: 6px;
  }
  .body {
    margin: 0 0 8px;
    font: inherit;
    white-space: pre-wrap;
  }
  .line {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 6px;
    margin-top: 4px;
  }
  .hash,
  .link {
    height: 20px;
    padding: 0 4px;
    color: var(--accent);
  }
  .fg {
    color: var(--fg);
  }
  .small {
    font-size: var(--fs-small);
  }
  .pill {
    border-radius: 3px;
    padding: 0 5px;
    background: var(--bg-surface);
  }
</style>
