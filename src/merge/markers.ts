import type { Text, ChangeSet } from '@codemirror/state';

export type Segment = { type: 'text'; text: string } | { type: 'conflict'; ours: string; base: string; theirs: string };

export function parse(src: string): Segment[] {
  const out: Segment[] = [];
  const lines = src.split(/(?<=\n)/);
  let text = '';
  let cur: { ours: string; base: string; theirs: string } | null = null;
  let part: 'ours' | 'base' | 'theirs' = 'ours';
  for (const line of lines) {
    const bare = line.replace(/\r?\n$/, '');
    if (!cur && bare.startsWith('<<<<<<<')) {
      if (text) out.push({ type: 'text', text });
      text = '';
      cur = { ours: '', base: '', theirs: '' };
      part = 'ours';
    } else if (cur && part === 'ours' && bare.startsWith('|||||||')) {
      part = 'base';
    } else if (cur && part !== 'theirs' && bare === '=======') {
      part = 'theirs';
    } else if (cur && part === 'theirs' && bare.startsWith('>>>>>>>')) {
      out.push({ type: 'conflict', ...cur });
      cur = null;
    } else if (cur) {
      cur[part] += line;
    } else {
      text += line;
    }
  }
  if (cur) text += render([{ type: 'conflict', ...cur }]);
  if (text) out.push({ type: 'text', text });
  return out;
}

export function render(segments: Segment[]): string {
  return segments
    .map((s) => (s.type === 'text' ? s.text : `<<<<<<< ours\n${s.ours}||||||| base\n${s.base}=======\n${s.theirs}>>>>>>> theirs\n`))
    .join('');
}

export const hasMarkers = (text: string) => /^(<<<<<<<|>>>>>>>)/m.test(text);

function countLines(doc: Text, first: number, last: number) {
  let count = 0;
  for (const line of doc.iterLines(first, last + 1)) if (hasMarkers(line)) count++;
  return count;
}

export const markerCount = (doc: Text) => countLines(doc, 1, doc.lines);

// Only inspect the old and new lines that intersect an edit, once per side.
export function markerDelta(before: Text, after: Text, changes: ChangeSet) {
  let delta = 0, lastA = 0, lastB = 0;
  changes.iterChangedRanges((fromA, toA, fromB, toB) => {
    const endA = before.lineAt(toA).number, endB = after.lineAt(toB).number;
    const startA = Math.max(lastA + 1, before.lineAt(fromA).number);
    const startB = Math.max(lastB + 1, after.lineAt(fromB).number);
    if (startA <= endA) delta -= countLines(before, startA, endA);
    if (startB <= endB) delta += countLines(after, startB, endB);
    lastA = endA;
    lastB = endB;
  });
  return delta;
}
