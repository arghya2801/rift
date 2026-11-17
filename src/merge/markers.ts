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
