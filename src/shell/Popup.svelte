<script lang="ts">
  import type { Snippet } from 'svelte';

  let { anchor, onclose, width = 360, children }: { anchor: HTMLElement; onclose: () => void; width?: number; children: Snippet } = $props();

  const r = $derived(anchor.getBoundingClientRect());
  const left = $derived(Math.max(8, Math.min(r.left, window.innerWidth - width - 8)));
  let el: HTMLDivElement;

  function outside(e: MouseEvent) {
    if (!el.contains(e.target as Node) && !anchor.contains(e.target as Node)) onclose();
  }
</script>

<svelte:window onmousedown={outside} onkeydown={(e) => e.key === 'Escape' && onclose()} onblur={onclose} />

<div class="popup" bind:this={el} style="left:{left}px; top:{r.bottom + 4}px; width:{width}px">
  {@render children()}
</div>

<style>
  .popup {
    position: fixed;
    z-index: 90;
    max-height: min(560px, calc(100vh - 80px));
    display: flex;
    flex-direction: column;
    background: var(--bg-chrome);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-pop);
    overflow: hidden;
  }
</style>
