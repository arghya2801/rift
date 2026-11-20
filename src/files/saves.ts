import * as Go from '../../wailsjs/go/main/App';

export class FileDraft {
  pending?: Promise<void>;
  users = 0;
  constructor(readonly dir: string, readonly path: string, public saved: string, public text = saved) {}

  save(): Promise<void> {
    if (!this.pending) this.pending = this.flush().finally(() => { this.pending = undefined; });
    return this.pending;
  }

  private async flush() {
    while (this.text !== this.saved) {
      const text = this.text;
      await Go.SaveFile(this.dir, this.path, text, this.saved);
      this.saved = text;
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
  if (draft.users === 0 && drafts.get(id) === draft && draft.text === draft.saved) drafts.delete(id);
}
