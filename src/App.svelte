<script lang="ts">
  import * as Go from '../wailsjs/go/main/App';
  import Icon from './lib/Icon.svelte';
  import ContextMenu from './lib/ContextMenu.svelte';
  import CommandPalette from './lib/CommandPalette.svelte';
  import { registerCommands } from './lib/commands.svelte';
  import { repo } from './lib/repo.svelte';
  import { activeTab, closeTab } from './lib/tabs.svelte';
  import { toast } from './lib/menu.svelte';
  import Dialog from './shell/Dialog.svelte';
  import Toasts from './shell/Toasts.svelte';
  import WorktreeWidget from './shell/WorktreeWidget.svelte';
  import BranchWidget from './shell/BranchWidget.svelte';
  import EditorArea from './shell/EditorArea.svelte';
  import Welcome from './home/Welcome.svelte';
  import CommitPanel from './changes/CommitPanel.svelte';
  import ProjectPanel from './files/ProjectPanel.svelte';
  import LogPanel from './log/LogPanel.svelte';
  import OpBanner from './merge/OpBanner.svelte';
  import { app, uiOf, refreshRepos, select, addRepo, remote, norm, repoOf } from './shell/app.svelte';

  const dir = $derived(app.current?.path ?? '');
  const r = $derived(dir ? repo(dir) : null);
  const ui = $derived(dir ? uiOf(dir) : null);

  $effect(() => {
    const branch = r?.branch;
    const wt = app.current;
    if (!branch || !wt) return;
    const detached = branch === '(detached)';
    if (wt.detached !== detached || (!detached && wt.branch !== branch)) refreshRepos();
  });

  function toggleLeft(k: 'commit' | 'project') {
    if (ui) ui.left = ui.left === k ? null : k;
  }
  function toggleBottom() {
    if (ui) ui.bottom = !ui.bottom;
  }

  function drag(e: PointerEvent, axis: 'x' | 'y') {
    const el = e.currentTarget as HTMLElement;
    el.setPointerCapture(e.pointerId);
    const host = el.parentElement!.getBoundingClientRect();
    const move = (ev: PointerEvent) => {
      if (axis === 'x') app.leftWidth = Math.max(220, Math.min(ev.clientX - host.left, host.width - 300));
      else app.bottomHeight = Math.max(0.12, Math.min((host.bottom - ev.clientY) / host.height, 0.85));
    };
    const up = () => {
      el.removeEventListener('pointermove', move);
      el.removeEventListener('pointerup', up);
    };
    el.addEventListener('pointermove', move);
    el.addEventListener('pointerup', up);
  }

  function onKey(e: KeyboardEvent) {
    if (!dir) return;
    if (e.altKey && !e.ctrlKey && !e.shiftKey) {
      if (e.key === '0') toggleLeft('commit');
      else if (e.key === '1') toggleLeft('project');
      else if (e.key === '9') toggleBottom();
      else return;
      e.preventDefault();
    } else if (e.ctrlKey && !e.shiftKey && e.key.toLowerCase() === 'w') {
      const t = activeTab(dir);
      if (t) {
        e.preventDefault();
        closeTab(dir, t.id);
      }
    }
  }

  const withDir = (f: (d: string) => unknown) => () => dir && f(dir);
  registerCommands('app', [
    { id: 'open-repo', title: 'Open repository…', run: addRepo },
    { id: 'welcome', title: 'Welcome screen', run: () => select(null) },
    { id: 'toggle-commit', title: 'Toggle Commit tool window', run: () => toggleLeft('commit') },
    { id: 'toggle-project', title: 'Toggle Project tool window', run: () => toggleLeft('project') },
    { id: 'toggle-git', title: 'Toggle Git tool window', run: toggleBottom },
    { id: 'fetch', title: 'Git: Fetch', run: withDir((d) => remote(d, 'Fetching', () => Go.Fetch(d))) },
    { id: 'pull', title: 'Git: Pull', run: withDir((d) => remote(d, 'Pulling', () => Go.Pull(d, false))) },
    { id: 'pull-rebase', title: 'Git: Pull with rebase', run: withDir((d) => remote(d, 'Pulling', () => Go.Pull(d, true))) },
    { id: 'push', title: 'Git: Push', run: withDir((d) => remote(d, 'Pushing', () => Go.Push(d, false))) },
    { id: 'force-push', title: 'Git: Force push (with lease)', run: withDir((d) => remote(d, 'Pushing', () => Go.Push(d, true))) },
    { id: 'stash', title: 'Git: Stash changes (incl. untracked)', run: withDir((d) => repo(d).run(Go.StashPush(d, '', true)).catch((e) => toast(String(e)))) },
    { id: 'atc', title: 'Open worktree in ATC', run: withDir((d) => Go.OpenInATC(d).catch((e) => toast(String(e)))) },
    { id: 'external', title: 'Open worktree in external editor', run: withDir((d) => Go.OpenExternal(d, '.', 0).catch((e) => toast(String(e)))) },
  ]);

  (async () => {
    const start = await Go.Initial();
    await refreshRepos();
    const wt = start && app.repos.flatMap((x) => x.worktrees ?? []).find((w) => norm(w.path) === norm(start));
    if (wt) select(wt);
  })();

  const openPalette = () => window.dispatchEvent(new KeyboardEvent('keydown', { key: 'p', ctrlKey: true, shiftKey: true }));
</script>

<svelte:window onkeydown={onKey} />

<ContextMenu />
<CommandPalette />
<Dialog />
<Toasts />

