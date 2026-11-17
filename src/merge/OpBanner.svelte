<script lang="ts">
  import * as Go from '../../wailsjs/go/main/App';
  import { repo } from '../lib/repo.svelte';
  import { openTab, mergeTab } from '../lib/tabs.svelte';
  import { sure, toast } from '../lib/menu.svelte';
  import Icon from '../lib/Icon.svelte';

  let { dir }: { dir: string } = $props();

  const r = $derived(repo(dir));
  const op = $derived(r.op);
  const conflicts = $derived(op?.conflicts ?? []);
  const name = (k: string) => k.charAt(0).toUpperCase() + k.slice(1);
  const act = (p: Promise<unknown>) => r.run(p).catch((e) => toast(String(e)));

  async function abort() {
    if (await sure(`Abort ${op!.kind}?`, 'Your branch goes back to where it was before it started.', 'Abort')) act(Go.OpAbort(dir));
  }
</script>

{#if op?.kind}
  <div class="bar">
    <span class="ico"><Icon name="warning" size={14} /></span>
    <span><strong>{name(op.kind)} in progress</strong>{#if conflicts.length}: {conflicts.length} conflict{conflicts.length === 1 ? '' : 's'}{/if}</span>
    {#if conflicts.length}
      <button class="link" onclick={() => openTab(dir, mergeTab(conflicts[0]))}>Resolve</button>
      {#each conflicts.slice(0, 4) as p (p)}
        <button class="link file mono" title={p} onclick={() => openTab(dir, mergeTab(p))}>{p.split('/').pop()}</button>
      {/each}
      {#if conflicts.length > 4}<span class="dim">+{conflicts.length - 4}</span>{/if}
    {/if}
    <span class="grow"></span>
    <button class="link" disabled={conflicts.length > 0} title={conflicts.length ? 'Resolve conflicts first' : ''} onclick={() => act(Go.OpContinue(dir))}>Continue</button>
    {#if op.kind === 'rebase' || op.kind === 'cherry-pick'}
      <button class="link" onclick={() => act(Go.OpSkip(dir))}>Skip</button>
    {/if}
    <button class="link danger" onclick={abort}>Abort</button>
  </div>
{/if}

<style>
  .bar {
    display: flex;
    align-items: center;
    gap: 10px;
    min-height: 32px;
    padding: 0 10px;
    background: color-mix(in srgb, var(--warn) 10%, var(--bg-panel));
    border-left: 3px solid var(--warn);
    border-bottom: 1px solid var(--line);
    flex-wrap: wrap;
  }
  .ico {
    color: var(--warn);
    display: flex;
  }
  .grow {
    flex: 1;
  }
  .link {
    height: 22px;
    padding: 0 4px;
    color: var(--accent);
  }
  .file {
    color: var(--fg);
    font-size: var(--fs-small);
  }
  .danger {
    color: var(--danger);
  }
</style>
