import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';
import CommitPanel from './changes/CommitPanel.svelte';
import LogPanel from './log/LogPanel.svelte';
import { menu } from './lib/menu.svelte';
import { Go } from './test/wails';
import * as fx from './test/fixtures';

beforeEach(() => {
  cleanup();
  Go.Snapshot.mockImplementation(async () => fx.snapshot());
});

async function row(name: string) {
  let el: HTMLElement | undefined;
  await waitFor(() => {
    el = screen.getAllByRole('treeitem').find((r) => r.textContent?.includes(name));
    expect(el).toBeTruthy();
  });
  return el!;
}

describe('CommitPanel', () => {
  it('checking an unstaged file stages exactly that path', async () => {
    render(CommitPanel, { dir: '/cp1' });
    const box = (await row('changed.ts')).querySelector('input[type=checkbox]') as HTMLInputElement;
    expect(box.checked).toBe(false);
    await fireEvent.click(box);
    await waitFor(() => expect(Go.Stage).toHaveBeenCalledWith('/cp1', ['src/changed.ts']));
  });

  it('unchecking a staged file unstages it', async () => {
    render(CommitPanel, { dir: '/cp2' });
    const box = (await row('staged.ts')).querySelector('input[type=checkbox]') as HTMLInputElement;
    expect(box.checked).toBe(true);
    await fireEvent.click(box);
    await waitFor(() => expect(Go.Unstage).toHaveBeenCalledWith('/cp2', ['src/staged.ts']));
  });

  it('Commit is disabled without a message; Ctrl+Enter commits', async () => {
    render(CommitPanel, { dir: '/cp3' });
    await row('staged.ts');
    const btn = screen.getByRole('button', { name: 'Commit' }) as HTMLButtonElement;
    expect(btn.disabled).toBe(true);
    const msg = screen.getByPlaceholderText(/Commit message/);
    await fireEvent.input(msg, { target: { value: 'feat: thing' } });
    expect(btn.disabled).toBe(false);
    await fireEvent.keyDown(msg, { key: 'Enter', ctrlKey: true });
    await waitFor(() => expect(Go.Commit).toHaveBeenCalledWith('/cp3', 'feat: thing', false));
  });
});

describe('LogPanel', () => {
  it('renders commits and opens a context menu on right-click', async () => {
    render(LogPanel, { dir: '/log1' });
    const subject = await screen.findByText('Add feature');
    expect(screen.getByText('Initial')).toBeTruthy();
    const r = subject.closest('[role=row]') as HTMLElement;
    await fireEvent.contextMenu(r);
    expect(menu.items.some((i) => i?.label === 'Cherry-Pick')).toBe(true);
  });

  it('selecting a commit loads its details', async () => {
    render(LogPanel, { dir: '/log2' });
    await fireEvent.click(await screen.findByText('Fix bug'));
    await waitFor(() => expect(Go.CommitFiles).toHaveBeenCalledWith('/log2', '2'.repeat(40)));
  });
});
