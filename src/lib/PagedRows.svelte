<script lang="ts" module>
  export const pageSize = 200;
</script>
<script lang="ts" generics="T">
  import type { Snippet } from 'svelte';
  let { items, keyFor, page = $bindable(0), onpage, children }: { items: T[]; keyFor: (item: T) => string; onpage?: (page: number) => void; page?: number; children: Snippet<[T, number]> } = $props();
  const pages = $derived(Math.ceil(items.length / pageSize));
  const current = $derived(Math.max(0, Math.min(page, pages - 1)));
  const start = $derived(current * pageSize);
  let timer: ReturnType<typeof setTimeout>;
  function select(target: number) {
    page = Math.max(0, Math.min(target, pages - 1));
    onpage?.(page);
  }
  function hover(event: DragEvent, target: number) {
    if (!event.dataTransfer?.types.includes('application/x-rift-rebase')) return;
    clearTimeout(timer);
    timer = setTimeout(() => { select(target); }, 550);
  }
  $effect(() => () => clearTimeout(timer));
</script>

{#if pages > 1}
  <div class="pages">
    <button aria-label="Previous page" disabled={!current} onclick={() => select(current - 1)}
      ondragenter={(e) => hover(e, current - 1)} ondragleave={() => clearTimeout(timer)}>Previous</button>
    <span aria-live="polite">{start + 1}-{Math.min(start + pageSize, items.length)} of {items.length}</span>
    <button aria-label="Next page" disabled={current + 1 >= pages} onclick={() => select(current + 1)}
      ondragenter={(e) => hover(e, current + 1)} ondragleave={() => clearTimeout(timer)}>Next</button>
  </div>
{/if}
{#each items.slice(start, start + pageSize) as item, i (keyFor(item))}
  {@render children(item, start + i)}
{/each}

<style>
  .pages { display: flex; align-items: center; gap: 8px; padding: 4px 8px; }
</style>
