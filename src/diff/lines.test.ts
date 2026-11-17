import { describe, expect, it } from 'vitest';
import { applyLines, revertLines } from './lines';

// a: x/old/y, b: x/new1/new2/y; one chunk replacing line 1 of A with lines 1-2 of B.
const a = 'x\nold\ny\n';
const b = 'x\nnew1\nnew2\ny\n';
const chunks = [{ fromA: 2, toA: 6, fromB: 2, toB: 12 }];

describe('lines', () => {
  it('stages only the selected added line and drops the old one', () => {
    expect(applyLines(a, b, chunks, 1, 1)).toBe('x\nnew1\ny\n');
  });
  it('keeps the old line when the selection starts below the chunk start', () => {
    expect(applyLines(a, b, chunks, 2, 2)).toBe('x\nold\nnew2\ny\n');
  });
  it('ignores chunks outside the selection', () => {
    expect(applyLines(a, b, chunks, 3, 3)).toBe(a);
  });
  it('reverts the selected line in B', () => {
    expect(revertLines(a, b, chunks, 1, 2)).toBe(a);
    expect(revertLines(a, b, chunks, 2, 2)).toBe('x\nnew1\ny\n');
  });
  it('handles a missing final newline', () => {
    expect(applyLines('x\n', 'x\ny', [{ fromA: 2, toA: 2, fromB: 2, toB: 4 }], 1, 1)).toBe('x\ny');
  });
});
