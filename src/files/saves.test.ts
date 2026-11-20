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
