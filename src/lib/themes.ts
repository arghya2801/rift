import { applyPalette, cssVariables, parsePalette, type Palette } from './theme';

import defaultTheme from '../themes/intellij-dark.json';

// Eager named imports keep just the display names in the initial bundle.
const names = import.meta.glob<string>('../themes/*.json', { eager: true, import: 'name' });
const loaders = import.meta.glob<string>('../themes/*.json', { query: '?raw', import: 'default' });
export const themeNames = Object.values(names).sort((a, b) => a.localeCompare(b));
export const DEFAULT_THEME = 'IntelliJ Dark';
const fallback = parsePalette(defaultTheme)!;
const cache = new Map<string, Promise<Palette>>();
const KEY = 'rift.theme';

export async function findTheme(name: string | null | undefined): Promise<Palette> {
  const wanted = name?.trim().toLowerCase();
  if (!wanted || wanted === DEFAULT_THEME.toLowerCase()) return fallback;
  const path = Object.keys(names).find((key) => names[key].toLowerCase() === wanted);
  if (!path) return fallback;
  let pending = cache.get(path);
  if (!pending) {
    pending = loaders[path]().then((raw) => parsePalette(JSON.parse(raw), names[path]) ?? fallback).catch((err) => {
      cache.delete(path);
      throw err;
    });
    cache.set(path, pending);
  }
  return pending;
}

export function savedTheme(): string {
  try {
    return localStorage.getItem(KEY) ?? DEFAULT_THEME;
  } catch {
    return DEFAULT_THEME;
  }
}

let generation = 0;
export async function applyTheme(name: string): Promise<Palette> {
  const id = ++generation;
  const p = await findTheme(name);
  if (id !== generation) return p;
  const root = document.documentElement;
  if (p.name === DEFAULT_THEME) {
    for (const v of Object.keys(cssVariables(p))) root.style.removeProperty(v);
    root.style.colorScheme = 'dark';
  } else applyPalette(p, root);
  try {
    localStorage.setItem(KEY, p.name);
  } catch {}
  return p;
}
