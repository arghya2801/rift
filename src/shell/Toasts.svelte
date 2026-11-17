<script lang="ts">
  import { toasts, dismiss } from '../lib/menu.svelte';
  import Icon from '../lib/Icon.svelte';
</script>

<div class="stack">
  {#each toasts as t (t.id)}
    <div class="toast {t.kind}" role="alert">
      <span class="ic"><Icon name={t.kind === 'error' ? 'warning' : 'check'} /></span>
      <span class="text">{t.text}</span>
      <button class="x" onclick={() => dismiss(t.id)} aria-label="Dismiss"><Icon name="close" size={14} /></button>
    </div>
  {/each}
</div>

<style>
  .stack {
    position: fixed;
    right: 16px;
    bottom: 36px;
    z-index: 95;
    display: flex;
    flex-direction: column;
    gap: 8px;
    width: min(420px, calc(100vw - 32px));
  }
  .toast {
    display: flex;
    gap: 10px;
    align-items: flex-start;
    padding: 10px 8px 10px 12px;
    border-radius: var(--radius-lg);
    background: var(--bg-chrome);
    box-shadow: var(--shadow-pop);
  }
  .ic {
    padding-top: 1px;
  }
  .error .ic {
    color: var(--danger);
  }
  .info .ic {
    color: var(--ok);
  }
  .text {
    flex: 1;
    white-space: pre-wrap;
    word-break: break-word;
    user-select: text;
    max-height: 30vh;
    overflow: auto;
  }
  .x {
    height: 22px;
    padding: 0 3px;
    color: var(--fg-dim);
  }
</style>
