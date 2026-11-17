<script lang="ts">
  import { untrack } from 'svelte';
  import { EditorView, keymap, highlightActiveLine, highlightActiveLineGutter } from '@codemirror/view';
  import { history, historyKeymap, indentWithTab } from '@codemirror/commands';
  import * as Go from '../../wailsjs/go/main/App';
  import { repo } from '../lib/repo.svelte';
  import { cmBase } from '../lib/cm';
  import { toast } from '../lib/menu.svelte';
  import Icon from '../lib/Icon.svelte';

  let { dir, rel, line = 1 }: { dir: string; rel: string; line?: number } = $props();

  let host: HTMLDivElement;
  let view: EditorView | undefined;
  let saved = '';
  let dirty = $state(false);
  let error = $state('');

  async function save() {
    if (!view || !dirty) return true;
    const text = view.state.doc.toString();
    try {
      await Go.WriteFile(dir, rel, text);
      saved = text;
      dirty = false;
      repo(dir).schedule();
    } catch (e) {
      toast(String(e));
    }
    return true;
  }

  async function reload() {
    if (!view || dirty) return;
    const text = await Go.ReadFile(dir, rel).catch(() => null);
    if (text === null || text === saved || dirty || !view) return;
    saved = text;
    view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: text } });
    dirty = false;
  }

  $effect(() => {
    const [d, r] = [dir, rel];
    let dead = false;
    (async () => {
      try {
        const [text, base] = await Promise.all([Go.ReadFile(d, r), cmBase(r)]);
        if (dead) return;
        saved = text;
        view = new EditorView({
          doc: text,
          parent: host,
          extensions: [
            keymap.of([{ key: 'Mod-s', run: () => (save(), true) }, ...historyKeymap, indentWithTab]),
            history(),
            highlightActiveLine(),
            highlightActiveLineGutter(),
            ...base,
            EditorView.updateListener.of((u) => u.docChanged && (dirty = u.state.doc.toString() !== saved)),
          ],
        });
        const ln = view.state.doc.line(Math.min(Math.max(1, line), view.state.doc.lines));
        view.dispatch({ selection: { anchor: ln.from }, effects: EditorView.scrollIntoView(ln.from, { y: 'center' }) });
        view.focus();
      } catch (e) {
        error = String(e);
      }
    })();
    return () => {
      dead = true;
      // Like IntelliJ: leaving the tab saves, so switching or closing never drops edits.
      if (view && dirty) Go.WriteFile(d, r, view.state.doc.toString()).then(() => repo(d).schedule(), (e) => toast(String(e)));
      view?.destroy();
      view = undefined;
    };
  });

  $effect(() => {
    repo(dir).snap;
    untrack(reload);
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
{#if error}<div class="error">{error}</div>{/if}
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
