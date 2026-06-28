#!/usr/bin/env bun
// Concatenate docs/[0-9]*.md (filename order) into docs/_all.md as one scrolling
// page, rewriting cross-doc links to in-page anchors so navigation stays on the page.
import { readdirSync, readFileSync, writeFileSync } from "node:fs";
import { join, dirname, basename } from "node:path";
import { fileURLToPath } from "node:url";

const DOCS = dirname(dirname(fileURLToPath(import.meta.url)));

const slug = (text: string) =>
  text.replace(/[^\w\- ]/g, "").trim().toLowerCase().replace(/ /g, "-");

const files = readdirSync(DOCS)
  .filter((f) => /^\d.*\.md$/.test(f))
  .sort()
  .map((f) => join(DOCS, f));

// basename (old name, no number prefix) -> anchor of its first heading
const firstAnchor: Record<string, string> = {};
for (const f of files) {
  const name = basename(f).replace(/^\d+-/, "");
  for (const line of readFileSync(f, "utf-8").split("\n")) {
    const m = line.match(/^#+\s+(.*)/);
    if (m) { firstAnchor[name] = slug(m[1]); break; }
  }
}

const link = /\]\(([a-z0-9-]+\.md)(#[a-z0-9-]+)?\)/g;
const fix = (_: string, target: string, frag?: string) =>
  "](" + (frag ? frag : "#" + (firstAnchor[target] ?? "")) + ")";

const parts = files.map((f) => readFileSync(f, "utf-8").replace(link, fix));
writeFileSync(join(DOCS, "_all.md"), parts.join("\n\n") + "\n", "utf-8");

// self-check: every cross-doc link became an anchor
const leftover = parts.join("\n").match(link);
if (leftover) throw new Error(`unrewritten links: ${leftover.slice(0, 5)}`);
console.log(`built _all.md from ${files.length} files`);
