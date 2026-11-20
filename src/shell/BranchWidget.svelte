<script lang="ts">
  import * as Go from '../../wailsjs/go/main/App';
  import type { git } from '../../wailsjs/go/models';
  import Icon from '../lib/Icon.svelte';
  import Popup from './Popup.svelte';
  import { repo } from '../lib/repo.svelte';
  import { openMenu, ask, sure, toast, type MenuItem } from '../lib/menu.svelte';
  import { remote } from './app.svelte';

  let { dir }: { dir: string } = $props();

  const r = $derived(repo(dir));
  let btn = $state<HTMLButtonElement>();
  let open = $state(false);
  let filter = $state('');
  let closed = $state<Record<string, boolean>>({});

  const current = $derived(r.branch === '(detached)' ? (r.status ? 'Detached HEAD' : '') : r.branch);
  const q = $derived(filter.trim().toLowerCase());
  const groups = $derived(
    (['local', 'remote', 'tag'] as const).map((k) => ({
      k,
      title: k === 'local' ? 'Local' : k === 'remote' ? 'Remote' : 'Tags',
      items: r.refs.filter((x) => x.kind === k && (!q || x.name.toLowerCase().includes(q))),
    })),
  );

  async function act(p: Promise<unknown>) {
    open = false;
    try {
      await r.run(p);
    } catch (e) {
      toast(String(e));
    }
  }

  async function newBranch(from: string) {
    open = false;
    const n = await ask(`New branch from ${from}`, '', 'Create');
    if (n) act(Go.CreateBranch(dir, n, from, true));
  }

  function items(x: git.Ref): MenuItem[] {
    const cur = current || 'HEAD';
    return [
      x.kind === 'remote'
        ? { label: 'Checkout as new local branch', run: () => act(Go.CheckoutRemote(dir, x.name)) }
        : { label: 'Checkout', run: () => act(Go.Checkout(dir, x.name)) },
      { label: `New branch from '${x.name}'…`, run: () => newBranch(x.name) },
      null,
      { label: `Merge '${x.name}' into '${cur}'`, run: () => act(Go.Merge(dir, x.name)) },
      { label: `Rebase '${cur}' onto '${x.name}'`, run: () => act(Go.Rebase(dir, x.name)) },
      ...(x.kind === 'local' && !x.current
        ? [
            null,
            {
              label: 'Rename…',
              run: async () => {
                const n = await ask(`Rename '${x.name}'`, x.name, 'Rename');
                if (n && n !== x.name) act(Go.RenameBranch(dir, x.name, n));
              },
            },
            {
              label: 'Delete',
              danger: true,
              run: async () => {
                open = false;
                try {
                  await r.run(Go.DeleteBranch(dir, x.name, false));
                } catch (e) {
                  if (await sure(`Force delete '${x.name}'?`, String(e), 'Force delete')) act(Go.DeleteBranch(dir, x.name, true));
                }
              },
            },
          ]
        : []),
    ];
  }
</script>

<button class="widget" bind:this={btn} onclick={() => ((open = !open), (filter = ''))} title="Branches">
  <Icon name="branch" size={15} />
  <span class="name">{current || '…'}</span>
  <Icon name="chevron-down" size={14} />
</button>

{#if open && btn}
  <Popup anchor={btn} onclose={() => (open = false)} width={380}>
    <div class="head">
      <!-- svelte-ignore a11y_autofocus -->
      <input placeholder="Search branches" bind:value={filter} autofocus />
    </div>
    <div class="actions">
      <button onclick={() => newBranch('HEAD')}><Icon name="plus" size={14} /> New branch…</button>
      <button onclick={() => ((open = false), remote(dir, 'Fetching', () => Go.Fetch(dir)))}><Icon name="fetch" size={14} /> Fetch</button>
    </div>
    <div class="list">
      {#each groups as g (g.k)}
        {#if g.items.length}
          <button class="group" onclick={() => (closed[g.k] = !closed[g.k])}>
            <Icon name={closed[g.k] && !q ? 'chevron-right' : 'chevron-down'} size={14} />
            {g.title}
          </button>
          {#if !closed[g.k] || q}
            {#each g.items as x (x.full)}
              <button class="ref" class:cur={x.current} onclick={(e) => openMenu(e, items(x))} oncontextmenu={(e) => openMenu(e, items(x))} title={x.upstream ? `${x.name} → ${x.upstream}` : x.name}>
                <Icon name={x.kind === 'tag' ? 'tag' : 'branch'} size={14} />
                <span class="grow">{x.name}</span>
                {#if x.track}<span class="dim small">{x.track.replace(/[[\]]/g, '')}</span>{/if}
                <Icon name="chevron-right" size={12} />
              </button>
            {/each}
          {/if}
        {/if}
      {/each}
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
  }
  .name {
    max-width: 260px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .head {
    padding: 8px 8px 4px;
  }
  .head input {
    width: 100%;
    height: 28px;
  }
  .actions {
    display: flex;
    flex-direction: column;
    padding: 2px 6px 4px;
    border-bottom: 1px solid var(--line);
  }
  .actions button,
  .group,
  .ref {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    height: var(--row-h);
    text-align: left;
    padding: 0 8px;
  }
  .list {
    overflow-y: auto;
    padding: 4px 6px 6px;
  }
  .group {
    color: var(--fg-dim);
    font-size: var(--fs-small);
    font-weight: 600;
  }
  .ref {
    padding-left: 22px;
  }
  .ref.cur .grow {
    color: var(--accent);
    font-weight: 600;
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
</style>
