import { execFileSync } from 'node:child_process';
import { mkdirSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

const base = join(tmpdir(), 'rift-play');
const repo = join(base, 'rift-play');
const wt = join(base, 'rift-play-wt');
rmSync(base, { recursive: true, force: true });
mkdirSync(repo, { recursive: true });

const git = (...args) => execFileSync('git', args, { cwd: repo, windowsHide: true, stdio: ['ignore', 'pipe', 'pipe'] }).toString();
const gitIn = (dir, ...args) => execFileSync('git', args, { cwd: dir, windowsHide: true, stdio: ['ignore', 'pipe', 'pipe'] }).toString();
const write = (dir, name, body) => writeFileSync(join(dir, name), body);
const commit = (msg) => { git('add', '-A'); git('commit', '-qm', msg); };

git('init', '-q', '-b', 'main');
git('config', 'user.name', 'Rift Play');
git('config', 'user.email', 'play@example.com');
git('config', 'commit.gpgsign', 'false');

write(repo, 'README.md', '# Play\n');
commit('Initial commit');
write(repo, 'app.txt', 'line 1\nline 2\nline 3\nline 4\nline 5\n');
commit('Add app.txt');
write(repo, 'shared.txt', 'base\n');
commit('Add shared.txt');

git('checkout', '-qb', 'feature');
write(repo, 'feature.txt', 'feature one\n');
commit('feat: first feature commit');
write(repo, 'feature.txt', 'feature one\nfeature two\n');
commit('feat: second feature commit');

git('checkout', '-q', 'main');
write(repo, 'app.txt', 'line 1\nline 2 changed on main\nline 3\nline 4\nline 5\n');
commit('Change app.txt on main');
git('merge', '-q', '--no-ff', '-m', 'Merge branch feature', 'feature');

git('config', 'core.autocrlf', 'true');
write(repo, 'crlf.txt', 'one\r\ntwo\r\nthree\r\n');
commit('Add CRLF file');

git('checkout', '-qb', 'conflict');
write(repo, 'shared.txt', 'changed on conflict branch\n');
commit('Change shared.txt on conflict');
git('checkout', '-q', 'main');
write(repo, 'shared.txt', 'changed on main\n');
commit('Change shared.txt on main');

git('worktree', 'add', '-q', '-b', 'wip', wt);
write(wt, 'app.txt', 'line 1\nline 2 changed on main\nline 3 edited in wip\nline 4\nline 5\nline 6 added\n');
write(wt, 'notes.txt', 'untracked file\n');
write(wt, 'staged.txt', 'staged file\n');
gitIn(wt, 'add', 'staged.txt');

try {
  git('merge', 'conflict');
} catch {}

console.log(`Open in Rift: ${repo}`);
console.log(`Worktree: ${wt}`);
