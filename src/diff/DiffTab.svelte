<script lang="ts">
  import { untrack } from 'svelte';
  import { MergeView, getChunks, goToNextChunk, goToPreviousChunk, unifiedMergeView, updateOriginalDoc, getOriginalDoc, originalDocChangeEffect } from '@codemirror/merge';
  import { EditorView, keymap } from '@codemirror/view';
  import { ChangeSet, EditorState, Prec, type Extension } from '@codemirror/state';
  import * as Go from '../../wailsjs/go/main/App';
  import type { git } from '../../wailsjs/go/models';
  import { openTab, fileTab, type EditorTab } from '../lib/tabs.svelte';
  import { repo } from '../lib/repo.svelte';
  import { toast } from '../lib/menu.svelte';
  import { cmBase } from '../lib/cm';
  import { partials } from '../changes/partial.svelte';
  import { applyLines, revertLines } from './lines';

  let { dir, tab }: { dir: string; tab: Extract<EditorTab, { kind: 'diff' }> } = $props();

  type Opts = { unified: boolean; collapse: boolean; words: boolean };
  const KEY = 'rift.diff';
  function loadOpts(): Opts {
    try {
      return { unified: false, collapse: true, words: true, ...JSON.parse(localStorage.getItem(KEY) ?? '{}') };
    } catch {
      return { unified: false, collapse: true, words: true };
    }
  }

  const r = $derived(repo(dir));
  const src = $derived(tab.source);
  // The side a hunk action writes into: a = index / partial, b = index when unstaging.
  const writable = $derived(src === 'unstaged' || src === 'head' ? 'a' : src === 'staged' ? 'b' : null);
  const labels = $derived(
    src === 'unstaged' ? ['Index', 'Working tree']
    : src === 'staged' ? ['HEAD', 'Staged']
    : src === 'head' ? ['HEAD + included', 'Working tree']
    : [`${tab.hash?.slice(0, 7)}^`, tab.hash?.slice(0, 7) ?? ''],
  );
  const verb = $derived(src === 'staged' ? 'Unstage' : src === 'head' ? 'Include' : 'Stage');
  const crumbs = $derived(tab.path.split('/'));

  let host: HTMLDivElement;
  let opts = $state<Opts>(loadOpts());
  let error = $state('');
  let binary = $state(false);
  let loading = $state(true);
  let count = $state(0);

  let base: Extension[] | null = null;
  let mv: MergeView | undefined;
  let uv: EditorView | undefined;
  const docs = { a: '', b: '' };
  let syncing = false;
  let seq = 0;

  const main = () => mv?.b ?? uv;

  function persist(text: string) {
    if (src === 'head') {
      if (!partials[dir]) partials[dir] = {};
      partials[dir][tab.path] = text;
      return;
    }
    r.run(Go.StageContent(dir, tab.path, text)).catch((e) => toast(String(e)));
  }

  function recount() {
    const v = main();
    count = v ? (getChunks(v.state)?.chunks.length ?? 0) : 0;
  }

  function delta(cur: string, next: string) {
    let s = 0;
    while (s < cur.length && s < next.length && cur[s] === next[s]) s++;
    let e = 0;
    while (e < cur.length - s && e < next.length - s && cur[cur.length - 1 - e] === next[next.length - 1 - e]) e++;
    return { from: s, to: cur.length - e, insert: next.slice(s, next.length - e) };
  }

  function setA(text: string) {
    if (text === docs.a) return;
    if (mv) mv.a.dispatch({ changes: delta(docs.a, text) });
    else if (uv) uv.dispatch({ effects: originalDocChangeEffect(uv.state, ChangeSet.of([delta(docs.a, text)], docs.a.length)) });
  }

  function setB(text: string) {
    const v = main();
    if (v && text !== docs.b) v.dispatch({ changes: delta(docs.b, text) });
  }

  const next = () => {
    const v = main();
    if (v) goToNextChunk({ state: v.state, dispatch: v.dispatch });
    return true;
  };
  const prev = () => {
    const v = main();
    if (v) goToPreviousChunk({ state: v.state, dispatch: v.dispatch });
    return true;
  };

  function openFile() {
    const v = main();
    openTab(dir, fileTab(tab.path, v ? v.state.doc.lineAt(v.state.selection.main.head).number : 1));
    return true;
  }

  function stageLines() {
    const v = main();
    if (!v || !writable) return true;
    const st = v.state;
    const chunks = getChunks(st)?.chunks ?? [];
    const sel = st.selection.main;
    const first = st.doc.lineAt(sel.from).number - 1;
    let last = st.doc.lineAt(sel.to).number - 1;
    if (sel.to > sel.from && st.doc.lineAt(sel.to).from === sel.to) last--;
    const b = st.doc.toString();
    if (writable === 'a') setA(applyLines(docs.a, b, chunks, first, last));
    else setB(revertLines(docs.a, b, chunks, first, last));
    return true;
  }

  function control(title: string, glyph: string) {
    const b = document.createElement('button');
    b.className = 'hunk';
    b.textContent = glyph;
    b.title = title;
    return b;
  }

  function destroy() {
    mv?.destroy();
    uv?.destroy();
    mv = uv = undefined;
  }

  function build() {
    if (!base || binary) return;
    destroy();
    const ro = EditorState.readOnly.of(true);
    const nav = Prec.highest(
      keymap.of([
        { key: 'F7', run: next },
        { key: 'Shift-F7', run: prev },
        { key: 'Alt-ArrowDown', run: next },
        { key: 'Alt-ArrowUp', run: prev },
        { key: 'Mod-Enter', run: stageLines },
        { key: 'F4', run: () => src !== 'commit' && openFile() },
      ]),
    );
    const watchA = EditorView.updateListener.of((u) => {
      if (!u.docChanged) return;
      docs.a = u.state.doc.toString();
      if (!syncing && writable === 'a') persist(docs.a);
      recount();
    });
    const watchB = EditorView.updateListener.of((u) => {
      if (u.docChanged) {
        docs.b = u.state.doc.toString();
        if (!syncing && writable === 'b') persist(docs.b);
      }
      if (uv && u.transactions.some((tr) => tr.effects.some((e) => e.is(updateOriginalDoc)))) {
        docs.a = getOriginalDoc(u.state).toString();
        if (!syncing && writable === 'a') persist(docs.a);
      }
      if (u.docChanged || u.transactions.some((tr) => tr.effects.length)) recount();
    });
    const collapse = opts.collapse ? { margin: 3, minSize: 6 } : undefined;
    if (opts.unified) {
      const want = writable === 'a' ? 'accept' : 'reject';
      uv = new EditorView({
        parent: host,
        state: EditorState.create({
          doc: docs.b,
          extensions: [
            ...base,
            ro,
            nav,
            watchB,
            unifiedMergeView({
              original: docs.a,
              highlightChanges: opts.words,
              collapseUnchanged: collapse,
              mergeControls: writable
                ? (type, action) => {
                    if (type !== want) return document.createElement('span');
                    const b = control(`${verb} hunk`, writable === 'a' ? '»' : '«');
                    b.onmousedown = action;
                    return b;
                  }
                : false,
            }),
          ],
        }),
      });
    } else {
      mv = new MergeView({
        a: { doc: docs.a, extensions: [...base, ro, nav, watchA] },
        b: { doc: docs.b, extensions: [...base, ro, nav, watchB] },
        parent: host,
        highlightChanges: opts.words,
        collapseUnchanged: collapse,
        revertControls: writable === 'a' ? 'b-to-a' : writable === 'b' ? 'a-to-b' : undefined,
        renderRevertControl: () => control(`${verb} hunk`, writable === 'a' ? '«' : '»'),
      });
    }
    recount();
  }

  async function sides(): Promise<git.Sides> {
    if (src === 'unstaged') return Go.FileSides(dir, tab.path, false);
    if (src === 'staged') return Go.FileSides(dir, tab.path, true);
    if (src === 'head') return Go.HeadSides(dir, tab.path);
    return Go.CommitSides(dir, tab.hash ?? '', tab.path, tab.oldPath ?? '');
  }

  async function reload() {
    const n = ++seq;
    try {
      const [s, ext] = await Promise.all([sides(), base ? Promise.resolve(base) : cmBase(tab.path)]);
      if (n !== seq) return;
      base = ext;
      error = '';
      const a = src === 'head' ? (partials[dir]?.[tab.path] ?? s.old) : s.old;
      if (s.binary !== binary) {
        binary = s.binary;
        if (binary) destroy();
      }
      if (binary) return;
      if (!mv && !uv) {
        docs.a = a;
        docs.b = s.new;
        build();
      } else {
        syncing = true;
        try {
          setA(a);
          setB(s.new);
        } finally {
          syncing = false;
        }
      }
    } catch (e) {
      error = String(e);
      toast(error);
    } finally {
      loading = false;
    }
  }

  function setOpt<K extends keyof Opts>(k: K, v: Opts[K]) {
    opts = { ...opts, [k]: v };
    try {
      localStorage.setItem(KEY, JSON.stringify(opts));
    } catch {}
    build();
  }

  $effect(() => {
    r.gitRev;
    r.snap;
    untrack(reload);
  });

  // Bumping seq makes a load still in flight stop instead of building into a closed tab.
  $effect(() => () => {
    seq++;
    destroy();
  });
