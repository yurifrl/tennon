#!/usr/bin/env bun
// Build two scrolling pages: docs/_all.md (user guide, served at /) and
// docs/arch/_all.md (design & internals, served at /arch). Each concatenates its
// own NN-*.md files (filename order). Cross-doc links are rewritten: same-bucket
// links become in-page anchors; cross-bucket links point at the other page + anchor.
import { readdirSync, readFileSync, writeFileSync } from "node:fs";
import { join, dirname, basename } from "node:path";
import { fileURLToPath } from "node:url";

const DOCS = dirname(dirname(fileURLToPath(import.meta.url)));

const slug = (text: string) =>
  text.replace(/[^\w\- ]/g, "").trim().toLowerCase().replace(/ /g, "-");

const listMd = (dir: string) =>
  readdirSync(dir).filter((f) => /^\d.*\.md$/.test(f)).sort().map((f) => join(dir, f));

const buckets = {
  user: { dir: DOCS, out: join(DOCS, "_all.md"), other: "arch/_all.md" },
  arch: { dir: join(DOCS, "arch"), out: join(DOCS, "arch", "_all.md"), other: "../_all.md" },
} as const;
type Bucket = keyof typeof buckets;

// unnumbered basename (the name used in cross-doc links) -> {bucket, first-heading anchor}
const reg: Record<string, { bucket: Bucket; anchor: string }> = {};
for (const b of Object.keys(buckets) as Bucket[]) {
  for (const f of listMd(buckets[b].dir)) {
    const name = basename(f).replace(/^\d+-/, "");
    for (const line of readFileSync(f, "utf-8").split("\n")) {
      const m = line.match(/^#+\s+(.*)/);
      if (m) { reg[name] = { bucket: b, anchor: slug(m[1]) }; break; }
    }
  }
}

const link = /\]\(([a-z0-9-]+\.md)(#[a-z0-9-]+)?\)/g;
const rewrite = (text: string, cur: Bucket) =>
  text.replace(link, (m, target: string, frag?: string) => {
    const e = reg[target];
    if (!e) return m; // unknown target -> leave; self-check below will catch it
    const anchor = frag ? frag : "#" + e.anchor;
    return e.bucket === cur ? "](" + anchor + ")" : "](" + buckets[cur].other + anchor + ")";
  });

for (const b of Object.keys(buckets) as Bucket[]) {
  const files = listMd(buckets[b].dir);
  const parts = files.map((f) => rewrite(readFileSync(f, "utf-8"), b));
  writeFileSync(buckets[b].out, parts.join("\n\n") + "\n", "utf-8");
  const leftover = parts.join("\n").match(link); // only bare name.md remains if a target is unknown
  if (leftover) throw new Error(`${b}: unrewritten links: ${leftover.slice(0, 5)}`);
  console.log(`built ${b} from ${files.length} files`);
}
