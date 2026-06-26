import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.join(path.dirname(fileURLToPath(import.meta.url)), '..', 'src', 'locales');

function load(locale) {
  const dir = path.join(root, locale);
  const keys = new Set();
  for (const file of fs.readdirSync(dir)) {
    if (!file.endsWith('.json')) {
      continue;
    }
    const json = JSON.parse(fs.readFileSync(path.join(dir, file), 'utf8'));
    for (const key of Object.keys(json)) {
      keys.add(`${file.replace('.json', '')}:${key}`);
    }
  }
  return keys;
}

const en = load('en');
const ru = load('ru');
const missingInRu = [...en].filter((k) => !ru.has(k));
const missingInEn = [...ru].filter((k) => !en.has(k));

if (missingInRu.length || missingInEn.length) {
  console.error('Locale key mismatch');
  missingInRu.forEach((k) => console.error(`  missing in ru: ${k}`));
  missingInEn.forEach((k) => console.error(`  missing in en: ${k}`));
  process.exit(1);
}

console.log(`Locale keys OK (${en.size} keys)`);
