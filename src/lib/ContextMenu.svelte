<script lang="ts">
  import { menu, closeMenu } from './menu.svelte';

  let el = $state<HTMLDivElement>();
  let pos = $state({ x: 0, y: 0 });

  $effect(() => {
    if (!menu.items.length || !el) return;
    const r = el.getBoundingClientRect();
    pos = { x: Math.min(menu.x, window.innerWidth - r.width - 6), y: Math.min(menu.y, window.innerHeight - r.height - 6) };
  });
</script>

<svelte:window onmousedown={(e) => el && !el.contains(e.target as Node) && closeMenu()} onblur={closeMenu} onkeydown={(e) => e.key === 'Escape' && closeMenu()} />

{#if menu.items.length}
  <div class="menu" bind:this={el} style="left:{pos.x || menu.x}px; top:{pos.y || menu.y}px" role="menu">
    {#each menu.items as item}
      {#if item}
        <button role="menuitem" class:danger={item.danger} onclick={() => { closeMenu(); item.run(); }}>
          <span>{item.label}</span>
          {#if item.hint}<span class="hint">{item.hint}</span>{/if}
        </button>
      {:else}
        <hr />
      {/if}
    {/each}
  </div>
{/if}

<style>
  .menu {
    position: fixed;
    z-index: 100;
    min-width: 200px;
    padding: 5px;
    background: var(--bg-chrome);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-pop);
    display: flex;
    flex-direction: column;
  }
  button {
    display: flex;
    justify-content: space-between;
    gap: 24px;
    height: var(--row-h);
    border: none;
    text-align: left;
    padding: 0 10px;
    white-space: nowrap;
  }
  button:hover {
    background: var(--accent);
    color: var(--on-accent);
  }
  .hint {
    color: var(--fg-dim);
  }
  button:hover .hint {
    color: inherit;
  }
  .danger {
    color: var(--danger);
  }
  hr {
    border: none;
    border-top: 1px solid var(--line);
    margin: 4px 6px;
  }
</style>
