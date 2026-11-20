// End-to-end suite: drives the real Svelte UI against the real Go backend (wails dev) in headless
// Edge/Chrome over CDP, and checks the on-disk git state after each action.
// Usage: node e2e/run.mjs [--only <name>] [--keep]
import { spawn, execFileSync } from 'node:child_process';
import { existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { createServer } from 'node:net';

const root = resolve(import.meta.dirname, '..');
const argv = process.argv.slice(2);
const only = argv.includes('--only') ? argv[argv.indexOf('--only') + 1] : null;
const keep = argv.includes('--keep');
const APP_PORT = 34135;
const VITE_PORT = 1430;
const URL = `http://localhost:${APP_PORT}`;
const BUDGET_MS = 3000;
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const win = process.platform === 'win32';

// Isolated temp tree: scratch repos and Rift's config dir, so the user's pinned repos are untouched.
const work = mkdtempSync(join(tmpdir(), 'rift-e2e-'));
const tmp = join(work, 'tmp');
const config = join(work, 'config');
mkdirSync(tmp, { recursive: true });
mkdirSync(config, { recursive: true });
const repo = join(tmp, 'rift-play', 'rift-play');
const wt = join(tmp, 'rift-play', 'rift-play-wt');
const slash = (p) => p.replaceAll('\\', '/');

async function makeRepos() {
  // The app's watcher holds handles on the old tree (Windows blocks deleting it): unwatch, then delete with retries.
  if (ws) {
    await js(`window.go?.main?.App?.Watch('')`).catch(() => {});
    await send('Page.navigate', { url: 'about:blank' });
    await sleep(800);
  }
  for (let i = 0; ; i++) {
    try {
      rmSync(join(tmp, 'rift-play'), { recursive: true, force: true });
      break;
    } catch (e) {
      // A git process the app just spawned can still have the folder open.
      if (i >= 60) throw e;
      await sleep(250);
    }
  }
  execFileSync(process.execPath, [join(root, 'scripts', 'play.mjs')], {
    env: { ...process.env, TEMP: tmp, TMP: tmp, TMPDIR: tmp },
    stdio: 'ignore',
    windowsHide: true,
  });
}

const git = (dir, ...args) => execFileSync('git', args, { cwd: dir, windowsHide: true, stdio: ['ignore', 'pipe', 'pipe'] }).toString();

function portFree(port, host = '127.0.0.1') {
  return new Promise((r) => {
    const s = createServer().once('error', (err) => r(err.code === 'EADDRNOTAVAIL' || err.code === 'EAFNOSUPPORT')).once('listening', () => s.close(() => r(true)));
    s.listen(port, host);
  });
}

// --- processes we own -------------------------------------------------------
const owned = [];
function killTree(p) {
  if (!p || p.exitCode !== null) return;
  try {
    if (win) execFileSync('taskkill', ['/PID', String(p.pid), '/T', '/F'], { stdio: 'ignore', windowsHide: true });
    else process.kill(-p.pid, 'SIGKILL');
  } catch {}
}
function cleanup() {
  if (keep) return;
  owned.forEach(killTree);
  try { rmSync(work, { recursive: true, force: true }); } catch {}
}
process.on('SIGINT', () => { cleanup(); process.exit(130); });

async function startApp() {
  if (!(await portFree(VITE_PORT)) || !(await portFree(VITE_PORT, '::1'))) throw new Error(`port ${VITE_PORT} (vite) is busy; stop the other dev server first`);
  if (!(await portFree(APP_PORT))) throw new Error(`port ${APP_PORT} is busy`);
  const env = { ...process.env };
  if (win) env.APPDATA = config;
  else if (process.platform === 'darwin') env.HOME = config;
  else env.XDG_CONFIG_HOME = config;
  const p = spawn('wails', ['dev', '-s', '-viteservertimeout', '60', '-devserver', `localhost:${APP_PORT}`, '-appargs', slash(repo)], { cwd: root, env, windowsHide: true, detached: !win });
  owned.push(p);
  let log = '';
  p.stdout.on('data', (d) => (log += d));
  p.stderr.on('data', (d) => (log += d));
  const t0 = Date.now();
  while (!/Watching \(sub\)/.test(log)) {
    if (p.exitCode !== null) throw new Error('wails dev exited:\n' + log.slice(-2000));
    if (Date.now() - t0 > 240_000) throw new Error('wails dev did not start in 240 s:\n' + log.slice(-2000));
    await sleep(500);
  }
}

// --- browser over CDP --------------------------------------------------------
function browserPath() {
  const c = win
    ? ['C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe', 'C:/Program Files/Microsoft/Edge/Application/msedge.exe', 'C:/Program Files/Google/Chrome/Application/chrome.exe']
    : process.platform === 'darwin'
      ? ['/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge', '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome', '/Applications/Chromium.app/Contents/MacOS/Chromium']
      : ['/usr/bin/microsoft-edge', '/usr/bin/google-chrome', '/usr/bin/chromium', '/usr/bin/chromium-browser'];
  const p = c.find(existsSync);
  if (!p) throw new Error('no Edge/Chrome found');
  return p;
}

let ws;
let seq = 0;
const pending = new Map();
let problems = [];
const IGNORED = /wails\/ipc\.js|reading 'nodes'/;

async function startBrowser() {
  const dataDir = join(work, 'browser');
  const p = spawn(browserPath(), ['--headless=new', '--remote-debugging-port=0', `--user-data-dir=${dataDir}`, '--window-size=1400,900', '--no-first-run', 'about:blank'], { stdio: 'ignore', windowsHide: true, detached: !win });
  owned.push(p);
  const portFile = join(dataDir, 'DevToolsActivePort');
  for (let i = 0; i < 80 && !existsSync(portFile); i++) await sleep(250);
  const port = readFileSync(portFile, 'utf8').split('\n')[0];
  const targets = await (await fetch(`http://127.0.0.1:${port}/json`)).json();
  ws = new WebSocket(targets.find((t) => t.type === 'page').webSocketDebuggerUrl);
  await new Promise((r) => (ws.onopen = r));
  ws.onmessage = (m) => {
    const msg = JSON.parse(m.data);
    if (msg.id && pending.has(msg.id)) { pending.get(msg.id)(msg); pending.delete(msg.id); return; }
    if (msg.method === 'Runtime.exceptionThrown') {
      const d = msg.params.exceptionDetails;
      const text = d.exception?.description ?? d.text;
      if (!IGNORED.test(text + (d.url ?? ''))) problems.push('exception: ' + text.split('\n').slice(0, 4).join(' | '));
    }
    if (msg.method === 'Runtime.consoleAPICalled' && ['error', 'warning', 'warn'].includes(msg.params.type)) {
      const text = msg.params.args.map((a) => a.value ?? a.description).join(' ');
      if (/svelte/i.test(text)) problems.push(`console.${msg.params.type}: ` + text.slice(0, 300));
    }
  };
  await send('Runtime.enable');
  await send('Page.enable');
  // --window-size is not honoured by every headless build; pin the viewport.
  await send('Emulation.setDeviceMetricsOverride', { width: 1400, height: 900, deviceScaleFactor: 1, mobile: false });
}

function send(method, params = {}) {
  return new Promise((r) => {
    const id = ++seq;
    pending.set(id, r);
    ws.send(JSON.stringify({ id, method, params }));
  });
}

async function js(expr) {
  const r = await send('Runtime.evaluate', { expression: expr, returnByValue: true, awaitPromise: true });
  if (r.result?.exceptionDetails) throw new Error('page eval failed: ' + (r.result.exceptionDetails.exception?.description ?? r.result.exceptionDetails.text));
  return r.result?.result?.value;
}

// Finder installed in the page: visible element by css, or by title/text (exact match preferred).
const FINDER = `window.__find = (q, scope) => {
  const root = scope ? window.__find(scope) : document;
  if (!root) return null;
  const vis = (e) => { const r = e.getBoundingClientRect(); return r.width > 0 && r.height > 0 && getComputedStyle(e).visibility !== 'hidden'; };
  if (q.startsWith('css:')) return [...root.querySelectorAll(q.slice(4))].find(vis) ?? null;
  const all = [...root.querySelectorAll('button,[role=button],[role=tab],[role=treeitem],[role=row],[role=option],a,label,input,textarea')].filter(vis);
  const t = (e) => (e.getAttribute('title') || e.getAttribute('placeholder') || '').trim();
  const x = (e) => (e.textContent ?? '').replace(/\\s+/g, ' ').trim();
  return all.find((e) => x(e) === q || t(e) === q) ?? all.find((e) => t(e).includes(q)) ?? all.find((e) => x(e).includes(q)) ?? null;
};`;

async function reload() {
  await send('Page.navigate', { url: URL });
  await sleep(500);
  await waitFor('document.readyState === "complete"', 15_000, 'page load');
  await js(FINDER);
}

async function waitFor(expr, timeout = 10_000, what = expr) {
  const t0 = Date.now();
  for (;;) {
    try {
      await js(FINDER);
      if (await js(`!!(${expr})`)) return Date.now() - t0;
    } catch {}
    if (Date.now() - t0 > timeout) throw new Error(`timed out (${timeout} ms) waiting for: ${what}`);
    await sleep(100);
  }
}

// Real mouse click at the element's center, so overlays and dead handlers are caught.
async function click(q, scope, button = 'left') {
  await js(FINDER);
  const pos = await js(`(() => { const e = window.__find(${JSON.stringify(q)}, ${JSON.stringify(scope ?? null)}); if (!e) return null; e.scrollIntoView({ block: 'center' }); const r = e.getBoundingClientRect(); const x = r.x + r.width / 2, y = r.y + r.height / 2; const top = document.elementFromPoint(x, y); return { x, y, covered: !(top === e || e.contains(top) || top?.contains(e)), top: top ? top.tagName + '.' + top.className : '' }; })()`);
  if (!pos) throw new Error(`not found: ${q}${scope ? ` in ${scope}` : ''}`);
  if (pos.covered) throw new Error(`"${q}" is covered by ${pos.top}`);
  for (const type of ['mouseMoved', 'mousePressed', 'mouseReleased'])
    await send('Input.dispatchMouseEvent', { type, x: pos.x, y: pos.y, button, clickCount: 1 });
  await sleep(150);
}

const KEYS = { Enter: [13, 'Enter', '\r'], F7: [118, 'F7'], Escape: [27, 'Escape'], End: [35, 'End'], s: [83, 'KeyS'] };
async function key(combo) {
  const parts = combo.split('+');
  const k = parts.pop();
  const mods = (parts.includes('Alt') ? 1 : 0) | (parts.includes('Ctrl') ? 2 : 0) | (parts.includes('Shift') ? 8 : 0);
  const [vk, code, text] = KEYS[k];
  await send('Input.dispatchKeyEvent', { type: 'keyDown', key: k, code, windowsVirtualKeyCode: vk, nativeVirtualKeyCode: vk, modifiers: mods, ...(text && !mods ? { text } : {}) });
  await send('Input.dispatchKeyEvent', { type: 'keyUp', key: k, code, windowsVirtualKeyCode: vk, nativeVirtualKeyCode: vk, modifiers: mods });
  await sleep(150);
}
const type = (text) => send('Input.insertText', { text });

async function untilGit(dir, args, pred, what, timeout = 10_000) {
  const t0 = Date.now();
  for (;;) {
    const out = git(dir, ...args);
    if (pred(out)) return out;
    if (Date.now() - t0 > timeout) throw new Error(`${what}; git ${args.join(' ')} gave:\n${out}`);
    await sleep(200);
  }
}

// --- app-level steps -----------------------------------------------------------
const branchIs = (b) => `window.__find('Branches')?.textContent.trim() === ${JSON.stringify(b)}`;
const fileRow = (name) => `[...document.querySelectorAll('[role=tree] .row.file .name')].some((e) => e.textContent.trim() === ${JSON.stringify(name)})`;

let fresh = false;
async function boot() {
  // The repos made during setup are still untouched for the first scenario.
  if (!fresh) await makeRepos();
  fresh = false;
  await reload();
  await waitFor(branchIs('main'), 20_000, 'toolbar shows branch main');
}

async function toWorktree(suffix, branch) {
  await click('Switch repository or worktree');
  await js(FINDER);
  const ok = await js(`(() => { const b = [...document.querySelectorAll('button.wt')].find((e) => e.title.replaceAll('\\\\', '/').endsWith('/' + ${JSON.stringify(suffix)})); if (!b) return false; b.setAttribute('data-e2e', 'wt'); return true; })()`);
  if (!ok) throw new Error(`worktree ${suffix} not in switcher`);
  const t0 = Date.now();
  await click('css:button[data-e2e=wt]');
  await waitFor(branchIs(branch), 10_000, `toolbar shows ${branch}`);
  return Date.now() - t0;
}

async function clickFile(name) {
  await js(`(() => { document.querySelectorAll('[data-e2e=row]').forEach((e) => e.removeAttribute('data-e2e')); const n = [...document.querySelectorAll('[role=tree] .row.file .name')].find((e) => e.textContent.trim() === ${JSON.stringify(name)}); n?.closest('[role=treeitem]').setAttribute('data-e2e', 'row'); })()`);
  await click('css:[data-e2e=row] .name');
}

async function toggleCheckbox(name) {
  await js(`(() => { document.querySelectorAll('[data-e2e=cb]').forEach((e) => e.removeAttribute('data-e2e')); const n = [...document.querySelectorAll('[role=tree] .row.file .name')].find((e) => e.textContent.trim() === ${JSON.stringify(name)}); n?.closest('[role=treeitem]').querySelector('input[type=checkbox]').setAttribute('data-e2e', 'cb'); })()`);
  await click('css:[data-e2e=cb]');
}

// --- scenarios -------------------------------------------------------------------
const scenarios = {
  async boot() {
    await waitFor(fileRow('shared.txt'), 10_000, 'Commit window lists shared.txt');
  },

  async 'worktree-switch'() {
    await toWorktree('rift-play-wt', 'wip');
    await waitFor(fileRow('notes.txt'), 10_000, 'wip files listed');
    await toWorktree('rift-play', 'main');
    await waitFor(`${fileRow('shared.txt')} && !${fileRow('notes.txt')}`, 10_000, 'main files listed again');
  },

  async 'tool-windows'() {
    await click('Project (Alt+1)');
    await waitFor(`[...document.querySelectorAll('[role=tree] .item')].some((e) => e.title === 'README.md')`, 10_000, 'Project tree shows README.md');
    await click('Commit (Alt+0)');
    await waitFor(`window.__find('Commit message')`, 5_000, 'Commit window back');
    await click('Commit (Alt+0)');
    await waitFor(`!window.__find('Commit message')`, 5_000, 'Commit window hidden');
    await click('Git (Alt+9)');
    await waitFor(`document.querySelectorAll('[role=row]').length > 3`, 10_000, 'log has commit rows');
    await click('Git (Alt+9)');
    await waitFor(`document.querySelectorAll('[role=row]').length === 0`, 5_000, 'log hidden');
  },

  async diff() {
    await toWorktree('rift-play-wt', 'wip');
    await waitFor(fileRow('app.txt'), 10_000, 'app.txt listed');
    await clickFile('app.txt');
    await waitFor(`document.querySelectorAll('.cm-editor').length >= 2`, 10_000, 'diff editors');
    await click('css:.cm-editor:last-of-type .cm-content');
    await key('F7');
    await waitFor(`document.querySelector('.cm-merge-revert button')`, 5_000, 'hunk control');
    await click('css:.cm-merge-revert button');
    await untilGit(wt, ['diff', '--cached', '--', 'app.txt'], (o) => o.trim() !== '', 'hunk not staged');
  },

  async 'stage-unstage'() {
    await toWorktree('rift-play-wt', 'wip');
    await waitFor(fileRow('notes.txt'), 10_000, 'notes.txt listed');
    await toggleCheckbox('notes.txt');
    await untilGit(wt, ['status', '--porcelain'], (o) => /^A  notes\.txt$/m.test(o), 'notes.txt not staged');
    await waitFor(fileRow('staged.txt'), 5_000, 'staged.txt listed');
    await toggleCheckbox('staged.txt');
    await untilGit(wt, ['status', '--porcelain'], (o) => /^\?\? staged\.txt$/m.test(o), 'staged.txt not unstaged');
  },

  async commit() {
    await toWorktree('rift-play-wt', 'wip');
    await click('Commit message');
    await type('e2e commit');
    await key('Ctrl+Enter');
    await untilGit(wt, ['log', '-1', '--format=%s'], (o) => o.trim() === 'e2e commit', 'commit not created');
    await untilGit(wt, ['diff', '--cached', '--name-only'], (o) => o.trim() === '', 'index not clean after commit');
  },

  async conflict() {
    await waitFor(`window.__find('Resolve')`, 10_000, 'conflict banner with Resolve');
    await click('Resolve');
    await waitFor(`document.querySelectorAll('.cm-editor').length >= 3`, 10_000, 'merge editors');
    const blocks = await js(`document.querySelectorAll('.blocks .block').length`);
    if (!blocks) throw new Error('no conflict blocks listed');
    for (let i = 0; i < blocks; i++) await click('Left', `css:.blocks .block:nth-child(${i + 1})`);
    await waitFor(`!window.__find('Apply')?.disabled`, 5_000, 'Apply enabled');
    await click('Apply');
    await untilGit(repo, ['status', '--porcelain'], (o) => !/^UU /m.test(o), 'still unmerged');
    const body = readFileSync(join(repo, 'shared.txt'), 'utf8');
    if (/^(<<<<<<<|=======|>>>>>>>)/m.test(body)) throw new Error('markers left in shared.txt:\n' + body);
  },

  async 'branch-popup'() {
    await toWorktree('rift-play-wt', 'wip');
    await click('Branches');
    await click('New branch…');
    await waitFor(`document.querySelector('dialog[open] input')`, 5_000, 'name dialog');
    await type('e2e-branch');
    await key('Enter');
    await untilGit(wt, ['branch', '--show-current'], (o) => o.trim() === 'e2e-branch', 'branch not created and checked out');
    await waitFor(branchIs('e2e-branch'), 10_000, 'toolbar shows e2e-branch');
    await click('Branches');
    await click('wip', 'css:.popup');
    await click('Checkout', 'css:.menu');
    await untilGit(wt, ['branch', '--show-current'], (o) => o.trim() === 'wip', 'checkout wip failed');
    await waitFor(branchIs('wip'), 10_000, 'toolbar shows wip');
  },

  async 'project-editor'() {
    await toWorktree('rift-play-wt', 'wip');
    await click('Project (Alt+1)');
    await waitFor(`[...document.querySelectorAll('[role=tree] .item')].some((e) => e.title === 'README.md')`, 10_000, 'README.md in tree');
    await click('css:[role=tree] .item[title="README.md"]');
    await waitFor(`document.querySelector('.cm-editor .cm-content')?.textContent.includes('# Play')`, 10_000, 'editor shows README');
    await click('css:.cm-editor .cm-content');
    await type('e2e edit ');
    await key('Ctrl+s');
    const t0 = Date.now();
    while (!readFileSync(join(wt, 'README.md'), 'utf8').includes('e2e edit')) {
      if (Date.now() - t0 > 5000) throw new Error('README.md not saved');
      await sleep(200);
    }
    // Unsaved edits must survive leaving the tab.
    await type('unsaved ');
    await click('css:[role=tab] .x');
    const t1 = Date.now();
    while (!readFileSync(join(wt, 'README.md'), 'utf8').includes('unsaved')) {
      if (Date.now() - t1 > 5000) throw new Error('unsaved edit lost when the tab closed');
      await sleep(200);
    }
  },

  async rebase() {
    await toWorktree('rift-play-wt', 'wip');
    await click('Git (Alt+9)');
    const mark = `(() => { const t = [...document.querySelectorAll('[role=row] .text')].find((e) => e.textContent === 'Add CRLF file'); t?.closest('[role=row]').setAttribute('data-e2e', 'commit'); return !!t; })()`;
    await waitFor(mark, 10_000, 'log shows Add CRLF file');
    await click('css:[data-e2e=commit] .text', null, 'right');
    await click('Interactively Rebase from Here…', 'css:.menu');
    await waitFor(`window.__find('Start Rebasing') && document.querySelectorAll('.table [role=row]').length === 2`, 10_000, 'rebase tab lists 2 commits');
    await click('Drop');
    await click('Start Rebasing');
    // The worktree is dirty, so this also proves the rebase autostashes and restores local changes.
    await untilGit(wt, ['log', '--format=%s', '-3'], (o) => o.startsWith('Change shared.txt on main') && !o.includes('Add CRLF file'), 'commit not dropped', 30_000);
    await untilGit(wt, ['status', '--porcelain'], (o) => /^ M app\.txt$/m.test(o) && /^A  staged\.txt$/m.test(o), 'local changes not restored after rebase', 15_000);
  },

  async changelists() {
    await toWorktree('rift-play-wt', 'wip');
    await waitFor(fileRow('app.txt'), 10_000, 'app.txt listed');
    const group = (n) => `[...document.querySelectorAll('.gname')].some((e) => e.textContent === ${JSON.stringify(n)})`;
    try {
      await js(`(() => { const s = document.querySelector('select[title="Commit model"]'); s.value = 'changelists'; s.dispatchEvent(new Event('change', { bubbles: true })); })()`);
      await waitFor(`${group('Unversioned Files')} && !${group('Staged')}`, 10_000, 'changelist groups');
      await toggleCheckbox('staged.txt');
      await click('Commit message');
      await type('e2e changelist commit');
      await key('Ctrl+Enter');
      await untilGit(wt, ['log', '-1', '--format=%s'], (o) => o.trim() === 'e2e changelist commit', 'changelist commit not created', 20_000);
      const files = git(wt, 'show', '--name-only', '--format=', 'HEAD').trim();
      if (files !== 'app.txt') throw new Error(`commit should hold only app.txt, has: ${files}`);
      await untilGit(wt, ['status', '--porcelain'], (o) => /^A  staged\.txt$/m.test(o) && /^\?\? notes\.txt$/m.test(o) && !/app\.txt/.test(o), 'unchecked files were touched');
    } finally {
      await js(`window.go.main.App.SetMode('staging')`).catch(() => {});
    }
  },

  async perf() {
    const sw = await toWorktree('rift-play-wt', 'wip');
    const rows = await waitFor(fileRow('app.txt'), BUDGET_MS * 2, 'rows after switch');
    const t0 = Date.now();
    await clickFile('app.txt');
    await waitFor(`document.querySelectorAll('.cm-editor').length >= 2`, BUDGET_MS * 2, 'editor after click');
    const open = Date.now() - t0;
    console.log(`    switch->branch ${sw} ms, ->rows +${rows} ms, file->editor ${open} ms`);
    if (sw + rows > BUDGET_MS || open > BUDGET_MS) throw new Error(`over budget ${BUDGET_MS} ms`);
  },
};

// --- runner ------------------------------------------------------------------------
const names = only ? [only] : Object.keys(scenarios);
if (only && !scenarios[only]) {
  console.error(`unknown scenario ${only}; one of: ${Object.keys(scenarios).join(', ')}`);
  process.exit(2);
}
let failed = 0;
let passed = 0;
const t0 = Date.now();
try {
  console.log('starting wails dev (isolated config) …');
  await makeRepos();
  fresh = true;
  await startApp();
  await startBrowser();
  for (const name of names) {
    const s0 = Date.now();
    problems = [];
    try {
      await boot();
      await scenarios[name]();
      await sleep(300);
      const toasts = await js(`[...document.querySelectorAll('.toast.error')].map((e) => e.textContent.trim())`);
      if (toasts?.length) problems.push('error toast: ' + toasts.join(' | '));
      if (problems.length) throw new Error(problems.join('\n      '));
      passed++;
      console.log(`PASS ${name} (${Date.now() - s0} ms)`);
    } catch (e) {
      failed++;
      console.log(`FAIL ${name} (${Date.now() - s0} ms)\n      ${String(e.message ?? e).replaceAll('\n', '\n      ')}`);
      if (problems.length && !String(e.message).includes(problems[0])) console.log('      also: ' + problems.join('\n      '));
    }
  }
} catch (e) {
  failed++;
  console.log('SETUP FAIL: ' + (e.message ?? e));
  console.log('Scenarios not run: ' + names.join(', '));
} finally {
  console.log(`${passed}/${names.length} passed in ${Math.round((Date.now() - t0) / 1000)} s${keep ? ` (kept running; scratch at ${work})` : ''}`);
  ws?.close();
  cleanup();
}
process.exit(failed ? 1 : 0);
