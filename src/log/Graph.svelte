<script lang="ts" module>
  export const LANE = 12;
  export const ROW = 24;
  const COLORS = ['#3574f0', '#5fb865', '#e6a23c', '#a571e6', '#f75464', '#2fb3bd', '#d17b5e', '#c77dbb'];
  export const color = (i: number) => COLORS[i % COLORS.length];
</script>

<script lang="ts">
  import type { git } from '../../wailsjs/go/models';

  let { row, width }: { row: git.Row; width: number } = $props();
  const x = (i: number) => i * LANE + LANE / 2 + 2;
  const mid = ROW / 2;
</script>

<svg {width} height={ROW}>
  {#each row.lines as l}
    {#if l.bottom}
      <path d="M{x(l.from)} {mid} C{x(l.from)} {ROW} {x(l.to)} {mid} {x(l.to)} {ROW}" stroke={color(l.to)} />
    {:else}
      <path d="M{x(l.from)} 0 C{x(l.from)} {mid} {x(l.to)} 0 {x(l.to)} {mid}" stroke={color(l.from)} />
    {/if}
  {/each}
  <circle cx={x(row.col)} cy={mid} r="3.5" fill={color(row.col)} />
</svg>

<style>
  svg {
    flex-shrink: 0;
    display: block;
  }
  path {
    fill: none;
    stroke-width: 1.6;
  }
</style>
