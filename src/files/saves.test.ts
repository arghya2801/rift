import { Text } from '@codemirror/state';
import { expect, it } from 'vitest';
import { FileDraft, openDraft, releaseDraft } from './saves';
import { Go } from '../test/wails';

it('serializes a newer edit behind an outstanding save', async () => {
  let finish!: () => void;
  Go.SaveFile.mockImplementationOnce(() => new Promise<void>((resolve) => { finish = resolve; }));
  const d = new FileDraft('/save', 'a', 'old', 'first');
  const pending = d.save();
  d.text = 'second';
  expect(d.save()).toBe(pending);
  finish();
  await pending;
  expect(Go.SaveFile).toHaveBeenCalledWith('/save', 'a', 'first', 'old');
  expect(Go.SaveFile).toHaveBeenCalledWith('/save', 'a', 'second', 'first');
  expect(d.saved).toBe('second');
});

it('retains the failed draft after the tab closes', async () => {
  const d = await openDraft('/recover', 'a');
  const disk = d.saved;
  d.text = 'unsigned draft';
  Go.SaveFile.mockRejectedValueOnce(new Error('disk changed'));
  await expect(d.save()).rejects.toThrow('disk changed');
  releaseDraft(d);
  const reopened = await openDraft('/recover', 'a');
  expect(reopened.text).toBe('unsigned draft');
  expect(reopened.saved).toBe(disk);
  await reopened.save();
  releaseDraft(reopened);
});

it('keeps edits made during an in-flight CRLF save with immutable editor documents', async () => {
  const d = new FileDraft('/repo', 'lines', 'a\r\nb');
  const original = Text.of(['a', 'b']);
  d.rememberSavedDoc(original);
  const first = original.replace(0, 1, Text.of(['A']));
  d.edit(first, '\r\n');
  let finish!: () => void;
  Go.SaveFile.mockImplementationOnce(() => new Promise<void>((resolve) => { finish = resolve; }));
  const pending = d.save();
  d.edit(first.replace(2, 3, Text.of(['B'])), '\r\n');
  finish();
  await pending;
  expect(d.saved).toBe('A\r\nB');
  expect(d.dirty).toBe(false);
});
