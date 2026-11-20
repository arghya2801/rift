import type { git } from '../../wailsjs/go/models';

export const partials = $state<Record<string, Record<string, { text: string; base: string }>>>({});

type CommitDraft = { repair?: git.IndexRepair; message: string; amend: boolean; pick: Record<string, boolean> };
const drafts = $state<Record<string, CommitDraft>>({});
export function commitDraft(dir: string) {
  if (!drafts[dir]) drafts[dir] = { message: '', amend: false, pick: {} };
  return drafts[dir];
}
