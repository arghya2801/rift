<script lang="ts">
  import { tick } from 'svelte';
  import { dialog } from '../lib/menu.svelte';

  let el = $state<HTMLDialogElement>();
  let input = $state<HTMLInputElement>();
  let value = $state('');

  $effect(() => {
    const req = dialog.req;
    if (!req || !el) return;
    value = req.input ?? '';
    if (!el.open) el.showModal();
    tick().then(() => {
      input?.focus();
      input?.select();
    });
  });

  function done(ok: boolean) {
    const req = dialog.req;
    dialog.req = null;
    el?.close();
    req?.resolve(ok ? (req.input !== undefined ? value : '') : null);
  }
</script>

<dialog
  bind:this={el}
  oncancel={(e) => {
    e.preventDefault();
    done(false);
  }}
>
  {#if dialog.req}
    {@const req = dialog.req}
    <form
      method="dialog"
      onsubmit={(e) => {
        e.preventDefault();
        if (req.input === undefined || value.trim()) done(true);
      }}
    >
      <h2>{req.title}</h2>
      {#if req.message}<p>{req.message}</p>{/if}
      {#if req.input !== undefined}
        <input bind:this={input} bind:value spellcheck="false" aria-label={req.title} />
      {/if}
      <div class="row">
        <button type="button" class="outline" onclick={() => done(false)}>Cancel</button>
        <button type="submit" class="primary" class:danger={req.danger} disabled={req.input !== undefined && !value.trim()}>{req.ok}</button>
      </div>
    </form>
  {/if}
</dialog>

<style>
  dialog {
    width: min(440px, calc(100vw - 32px));
    padding: 0;
    border: none;
    border-radius: var(--radius-lg);
    background: var(--bg-chrome);
    color: var(--fg);
    box-shadow: var(--shadow-pop);
  }
  dialog::backdrop {
    background: rgb(0 0 0 / 0.35);
  }
  form {
    padding: 16px 18px 14px;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  h2 {
    margin: 0;
    font-size: 13px;
    font-weight: 600;
    color: var(--fg-bright);
  }
  p {
    margin: 0;
    color: var(--fg);
    white-space: pre-wrap;
    user-select: text;
    max-height: 40vh;
    overflow: auto;
  }
  input {
    width: 100%;
    height: 28px;
  }
  .row {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    margin-top: 4px;
  }
  .row button {
    min-width: 72px;
  }
  .primary.danger {
    background: var(--danger-strong);
  }
</style>
