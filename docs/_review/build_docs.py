#!/usr/bin/env python3
"""Concatenate docs/[0-9]*.md (filename order) into docs/_all.md as one scrolling
page, rewriting cross-doc links to in-page anchors so navigation stays on the page."""
import re, glob, os

DOCS = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))


def slug(text):
    return re.sub(r"[^\w\- ]", "", text).strip().lower().replace(" ", "-")


files = sorted(glob.glob(os.path.join(DOCS, "[0-9]*.md")))

# basename (old name, no number prefix) -> anchor of its first heading
first_anchor = {}
for f in files:
    name = re.sub(r"^\d+-", "", os.path.basename(f))
    with open(f, encoding="utf-8") as fh:
        for line in fh:
            m = re.match(r"#+\s+(.*)", line)
            if m:
                first_anchor[name] = slug(m.group(1))
                break

link = re.compile(r"\]\(([a-z0-9-]+\.md)(#[a-z0-9-]+)?\)")


def fix(m):
    target, frag = m.group(1), m.group(2)
    return "](" + (frag if frag else "#" + first_anchor.get(target, "")) + ")"


parts = []
for f in files:
    with open(f, encoding="utf-8") as fh:
        parts.append(link.sub(fix, fh.read()))

with open(os.path.join(DOCS, "_all.md"), "w", encoding="utf-8") as out:
    out.write("\n\n".join(parts) + "\n")

# self-check: every cross-doc link became an anchor
leftover = link.findall("\n".join(parts))
assert not leftover, f"unrewritten links: {leftover[:5]}"
print(f"built _all.md from {len(files)} files")
