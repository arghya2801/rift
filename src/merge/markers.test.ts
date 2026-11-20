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

test('preserves incomplete and illustrative marker lines', () => {
  for (const text of ['<<<<<<< example\n', '<<<<<<< ours\npartial\n']) {
    expect(render(parse(text))).toBe(text);
  }
  expect(hasMarkers('<<<<<<< example\n')).toBe(false);
});

test('distinguishes generated blocks from literal marker examples', () => {
  const literal = '<<<<<<< ours\nexample\n=======\n>>>>>>> theirs\n';
  const block = { type: 'conflict', ours: 'ours\n', base: 'base\n', theirs: 'theirs\n' } as const;
  expect(hasMarkers(literal, 8)).toBe(false);
  const source = literal + render([block], 8);
  expect(parse(source, 8)).toEqual([{ type: 'text', text: literal }, block]);
  expect(hasMarkers(source, 8)).toBe(true);
});
