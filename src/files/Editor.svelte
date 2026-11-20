<script lang="ts">
  import { FileDraft, openDraft, releaseDraft } from './saves';
  import { untrack } from 'svelte';
  import { EditorView, keymap, highlightActiveLine, highlightActiveLineGutter } from '@codemirror/view';
  import { history, historyKeymap, indentWithTab } from '@codemirror/commands';
  import * as Go from '../../wailsjs/go/main/App';
  import { repo, touches } from '../lib/repo.svelte';
  import { cmBase } from '../lib/cm';
  import { sure, toast } from '../lib/menu.svelte';
  import Icon from '../lib/Icon.svelte';

  let { dir, rel, line = 1 }: { dir: string; rel: string; line?: number } = $props();

  let host: HTMLDivElement;
  let view: EditorView | undefined;
  let draft: FileDraft | undefined;
  let eol = '\n';
  let dirty = $state(false);
  let error = $state('');

  async function save() {
    const current = draft;
    if (!current || (!current.pending && !current.dirty)) return true;
    try {
      await current.save();
      if (draft === current) { dirty = current.dirty; error = ''; }
      repo(current.dir).schedule(false, false, [current.path]);
    } catch (e) {
      if (draft === current) error = String(e);
      toast(String(e));
    }
    return true;
  }

  async function reconcile(keepDraft: boolean) {
    const current = draft;
    if (!current || !view || current.pending) return;
    try {
      const disk = await Go.ReadFile(dir, rel);
      const ok = await sure(keepDraft ? 'Replace the disk version?' : 'Discard the editor draft?', rel, keepDraft ? 'Replace' : 'Reload');
      if (!ok || draft !== current || !view) return;
      current.saved = disk;
      if (keepDraft) { await save(); return; }
      current.text = disk;
      eol = disk.includes('\r\n') ? '\r\n' : '\n';
      view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: disk } });
      current.rememberSavedDoc(view.state.doc);
      dirty = false;
      error = '';
    } catch (e) {
      error = String(e);
    }
  }

  let reloadId = 0;
  async function reload() {
    const current = draft;
    if (!view || !current || current.dirty || current.pending) return;
    const previous = current.saved;
    const id = ++reloadId;
    try {
      const text = await Go.ReadFile(dir, rel);
      if (!view || current !== draft || id !== reloadId || current.dirty || current.saved !== previous) return;
      error = '';
      if (text === previous) return;
      eol = text.includes('\r\n') ? '\r\n' : '\n';
      current.saved = current.text = text;
      view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: text } });
      current.rememberSavedDoc(view.state.doc);
      dirty = false;
    } catch (e) {
      if (id === reloadId && current === draft) error = String(e);
    }
  }

  $effect(() => {
    const [d, r] = [dir, rel];
    let dead = false;
    (async () => {
      try {
        const base = await cmBase(r);
        if (dead) return;
        const current = await openDraft(d, r);
        if (dead) { releaseDraft(current); return; }
        draft = current;
        const text = current.text;
        eol = text.includes('\r\n') ? '\r\n' : '\n';
        dirty = text !== current.saved;
        view = new EditorView({
          doc: text,
          parent: host,
          extensions: [
            keymap.of([{ key: 'Mod-s', run: () => (save(), true) }, ...historyKeymap, indentWithTab]),
            history(),
            highlightActiveLine(),
            highlightActiveLineGutter(),
            ...base,
            EditorView.updateListener.of((u) => {
              if (!u.docChanged) return;
              current.edit(u.state.doc, eol);
              dirty = current.dirty;
            }),
          ],
        });
        if (!dirty) current.rememberSavedDoc(view.state.doc);
        const ln = view.state.doc.line(Math.min(Math.max(1, line), view.state.doc.lines));
        view.dispatch({ selection: { anchor: ln.from }, effects: EditorView.scrollIntoView(ln.from, { y: 'center' }) });
        view.focus();
      } catch (e) {
        error = String(e);
      }
    })();
    return () => {
      dead = true;
      const current = draft;
      if (current) save().finally(() => releaseDraft(current));
      draft = undefined;
      view?.destroy();
      view = undefined;
    };
  });

  $effect(() => {
    const path = rel;
    return repo(dir).onFiles((_refs, _index, paths) => { if (touches(path, paths)) untrack(reload); });
  });

  function external() {
    const ln = view ? view.state.doc.lineAt(view.state.selection.main.head).number : line;
    Go.OpenExternal(dir, rel, ln).catch((e) => toast(String(e)));
  }

  const crumbs = $derived(rel.split('/'));
</script>

<svelte:window onblur={save} />

<div class="crumbs">
  {#each crumbs as c, i}
    {#if i}<Icon name="chevron-right" size={12} />{/if}
    <span class:leaf={i === crumbs.length - 1}>{c}</span>
  {/each}
  {#if dirty}<span class="dot" title="Unsaved changes (Ctrl+S)"></span>{/if}
  <span class="grow"></span>
  <button class="icon" title="Save (Ctrl+S)" disabled={!dirty} onclick={save}><Icon name="check" size={14} /></button>
  <button class="icon" title="Open in external editor" onclick={external}><Icon name="external" size={14} /></button>
</div>
{#if error}<div class="error" role="alert">
  {error}
  {#if dirty}
    <button onclick={() => reconcile(false)}>Reload disk version</button>
    <button onclick={() => reconcile(true)}>Keep editor version</button>
  {/if}
</div>{/if}
<div class="host" bind:this={host}></div>

<style>
  .crumbs {
    display: flex;
    align-items: center;
    gap: 3px;
    height: 28px;
    padding: 0 6px 0 12px;
    font-size: var(--fs-small);
    color: var(--fg-dim);
    background: var(--bg-editor);
    border-bottom: 1px solid var(--line);
    white-space: nowrap;
  }
  .leaf {
    color: var(--fg);
  }
  .dot {
    width: 7px;
    height: 7px;
    border-radius: 50%;
    background: var(--accent);
    margin-left: 6px;
  }
  .grow {
    flex: 1;
  }
  .icon {
    width: 24px;
    height: 22px;
    padding: 0;
    display: grid;
    place-items: center;
    color: var(--fg-dim);
  }
  .error {
    padding: 6px 12px;
    color: var(--danger);
  }
  .host {
    flex: 1;
    min-height: 0;
    overflow: hidden;
    background: var(--bg-editor);
  }
  .host :global(.cm-editor) {
    height: 100%;
  }
</style>
