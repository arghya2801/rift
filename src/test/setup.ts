import { vi, beforeEach, afterEach } from 'vitest';

vi.mock('../../wailsjs/go/main/App', async () => (await import('./wails')).Go);
vi.mock('../../wailsjs/runtime/runtime', async () => (await import('./wails')).runtime);

// Svelte runtime warnings and uncaught errors (state_unsafe_mutation, assignment_value_stale, ...)
// froze the real UI once; here they fail the test that caused them.
const bad = /svelte|state_unsafe_mutation|assignment_value_stale|Uncaught/i;
let problems: string[] = [];
const text = (args: unknown[]) => args.map((a) => (a instanceof Error ? `${a.message}\n${a.stack}` : String(a))).join(' ');

const origError = console.error;
const origWarn = console.warn;
console.error = (...args: unknown[]) => {
  if (bad.test(text(args))) problems.push('console.error: ' + text(args));
  origError(...args);
};
console.warn = (...args: unknown[]) => {
  if (bad.test(text(args))) problems.push('console.warn: ' + text(args));
  origWarn(...args);
};
if (typeof window !== 'undefined') {
  window.addEventListener('error', (e) => problems.push('window error: ' + (e.error?.stack ?? e.message)));
  window.addEventListener('unhandledrejection', (e) => problems.push('unhandled rejection: ' + String(e.reason?.stack ?? e.reason)));
  // jsdom gaps CodeMirror and layout code touch.
  const proto = window.Range?.prototype as unknown as Record<string, unknown> | undefined;
  if (proto && !proto.getClientRects) proto.getClientRects = () => ({ length: 0, item: () => null, [Symbol.iterator]: [][Symbol.iterator] });
  if (proto && !proto.getBoundingClientRect) proto.getBoundingClientRect = () => ({ x: 0, y: 0, width: 0, height: 0, top: 0, left: 0, right: 0, bottom: 0 });
  if (!('ResizeObserver' in window)) (window as unknown as Record<string, unknown>).ResizeObserver = class { observe() {} unobserve() {} disconnect() {} };
  const dlg = window.HTMLDialogElement?.prototype;
  if (dlg && !dlg.showModal) {
    dlg.showModal = function (this: HTMLDialogElement) { this.open = true; };
    dlg.show = function (this: HTMLDialogElement) { this.open = true; };
    dlg.close = function (this: HTMLDialogElement) { this.open = false; this.dispatchEvent(new Event('close')); };
  }
  if (!Element.prototype.scrollIntoView) Element.prototype.scrollIntoView = () => {};
  if (!Element.prototype.setPointerCapture) Element.prototype.setPointerCapture = () => {};
}

beforeEach(() => {
  problems = [];
});

afterEach(() => {
  const p = problems;
  problems = [];
  if (p.length) throw new Error('Svelte runtime problems:\n' + p.join('\n---\n'));
});
