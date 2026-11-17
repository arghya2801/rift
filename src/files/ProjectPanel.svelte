<script lang="ts">
  import FileTree from './FileTree.svelte';
  import Icon from '../lib/Icon.svelte';

  let { dir }: { dir: string } = $props();

  let collapse = $state(0);
  let reloadKey = $state(0);
  const name = $derived(dir.split(/[\\/]/).pop());
</script>

<div class="panel">
  <div class="header">
    <span class="title">Project</span>
    <span class="dim">{name}</span>
    <span class="grow"></span>
    <button class="icon" title="Refresh" onclick={() => reloadKey++}><Icon name="refresh" size={14} /></button>
    <button class="icon" title="Collapse All" onclick={() => collapse++}><Icon name="collapse-all" size={14} /></button>
  </div>
  <div class="body"><FileTree {dir} {collapse} {reloadKey} /></div>
</div>

<style>
  .panel {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
    background: var(--bg-panel);
  }
  .header {
    display: flex;
    align-items: center;
    gap: 8px;
    height: 34px;
    padding: 0 6px 0 12px;
    flex-shrink: 0;
  }
  .title {
    font-weight: 600;
  }
  .grow {
    flex: 1;
  }
  .icon {
    width: 24px;
    height: 24px;
    padding: 0;
    display: grid;
    place-items: center;
    color: var(--fg-dim);
  }
  .body {
    flex: 1;
    min-height: 0;
  }
</style>
