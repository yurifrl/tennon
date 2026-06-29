/* Shared annotation widget for the Tennon docs + mockup.
   Adds a "+ Comment" button. In comment mode, hovering highlights a text block;
   click it (or select text first) to attach a comment. Comments POST to /api/comment
   and are appended to comments.md with the original text. */
(function () {
  if (window.__annot) return; window.__annot = true;
  const SURFACE = /Docs/i.test(document.title) ? 'docs' : 'mock';
  const IGNORE = '#annot-bar,#annot-pop,#sidebar,.sidebar,header,.app-name,form,select,input,textarea,.search';
  const BLOCK = 'p,li,h1,h2,h3,h4,h5,h6,td,th,pre,blockquote';
  const INLINE = 'a,button,b,span,div';

  const style = document.createElement('style');
  style.textContent = `
  #annot-bar{position:fixed;top:10px;right:14px;z-index:100000}
  #annot-bar button{font:600 13px/1 -apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;background:#1b212d;color:#e6e9ef;border:1px solid #28303f;border-radius:999px;padding:9px 15px;cursor:pointer;box-shadow:0 4px 16px rgba(0,0,0,.45)}
  #annot-bar button.on{background:#00c29f;color:#04121a;border-color:transparent}
  body.annot-on{cursor:crosshair}
  body.annot-on .annot-hover{outline:2px solid #00c29f !important;outline-offset:2px;background:rgba(0,194,159,.08) !important;border-radius:4px}
  #annot-pop{position:fixed;z-index:100001;width:320px;max-width:92vw;background:#141821;border:1px solid #28303f;border-radius:12px;box-shadow:0 20px 60px rgba(0,0,0,.6);padding:14px;font:14px/1.5 -apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;color:#e6e9ef}
  #annot-pop .lbl{font-size:11px;text-transform:uppercase;letter-spacing:.6px;color:#6b7587;margin-bottom:4px}
  #annot-pop .q{font-size:12px;color:#9aa4b5;background:#0d0f14;border:1px solid #28303f;border-radius:8px;padding:8px;max-height:88px;overflow:auto;margin-bottom:10px;white-space:pre-wrap}
  #annot-pop textarea{width:100%;box-sizing:border-box;background:#0d0f14;border:1px solid #28303f;border-radius:8px;color:#e6e9ef;padding:8px;font:14px -apple-system,sans-serif;min-height:72px;outline:none;resize:vertical}
  #annot-pop textarea:focus{border-color:#00c29f}
  #annot-pop .row{display:flex;gap:8px;justify-content:flex-end;margin-top:10px}
  #annot-pop button{font:600 13px -apple-system,sans-serif;border-radius:8px;padding:7px 13px;cursor:pointer;border:1px solid #28303f;background:#1b212d;color:#e6e9ef}
  #annot-pop button.save{background:#00c29f;color:#04121a;border-color:transparent}
  #annot-pop .hint{font-size:11px;color:#6b7587;margin-top:6px;text-align:right}
  #annot-toast{position:fixed;bottom:18px;left:50%;transform:translateX(-50%);z-index:100002;background:#00c29f;color:#04121a;font:600 13px -apple-system,sans-serif;padding:9px 16px;border-radius:999px;opacity:0;transition:opacity .2s;pointer-events:none}
  #annot-toast.show{opacity:1}
  `;
  document.head.appendChild(style);

  const bar = document.createElement('div'); bar.id = 'annot-bar';
  const btn = document.createElement('button');
  btn.textContent = '+ Comment';
  btn.title = 'Comment mode: click, then select text or click a block to comment';
  bar.appendChild(btn); document.body.appendChild(bar);
  const toast = document.createElement('div'); toast.id = 'annot-toast'; document.body.appendChild(toast);
  const showToast = m => { toast.textContent = m; toast.classList.add('show'); setTimeout(() => toast.classList.remove('show'), 1800); };

  let mode = false, hoverEl = null, pop = null;
  btn.onclick = () => {
    mode = !mode;
    document.body.classList.toggle('annot-on', mode);
    btn.classList.toggle('on', mode);
    btn.textContent = mode ? '× Done' : '+ Comment';
    if (!mode) { clearHover(); closePop(); }
  };

  function eligible(t) {
    if (!t || (t.closest && t.closest(IGNORE))) return null;
    return (t.closest && (t.closest(BLOCK) || t.closest(INLINE))) || null;
  }
  function clearHover() { if (hoverEl) { hoverEl.classList.remove('annot-hover'); hoverEl = null; } }

  document.addEventListener('mouseover', e => {
    if (!mode || pop) return;
    const el = eligible(e.target);
    if (el !== hoverEl) { clearHover(); if (el) { hoverEl = el; el.classList.add('annot-hover'); } }
  }, true);

  document.addEventListener('click', e => {
    if (!mode) return;
    if (e.target.closest('#annot-bar,#annot-pop')) return;
    const el = eligible(e.target);
    if (!el) return;
    e.preventDefault(); e.stopPropagation();
    const sel = (window.getSelection && window.getSelection().toString().trim()) || '';
    const original = (sel || el.innerText || el.textContent || '').trim().replace(/\s+\n/g, '\n').slice(0, 2000);
    openPop(e.clientX, e.clientY, original);
  }, true);

  function context() {
    const t = document.getElementById('pgTitle');
    if (t && t.textContent.trim()) return t.textContent.trim();
    const h = document.querySelector('.markdown-section h1, .markdown-section h2');
    if (h && h.textContent.trim()) return h.textContent.trim();
    return document.title;
  }

  function openPop(x, y, original) {
    closePop();
    pop = document.createElement('div'); pop.id = 'annot-pop';
    pop.innerHTML = '<div class="lbl">Original text</div><div class="q"></div>' +
      '<textarea placeholder="Your comment…"></textarea>' +
      '<div class="row"><button class="cancel">Cancel</button><button class="save">Save comment</button></div>' +
      '<div class="hint">⌘/Ctrl + Enter to save · Esc to cancel</div>';
    pop.querySelector('.q').textContent = original || '(no text)';
    document.body.appendChild(pop);
    const w = pop.offsetWidth, h = pop.offsetHeight;
    pop.style.left = Math.max(8, Math.min(x, innerWidth - w - 12)) + 'px';
    pop.style.top = Math.max(8, Math.min(y + 8, innerHeight - h - 12)) + 'px';
    const ta = pop.querySelector('textarea'); ta.focus();
    pop.querySelector('.cancel').onclick = closePop;
    pop.querySelector('.save').onclick = () => save(original, ta.value);
    ta.addEventListener('keydown', ev => {
      if ((ev.metaKey || ev.ctrlKey) && ev.key === 'Enter') save(original, ta.value);
      else if (ev.key === 'Escape') closePop();
    });
  }
  function closePop() { if (pop) { pop.remove(); pop = null; } clearHover(); }

  function save(original, comment) {
    comment = (comment || '').trim();
    if (!comment) { showToast('Comment is empty'); return; }
    fetch('/api/comment', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ surface: SURFACE, page: location.pathname + location.hash, context: context(), original, comment })
    }).then(r => { if (!r.ok) throw 0; showToast('Comment saved ✓'); })
      .catch(() => showToast('Save failed'))
      .finally(closePop);
  }
})();
