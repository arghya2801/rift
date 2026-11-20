import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';
import FileTree from './files/FileTree.svelte';
import RebaseView from './rebase/RebaseView.svelte';
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

it('keeps a 10,000-file folder scrollable with a bounded number of mounted rows', async () => {
  const files = Array.from({ length: 10000 }, (_, i) => ({ ...fx.files[1], path: 'src/file-' + i }));
  Go.Snapshot.mockResolvedValueOnce(fx.snapshot({ files }));
  render(CommitPanel, { dir: '/large-folder' });
  await row('file-0');
  await fireEvent.click(screen.getByTitle('Group by directory'));
  const tree = screen.getByRole('tree');
  tree.scrollTop = 5000 * 24;
  await fireEvent.scroll(tree);
  await row('file-4998');
  expect(screen.getAllByRole('treeitem').length).toBeLessThan(40);
});

it('keeps later project files reachable by keyboard without mounting the whole directory', async () => {
  Go.ListDir.mockResolvedValueOnce(Array.from({ length: 401 }, (_, i) => ({ name: 'item-' + i, isDir: false })));
  render(FileTree, { dir: '/paged-files' });
  await screen.findByTitle('item-0');
  expect(screen.getAllByRole('treeitem')).toHaveLength(200);
  const next = screen.getByRole('button', { name: 'Next page' });
  await fireEvent.click(next);
  await fireEvent.click(next);
  expect(screen.getAllByRole('treeitem')).toHaveLength(1);
  expect(screen.getByTitle('item-400').tabIndex).toBe(0);
});

it('rebase keyboard moves cross a page boundary and keep the row selected', async () => {
  const items = Array.from({ length: 201 }, (_, i) => ({ hash: String(i), subject: 'subject-' + i, action: 'pick', message: '' }));
  Go.RebaseTodo.mockResolvedValueOnce(items);
  render(RebaseView, { dir: '/paged-rebase', base: '', ondone: () => {} });
  await screen.findByText('subject-0');
  await fireEvent.click(screen.getByRole('button', { name: 'Next page' }));
  expect(screen.getByText('subject-200').closest('[role=row]')?.getAttribute('aria-selected')).toBe('true');
  await fireEvent.keyDown(screen.getByRole('grid'), { key: 'ArrowUp', altKey: true });
  expect(screen.getByText('subject-200').closest('[role=row]')?.getAttribute('aria-selected')).toBe('true');
  expect(screen.getAllByRole('row')).toHaveLength(200);
});
