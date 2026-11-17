<script lang="ts" module>
  const P: Record<string, string> = {
    commit: '<circle cx="8" cy="8" r="2.5"/><path d="M1.5 8h4M10.5 8h4"/>',
    project: '<path d="M2 4.5h4l1.2 1.5H14v6.5H2z"/>',
    git: '<circle cx="4.5" cy="3.5" r="1.5"/><circle cx="4.5" cy="12.5" r="1.5"/><circle cx="11.5" cy="6.5" r="1.5"/><path d="M4.5 5v6M11.5 8c0 2-3 2-6.5 3.5"/>',
    branch: '<circle cx="5" cy="3.5" r="1.5"/><circle cx="5" cy="12.5" r="1.5"/><circle cx="11" cy="4.5" r="1.5"/><path d="M5 5v6M11 6c0 3-6 2.5-6 5"/>',
    folder: '<path d="M2 4h4l1.2 1.5H14v7H2z"/>',
    'folder-open': '<path d="M2 12.5V4h4l1.2 1.5H13v1.5"/><path d="M2 12.5l1.8-5h11l-1.8 5z"/>',
    file: '<path d="M4 1.5h5l3 3v10H4z"/><path d="M9 1.5v3h3"/>',
    'chevron-right': '<path d="M6 4l4 4-4 4"/>',
    'chevron-down': '<path d="M4 6l4 4 4-4"/>',
    close: '<path d="M4.5 4.5l7 7M11.5 4.5l-7 7"/>',
    refresh: '<path d="M13 8a5 5 0 1 1-1.5-3.5"/><path d="M12 2v3h-3"/>',
    fetch: '<path d="M8 2v7M5 6l3 3 3-3"/><path d="M2.5 10.5v3h11v-3"/>',
    pull: '<path d="M8 2v10M4 8l4 4 4-4"/>',
    push: '<path d="M8 14V4M4 8l4-4 4 4"/>',
    search: '<circle cx="7" cy="7" r="4.5"/><path d="M10.5 10.5l3.5 3.5"/>',
    plus: '<path d="M8 3v10M3 8h10"/>',
    minus: '<path d="M3 8h10"/>',
    check: '<path d="M3 8.5l3.5 3.5L13 4.5"/>',
    settings: '<circle cx="8" cy="8" r="2.2"/><path d="M8 1.5v2M8 12.5v2M1.5 8h2M12.5 8h2M3.4 3.4l1.4 1.4M11.2 11.2l1.4 1.4M3.4 12.6l1.4-1.4M11.2 4.8l1.4-1.4"/>',
    more: '<circle cx="3.5" cy="8" r=".8"/><circle cx="8" cy="8" r=".8"/><circle cx="12.5" cy="8" r=".8"/>',
    tag: '<path d="M2 2h6l6 6-6 6-6-6z"/><circle cx="5.5" cy="5.5" r="1"/>',
    stash: '<path d="M2.5 5.5h11v8h-11zM4 3h8M5.5 8.5h5"/>',
    merge: '<circle cx="4.5" cy="3.5" r="1.5"/><circle cx="4.5" cy="12.5" r="1.5"/><circle cx="11.5" cy="12.5" r="1.5"/><path d="M4.5 5v6M4.5 6c0 3 7 2 7 5"/>',
    rebase: '<circle cx="4.5" cy="3.5" r="1.5"/><circle cx="11.5" cy="8" r="1.5"/><circle cx="11.5" cy="12.5" r="1.5"/><path d="M4.5 5v9M11.5 9.5v1.5M6 3.5c4 0 5.5 1 5.5 3"/>',
    undo: '<path d="M5 3L2 6l3 3"/><path d="M2 6h7.5a4 4 0 0 1 0 8H6"/>',
    diff: '<path d="M5 2v8M1 6h8M7 13h8"/>',
    eye: '<path d="M1.5 8s2.5-4.5 6.5-4.5S14.5 8 14.5 8 12 12.5 8 12.5 1.5 8 1.5 8z"/><circle cx="8" cy="8" r="2"/>',
    external: '<path d="M9 2.5h4.5V7M13.5 2.5L7 9"/><path d="M11.5 9.5v4h-9v-9h4"/>',
    filter: '<path d="M2 3h12l-4.5 5.5V13l-3-1.5v-3z"/>',
    'expand-all': '<path d="M4 3l4 4 4-4M4 9l4 4 4-4"/>',
    'collapse-all': '<path d="M4 7l4-4 4 4M4 13l4-4 4 4"/>',
    warning: '<path d="M8 2l6.5 11.5h-13z"/><path d="M8 6.5v3.5M8 12v.01"/>',
  };
</script>

<script lang="ts">
  let { name, size = 16 }: { name: string; size?: number } = $props();
</script>

<svg width={size} height={size} viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
  {@html P[name] ?? ''}
</svg>

<style>
  svg {
    flex-shrink: 0;
    display: block;
  }
</style>
