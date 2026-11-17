import { applyPalette, cssVariables, parsePalette, type Palette } from './theme';

const files = import.meta.glob<Record<string, unknown>>('../themes/*.json', { eager: true, import: 'default' });

export const themes: Palette[] = Object.entries(files)
  .flatMap(([path, raw]) => {
    const p = parsePalette(raw, path.split('/').pop()!.replace(/\.json$/i, ''));
    return p ? [p] : [];
  })
  .sort((a, b) => a.name.localeCompare(b.name));

export const themeNames = themes.map((t) => t.name);
export const DEFAULT_THEME = 'IntelliJ Dark';
const KEY = 'rift.theme';

export function findTheme(name: string | null | undefined): Palette {
  const wanted = (name ?? '').trim().toLowerCase();
  return themes.find((t) => t.name.toLowerCase() === wanted) ?? themes.find((t) => t.name === DEFAULT_THEME) ?? themes[0];
}

export function savedTheme(): string {
  try {
    return localStorage.getItem(KEY) ?? DEFAULT_THEME;
  } catch {
    return DEFAULT_THEME;
  }
}

export function applyTheme(name: string): Palette {
  const p = findTheme(name);
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
