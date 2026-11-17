export type MenuItem = { label: string; run: () => unknown; danger?: boolean; hint?: string } | null;

export const menu = $state<{ x: number; y: number; items: MenuItem[] }>({ x: 0, y: 0, items: [] });

export function openMenu(e: MouseEvent, items: MenuItem[]) {
  e.preventDefault();
  e.stopPropagation();
  menu.x = e.clientX;
  menu.y = e.clientY;
  menu.items = items;
}

export function closeMenu() {
  menu.items = [];
}

// In-app dialogs, rendered by src/shell/Dialog.svelte.
export type DialogReq = { title: string; message?: string; input?: string; ok: string; danger?: boolean; resolve: (v: string | null) => void };
export const dialog = $state<{ req: DialogReq | null }>({ req: null });

function open(req: Omit<DialogReq, 'resolve'>): Promise<string | null> {
  dialog.req?.resolve(null);
  return new Promise((resolve) => {
    dialog.req = { ...req, resolve };
  });
}

// Resolves to the trimmed text, or '' when cancelled.
export async function ask(title: string, value = '', ok = 'OK'): Promise<string> {
  return ((await open({ title, input: value, ok })) ?? '').trim();
}

export async function sure(title: string, message = '', ok = 'OK', danger = true): Promise<boolean> {
  return (await open({ title, message, ok, danger })) !== null;
}

// Notification balloons, rendered by the shell.
export type Toast = { id: number; text: string; kind: 'error' | 'info' };
export const toasts = $state<Toast[]>([]);
let seq = 0;
export function toast(text: string, kind: Toast['kind'] = 'error') {
  const id = ++seq;
  toasts.push({ id, text, kind });
  setTimeout(() => dismiss(id), kind === 'error' ? 8000 : 4000);
}
export function dismiss(id: number) {
  const i = toasts.findIndex((t) => t.id === id);
  if (i >= 0) toasts.splice(i, 1);
}
