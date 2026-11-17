import { expect, test } from 'vitest';
import { parse, render, hasMarkers } from './markers';

test('parses diff3 conflicts and round-trips', () => {
  const src = 'a\n<<<<<<< ours\nO\n||||||| base\nB\n=======\nT\n>>>>>>> theirs\nz\n';
  const s = parse(src);
  expect(s).toEqual([
    { type: 'text', text: 'a\n' },
    { type: 'conflict', ours: 'O\n', base: 'B\n', theirs: 'T\n' },
    { type: 'text', text: 'z\n' },
  ]);
  expect(render(s)).toBe(src);
  expect(hasMarkers(src)).toBe(true);
  expect(hasMarkers('a\nb\n')).toBe(false);
});
