<script lang="ts">
  import * as Go from '../../wailsjs/go/main/App';
  import type { git } from '../../wailsjs/go/models';
  import { repo } from '../lib/repo.svelte';
  import { openMenu, ask, sure, toast } from '../lib/menu.svelte';
  import Icon from '../lib/Icon.svelte';

  let { dir, selected, onpick }: { dir: string; selected: string; onpick: (ref: string) => void } = $props();

  const r = $derived(repo(dir));
  let q = $state('');
  let open = $state<Record<string, boolean>>({ local: true, remote: true, tag: false, stash: false });

  const act = (p: Promise<unknown>) => r.run(p).catch((e) => toast(String(e)));
  const match = (s: string) => !q || s.toLowerCase().includes(q.toLowerCase());

  const current = $derived(r.refs.find((x) => x.current));
  const local = $derived(r.refs.filter((x) => x.kind === 'local' && match(x.name)));
  const tags = $derived(r.refs.filter((x) => x.kind === 'tag' && match(x.name)));
  const remotes = $derived.by(() => {
    const m = new Map<string, git.Ref[]>();
    for (const x of r.refs) {
      if (x.kind !== 'remote' || !match(x.name)) continue;
      const remote = x.name.split('/')[0];
      if (!m.has(remote)) m.set(remote, []);
      m.get(remote)!.push(x);
    }
    return [...m];
  });
  const stashes = $derived(r.stashes.filter((s) => match(s.subject)));

  function checkout(x: git.Ref) {
    act(x.kind === 'remote' ? Go.CheckoutRemote(dir, x.name) : Go.Checkout(dir, x.name));
  }

  function branchMenu(e: MouseEvent, x: git.Ref) {
    const cur = current?.name ?? 'HEAD';
    openMenu(e, [
      { label: x.kind === 'remote' ? 'Checkout as local branch' : 'Checkout', run: () => checkout(x) },
      {
        label: `New Branch from '${x.name}'…`,
        run: async () => {
          const n = await ask('New branch name', '', 'Create');
          if (n) act(Go.CreateBranch(dir, n, x.name, true));
        },
      },
      null,
      { label: `Merge '${x.name}' into '${cur}'`, run: () => act(Go.Merge(dir, x.name)) },
      { label: `Rebase '${cur}' onto '${x.name}'`, run: () => act(Go.Rebase(dir, x.name)) },
      ...(x.kind === 'local'
        ? [
            null,
            {
              label: 'Rename…',
              run: async () => {
                const n = await ask('Rename branch', x.name, 'Rename');
                if (n && n !== x.name) act(Go.RenameBranch(dir, x.name, n));
              },
            },
            {
              label: 'Delete',
              danger: true,
              run: async () => {
                if (!(await sure(`Delete branch '${x.name}'?`, '', 'Delete'))) return;
                try {
                  await r.run(Go.DeleteBranch(dir, x.name, false));
                } catch {
                  if (await sure(`'${x.name}' is not fully merged`, 'Delete it anyway?', 'Force delete')) act(Go.DeleteBranch(dir, x.name, true));
                }
              },
            },
          ]
        : []),
    ]);
  }

  function stashMenu(e: MouseEvent, s: git.Stash) {
    openMenu(e, [
      { label: 'Apply', run: () => act(Go.StashAction(dir, 'apply', s.ref)) },
      { label: 'Pop', run: () => act(Go.StashAction(dir, 'pop', s.ref)) },
      { label: 'Drop', danger: true, run: async () => (await sure(`Drop ${s.ref}?`, s.subject, 'Drop')) && act(Go.StashAction(dir, 'drop', s.ref)) },
    ]);
  }
</script>

