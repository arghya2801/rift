export type ChunkLike = { fromA: number; toA: number; fromB: number; toB: number };

const split = (t: string) => t.match(/[^\n]*\n|[^\n]+$/g) ?? [];
const nl = (t: string, off: number) => {
  let n = 0;
  for (let i = 0; i < off && i < t.length; i++) if (t[i] === '\n') n++;
  return n;
};
const range = (t: string, n: number, from: number, to: number) => [nl(t, from), from === to ? nl(t, from) : to > t.length ? n : nl(t, to)];

function join(lines: string[], eol: string) {
  return lines.map((l, i) => (i < lines.length - 1 && !l.endsWith('\n') ? l + eol : l)).join('');
}

type Part = { a0: number; a1: number; b0: number; b1: number; drop: boolean; picked: (i: number) => boolean };

// selFrom/selTo are 0-based inclusive line numbers in B.
function parts(a: string, b: string, chunks: readonly ChunkLike[], selFrom: number, selTo: number) {
  const la = split(a), lb = split(b);
  const out: Part[] = [];
  for (const c of chunks) {
    const [a0, a1] = range(a, la.length, c.fromA, c.toA);
    const [b0, b1] = range(b, lb.length, c.fromB, c.toB);
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
    res.push(...la.slice(pos, p.a0));
    if (!p.drop) res.push(...la.slice(p.a0, p.a1));
    for (let i = p.b0; i < p.b1; i++) if (p.picked(i)) res.push(lb[i]);
    pos = p.a1;
  }
  res.push(...la.slice(pos));
  return join(res, eolOf(a + b));
}

// New B text with the selected B lines of each touched chunk reverted to A (unstage selected lines).
export function revertLines(a: string, b: string, chunks: readonly ChunkLike[], selFrom: number, selTo: number): string {
  const { la, lb, out } = parts(a, b, chunks, selFrom, selTo);
  const res: string[] = [];
  let pos = 0;
  for (const p of out) {
    res.push(...lb.slice(pos, p.b0));
    if (p.drop) res.push(...la.slice(p.a0, p.a1));
    for (let i = p.b0; i < p.b1; i++) if (!p.picked(i)) res.push(lb[i]);
    pos = p.b1;
  }
  res.push(...lb.slice(pos));
  return join(res, eolOf(a + b));
}
