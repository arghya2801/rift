<script lang="ts">
  import * as Go from '../../wailsjs/go/main/App';
  import { git } from '../../wailsjs/go/models';
  import { repo } from '../lib/repo.svelte';
  import Icon from '../lib/Icon.svelte';

  let { dir, base, ondone }: { dir: string; base: string; ondone: () => void } = $props();

  const ACTIONS = ['pick', 'edit', 'reword', 'squash', 'fixup', 'drop'];

  let items = $state<git.TodoItem[]>([]);
  let error = $state('');
  let busy = $state(false);
  let dragFrom = $state(-1);
  let sel = $state(0);

  $effect(() => {
    Go.RebaseTodo(dir, base).then((t) => (items = t), (e) => (error = String(e)));
  });

  function move(from: number, to: number) {
    if (from < 0 || from >= items.length || to < 0 || to >= items.length || from === to) return;
    const [it] = items.splice(from, 1);
    items.splice(to, 0, it);
    sel = to;
  }

  function setAction(a: string) {
    const it = items[sel];
    if (!it || (sel === 0 && (a === 'squash' || a === 'fixup'))) return;
    it.action = it.action === a && a !== 'pick' ? 'pick' : a;
  }

  function keys(e: KeyboardEvent) {
    if (e.altKey && (e.key === 'ArrowUp' || e.key === 'ArrowDown')) {
      e.preventDefault();
      move(sel, sel + (e.key === 'ArrowUp' ? -1 : 1));
    } else if (e.key === 'ArrowUp' || e.key === 'ArrowDown') {
      e.preventDefault();
      sel = Math.max(0, Math.min(items.length - 1, sel + (e.key === 'ArrowUp' ? -1 : 1)));
    }
  }

  const invalid = $derived(
    items.length === 0 ? 'Nothing to rebase' : ['squash', 'fixup'].includes(items.find((item) => item.action !== 'drop')?.action ?? '') ? 'The first commit cannot be squash or fixup' : '',
  );

  async function start() {
    busy = true;
    error = '';
    try {
      await repo(dir).run(Go.RunRebase(dir, base, items.map((i) => git.TodoItem.createFrom(i))));
      ondone();
    } catch (e) {
      error = String(e);
    }
    busy = false;
  }
</script>

<div class="toolbar">
  {#each ACTIONS as a}
    <button class:on={items[sel]?.action === a} disabled={!items[sel] || (sel === 0 && (a === 'squash' || a === 'fixup'))} onclick={() => setAction(a)}>{a[0].toUpperCase() + a.slice(1)}</button>
  {/each}
  <span class="sep"></span>
  <button class="icon" title="Move up (Alt+Up)" disabled={sel === 0} onclick={() => move(sel, sel - 1)}><Icon name="push" size={14} /></button>
  <button class="icon" title="Move down (Alt+Down)" disabled={sel >= items.length - 1} onclick={() => move(sel, sel + 1)}><Icon name="pull" size={14} /></button>
  <span class="grow"></span>
  <span class="dim small">onto {base ? base.slice(0, 8) : 'root'} · oldest first</span>
</div>
{#if error || invalid}<div class="error">{error || invalid}</div>{/if}
<div class="table" role="grid" tabindex="0" onkeydown={keys}>
  <div class="thead"><span class="c-grip"></span><span class="c-action">Action</span><span class="c-hash">Hash</span><span>Subject</span></div>
  {#each items as it, i (it.hash)}
    <div
      class="row"
      class:sel={sel === i}
      class:drop={it.action === 'drop'}
      class:dragging={dragFrom === i}
      draggable="true"
      role="row"
      tabindex="-1"
      aria-selected={sel === i}
      onclick={() => (sel = i)}
      onkeydown={() => {}}
      ondragstart={() => (dragFrom = i)}
      ondragover={(e) => e.preventDefault()}
      ondrop={() => {
        move(dragFrom, i);
        dragFrom = -1;
      }}
      ondragend={() => (dragFrom = -1)}
    >
      <span class="c-grip"><Icon name="more" size={14} /></span>
      <span class="c-action">
        <select bind:value={it.action} onclick={(e) => e.stopPropagation()}>
          {#each ACTIONS as a}<option value={a} disabled={i === 0 && (a === 'squash' || a === 'fixup')}>{a}</option>{/each}
        </select>
      </span>
      <span class="c-hash mono">{it.hash.slice(0, 8)}</span>
      <span class="subject">{it.subject}</span>
    </div>
    {#if it.action === 'reword'}
      <div class="msg"><textarea bind:value={it.message} rows="3"></textarea></div>
    {/if}
  {/each}
</div>
<div class="footer">
  <span class="grow"></span>
  <button class="outline" onclick={ondone} disabled={busy}>Cancel</button>
  <button class="primary" onclick={start} disabled={busy || !!invalid}>{busy ? 'Rebasing…' : 'Start Rebasing'}</button>
</div>

<style>
  .toolbar,
  .footer {
    display: flex;
    gap: 4px;
    align-items: center;
    padding: 4px 8px;
    background: var(--bg-panel);
  }
  .toolbar {
    border-bottom: 1px solid var(--line);
  }
  .footer {
    border-top: 1px solid var(--line);
    padding: 8px 12px;
    gap: 8px;
  }
  .toolbar button.on {
    background: var(--selection);
  }
  .icon {
    width: 26px;
    padding: 0;
    display: grid;
    place-items: center;
    color: var(--fg-dim);
  }
  .sep {
    width: 1px;
    height: 16px;
    background: var(--line);
    margin: 0 4px;
  }
  .grow {
    flex: 1;
  }
  .small {
    font-size: var(--fs-small);
  }
  .error {
    padding: 6px 12px;
    color: var(--danger);
    white-space: pre-wrap;
  }
  .table {
    flex: 1;
    overflow-y: auto;
    outline: none;
    background: var(--bg-editor);
  }
  .thead,
  .row {
    display: flex;
    align-items: center;
    gap: 8px;
    height: 28px;
    padding: 0 10px;
  }
  .thead {
    color: var(--fg-dim);
    font-size: var(--fs-small);
    border-bottom: 1px solid var(--line);
    position: sticky;
    top: 0;
    background: var(--bg-editor);
  }
  .row:hover {
    background: var(--bg-hover);
  }
  .row.sel {
    background: var(--selection);
  }
  .row.dragging {
    opacity: 0.5;
  }
  .row.drop .subject {
    text-decoration: line-through;
    color: var(--fg-dim);
  }
  .c-grip {
    width: 16px;
    color: var(--fg-dim);
    cursor: grab;
    display: flex;
  }
  .c-action {
    width: 90px;
  }
  .c-action select {
    height: 22px;
    padding: 0 4px;
    width: 100%;
  }
  .c-hash {
    width: 72px;
    color: var(--fg-dim);
  }
  .subject {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .msg {
    padding: 4px 10px 8px 140px;
  }
  textarea {
    width: 100%;
    resize: vertical;
  }
</style>
