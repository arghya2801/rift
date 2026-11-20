import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor, within, cleanup } from '@testing-library/svelte';
import App from './App.svelte';
import { Go } from './test/wails';
import { app, select } from './shell/app.svelte';
import { ask } from './lib/menu.svelte';
import * as fx from './test/fixtures';

beforeEach(() => {
  cleanup();
  select(null);
  for (const k of Object.keys(app.ui)) delete app.ui[k];
  Go.Snapshot.mockImplementation(async () => fx.snapshot());
});

async function fileRow(name: string) {
  let row: HTMLElement | undefined;
  await waitFor(() => {
    row = screen.getAllByRole('treeitem').find((r) => r.textContent?.includes(name));
    expect(row).toBeTruthy();
  });
  return row!;
}

async function openWorktree() {
  render(App);
  const wt = await screen.findByRole('button', { name: /feat/ });
  await fireEvent.click(wt);
  await screen.findByTitle('Commit (Alt+0)');
}

describe('App', () => {
  it('shows pinned repos and worktrees on the welcome screen', async () => {
    render(App);
    expect(await screen.findByText('demo')).toBeTruthy();
    expect(screen.getByRole('button', { name: /main/ })).toBeTruthy();
    expect(screen.getByRole('button', { name: /feat/ })).toBeTruthy();
  });

  it('opening a worktree shows the toolbar branch and the Commit tool window rows', async () => {
    await openWorktree();
    expect(screen.getByTitle('Branches').textContent).toContain('main');
    await waitFor(() => expect(screen.getAllByRole('treeitem').some((r) => r.textContent?.includes('changed.ts'))).toBe(true));
    expect(Go.Watch).toHaveBeenCalledWith(fx.WT);
  });

  it('stripe buttons toggle the tool windows', async () => {
    await openWorktree();
    const tree = () => screen.queryByRole('tree');
    expect(tree()).toBeTruthy();
    await fireEvent.click(screen.getByTitle('Commit (Alt+0)'));
    expect(tree()).toBe(null);
    await fireEvent.click(screen.getByTitle('Project (Alt+1)'));
    await waitFor(() => expect(screen.getByText('notes.txt')).toBeTruthy());
    await fireEvent.click(screen.getByTitle('Git (Alt+9)'));
    await waitFor(() => expect(screen.getByText('Add feature')).toBeTruthy());
    expect(Go.LogPage).toHaveBeenCalled();
  });

  it('clicking a changed file opens a diff tab', async () => {
    await openWorktree();
    await fireEvent.click(await fileRow('changed.ts'));
    const tab = await screen.findByRole('tab', { name: /changed\.ts/ });
    expect(tab.getAttribute('aria-selected')).toBe('true');
    await waitFor(() => expect(Go.FileSides).toHaveBeenCalledWith(fx.WT, 'src/changed.ts', false, ''));
  });

  it('branch popup lists local and remote branches', async () => {
    await openWorktree();
    expect(screen.queryByTitle('origin/main')).toBe(null);
    await fireEvent.click(screen.getByTitle('Branches'));
    expect(await screen.findByTitle('origin/main')).toBeTruthy();
    expect(screen.getAllByTitle('feat').length).toBeGreaterThan(0);
  });

  it('merge banner Resolve opens a merge tab', async () => {
    Go.Snapshot.mockImplementation(async () => fx.snapshot({ op: 'merge' }));
    await openWorktree();
    await fireEvent.click(await screen.findByRole('button', { name: 'Resolve' }));
    expect(await screen.findByRole('tab', { name: /shared\.txt/ })).toBeTruthy();
    await waitFor(() => expect(Go.ConflictSides).toHaveBeenCalledWith(fx.WT, 'shared.txt'));
  });

  it('dialog: ask shows an input and Enter resolves it', async () => {
    render(App);
    const p = ask('Branch name', 'x');
    const input = (await screen.findByRole('textbox', { name: 'Branch name' })) as HTMLInputElement;
    expect(input.value).toBe('x');
    await fireEvent.input(input, { target: { value: 'topic' } });
    await fireEvent.submit(input.form!);
    expect(await p).toBe('topic');
    await waitFor(() => expect(screen.queryByDisplayValue('topic')).toBe(null));
  });

  it('switching worktrees keeps each one usable', async () => {
    await openWorktree();
    await fireEvent.click(screen.getByTitle('Switch repository or worktree'));
    await waitFor(() => expect(document.querySelector('button.wt')).toBeTruthy());
    const main = [...document.querySelectorAll<HTMLButtonElement>('button.wt')].find((b) => b.title === fx.MAIN)!;
    await fireEvent.click(main);
    await waitFor(() => expect(Go.Watch).toHaveBeenLastCalledWith(fx.MAIN));
    expect(within(document.body).getByTitle('Commit (Alt+0)')).toBeTruthy();
  });
});
