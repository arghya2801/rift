<script lang="ts">
  import { EditorView, keymap } from '@codemirror/view';
  import { EditorState } from '@codemirror/state';
  import { history, historyKeymap } from '@codemirror/commands';
  import * as Go from '../../wailsjs/go/main/App';
  import { repo } from '../lib/repo.svelte';
  import { cmBase } from '../lib/cm';
  import { sure, toast } from '../lib/menu.svelte';
  import Icon from '../lib/Icon.svelte';
  import { parse, render, hasMarkers, type Segment } from './markers';

  let { dir, path, ondone }: { dir: string; path: string; ondone: () => void } = $props();

  type Choice = 'ours' | 'theirs' | 'both' | null;

  let oursEl: HTMLDivElement;
  let resultEl: HTMLDivElement;
  let theirsEl: HTMLDivElement;
  let segments = $state<Segment[]>([]);
  let choices = $state<Choice[]>([]);
  let edited = $state(false);
  let unresolved = $state(true);
  let error = $state('');
  let result: EditorView | undefined;
  let syncing = false;

  const r = $derived(repo(dir));
  const conflicts = $derived(segments.filter((s) => s.type === 'conflict'));
  const left = $derived(`Your version${r.branch ? ` (${r.branch})` : ''}`);
  const right = $derived(r.op?.kind === 'rebase' ? 'Commit being rebased' : 'Incoming changes');

  function build(): string {
    let i = 0;
    return segments
      .map((s) => {
        if (s.type === 'text') return s.text;
        const c = choices[i++];
        if (c === 'ours') return s.ours;
        if (c === 'theirs') return s.theirs;
        if (c === 'both') return s.ours + s.theirs;
        return render([s]);
      })
      .join('');
  }

  function setResult(text: string) {
    if (!result) return;
    syncing = true;
    result.dispatch({ changes: { from: 0, to: result.state.doc.length, insert: text } });
    syncing = false;
    unresolved = hasMarkers(text);
  }

  function choose(i: number, c: Choice) {
    choices[i] = c;
    if (!edited) setResult(build());
  }

  function jump(dirn: 1 | -1) {
    if (!result) return;
    const doc = result.state.doc.toString();
    const head = result.state.selection.main.head;
    const starts = [...doc.matchAll(/^<<<<<<<.*$/gm)].map((m) => m.index!);
    const target = dirn > 0 ? (starts.find((p) => p > head) ?? starts[0]) : ([...starts].reverse().find((p) => p < head) ?? starts.at(-1));
    if (target === undefined) return;
    result.dispatch({ selection: { anchor: target }, effects: EditorView.scrollIntoView(target, { y: 'center' }) });
    result.focus();
  }

  async function save() {
    try {
      await r.run(Go.ResolveFile(dir, path, result!.state.doc.toString()));
      ondone();
    } catch (e) {
      toast(String(e));
    }
  }

  async function acceptAll(side: 'ours' | 'theirs') {
    if (!(await sure(`Accept ${side === 'ours' ? 'your version' : 'incoming changes'} for the whole file?`, path, 'Accept', false))) return;
    try {
      await r.run(Go.AcceptSide(dir, path, side));
      ondone();
    } catch (e) {
      toast(String(e));
    }
  }

  $effect(() => {
    const views: EditorView[] = [];
    let dead = false;
    (async () => {
      try {
        const [c, base] = await Promise.all([Go.ConflictSides(dir, path), cmBase(path)]);
        if (dead) return;
        segments = parse(c.merged);
        choices = segments.filter((s) => s.type === 'conflict').map(() => null);
        const ro = [...base, EditorState.readOnly.of(true)];
        views.push(new EditorView({ doc: c.ours, extensions: ro, parent: oursEl }));
        views.push(new EditorView({ doc: c.theirs, extensions: ro, parent: theirsEl }));
        result = new EditorView({
          doc: build(),
          parent: resultEl,
          extensions: [
            history(),
            keymap.of(historyKeymap),
            ...base,
            EditorView.updateListener.of((u) => {
              if (!u.docChanged) return;
              unresolved = hasMarkers(u.state.doc.toString());
              if (!syncing) edited = true;
            }),
          ],
        });
        views.push(result);
        unresolved = hasMarkers(result.state.doc.toString());
      } catch (e) {
        error = String(e);
      }
    })();
    return () => {
      dead = true;
      views.forEach((v) => v.destroy());
      result = undefined;
    };
  });
