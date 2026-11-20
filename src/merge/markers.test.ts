import { ChangeSet, Text } from '@codemirror/state';
import { markerCount, markerDelta } from './markers';
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

test('incremental marker detection matches rescanning split and joined lines', () => {
  const doc = Text.of(['<<<<<<< ours','a','b','>>>>>>> theirs','end']);
  for (const spec of [[{from: 1, to: 4}, {from: 5, to: 6}], [{from: 0, to: 9, insert: 'new\n<<<<<<<'}], [{from: 8, to: 18, insert: '\n>>>>>>>'}]]) {
    const change = ChangeSet.of(spec, doc.length);
    const after = change.apply(doc);
    expect(markerCount(doc) + markerDelta(doc, after, change)).toBe(markerCount(after));
  }
});