<div class="frame">
  <header class="toolbar">
    <WorktreeWidget />
    {#if dir}<BranchWidget {dir} />{/if}
    <span class="grow"></span>
    {#if dir}
      <button class="tb" title="Fetch" disabled={!!app.busy} onclick={() => remote(dir, 'Fetching', () => Go.Fetch(dir))}><Icon name="fetch" /></button>
      <button class="tb" title="Pull" disabled={!!app.busy} onclick={() => remote(dir, 'Pulling', () => Go.Pull(dir, false))}><Icon name="pull" /></button>
      <button class="tb" title="Push" disabled={!!app.busy} onclick={() => remote(dir, 'Pushing', () => Go.Push(dir, false))}><Icon name="push" /></button>
      <span class="sep"></span>
    {/if}
    <button class="tb" title="Search everywhere (Ctrl+Shift+P)" onclick={openPalette}><Icon name="search" /></button>
  </header>

  {#if !dir || !ui || !r}
    <Welcome />
  {:else}
    <div class="workspace">
      <nav class="stripe">
        <button class="st" class:on={ui.left === 'commit'} title="Commit (Alt+0)" onclick={() => toggleLeft('commit')}><Icon name="commit" size={18} /></button>
        <button class="st" class:on={ui.left === 'project'} title="Project (Alt+1)" onclick={() => toggleLeft('project')}><Icon name="project" size={18} /></button>
        <span class="grow"></span>
        <button class="st" class:on={ui.bottom} title="Git (Alt+9)" onclick={toggleBottom}><Icon name="git" size={18} /></button>
      </nav>
      <div class="main">
        <div class="upper" style="flex-basis: {ui.bottom ? (1 - app.bottomHeight) * 100 : 100}%">
          {#if ui.left}
            <aside class="tool" style="width:{app.leftWidth}px">
              {#key dir}
                {#if ui.left === 'commit'}<CommitPanel {dir} />{:else}<ProjectPanel {dir} />{/if}
              {/key}
            </aside>
            <div class="resize-x" role="separator" aria-orientation="vertical" onpointerdown={(e) => drag(e, 'x')}></div>
          {/if}
          <div class="center">
            {#key dir}
              <OpBanner {dir} />
              <EditorArea {dir} />
            {/key}
          </div>
        </div>
        {#if ui.bottom}
          <div class="resize-y" role="separator" aria-orientation="horizontal" onpointerdown={(e) => drag(e, 'y')}></div>
          <div class="bottom" style="flex-basis: {app.bottomHeight * 100}%">
            {#key dir}<LogPanel {dir} />{/key}
          </div>
        {/if}
      </div>
    </div>
  {/if}

  <footer class="status">
    {#if dir && r}
      <span class="crumb" title={dir}>{repoOf(app.current)?.name ?? ''} › {dir}</span>
      <span class="grow"></span>
      {#if app.busy}<span class="busy">{app.busy}</span>{/if}
      {#if r.op?.kind}<span class="warn">{r.op.kind} in progress</span>{/if}
      {#if r.status?.upstream}<span>↑{r.status.ahead} ↓{r.status.behind}</span>{/if}
      <span class="br"><Icon name="branch" size={13} /> {r.branch}</span>
    {:else}
      <span class="grow"></span>
      {#if app.busy}<span class="busy">{app.busy}</span>{/if}
    {/if}
  </footer>
</div>

<style>
  .frame {
    height: 100%;
    display: flex;
    flex-direction: column;
  }
  .toolbar {
    display: flex;
    align-items: center;
    gap: 2px;
    height: 40px;
    flex-shrink: 0;
    padding: 0 8px;
    background: var(--bg-panel);
    border-bottom: 1px solid var(--line);
  }
  .tb {
    height: 28px;
    width: 30px;
    padding: 0;
    display: grid;
    place-items: center;
  }
  .sep {
    width: 1px;
    height: 18px;
    margin: 0 6px;
    background: var(--line);
  }
  .grow {
    flex: 1;
  }
  .workspace {
    flex: 1;
    min-height: 0;
    display: flex;
  }
  .stripe {
    width: 40px;
    flex-shrink: 0;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
    padding: 6px 0;
    background: var(--bg-panel);
    border-right: 1px solid var(--line);
  }
  .st {
    width: 30px;
    height: 30px;
    padding: 0;
    display: grid;
    place-items: center;
    color: var(--fg-dim);
  }
  .st.on {
    background: var(--selection);
    color: var(--fg-bright);
  }
  .main {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
  }
  .upper {
    min-height: 0;
    flex-grow: 1;
    flex-shrink: 1;
    display: flex;
  }
  .tool {
    flex-shrink: 0;
    min-width: 0;
    display: flex;
    flex-direction: column;
    background: var(--bg-panel);
  }
  .center {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
  }
  .resize-x {
    width: 1px;
    flex-shrink: 0;
    background: var(--line);
    cursor: col-resize;
    position: relative;
  }
  .resize-x::after {
    content: '';
    position: absolute;
    inset: 0 -3px;
  }
  .resize-y {
    height: 1px;
    flex-shrink: 0;
    background: var(--line);
    cursor: row-resize;
    position: relative;
  }
  .resize-y::after {
    content: '';
    position: absolute;
    inset: -3px 0;
  }
  .bottom {
    min-height: 0;
    flex-grow: 0;
    flex-shrink: 0;
    display: flex;
    flex-direction: column;
    background: var(--bg-panel);
  }
  .status {
    display: flex;
    align-items: center;
    gap: 14px;
    height: 24px;
    flex-shrink: 0;
    padding: 0 10px;
    font-size: var(--fs-small);
    color: var(--fg-dim);
    background: var(--bg-panel);
    border-top: 1px solid var(--line);
  }
  .crumb {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    min-width: 0;
  }
  .br {
    display: flex;
    align-items: center;
    gap: 4px;
    color: var(--fg);
  }
  .warn {
    color: var(--warn);
  }
  .busy {
    color: var(--accent);
  }
</style>
