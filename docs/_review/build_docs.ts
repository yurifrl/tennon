#!/usr/bin/env bun
// Build two scrolling pages: docs/README.md (user guide, served at /) and
// docs/arch/README.md (design & internals, served at /arch). Each concatenates its
// own NN-*.md files (filename order). Cross-doc links are rewritten: same-bucket
// links become in-page anchors; cross-bucket links point at the other page + anchor.
// README.md is docsify's folder-index name, so /arch/ routes to it natively
// (a _all.md page can't be linked from the sidebar — docsify ignores _-prefixed files).
//
// Sources write anchors the GitHub way (`crds.md#policy`, `#4-connect-an-agent`). Docsify
// derives heading ids differently (a leading digit gets `_`, runs of `-` collapse, and a
// heading repeated anywhere on the concatenated page gets a `-N` suffix), so every anchor
// is resolved against the target file's headings and emitted as the id docsify will
// generate. The build fails on any link whose file or heading does not exist.
import { existsSync, readdirSync, readFileSync, writeFileSync } from "node:fs";
import { join, dirname, basename, relative } from "node:path";
import { fileURLToPath } from "node:url";

const DOCS = dirname(dirname(fileURLToPath(import.meta.url)));

const listMd = (dir: string) =>
  readdirSync(dir).filter((f) => /^\d.*\.md$/.test(f)).sort().map((f) => join(dir, f));

const buckets = {
  user: { dir: DOCS, out: join(DOCS, "README.md"), other: "arch/README.md" },
  arch: { dir: join(DOCS, "arch"), out: join(DOCS, "arch", "README.md"), other: "../README.md" },
} as const;
type Bucket = keyof typeof buckets;

// Heading text with inline markdown removed: `code`, **bold**, [text](url) -> text.
const plain = (md: string) =>
  md.replace(/\[([^\]]*)\]\([^)]*\)/g, "$1").replace(/[`*]/g, "").trim();

// The anchor GitHub generates for a heading (what the sources link to).
const githubSlug = (text: string) =>
  plain(text).toLowerCase().replace(/[^\p{L}\p{N}_\- ]/gu, "").replace(/ /g, "-");

// The id docsify@4 generates (lib/render/slugify.js), given the heading as marked renders it.
const DOCSIFY_STRIP = /[\u2000-\u206F\u2E00-\u2E7F\\'!"#$%&()*+,./:;<=>?@[\]^`{|}~]/g;
const docsifySlug = (text: string) =>
  plain(text)
    .replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;").replace(/'/g, "&#39;")
    .toLowerCase().replace(DOCSIFY_STRIP, "").replace(/\s/g, "-").replace(/-+/g, "-")
    .replace(/^(\d)/, "_$1");

// Both generators suffix a repeated slug with -1, -2, ...
const dedupe = (seen: Map<string, number>, slug: string) => {
  const n = seen.has(slug) ? seen.get(slug)! + 1 : 0;
  seen.set(slug, n);
  return n > 0 ? `${slug}-${n}` : slug;
};

// Split a file into lines tagged with whether they sit inside a ``` fence.
const lines = (text: string) => {
  let fenced = false;
  return text.split("\n").map((line) => {
    const isFence = /^\s*(```|~~~)/.test(line);
    const inCode = fenced || isFence;
    if (isFence) fenced = !fenced;
    return { line, inCode };
  });
};

type Doc = {
  path: string; bucket: Bucket; text: string;
  first: string;                 // docsify id of the file's first heading
  anchors: Map<string, string>;  // github slug -> docsify id on the built page
};

// unnumbered basename (the name used in cross-doc links) -> Doc
const reg: Record<string, Doc> = {};
const docs: Doc[] = [];
for (const b of Object.keys(buckets) as Bucket[]) {
  const pageIds = new Map<string, number>(); // docsify dedupes across the whole built page
  for (const path of listMd(buckets[b].dir)) {
    const text = readFileSync(path, "utf-8");
    const fileIds = new Map<string, number>(); // github dedupes within one file
    const doc: Doc = { path, bucket: b, text, first: "", anchors: new Map() };
    for (const { line, inCode } of lines(text)) {
      const m = !inCode && line.match(/^#{1,6}\s+(.*?)\s*#*\s*$/);
      if (!m) continue;
      const id = dedupe(pageIds, docsifySlug(m[1]));
      doc.anchors.set(dedupe(fileIds, githubSlug(m[1])), id);
      doc.first ||= id;
    }
    const name = basename(path).replace(/^\d+-/, "");
    if (reg[name]) throw new Error(`duplicate doc name ${name}: ${reg[name].path} and ${path}`);
    reg[name] = doc;
    docs.push(doc);
  }
}

const errors: string[] = [];
const link = /\]\(([^)\s]+)\)/g;

const rewrite = (doc: Doc) =>
  lines(doc.text).map(({ line, inCode }, i) => {
    if (inCode) return line;
    return line.replace(link, (m, href: string) => {
      const bad = (why: string) => {
        errors.push(`${relative(DOCS, doc.path)}:${i + 1}: ${m.slice(1)} — ${why}`);
        return m;
      };
      if (/^[a-z][a-z0-9+.-]*:/i.test(href)) return m; // http:, mailto:, ...
      const [target, frag] = href.split("#") as [string, string | undefined];

      if (!target) { // same-file anchor
        const id = doc.anchors.get(frag ?? "");
        return id ? `](#${id})` : bad("no such heading in this file");
      }
      const to = reg[target];
      if (to) { // cross-doc link by unnumbered name
        const id = frag === undefined ? to.first : to.anchors.get(frag);
        if (!id) return bad(`no such heading in ${relative(DOCS, to.path)}`);
        return to.bucket === doc.bucket ? `](#${id})` : `](${buckets[doc.bucket].other}#${id})`;
      }
      if (/^[a-z0-9-]+\.md$/.test(target)) return bad("no such doc");
      // anything else is a plain relative file (e.g. arch/README.md, mock.html)
      return existsSync(join(dirname(doc.path), target)) ? m : bad("file not found");
    });
  }).join("\n");

for (const b of Object.keys(buckets) as Bucket[]) {
  const files = docs.filter((d) => d.bucket === b);
  writeFileSync(buckets[b].out, files.map(rewrite).join("\n\n") + "\n", "utf-8");
  console.log(`built ${b} from ${files.length} files`);
}

if (errors.length) {
  console.error(`\n${errors.length} broken link(s):\n` + errors.map((e) => "  " + e).join("\n"));
  process.exit(1);
}