</script>

<div class="diff">
  <div class="bar">
    <div class="crumb mono" title={tab.path}>
      {#each crumbs.slice(0, -1) as c}<span class="dim">{c}/</span>{/each}<span class="name">{crumbs.at(-1)}</span>
    </div>
    <div class="group">
      <button class="icon" title="Previous difference (Shift+F7)" onclick={prev} disabled={!count}>↑</button>
      <button class="icon" title="Next difference (F7)" onclick={next} disabled={!count}>↓</button>
      <span class="dim count">{count ? `${count} difference${count === 1 ? '' : 's'}` : loading ? '' : 'No differences'}</span>
    </div>
    <div class="group">
      <select value={opts.unified ? 'u' : 's'} onchange={(e) => setOpt('unified', e.currentTarget.value === 'u')}>
        <option value="s">Side-by-side</option>
        <option value="u">Unified</option>
      </select>
      <button class:on={opts.collapse} onclick={() => setOpt('collapse', !opts.collapse)} title="Collapse unchanged fragments">Collapse unchanged</button>
      <button class:on={opts.words} onclick={() => setOpt('words', !opts.words)} title="Highlight changed words">Highlight words</button>
    </div>
    <span class="grow"></span>
    {#if src !== 'commit'}
      <button class="outline" onclick={openFile} title="Edit this file at the cursor line (F4)">Open file</button>
    {/if}
    {#if writable && !binary}
      <button class="outline" onclick={stageLines} title="Select lines on the right side, then Ctrl+Enter">{verb} selected lines</button>
    {/if}
  </div>
  {#if !binary}
    <div class="labels" class:one={opts.unified}>
      {#if opts.unified}
        <span>{labels[0]} → {labels[1]}</span>
      {:else}
        <span>{labels[0]}</span><span>{labels[1]}</span>
      {/if}
    </div>
  {/if}
  {#if error}<div class="error">{error}</div>{/if}
  {#if binary}<div class="empty">Binary file, contents not shown</div>{/if}
  <div class="host" bind:this={host}></div>
</div>

<style>
  .diff {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    background: var(--bg-editor);
  }
  .bar {
    height: 32px;
    flex-shrink: 0;
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 0 8px 0 12px;
    border-bottom: 1px solid var(--line);
    background: var(--bg-panel);
    white-space: nowrap;
  }
  .crumb {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    font-size: var(--fs-small);
  }
  .name {
    color: var(--fg-bright);
  }
  .group {
    display: flex;
    align-items: center;
    gap: 2px;
  }
  .icon {
    width: 26px;
    padding: 0;
  }
  .count {
    margin-left: 6px;
    font-size: var(--fs-small);
  }
  select {
    height: 24px;
    padding: 0 6px;
    background: transparent;
    border-color: transparent;
  }
  select:hover {
    background: var(--bg-hover);
  }
  .group button:not(.icon) {
    font-size: var(--fs-small);
    color: var(--fg-dim);
  }
  .group button.on {
    color: var(--fg);
    background: color-mix(in srgb, var(--accent) 22%, transparent);
  }
  .grow {
    flex: 1;
  }
  .labels {
    display: grid;
    grid-template-columns: 1fr 1fr;
    flex-shrink: 0;
    font-size: var(--fs-small);
    color: var(--fg-dim);
    border-bottom: 1px solid var(--line);
  }
  .labels.one {
    grid-template-columns: 1fr;
  }
  .labels span {
    padding: 3px 12px;
  }
  .error {
    padding: 6px 12px;
    color: var(--danger);
  }
  .empty {
    margin: auto;
    color: var(--fg-dim);
  }
  .host {
    flex: 1;
    min-height: 0;
    overflow: auto;
  }
  .host :global(.cm-mergeView) {
    min-height: 100%;
  }
  .host :global(.cm-mergeViewEditors) {
    min-height: 100%;
  }
  .host :global(.cm-editor) {
    height: auto;
  }
  .host :global(> .cm-editor) {
    min-height: 100%;
  }
  .host :global(.cm-merge-b .cm-changedLine),
  .host :global(.cm-insertedLine) {
    background: rgba(73, 156, 84, 0.18);
  }
  .host :global(.cm-merge-a .cm-changedLine),
  .host :global(.cm-deletedChunk) {
    background: rgba(201, 79, 79, 0.18);
  }
  .host :global(.cm-merge-b .cm-changedText),
  .host :global(.cm-insertedLine .cm-changedText) {
    background: rgba(73, 156, 84, 0.42);
  }
  .host :global(.cm-merge-a .cm-changedText),
  .host :global(.cm-deletedText),
  .host :global(.cm-deletedChunk .cm-deletedText) {
    background: rgba(201, 79, 79, 0.42);
  }
  .host :global(.cm-merge-b .cm-changedLineGutter),
  .host :global(.cm-insertedLineGutter) {
    background: #549159;
  }
  .host :global(.cm-merge-a .cm-changedLineGutter),
  .host :global(.cm-deletedLineGutter) {
    background: #c94f4f;
  }
  .host :global(.cm-collapsedLines) {
    background: var(--bg-panel);
    color: var(--fg-dim);
    font-family: var(--font-ui);
    font-size: var(--fs-small);
  }
  .host :global(.cm-merge-revert) {
    width: 24px;
    background: var(--bg-panel);
    border-left: 1px solid var(--line);
    border-right: 1px solid var(--line);
  }
  .host :global(button.hunk) {
    width: 18px;
    height: 18px;
    padding: 0;
    border: none;
    border-radius: 3px;
    background: transparent;
    color: var(--fg-dim);
    font-size: 13px;
    line-height: 16px;
    cursor: pointer;
  }
  .host :global(button.hunk:hover) {
    background: var(--bg-hover);
    color: var(--fg-bright);
  }
  .host :global(.cm-deletedChunk button.hunk) {
    margin-left: 4px;
  }
</style>
