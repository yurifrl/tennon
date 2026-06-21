#!/usr/bin/env python3
"""Static file server for docs/ + an append-only comment endpoint.
GET: serves files from this directory (docs + mockup.html).
POST /api/comment: appends {original text, comment} to comments.md."""
import json, os, datetime
from functools import partial
from http.server import SimpleHTTPRequestHandler, ThreadingHTTPServer

ROOT = os.path.dirname(os.path.abspath(__file__))
COMMENTS = os.path.join(ROOT, "comments.md")
HOST, PORT = "127.0.0.1", 8800

if not os.path.exists(COMMENTS):
    with open(COMMENTS, "w", encoding="utf-8") as f:
        f.write("# MCPHub Review Comments\n\n"
                "Append-only log. Each entry: timestamp, surface, page, the original text, and the comment.\n")


class Handler(SimpleHTTPRequestHandler):
    def do_POST(self):
        if self.path != "/api/comment":
            self.send_error(404); return
        try:
            n = int(self.headers.get("Content-Length", 0))
            data = json.loads(self.rfile.read(n) or b"{}")
        except Exception:
            self.send_error(400, "bad json"); return

        surface = str(data.get("surface", "?"))[:40]
        page = str(data.get("page", ""))[:300]
        ctx = str(data.get("context", ""))[:200]
        original = str(data.get("original", ""))[:5000]
        comment = str(data.get("comment", "")).strip()[:5000]
        if not comment:
            self.send_error(400, "empty comment"); return

        ts = datetime.datetime.now().astimezone().isoformat(timespec="seconds")
        quoted = "\n".join("> " + ln for ln in original.splitlines()) or "> (no text)"
        loc = " · ".join(p for p in [surface, ctx, page] if p)
        entry = f"\n## {ts} — {loc}\n\n**Original:**\n\n{quoted}\n\n**Comment:** {comment}\n\n---\n"
        with open(COMMENTS, "a", encoding="utf-8") as f:
            f.write(entry)

        body = json.dumps({"ok": True}).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, *a):  # quiet
        pass


if __name__ == "__main__":
    httpd = ThreadingHTTPServer((HOST, PORT), partial(Handler, directory=ROOT))
    print(f"serving {ROOT} on http://{HOST}:{PORT}  (POST /api/comment -> comments.md)")
    httpd.serve_forever()