{#snippet group(key: string, title: string, icon: string, count: number, depth = 0)}
  <button class="row group" style="padding-left:{6 + depth * 16}px" onclick={() => (open[key] = !open[key])}>
    <Icon name={open[key] ? 'chevron-down' : 'chevron-right'} size={14} />
    <Icon name={icon} size={14} />
    <span class="name">{title}</span>
    <span class="dim">{count}</span>
  </button>
{/snippet}

{#snippet ref(x: git.Ref, depth: number, label = x.name)}
  <div
    class="row"
    class:sel={selected === x.name}
    style="padding-left:{6 + depth * 16}px"
    title={x.upstream ? `${x.name} → ${x.upstream} ${x.track}` : x.name}
    role="treeitem"
    aria-selected={selected === x.name}
    tabindex="0"
    onclick={() => onpick(selected === x.name ? '' : x.name)}
    ondblclick={() => checkout(x)}
    oncontextmenu={(e) => branchMenu(e, x)}
    onkeydown={(e) => e.key === 'Enter' && checkout(x)}
  >
    <span class="ico" class:cur={x.current}><Icon name={x.kind === 'tag' ? 'tag' : 'branch'} size={14} /></span>
    <span class="name" class:cur={x.current}>{label}</span>
    {#if x.track}<span class="dim small">{x.track.replace(/[[\]]/g, '')}</span>{/if}
  </div>
{/snippet}

<div class="pane">
  <div class="search">
    <Icon name="search" size={14} />
    <input placeholder="Search branches" bind:value={q} />
  </div>
  <div class="tree" role="tree">
    {#if current}
      <div class="head-row dim small">HEAD (Current Branch)</div>
      {@render ref(current, 0)}
    {/if}
    {@render group('local', 'Local', 'folder', local.length)}
    {#if open.local}
      {#each local as x (x.full)}{@render ref(x, 1)}{/each}
    {/if}
    {@render group('remote', 'Remote', 'folder', remotes.reduce((n, [, l]) => n + l.length, 0))}
    {#if open.remote}
      {#each remotes as [name, list] (name)}
        {@render group(`remote:${name}`, name, 'folder', list.length, 1)}
        {#if open[`remote:${name}`] ?? true}
          {#each list as x (x.full)}{@render ref(x, 2, x.name.slice(name.length + 1))}{/each}
        {/if}
      {/each}
    {/if}
    {@render group('tag', 'Tags', 'tag', tags.length)}
    {#if open.tag}
      {#each tags as x (x.full)}{@render ref(x, 1)}{/each}
    {/if}
    {@render group('stash', 'Stashes', 'stash', stashes.length)}
    {#if open.stash}
      {#each stashes as s (s.ref)}
        <div class="row" style="padding-left:22px" title={s.ref} role="treeitem" aria-selected="false" tabindex="0" onclick={(e) => stashMenu(e, s)} oncontextmenu={(e) => stashMenu(e, s)} onkeydown={() => {}}>
          <span class="ico"><Icon name="stash" size={14} /></span>
          <span class="name">{s.subject}</span>
        </div>
      {/each}
    {/if}
  </div>
</div>

<style>
  .pane {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
  }
  .search {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 6px 8px;
    color: var(--fg-dim);
  }
  .search input {
    flex: 1;
    min-width: 0;
    height: 24px;
  }
  .tree {
    flex: 1;
    overflow: auto;
    padding-bottom: 8px;
  }
  .row {
    display: flex;
    align-items: center;
    gap: 5px;
    width: 100%;
    height: var(--row-h);
    padding-right: 8px;
    border: none;
    border-radius: 0;
    white-space: nowrap;
    cursor: default;
    text-align: left;
  }
  .row:hover {
    background: var(--bg-hover);
  }
  .row.sel {
    background: var(--selection);
  }
  .group {
    color: var(--fg);
  }
  .group :global(svg:first-child) {
    color: var(--fg-dim);
  }
  .head-row {
    padding: 4px 8px 0;
  }
  .ico {
    color: var(--fg-dim);
    display: flex;
  }
  .ico.cur {
    color: var(--warn);
  }
  .name {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .name.cur {
    font-weight: 600;
  }
  .small {
    font-size: var(--fs-small);
  }
</style>
