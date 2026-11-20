import type { Text, ChangeSet } from '@codemirror/state';

export type Segment = { type: 'text'; text: string } | { type: 'conflict'; ours: string; base: string; theirs: string };

export function parse(src: string, size = 7): Segment[] {
  const out: Segment[] = [];
  const lines = src.split(/(?<=\n)/);
  let text = '';
  let raw = '';
  let cur: { ours: string; base: string; theirs: string } | null = null;
  let part: 'ours' | 'base' | 'theirs' = 'ours';
  for (const line of lines) {
    const bare = line.replace(/\r?\n$/, '');
    if (cur) raw += line;
    if (!cur && bare === '<'.repeat(size) + ' ours') {
      raw = line;
      if (text) out.push({ type: 'text', text });
      text = '';
      cur = { ours: '', base: '', theirs: '' };
      part = 'ours';
    } else if (cur && part === 'ours' && bare === '|'.repeat(size) + ' base') {
      part = 'base';
    } else if (cur && part !== 'theirs' && bare === '='.repeat(size)) {
      part = 'theirs';
    } else if (cur && part === 'theirs' && bare === '>'.repeat(size) + ' theirs') {
      out.push({ type: 'conflict', ...cur });
      cur = null;
    } else if (cur) {
      cur[part] += line;
    } else {
      text += line;
    }
  }
  if (cur) text += raw;
  if (text) out.push({ type: 'text', text });
  return out;
}

export function render(segments: Segment[], size = 7): string {
  return segments
    .map((s) => (s.type === 'text' ? s.text : `${'<'.repeat(size)} ours\n${s.ours}${'|'.repeat(size)} base\n${s.base}${'='.repeat(size)}\n${s.theirs}${'>'.repeat(size)} theirs\n`))
    .join('');
}

export function markerStarts(text: string, size = 7): number[] {
  return Array.from(text.matchAll(new RegExp('^<{' + size + '} ours\\r?$', 'gm')), (match) => match.index!);
}

export function hasMarkers(text: string, size = 7): boolean {
  return new RegExp('^(?:<{' + size + '} ours|[|]{' + size + '} base|={' + size + '}|>{' + size + '} theirs)\\r?$', 'm').test(text);
}

function countLines(doc: Text, first: number, last: number, match: (line: string) => boolean) {
  let count = 0;
  for (const line of doc.iterLines(first, last + 1)) if (match(line)) count++;
  return count;
}

export const markerCount = (doc: Text, match: (line: string) => boolean = hasMarkers) => countLines(doc, 1, doc.lines, match);

// Only inspect the old and new lines that intersect an edit, once per side.
export function markerDelta(before: Text, after: Text, changes: ChangeSet, match: (line: string) => boolean = hasMarkers) {
  let delta = 0, lastA = 0, lastB = 0;
  changes.iterChangedRanges((fromA, toA, fromB, toB) => {
    const endA = before.lineAt(toA).number, endB = after.lineAt(toB).number;
    const startA = Math.max(lastA + 1, before.lineAt(fromA).number);
    const startB = Math.max(lastB + 1, after.lineAt(fromB).number);
    if (startA <= endA) delta -= countLines(before, startA, endA, match);
    if (startB <= endB) delta += countLines(after, startB, endB, match);
    lastA = endA;
    lastB = endB;
  });
  return delta;
}
