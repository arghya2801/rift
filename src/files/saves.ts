import { Text } from '@codemirror/state';
import * as Go from '../../wailsjs/go/main/App';

export class FileDraft {
  pending?: Promise<void>;
  users = 0;
  private doc?: Text;
  private baseline?: Text;
  private baselineText?: string;
  private eol = '\n';
  constructor(readonly dir: string, readonly path: string, public saved: string, private value = saved) {}

  get text() { return this.doc ? this.doc.sliceString(0, this.doc.length, this.eol) : this.value; }
  set text(value: string) { this.doc = undefined; this.value = value; }

  rememberSavedDoc(doc: Text) {
    this.baseline = doc;
    this.baselineText = this.saved;
  }

  edit(doc: Text, eol: string) { this.doc = doc; this.eol = eol; }
  get dirty() {
    if (!this.doc) return this.value !== this.saved;
    if (!this.baseline || this.baselineText !== this.saved) {
      this.baseline = Text.of(this.saved.split(/\r?\n/));
      this.baselineText = this.saved;
    }
    return !this.doc.eq(this.baseline);
  }

  save(): Promise<void> {
    if (!this.pending) this.pending = this.flush().finally(() => { this.pending = undefined; });
    return this.pending;
  }

  private async flush() {
    while (this.dirty) {
      const written = this.doc;
      const text = this.text;
      await Go.SaveFile(this.dir, this.path, text, this.saved);
      this.saved = text;
      if (written) this.rememberSavedDoc(written);
    }
  }
}

const drafts = new Map<string, FileDraft>();
const key = (dir: string, path: string) => JSON.stringify([dir, path]);

export async function openDraft(dir: string, path: string) {
  const id = key(dir, path);
  const existing = drafts.get(id);
  if (existing) { existing.users++; return existing; }
  const disk = await Go.ReadFile(dir, path);
  const draft = drafts.get(id) ?? new FileDraft(dir, path, disk);
  draft.users++;
  drafts.set(id, draft);
  return draft;
}

export function releaseDraft(draft: FileDraft) {
  const id = key(draft.dir, draft.path);
  draft.users--;
  if (draft.users === 0 && drafts.get(id) === draft && !draft.dirty) drafts.delete(id);
}
