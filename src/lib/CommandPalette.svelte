<script lang="ts">
  import { tick } from 'svelte';
  import { fuzzyScore } from './fuzzy';
  import { commands, type Command } from './commands.svelte';
  import { applyTheme, savedTheme, themeNames } from './themes';

  applyTheme(savedTheme());

  const builtins: Command[] = themeNames.map((n) => ({ id: `theme:${n}`, title: `Theme: ${n}`, run: () => applyTheme(n) }));

  let dialog: HTMLDialogElement;
  let input = $state<HTMLInputElement>();
  let list = $state<HTMLElement>();
  let query = $state('');
  let index = $state(0);
  let open = $state(false);

  const results = $derived.by(() => {
    const all = [...commands(), ...builtins];
    if (!query.trim()) return all.slice(0, 60);
    return all
      .flatMap((c) => {
        const score = fuzzyScore(c.title, query);
        return score === null ? [] : [{ c, score }];
      })
      .sort((a, b) => b.score - a.score)
      .slice(0, 60)
      .map((r) => r.c);
  });

  $effect(() => {
    void query;
    index = 0;
  });

  async function show() {
    query = '';
    open = true;
    dialog.showModal();
    await tick();
    input?.focus();
  }

  function close() {
    open = false;
    dialog.close();
  }

  function run(c: Command | undefined) {
    if (!c) return;
    close();
    c.run();
  }

  async function move(by: number) {
    if (!results.length) return;
    index = (index + by + results.length) % results.length;
    await tick();
    list?.querySelector('[aria-selected="true"]')?.scrollIntoView({ block: 'nearest' });
  }

  function onGlobal(e: KeyboardEvent) {
    const k = e.key.toLowerCase();
    if (e.ctrlKey && ((e.shiftKey && k === 'p') || (!e.shiftKey && k === 'k'))) {
      e.preventDefault();
      if (open) close();
      else show();
    }
  }

  function onKey(e: KeyboardEvent) {
    e.stopPropagation();
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
      e.preventDefault();
      move(e.key === 'ArrowDown' ? 1 : -1);
    } else if (e.key === 'Enter') {
      e.preventDefault();
      run(results[index]);
    }
  }
</script>

<svelte:window onkeydown={onGlobal} />

<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_noninteractive_element_interactions -->
<dialog bind:this={dialog} aria-label="Command palette" onclose={() => (open = false)} onclick={(e) => e.target === dialog && close()}>
  <input
    bind:this={input}
    bind:value={query}
    onkeydown={onKey}
    placeholder="Run a command"
    spellcheck="false"
    role="combobox"
    aria-expanded="true"
    aria-controls="palette-list"
  />
  <div class="list" id="palette-list" role="listbox" bind:this={list}>
    {#each results as c, i (c.id)}
      <div class="row" role="option" tabindex="-1" aria-selected={i === index} onmousemove={() => (index = i)} onclick={() => run(c)}>
        {c.title}
      </div>
    {:else}
      <p class="empty">Nothing matches.</p>
    {/each}
  </div>
</dialog>

<style>
  dialog {
    width: min(560px, calc(100vw - 32px));
    max-width: none;
    margin: 10vh auto auto;
    padding: 0;
    overflow: hidden;
    border: 1px solid var(--border);
    border-radius: 10px;
    background: var(--bg-chrome);
    color: var(--fg);
    box-shadow: 0 10px 32px rgb(0 0 0 / 0.35);
  }
  dialog::backdrop {
    background: color-mix(in srgb, var(--bg) 45%, transparent);
  }
  input {
    width: 100%;
    box-sizing: border-box;
    padding: 12px 14px;
    border: none;
    border-bottom: 1px solid var(--border);
    background: transparent;
    color: var(--fg-bright);
    font: inherit;
    font-size: 14px;
    outline: none;
  }
  .list {
    max-height: min(420px, 60vh);
    overflow-y: auto;
    padding: 6px;
  }
  .row {
    padding: 6px 10px;
    border-radius: 6px;
    cursor: pointer;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .row[aria-selected='true'] {
    background: color-mix(in srgb, var(--accent) 18%, transparent);
    color: var(--fg-bright);
  }
  .empty {
    margin: 0;
    padding: 14px 10px;
    color: var(--fg-dim);
    font-size: 12px;
  }
</style>
