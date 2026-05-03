// tool.js — generic driver for /merge-pdf, /split-pdf, /remove-pages-pdf,
// /remove-empty-pages, /rotate-pdf, /compress-pdf.
//
// Reads the tool config from <script id="tool-config" type="application/json">
// and dispatches on the slug that the server set on <section data-slug>.
// Each tool has its own input shape (single PDF, multi-PDF, page ranges,
// rotation angle…) but they all share: drop zone -> action button ->
// download. The download URL comes back in the API JSON.

(function () {
  const root = document.querySelector(".tool[data-slug]");
  if (!root) return;
  const slug = root.dataset.slug;
  const cfgEl = document.getElementById("tool-config");
  const cfg = cfgEl ? JSON.parse(cfgEl.textContent) : {};
  const ui = (cfg.ui) || {};
  const app = document.getElementById("tool-app");
  if (!app) return;

  // Build a generic upload form per slug.
  const layouts = {
    "merge-pdf":         { multi: true,  extras: [] },
    "split-pdf":         { multi: false, extras: ["splitMode"] },
    "remove-pages-pdf":  { multi: false, extras: ["pages"] },
    "remove-empty-pages":{ multi: false, extras: [] },
    "rotate-pdf":        { multi: false, extras: ["angle", "pages"] },
    "compress-pdf":      { multi: false, extras: [] },
  };
  const layout = layouts[slug];
  if (!layout) {
    app.innerHTML = `<p class="status error">Tool not configured.</p>`;
    return;
  }

  // ----- render -----
  app.innerHTML = `
    <h2>1. Upload</h2>
    <label class="dropzone" id="dz">
      <input id="file" type="file" accept="application/pdf" hidden${layout.multi ? " multiple" : ""}>
      <div class="dz-empty" style="display:flex;flex-direction:column;align-items:center;gap:12px;">
        <span class="dz-ico">
          <svg width="30" height="30" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>
        </span>
        <span class="dz-cta">${esc(ui.uploadCta || "Choose PDF or drag here")}</span>
        <span class="dz-hint">${esc(ui.uploadHint || "PDFs only")}</span>
      </div>
      <ul class="dz-list" id="dzList"></ul>
    </label>
    <p id="uploadStatus" class="status" aria-live="polite"></p>

    ${layout.extras.length ? `<h2>2. Options</h2>` : ""}
    ${renderExtras(layout.extras)}

    <h2>${layout.extras.length ? "3" : "2"}. Run</h2>
    <button id="runBtn" class="btn primary big">${esc(ui.actionBtn || "Process")}
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12h14"/><path d="m12 5 7 7-7 7"/></svg>
    </button>
    <p id="runStatus" class="status" aria-live="polite"></p>

    <div id="result" class="result-card hidden"></div>
  `;

  function renderExtras(list) {
    return list.map((k) => {
      switch (k) {
        case "pages":
          return `<label class="card-input">Pages
            <input id="pages" type="text" placeholder="e.g. 1,3-5,9" autocomplete="off">
            <span class="hint">Comma-separated; ranges allowed.</span>
          </label>`;
        case "angle":
          return `<label class="card-input">Rotation
            <select id="angle">
              <option value="90">90° clockwise</option>
              <option value="180">180°</option>
              <option value="270">90° counter-clockwise</option>
            </select>
          </label>`;
        case "splitMode":
          return `<label class="card-input">Split mode
            <select id="mode">
              <option value="per_page">Every page → its own PDF</option>
              <option value="every_n">Every N pages</option>
              <option value="at_pages">At specific page numbers</option>
            </select>
          </label>
          <label class="card-input" id="modeN-row" style="display:none;">N
            <input id="modeN" type="number" min="1" value="1">
          </label>
          <label class="card-input" id="modeAt-row" style="display:none;">Split before pages
            <input id="modeAt" type="text" placeholder="e.g. 4,9,15">
            <span class="hint">Each number starts a new chunk.</span>
          </label>`;
        default:
          return "";
      }
    }).join("");
  }

  // ----- file selection -----
  const dz = document.getElementById("dz");
  const fileInput = document.getElementById("file");
  const dzList = document.getElementById("dzList");
  let files = [];

  ["dragenter","dragover"].forEach(ev =>
    dz.addEventListener(ev, e => { e.preventDefault(); dz.classList.add("drag"); }));
  ["dragleave","drop"].forEach(ev =>
    dz.addEventListener(ev, e => { e.preventDefault(); dz.classList.remove("drag"); }));
  dz.addEventListener("drop", e => addFiles(e.dataTransfer.files));
  fileInput.addEventListener("change", () => addFiles(fileInput.files));

  function addFiles(list) {
    const arr = Array.from(list);
    if (!layout.multi) {
      files = arr.slice(0, 1);
    } else {
      files = files.concat(arr);
    }
    paintList();
  }

  function paintList() {
    if (files.length === 0) {
      dz.classList.remove("has-file");
      dzList.innerHTML = "";
      return;
    }
    dz.classList.add("has-file");
    dzList.innerHTML = files.map((f, i) => `
      <li>
        <span class="file-ico">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><path d="M14 2v6h6"/></svg>
        </span>
        <span class="file-name">${esc(f.name)}</span>
        <span class="file-size">${fmtBytes(f.size)}</span>
        ${layout.multi ? `<button type="button" class="dz-rm" data-i="${i}" aria-label="Remove">&times;</button>` : ""}
      </li>`).join("");
  }
  dzList.addEventListener("click", e => {
    const b = e.target.closest(".dz-rm");
    if (!b) return;
    e.preventDefault();
    files.splice(Number(b.dataset.i), 1);
    paintList();
  });

  // ----- split mode toggle -----
  const modeSel = document.getElementById("mode");
  if (modeSel) {
    const showN  = () => document.getElementById("modeN-row").style.display = modeSel.value === "every_n" ? "" : "none";
    const showAt = () => document.getElementById("modeAt-row").style.display = modeSel.value === "at_pages" ? "" : "none";
    modeSel.addEventListener("change", () => { showN(); showAt(); });
  }

  // ----- run -----
  document.getElementById("runBtn").addEventListener("click", run);

  async function run() {
    if (files.length === 0) {
      return setStatus("uploadStatus", "Choose a PDF first.", "error");
    }
    if (slug === "merge-pdf" && files.length < 2) {
      return setStatus("uploadStatus", "Merge needs at least 2 PDFs.", "error");
    }
    setStatus("runStatus", `<span class="spinner"></span>Processing… this can take a few seconds.`, "", true);
    const btn = document.getElementById("runBtn");
    btn.disabled = true;
    try {
      const fd = new FormData();
      for (const f of files) fd.append("pdf", f);

      // Per-slug extras.
      if (slug === "remove-pages-pdf") {
        const v = document.getElementById("pages").value.trim();
        if (!v) throw new Error("Specify pages to remove.");
        fd.append("pages", v);
      }
      if (slug === "rotate-pdf") {
        fd.append("angle", document.getElementById("angle").value);
        fd.append("pages", document.getElementById("pages").value.trim());
      }
      if (slug === "split-pdf") {
        const mode = document.getElementById("mode").value;
        fd.append("mode", mode);
        if (mode === "every_n") fd.append("n", document.getElementById("modeN").value);
        if (mode === "at_pages") fd.append("at", document.getElementById("modeAt").value);
      }

      const endpoint = endpoints[slug];
      const r = await fetch(endpoint, { method: "POST", body: fd });
      const j = await r.json();
      if (!r.ok) throw new Error(j.error || "request failed");
      showResult(j);
      setStatus("runStatus", "Done.", "ok");
    } catch (err) {
      setStatus("runStatus", err.message, "error");
    } finally {
      btn.disabled = false;
    }
  }

  const endpoints = {
    "merge-pdf":          "/api/tools/merge",
    "split-pdf":          "/api/tools/split",
    "remove-pages-pdf":   "/api/tools/remove-pages",
    "remove-empty-pages": "/api/tools/remove-empty-pages",
    "rotate-pdf":         "/api/tools/rotate",
    "compress-pdf":       "/api/tools/compress",
  };

  function showResult(j) {
    const el = document.getElementById("result");
    el.classList.remove("hidden");
    let extra = "";
    if (typeof j.removed !== "undefined" && Array.isArray(j.removed)) {
      extra = `<p class="result-detail">Removed ${j.removed.length} blank page${j.removed.length === 1 ? "" : "s"}: ${j.removed.join(", ") || "—"}</p>`;
    }
    if (typeof j.parts !== "undefined") {
      extra = `<p class="result-detail">Split into <strong>${j.parts}</strong> part${j.parts === 1 ? "" : "s"} (zip).</p>`;
    }
    if (typeof j.sourceBytes !== "undefined" && typeof j.outputBytes !== "undefined") {
      const saved = Math.max(0, j.sourceBytes - j.outputBytes);
      const pct = j.sourceBytes ? Math.round(saved * 100 / j.sourceBytes) : 0;
      extra = `<p class="result-detail">Reduced from <strong>${fmtBytes(j.sourceBytes)}</strong> to <strong>${fmtBytes(j.outputBytes)}</strong> — saved ${fmtBytes(saved)} (${pct}%).</p>`;
    }
    el.innerHTML = `
      <div class="thanks-burst" aria-hidden="true">
        <svg width="56" height="56" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>
      </div>
      <h2>${esc(ui.successTitle || "Done")}</h2>
      ${extra}
      <p><a class="btn primary big" href="${j.download}" download="${esc(ui.downloadName || "output.pdf")}">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
        Download
      </a></p>
    `;
    el.scrollIntoView({ behavior: "smooth", block: "start" });
  }

  // ----- helpers -----
  function setStatus(id, msg, kind, html) {
    const el = document.getElementById(id);
    if (!el) return;
    if (html) el.innerHTML = msg; else el.textContent = msg;
    el.className = "status" + (kind ? " " + kind : "");
  }
  function fmtBytes(n) {
    if (!n && n !== 0) return "";
    if (n < 1024) return n + " B";
    if (n < 1024 * 1024) return (n / 1024).toFixed(1) + " KB";
    return (n / 1024 / 1024).toFixed(2) + " MB";
  }
  function esc(s) {
    return String(s == null ? "" : s)
      .replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;").replace(/'/g, "&#39;");
  }
})();
