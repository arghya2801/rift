import { EditorView } from '@codemirror/view';
import type { Extension } from '@codemirror/state';

export const ANSI = [
  'black',
  'red',
  'green',
  'yellow',
  'blue',
  'magenta',
  'cyan',
  'white',
  'brightBlack',
  'brightRed',
  'brightGreen',
  'brightYellow',
  'brightBlue',
  'brightMagenta',
  'brightCyan',
  'brightWhite',
] as const;

export type AnsiName = (typeof ANSI)[number];

export interface Palette extends Record<AnsiName, string> {
  name: string;
  background: string;
  foreground: string;
  cursor: string;
  selection: string;
}

const HEX = /^#(?:[0-9a-f]{3}|[0-9a-f]{6})$/i;

// Accepts Windows Terminal (`cursorColor`, `purple`) and VS Code (`terminal.ansiX`) spellings.
export function parsePalette(raw: unknown, fallbackName = ''): Palette | null {
  if (typeof raw !== 'object' || raw === null) return null;
  const o = raw as Record<string, unknown>;
  const pick = (...keys: string[]): string | null => {
    for (const k of keys) {
      const v = o[k] ?? o[`terminal.ansi${k[0].toUpperCase()}${k.slice(1)}`];
      if (typeof v === 'string' && HEX.test(v.trim())) return v.trim().toLowerCase();
    }
    return null;
  };
  const background = pick('background');
  const foreground = pick('foreground');
  if (!background || !foreground) return null;
  const name = typeof o.name === 'string' && o.name.trim() ? o.name.trim() : fallbackName;
  if (!name) return null;
  const colours = {} as Record<AnsiName, string>;
  for (const key of ANSI) {
    const alt = key === 'magenta' ? 'purple' : key === 'brightMagenta' ? 'brightPurple' : key;
    colours[key] = pick(key, alt) ?? foreground;
  }
  return {
    ...colours,
    name,
    background,
    foreground,
    cursor: pick('cursor', 'cursorColor') ?? foreground,
    selection: pick('selection', 'selectionBackground') ?? colours.brightBlack,
  };
}

// Chrome surfaces are mixed from the palette, so light themes work without extra keys.
export function cssVariables(p: Palette): Record<string, string> {
  const lift = (pct: number) => `color-mix(in srgb, ${p.background} ${pct}%, ${p.foreground})`;
  const dim = (pct: number) => `color-mix(in srgb, ${p.foreground} ${pct}%, ${p.background})`;
  const light = isLight(p);
  return {
    '--bg': p.background,
    '--fg': dim(88),
    '--bg-chrome': lift(94),
    '--bg-surface': lift(88),
    '--bg-hover': lift(82),
    '--border': lift(78),
    '--fg-bright': p.foreground,
    '--fg-dim': dim(light ? 86 : 62),
    '--fg-faint': dim(light ? 84 : 55),
    '--accent': p.brightBlue,
    '--accent-strong': p.blue,
    '--on-accent': onAccent(p),
    '--danger': p.brightRed,
    '--danger-strong': p.red,
    '--warn': p.brightYellow,
    '--ok': p.brightGreen,
    '--info': p.brightMagenta,
    '--selection': p.selection,
  };
}

export function isLight(p: Palette): boolean {
  return luminance(p.background) > 0.4;
}

function onAccent(p: Palette): string {
  const ratio = (c: string) => {
    const [a, b] = [luminance(c), luminance(p.brightBlue)].sort((x, y) => y - x);
    return (a + 0.05) / (b + 0.05);
  };
  if (ratio(p.background) >= 4.5) return p.background;
  return ratio('#ffffff') >= ratio('#000000') ? '#ffffff' : '#000000';
}

function luminance(hex: string): number {
  const [r, g, b] = rgb(hex).map((c) => {
    const v = c / 255;
    return v <= 0.04045 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4;
  });
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

function rgb(hex: string): [number, number, number] {
  const h = hex.replace('#', '');
  const full = h.length === 3 ? [...h].map((c) => c + c).join('') : h;
  return [0, 2, 4].map((i) => parseInt(full.slice(i, i + 2), 16) || 0) as [number, number, number];
}

export function applyPalette(p: Palette, root: HTMLElement) {
  for (const [name, value] of Object.entries(cssVariables(p))) root.style.setProperty(name, value);
  root.style.colorScheme = isLight(p) ? 'light' : 'dark';
}

// Editor chrome only; syntax colours still come from the editor's highlight style.
export function cmTheme(p: Palette): Extension {
  return EditorView.theme(
    {
      '&': { color: p.foreground, backgroundColor: p.background },
      '.cm-content': { caretColor: p.cursor },
      '.cm-cursor, .cm-dropCursor': { borderLeftColor: p.cursor },
      '&.cm-focused .cm-selectionBackground, .cm-selectionBackground, ::selection': { backgroundColor: p.selection },
      '.cm-gutters': { backgroundColor: p.background, color: p.brightBlack, border: 'none' },
      '.cm-activeLine': { backgroundColor: `color-mix(in srgb, ${p.foreground} 5%, transparent)` },
      '.cm-activeLineGutter': { backgroundColor: `color-mix(in srgb, ${p.foreground} 8%, transparent)` },
    },
    { dark: !isLight(p) },
  );
}
