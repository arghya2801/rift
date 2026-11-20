<script lang="ts" module>
  function lazy<T>(load: () => Promise<{ default: T }>) {
    let pending: Promise<T> | undefined;
    return () => pending ??= load().then((m) => m.default).catch((err) => {
      pending = undefined;
      throw err;
    });
  }
  // CodeMirror-heavy views load on first use, keeping startup parse small.
  const DiffTab = lazy(() => import('../diff/DiffTab.svelte'));
  const Editor = lazy(() => import('../files/Editor.svelte'));
  const MergeView = lazy(() => import('../merge/MergeView.svelte'));
  const RebaseView = lazy(() => import('../rebase/RebaseView.svelte'));
</script>

<script lang="ts">
  import Icon from '../lib/Icon.svelte';
  import { tabsOf, activeTab, focusTab, closeTab, previewOf, pinTab, type EditorTab } from '../lib/tabs.svelte';
  let { dir }: { dir: string } = $props();

  const tabs = $derived(tabsOf(dir));
  const active = $derived(activeTab(dir));
  const icon = (t: EditorTab) => (t.kind === 'file' ? 'file' : t.kind);
</script>

<div class="area">
  {#if tabs.length}
    <div class="bar" role="tablist">
      {#each tabs as t (t.id)}
        <div
          class="tab"
          class:on={active?.id === t.id}
          class:preview={previewOf(dir) === t.id}
          role="tab"
          tabindex="0"
          aria-selected={active?.id === t.id}
          title={'path' in t ? t.path : t.title}
          onclick={() => focusTab(dir, t.id)}
          ondblclick={() => pinTab(dir, t.id)}
          onauxclick={(e) => e.button === 1 && closeTab(dir, t.id)}
          onkeydown={(e) => e.key === 'Enter' && focusTab(dir, t.id)}
        >
          <span class="ic ic-{t.kind}"><Icon name={icon(t)} size={14} /></span>
          <span class="title">{t.title}</span>
          <button class="x" aria-label="Close" onclick={(e) => (e.stopPropagation(), closeTab(dir, t.id))}><Icon name="close" size={12} /></button>
        </div>
      {/each}
    </div>
  {/if}
  <div class="body">
    {#if active}
      {#key active.id}
        {#if active.kind === 'diff'}
          {@const t = active}
          {#await DiffTab() then C}<C {dir} tab={t} />{/await}
        {:else if active.kind === 'file'}
          {@const t = active}
          {#await Editor() then C}<C {dir} rel={t.path} line={t.line} />{/await}
        {:else if active.kind === 'merge'}
          {@const t = active}
          {#await MergeView() then C}<C {dir} path={t.path} ondone={() => closeTab(dir, t.id)} />{/await}
        {:else if active.kind === 'rebase'}
          {@const t = active}
          {#await RebaseView() then C}<C {dir} base={t.base} ondone={() => closeTab(dir, t.id)} />{/await}
        {/if}
      {/key}
    {:else}
      <div class="empty">
        <dl>
          <dt>Commit tool window</dt><dd>Alt+0</dd>
          <dt>Project tool window</dt><dd>Alt+1</dd>
          <dt>Git log</dt><dd>Alt+9</dd>
          <dt>Search everywhere / commands</dt><dd>Ctrl+Shift+P</dd>
          <dt>Close tab</dt><dd>Ctrl+W</dd>
        </dl>
      </div>
    {/if}
  </div>
</div>

<style>
  .area {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    background: var(--bg-editor);
  }
  .bar {
    display: flex;
    flex-shrink: 0;
    height: 36px;
    overflow-x: auto;
    overflow-y: hidden;
    border-bottom: 1px solid var(--line);
    scrollbar-width: none;
  }
  .tab {
    position: relative;
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 0 6px 0 12px;
    white-space: nowrap;
    cursor: pointer;
    color: var(--fg);
  }
  .tab:hover {
    background: var(--bg-hover);
  }
  .tab.on {
    color: var(--fg-bright);
  }
  .tab.preview .title {
    font-style: italic;
  }
  .tab.on::after {
    content: '';
    position: absolute;
    left: 0;
    right: 0;
    bottom: 0;
    height: 2px;
    background: var(--accent);
  }
  .ic {
    color: var(--fg-dim);
  }
  .ic-diff {
    color: var(--vcs-modified);
  }
  .ic-merge {
    color: var(--vcs-conflict);
  }
  .title {
    max-width: 220px;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .x {
    height: 20px;
    padding: 0 3px;
    color: var(--fg-dim);
    visibility: hidden;
  }
  .tab:hover .x,
  .tab.on .x {
    visibility: visible;
  }
  .body {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
  }
  .empty {
    margin: auto;
    color: var(--fg-dim);
  }
  dl {
    display: grid;
    grid-template-columns: auto auto;
    gap: 10px 24px;
    margin: 0;
  }
  dt {
    text-align: right;
  }
  dd {
    margin: 0;
    color: var(--accent);
  }
</style>
