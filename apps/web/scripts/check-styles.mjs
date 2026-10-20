import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { readFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import postcss from 'postcss';
import tailwindcss from 'tailwindcss';
import autoprefixer from 'autoprefixer';
import config from '../postcss.config.js';

if (process.argv.includes('--compile')) {
  const source = fileURLToPath(new URL('../src/index.css', import.meta.url));
  const result = await postcss([
    tailwindcss(config.plugins.tailwindcss),
    autoprefixer(config.plugins.autoprefixer),
  ]).process(await readFile(source, 'utf8'), { from: source });

  const rules = new Map();
  result.root.walkRules((rule) => rules.set(rule.selector, rule));
  for (const selector of ['.bg-navigation', '.bg-canvas', '.bg-accent', '.hidden', '.min-h-11', '.lg\\:flex']) {
    assert.ok(rules.has(selector), `Missing app style ${selector} when launched from ${process.cwd()}`);
  }
  const desktop = rules.get('.lg\\:flex');
  assert.equal(desktop.parent.name, 'media');
  assert.equal(desktop.parent.params, '(min-width: 1024px)');
  assert.ok(desktop.nodes.some((node) => node.prop === 'display' && node.value === 'flex'));
} else {
  const script = fileURLToPath(import.meta.url);
  for (const [label, directory] of [['repository root', '../../..'], ['web app', '..']]) {
    execFileSync(process.execPath, [script, '--compile'], {
      cwd: fileURLToPath(new URL(`${directory}/`, import.meta.url)),
      stdio: 'inherit',
    });
    console.log(`App styles generated from ${label}`);
  }
}
