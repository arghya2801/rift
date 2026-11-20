export type ChunkLike = { fromA: number; toA: number; fromB: number; toB: number };

const split = (t: string) => t.match(/[^\n]*\n|[^\n]+$/g) ?? [];
// CodeMirror chunks are ordered and non-overlapping on each side.
function lineCursor(text: string, count: number) {
  let pos = 0, line = 0;
  return (offset: number) => {
    if (offset > text.length) return count;
    while (pos < offset) if (text.charCodeAt(pos++) === 10) line++;
    return line;
  };
}

function join(lines: string[], eol: string) {
  return lines.map((l, i) => (i < lines.length - 1 && !l.endsWith('\n') ? l + eol : l)).join('');
}

type Part = { a0: number; a1: number; b0: number; b1: number; drop: boolean; picked: (i: number) => boolean };

// selFrom/selTo are 0-based inclusive line numbers in B.
function parts(a: string, b: string, chunks: readonly ChunkLike[], selFrom: number, selTo: number) {
  const la = split(a), lb = split(b);
  const out: Part[] = [];
  const lineA = lineCursor(a, la.length), lineB = lineCursor(b, lb.length);
  for (const c of chunks) {
    const [a0, a1] = [lineA(c.fromA), lineA(c.toA)];
    const [b0, b1] = [lineB(c.fromB), lineB(c.toB)];
    const touches = b1 > b0 ? selFrom < b1 && selTo >= b0 : selFrom <= b0 && selTo >= b0 - 1;
    if (!touches) continue;
    const drop = b1 > b0 ? selFrom <= b0 && selTo >= b0 : true;
    out.push({ a0, a1, b0, b1, drop, picked: (i) => i >= selFrom && i <= selTo });
  }
  return { la, lb, out };
}

const eolOf = (t: string) => (t.includes('\r\n') ? '\r\n' : '\n');

// New A text with the selected B lines of each touched chunk applied (stage selected lines).
export function applyLines(a: string, b: string, chunks: readonly ChunkLike[], selFrom: number, selTo: number): string {
  const { la, lb, out } = parts(a, b, chunks, selFrom, selTo);
  const res: string[] = [];
  let pos = 0;
  for (const p of out) {
    for (let i = pos; i < p.a0; i++) res.push(la[i]);
    if (!p.drop) for (let i = p.a0; i < p.a1; i++) res.push(la[i]);
    for (let i = p.b0; i < p.b1; i++) if (p.picked(i)) res.push(lb[i]);
    pos = p.a1;
  }
  for (let i = pos; i < la.length; i++) res.push(la[i]);
  return join(res, eolOf(a + b));
}

// New B text with the selected B lines of each touched chunk reverted to A (unstage selected lines).
export function revertLines(a: string, b: string, chunks: readonly ChunkLike[], selFrom: number, selTo: number): string {
  const { la, lb, out } = parts(a, b, chunks, selFrom, selTo);
  const res: string[] = [];
  let pos = 0;
  for (const p of out) {
    for (let i = pos; i < p.b0; i++) res.push(lb[i]);
    if (p.drop) for (let i = p.a0; i < p.a1; i++) res.push(la[i]);
    for (let i = p.b0; i < p.b1; i++) if (!p.picked(i)) res.push(lb[i]);
    pos = p.b1;
  }
  for (let i = pos; i < lb.length; i++) res.push(lb[i]);
  return join(res, eolOf(a + b));
}
