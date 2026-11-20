// Editor tabs in the center area, per worktree, like IntelliJ's editor tabs.
export type EditorTab =
  | { kind: 'diff'; id: string; title: string; path: string; source: 'unstaged' | 'staged' | 'head' | 'commit'; hash?: string; oldPath?: string }
  | { kind: 'file'; id: string; title: string; path: string; line?: number }
  | { kind: 'merge'; id: string; title: string; path: string }
  | { kind: 'rebase'; id: string; title: string; base: string };

type Group = { tabs: EditorTab[]; active: string | null; preview: string | null };

const groups = $state<Record<string, Group>>({});

// Writers create the group; readers (used in $derived) must not.
function group(dir: string): Group {
  if (!groups[dir]) groups[dir] = { tabs: [], active: null, preview: null };
  return groups[dir];
}

export function forgetTabs(dir: string) { delete groups[dir]; }

const none: EditorTab[] = [];
export const tabsOf = (dir: string) => groups[dir]?.tabs ?? none;
export const activeTab = (dir: string) => {
  const g = groups[dir];
  return g?.tabs.find((t) => t.id === g.active) ?? null;
};

// Opens or focuses a tab. A preview (single-click) replaces the previous preview tab, like IntelliJ.
export function openTab(dir: string, tab: EditorTab, preview = false) {
  const g = group(dir);
  const i = g.tabs.findIndex((t) => t.id === tab.id);
  if (i >= 0) {
    g.tabs[i] = tab;
    if (!preview) pinTab(dir, tab.id);
  } else {
    const p = preview ? g.tabs.findIndex((t) => t.id === g.preview) : -1;
    if (p >= 0) g.tabs[p] = tab;
    else g.tabs.push(tab);
    if (preview) g.preview = tab.id;
  }
  g.active = tab.id;
}

export const previewOf = (dir: string) => groups[dir]?.preview ?? null;
export function pinTab(dir: string, id: string) {
  const g = group(dir);
  if (g.preview === id) g.preview = null;
}

export function focusTab(dir: string, id: string) {
  group(dir).active = id;
}

export function closeTab(dir: string, id: string) {
  const g = group(dir);
  const i = g.tabs.findIndex((t) => t.id === id);
  if (i < 0) return;
  pinTab(dir, id);
  g.tabs.splice(i, 1);
  if (g.active === id) g.active = g.tabs[Math.min(i, g.tabs.length - 1)]?.id ?? null;
}

const leaf = (p: string) => p.split('/').pop() ?? p;
export const diffTab = (path: string, source: 'unstaged' | 'staged' | 'head', ): EditorTab => ({ kind: 'diff', id: `diff:${source}:${path}`, title: leaf(path), path, source });
export const commitDiffTab = (hash: string, path: string, oldPath?: string): EditorTab => ({ kind: 'diff', id: `diff:${hash}:${path}`, title: `${leaf(path)} @ ${hash.slice(0, 7)}`, path, source: 'commit', hash, oldPath });
export const fileTab = (path: string, line?: number): EditorTab => ({ kind: 'file', id: `file:${path}`, title: leaf(path), path, line });
export const mergeTab = (path: string): EditorTab => ({ kind: 'merge', id: `merge:${path}`, title: `Merge ${leaf(path)}`, path });
export const rebaseTab = (base: string): EditorTab => ({ kind: 'rebase', id: 'rebase', title: 'Interactive rebase', base });