</script>

<div class="toolbar">
  <button class="icon" onclick={() => jump(-1)} title="Previous conflict"><Icon name="push" size={14} /></button>
  <button class="icon" onclick={() => jump(1)} title="Next conflict"><Icon name="pull" size={14} /></button>
  <span class="sep"></span>
  <button onclick={() => acceptAll('ours')}>Accept Left</button>
  <button onclick={() => acceptAll('theirs')}>Accept Right</button>
  <span class="dim small">{unresolved ? `${conflicts.length} conflict${conflicts.length === 1 ? '' : 's'}` : 'All conflicts resolved'}{edited ? ' · edited' : ''}</span>
  <span class="grow"></span>
  <button class="primary" disabled={unresolved} title={unresolved ? 'Resolve every conflict first' : ''} onclick={save}>Apply</button>
</div>
{#if error}<div class="error">{error}</div>{/if}
{#if conflicts.length && !edited}
  <div class="blocks">
    {#each conflicts as _, i}
      <span class="block">
        <span class="dim">#{i + 1}</span>
        {#each [['ours', 'Left'], ['theirs', 'Right'], ['both', 'Both']] as const as [c, label]}
          <button class:on={choices[i] === c} onclick={() => choose(i, choices[i] === c ? null : c)}>{label}</button>
        {/each}
      </span>
    {/each}
  </div>
{/if}
<div class="panes">
  <div class="pane"><div class="label">{left}</div><div class="cm" bind:this={oursEl}></div></div>
  <div class="pane mid"><div class="label">Result</div><div class="cm" bind:this={resultEl}></div></div>
  <div class="pane"><div class="label">{right}</div><div class="cm" bind:this={theirsEl}></div></div>
</div>

<style>
  .toolbar,
  .blocks {
    display: flex;
    gap: 4px;
    align-items: center;
    padding: 4px 8px;
    border-bottom: 1px solid var(--line);
    background: var(--bg-panel);
    flex-wrap: wrap;
  }
  .icon {
    width: 26px;
    padding: 0;
    display: grid;
    place-items: center;
    color: var(--fg-dim);
  }
  .sep {
    width: 1px;
    height: 16px;
    background: var(--line);
    margin: 0 4px;
  }
  .grow {
    flex: 1;
  }
  .small {
    font-size: var(--fs-small);
    margin-left: 6px;
  }
  .block {
    display: inline-flex;
    gap: 2px;
    align-items: center;
    margin-right: 10px;
    font-size: var(--fs-small);
  }
  .block button {
    height: 22px;
    padding: 0 6px;
    font-size: var(--fs-small);
  }
  button.on {
    background: var(--selection);
  }
  .panes {
    flex: 1;
    display: flex;
    min-height: 0;
    background: var(--bg-editor);
  }
  .pane {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
  }
  .pane.mid {
    border-left: 1px solid var(--line);
    border-right: 1px solid var(--line);
  }
  .label {
    height: 26px;
    display: flex;
    align-items: center;
    padding: 0 10px;
    color: var(--fg-dim);
    font-size: var(--fs-small);
    border-bottom: 1px solid var(--line);
  }
  .cm {
    flex: 1;
    overflow: hidden;
    min-height: 0;
  }
  .cm :global(.cm-editor) {
    height: 100%;
  }
  .error {
    padding: 6px 12px;
    color: var(--danger);
  }
</style>
