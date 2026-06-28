#!/usr/bin/env bun
// Static file server for docs/ + an append-only comment endpoint.
// GET: serves files from the docs/ directory (docs + mock.html).
// POST /api/comment: appends {original text, comment, state/type/params} to comments.md.
import { createServer } from "node:http";
import { createReadStream, existsSync, appendFileSync, writeFileSync, statSync } from "node:fs";
import { join, dirname, normalize, extname } from "node:path";
import { fileURLToPath } from "node:url";

const HERE = dirname(fileURLToPath(import.meta.url));
const ROOT = dirname(HERE); // serve docs/
const COMMENTS = join(ROOT, "comments.md");
const HOST = "127.0.0.1";
const PORT = 8800;

const MIME: Record<string, string> = {
  ".html": "text/html", ".css": "text/css", ".js": "text/javascript",
  ".json": "application/json", ".md": "text/markdown; charset=utf-8",
  ".svg": "image/svg+xml", ".png": "image/png", ".jpg": "image/jpeg",
  ".gif": "image/gif", ".ico": "image/x-icon", ".woff2": "font/woff2",
};

if (!existsSync(COMMENTS)) {
  writeFileSync(COMMENTS,
    "# MCPHub Review Comments\n\n" +
    "Append-only log. Each entry: timestamp, surface, page, the original text, and the comment.\n",
    "utf-8");
}

const clip = (v: unknown, n: number) => String(v ?? "").slice(0, n);

const server = createServer((req, res) => {
  if (req.method === "POST") {
    if (req.url !== "/api/comment") { res.writeHead(404).end(); return; }
    let raw = "";
    req.on("data", (c) => { raw += c; if (raw.length > 1 << 20) req.destroy(); });
    req.on("end", () => {
      let data: any;
      try { data = JSON.parse(raw || "{}"); }
      catch { res.writeHead(400).end("bad json"); return; }

      const surface = clip(data.surface ?? "?", 40);
      const page = clip(data.page, 300);
      const ctx = clip(data.context, 200);
      const original = clip(data.original, 5000);
      const comment = clip(data.comment, 5000).trim();
      const ctype = clip(data.type, 40);
      const params = clip(data.params, 300);
      if (!comment) { res.writeHead(400).end("empty comment"); return; }

      const ts = localIso();
      const quoted = original.split("\n").map((ln) => "> " + ln).join("\n") || "> (no text)";
      const loc = [surface, ctx, page].filter(Boolean).join(" · ");
      const meta = `- **state:** open\n- **type:** ${ctype}\n- **params:** ${params}\n`;
      const entry = `\n## ${ts} — ${loc}\n\n${meta}\n**Original:**\n\n${quoted}\n\n**Comment:** ${comment}\n\n---\n`;
      appendFileSync(COMMENTS, entry, "utf-8");

      const body = JSON.stringify({ ok: true });
      res.writeHead(200, { "Content-Type": "application/json", "Content-Length": Buffer.byteLength(body) });
      res.end(body);
    });
    return;
  }

  // GET: serve a file from ROOT, no path traversal.
  let pathname = decodeURIComponent((req.url || "/").split("?")[0]);
  if (pathname.endsWith("/")) pathname += "index.html";
  const full = normalize(join(ROOT, pathname));
  if (!full.startsWith(ROOT) || !existsSync(full) || !statSync(full).isFile()) {
    res.writeHead(404).end("not found"); return;
  }
  res.writeHead(200, { "Content-Type": MIME[extname(full)] || "application/octet-stream" });
  createReadStream(full).pipe(res);
});

function localIso(): string {
  // ISO-8601 with local offset, seconds precision (matches Python isoformat).
  const d = new Date();
  const p = (n: number) => String(n).padStart(2, "0");
  const off = -d.getTimezoneOffset();
  const sign = off >= 0 ? "+" : "-";
  const oa = Math.abs(off);
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}` +
    `T${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}` +
    `${sign}${p(Math.floor(oa / 60))}:${p(oa % 60)}`;
}

server.listen(PORT, HOST, () => {
  console.log(`serving ${ROOT} on http://${HOST}:${PORT}  (POST /api/comment -> comments.md)`);
});
