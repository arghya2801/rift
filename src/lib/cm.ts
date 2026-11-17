import { EditorView, lineNumbers, drawSelection, highlightSpecialChars, keymap } from '@codemirror/view';
import type { Extension } from '@codemirror/state';
import { HighlightStyle, LanguageDescription, bracketMatching, syntaxHighlighting } from '@codemirror/language';
import { languages } from '@codemirror/language-data';
import { defaultKeymap } from '@codemirror/commands';
import { searchKeymap, highlightSelectionMatches } from '@codemirror/search';
import { tags as t } from '@lezer/highlight';
import { findTheme, savedTheme } from './themes';
import { isLight } from './theme';

export const cmHighlight = HighlightStyle.define([
  { tag: [t.keyword, t.modifier, t.controlKeyword, t.operatorKeyword, t.definitionKeyword, t.moduleKeyword], color: '#cf8e6d' },
  { tag: [t.bool, t.null, t.atom], color: '#cf8e6d' },
  { tag: [t.string, t.special(t.string)], color: '#6aab73' },
  { tag: [t.regexp, t.escape], color: '#42c3d4' },
  { tag: [t.number], color: '#2aacb8' },
  { tag: [t.comment, t.lineComment, t.blockComment], color: '#7a7e85', fontStyle: 'italic' },
  { tag: t.docComment, color: '#5f826b', fontStyle: 'italic' },
  { tag: [t.function(t.variableName), t.function(t.propertyName)], color: '#56a8f5' },
  { tag: [t.definition(t.variableName)], color: '#bcbec4' },
  { tag: [t.typeName, t.className, t.namespace], color: '#bcbec4' },
  { tag: [t.propertyName, t.attributeName], color: '#c77dbb' },
  { tag: [t.tagName], color: '#d5b778' },
  { tag: [t.meta, t.annotation], color: '#b3ae60' },
  { tag: [t.constant(t.variableName), t.standard(t.variableName)], color: '#c77dbb' },
  { tag: t.heading, fontWeight: 'bold', color: '#cf8e6d' },
  { tag: t.link, textDecoration: 'underline', color: '#56a8f5' },
  { tag: t.emphasis, fontStyle: 'italic' },
  { tag: t.strong, fontWeight: 'bold' },
  { tag: t.invalid, color: '#f75464' },
]);

const chrome = (dark: boolean) =>
  EditorView.theme(
    {
      '&': { color: 'var(--fg)', backgroundColor: 'var(--bg-editor)', fontSize: '13px', height: '100%' },
      '.cm-scroller': { fontFamily: 'var(--font-mono)', lineHeight: '1.5' },
      '.cm-content': { caretColor: 'var(--fg)' },
      '.cm-cursor': { borderLeftColor: 'var(--fg)' },
      '&.cm-focused .cm-selectionBackground, .cm-selectionBackground, ::selection': { backgroundColor: 'var(--selection) !important' },
      '.cm-gutters': { backgroundColor: 'var(--bg-editor)', color: 'var(--fg-faint)', border: 'none' },
      '.cm-lineNumbers .cm-gutterElement': { padding: '0 10px 0 14px' },
      '.cm-activeLine': { backgroundColor: 'color-mix(in srgb, var(--fg) 4%, transparent)' },
      '.cm-activeLineGutter': { backgroundColor: 'transparent', color: 'var(--fg)' },
      '.cm-matchingBracket': { backgroundColor: 'color-mix(in srgb, var(--accent) 30%, transparent)', outline: 'none' },
      '.cm-searchMatch': { backgroundColor: 'color-mix(in srgb, var(--warn) 30%, transparent)' },
      '.cm-panels': { backgroundColor: 'var(--bg-panel)', color: 'var(--fg)', borderColor: 'var(--line)' },
    },
    { dark },
  );

export async function cmLanguage(path: string): Promise<Extension> {
  return (await LanguageDescription.matchFilename(languages, path)?.load()) ?? [];
}

// Lean read/edit base: no history, folding or autocomplete. Editors add those on top.
export async function cmBase(path: string): Promise<Extension[]> {
  return [
    lineNumbers(),
    highlightSpecialChars(),
    drawSelection(),
    bracketMatching(),
    highlightSelectionMatches(),
    syntaxHighlighting(cmHighlight),
    keymap.of([...searchKeymap, ...defaultKeymap]),
    chrome(!isLight(findTheme(savedTheme()))),
    await cmLanguage(path),
  ];
}
