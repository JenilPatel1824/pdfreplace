// PDF Text Replace — editor flow
//   upload -> find (returns bboxes) -> select pages -> replace -> download

const $ = (sel) => document.querySelector(sel);

const stepEls = document.querySelectorAll(".stepper li");
function setStep(n) {
  stepEls.forEach((el) => {
    const k = +el.dataset.step;
    el.classList.toggle("active", k === n);
    el.classList.toggle("done", k < n);
  });
}

const state = { id: null, pages: 0, matches: [], selected: new Set() };
setStep(1);

// ---------- Step 1 — upload ----------
const dz = $("#dropzone");
const pdfInput = $("#pdfInput");
const uploadStatus = $("#uploadStatus");
const fileNameEl = $("#fileName");
const fileSizeEl = $("#fileSize");
const fileClearBtn = $("#fileClear");

["dragenter", "dragover"].forEach((ev) =>
  dz.addEventListener(ev, (e) => { e.preventDefault(); dz.classList.add("drag"); })
);
["dragleave", "drop"].forEach((ev) =>
  dz.addEventListener(ev, (e) => { e.preventDefault(); dz.classList.remove("drag"); })
);
dz.addEventListener("drop", (e) => {
  if (e.dataTransfer.files[0]) handleFile(e.dataTransfer.files[0]);
});
pdfInput.addEventListener("change", () => pdfInput.files[0] && handleFile(pdfInput.files[0]));

fileClearBtn.addEventListener("click", (e) => {
  e.preventDefault();
  pdfInput.value = "";
  dz.classList.remove("has-file");
  state.id = null; state.pages = 0;
  $("#step-find").classList.add("hidden");
  $("#step-pages").classList.add("hidden");
  $("#step-download").classList.add("hidden");
  setStatus(uploadStatus, "");
  setStep(1);
});

function fmtBytes(n) {
  if (n < 1024) return n + " B";
  if (n < 1024 * 1024) return (n / 1024).toFixed(1) + " KB";
  return (n / 1024 / 1024).toFixed(2) + " MB";
}

async function handleFile(file) {
  if (!file.name.toLowerCase().endsWith(".pdf")) {
    return setStatus(uploadStatus, "Only PDF files are supported.", "error");
  }
  if (file.size > 25 * 1024 * 1024) {
    return setStatus(uploadStatus, "File too large (25 MB max).", "error");
  }
  fileNameEl.textContent = file.name;
  fileSizeEl.textContent = fmtBytes(file.size);
  dz.classList.add("has-file");
  setStatus(uploadStatus, '<span class="spinner"></span>Uploading…', "", true);

  const fd = new FormData();
  fd.append("pdf", file);
  try {
    const r = await fetch("/api/upload", { method: "POST", body: fd });
    const j = await r.json();
    if (!r.ok) throw new Error(j.error || "upload failed");
    state.id = j.id;
    state.pages = j.pages;
    setStatus(uploadStatus, `Uploaded — ${j.pages} pages.`, "ok");
    $("#step-find").classList.remove("hidden");
    setStep(2);
    $("#oldText").focus();
    $("#step-find").scrollIntoView({ behavior: "smooth", block: "start" });
  } catch (err) {
    setStatus(uploadStatus, err.message, "error");
    dz.classList.remove("has-file");
  }
}

// ---------- Step 2 — find ----------
$("#findBtn").addEventListener("click", findText);
[ $("#oldText"), $("#newText") ].forEach((el) => {
  el.addEventListener("keydown", (e) => { if (e.key === "Enter") findText(); });
  el.addEventListener("input", updateDiffWarning);
});

// Live char-count diff warning. Severity:
//   ok        — replacement fits comfortably
//   warn      — moderately longer (1.3×–2× length OR +5 chars), font will shrink
//   danger    — much longer (>2× length OR +10 chars), result almost certainly looks bad
function updateDiffWarning() {
  const o = $("#oldText").value.trim();
  const n = $("#newText").value;
  const el = $("#diffWarn");
  if (!o || !n) { setStatus(el, ""); return; }

  const oLen = o.length;
  const nLen = n.length;
  const ratio = nLen / Math.max(oLen, 1);
  const diff = nLen - oLen;

  if (nLen <= oLen) { setStatus(el, ""); return; }

  if (ratio >= 2 || diff >= 10) {
    setStatus(el,
      `Replacement is ${nLen} characters but the original is only ${oLen}. ` +
      `That's ${diff} extra characters — the new text will be shrunk hard to fit, or it will overflow into adjacent text. ` +
      `Try a shorter replacement, edit the surrounding text separately, or use a different tool.`,
      "danger");
    return;
  }
  if (ratio >= 1.3 || diff >= 5) {
    setStatus(el,
      `Replacement is ${diff} characters longer than the original. The font will be shrunk slightly so it fits — readable, but visibly smaller than the surrounding text.`,
      "warn");
    return;
  }
  setStatus(el, "");
}

async function findText() {
  const oldText = $("#oldText").value.trim();
  const newText = $("#newText").value;
  if (!state.id) return setStatus($("#findStatus"), "Upload a PDF first.", "error");
  if (!oldText) return setStatus($("#findStatus"), "Enter the old text.", "error");

  const btn = $("#findBtn");
  btn.disabled = true;
  setStatus($("#findStatus"), '<span class="spinner"></span>Scanning every page…', "", true);
  try {
    const r = await fetch("/api/find", {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({ id: state.id, old: oldText }),
    });
    const j = await r.json();
    if (!r.ok) throw new Error(j.error || "find failed");
    state.matches = j.matches || [];
    if (state.matches.length === 0) {
      return setStatus($("#findStatus"),
        "No matches. Check capitalisation and spacing. If the PDF is a scan, the text isn't selectable.",
        "warn");
    }
    const pagesWithHits = new Set(state.matches.map((m) => m.page));
    const baseMsg = `${state.matches.length} matches across ${pagesWithHits.size} page${pagesWithHits.size === 1 ? "" : "s"}.`;
    setStatus($("#findStatus"), baseMsg, "ok");
    updateDiffWarning(); // re-render the persistent length warning
    await renderPagesGrid();
    $("#step-pages").classList.remove("hidden");
    setStep(3);
    $("#step-pages").scrollIntoView({ behavior: "smooth", block: "start" });
  } catch (err) {
    setStatus($("#findStatus"), err.message, "error");
  } finally {
    btn.disabled = false;
  }
}

// ---------- Step 3 — page grid ----------
async function renderPagesGrid() {
  const grid = $("#pagesGrid");
  grid.innerHTML = "";
  state.selected = new Set();
  const matchesByPage = {};
  state.matches.forEach((m) => (matchesByPage[m.page] ||= []).push(m));

  const pagesWithHits = Object.keys(matchesByPage).map(Number).sort((a, b) => a - b);

  for (const page of pagesWithHits) {
    const card = document.createElement("div");
    card.className = "page-card selected";
    card.dataset.page = page;
    state.selected.add(page);

    const wrap = document.createElement("div");
    wrap.className = "pg-wrap";

    const img = document.createElement("img");
    img.alt = `Page ${page} preview`;
    img.loading = "lazy";
    img.src = `/api/page-image/${state.id}/${page}`;
    wrap.appendChild(img);

    const check = document.createElement("div");
    check.className = "pg-check";
    wrap.appendChild(check);

    card.appendChild(wrap);

    img.addEventListener("load", () => drawHighlights(wrap, img, matchesByPage[page]));

    const lbl = document.createElement("div");
    lbl.className = "pg-label";
    lbl.innerHTML = `<span>Page ${page}</span><span class="pg-hits">${matchesByPage[page].length}×</span>`;
    card.appendChild(lbl);

    card.addEventListener("click", () => {
      const sel = card.classList.toggle("selected");
      if (sel) state.selected.add(page); else state.selected.delete(page);
    });
    grid.appendChild(card);
  }
}

function drawHighlights(wrap, img, matches) {
  // pdftotext bboxes are in PDF points; image is rendered at 110 DPI.
  const scale = img.clientWidth / matches[0].pageW;
  matches.forEach((m) => {
    const hl = document.createElement("div");
    hl.className = "hl";
    hl.style.left = (m.x0 * scale) + "px";
    hl.style.top = (m.y0 * scale) + "px";
    hl.style.width = ((m.x1 - m.x0) * scale) + "px";
    hl.style.height = ((m.y1 - m.y0) * scale) + "px";
    wrap.appendChild(hl);
  });
}

$("#selAll").addEventListener("click", () => {
  document.querySelectorAll(".page-card").forEach((c) => {
    c.classList.add("selected");
    state.selected.add(+c.dataset.page);
  });
});
$("#selNone").addEventListener("click", () => {
  document.querySelectorAll(".page-card").forEach((c) => c.classList.remove("selected"));
  state.selected.clear();
});

// ---------- Step 4 — replace + download ----------
$("#replaceBtn").addEventListener("click", async () => {
  if (state.selected.size === 0) {
    return setStatus($("#replaceStatus"), "Select at least one page.", "error");
  }
  const btn = $("#replaceBtn");
  btn.disabled = true;
  setStatus($("#replaceStatus"), '<span class="spinner"></span>Replacing… this can take a few seconds.', "", true);
  const mode = (document.querySelector('input[name="mode"]:checked') || {}).value || "vector";
  const bold = $("#boldToggle").checked;
  const italic = $("#italicToggle").checked;
  try {
    const r = await fetch("/api/replace", {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({
        id: state.id,
        old: $("#oldText").value.trim(),
        new: $("#newText").value,
        pages: [...state.selected].sort((a, b) => a - b),
        mode,
        bold,
        italic,
      }),
    });
    const j = await r.json();
    if (!r.ok) throw new Error(j.error || "replace failed");
    $("#downloadLink").href = j.download;
    $("#step-download").classList.remove("hidden");
    setStep(4);
    $("#step-download").scrollIntoView({ behavior: "smooth", block: "start" });
    setStatus($("#replaceStatus"), "Done.", "ok");
  } catch (err) {
    setStatus($("#replaceStatus"), err.message, "error");
  } finally {
    btn.disabled = false;
  }
});

function setStatus(el, msg, kind, html) {
  if (html) el.innerHTML = msg;
  else el.textContent = msg;
  el.className = "status" + (kind ? " " + kind : "");
}

// ---------- Thank-you screen ----------
$("#downloadLink").addEventListener("click", () => {
  // Wait a tick so the browser actually starts the download before we
  // swap the UI; otherwise some browsers cancel the navigation.
  setTimeout(() => {
    const dlHref = $("#downloadLink").getAttribute("href");
    const thanks = $("#step-thanks");
    $("#downloadAgain").setAttribute("href", dlHref);
    $("#step-download").classList.add("hidden");
    thanks.classList.remove("hidden");

    const shareUrl = encodeURIComponent(window.location.origin + "/");
    const shareText = encodeURIComponent(
      "Free PDF text replace — find & replace text in any PDF, no signup."
    );
    $("#shareTwitter").href =
      `https://twitter.com/intent/tweet?text=${shareText}&url=${shareUrl}`;
    $("#shareFacebook").href =
      `https://www.facebook.com/sharer/sharer.php?u=${shareUrl}`;
    $("#shareLinkedIn").href =
      `https://www.linkedin.com/sharing/share-offsite/?url=${shareUrl}`;

    thanks.scrollIntoView({ behavior: "smooth", block: "start" });
  }, 120);
});

$("#shareCopy").addEventListener("click", async (e) => {
  e.preventDefault();
  try {
    await navigator.clipboard.writeText(window.location.origin + "/");
    e.target.textContent = "Link copied ✓";
    setTimeout(() => (e.target.textContent = "Copy link"), 2000);
  } catch {
    e.target.textContent = "Couldn't copy";
  }
});

$("#restartFlow").addEventListener("click", () => {
  pdfInput.value = "";
  state.id = null;
  state.pages = 0;
  state.matches = [];
  state.selected.clear();
  $("#oldText").value = "";
  $("#newText").value = "";
  $("#pagesGrid").innerHTML = "";
  dz.classList.remove("has-file");
  for (const id of ["#step-find", "#step-pages", "#step-download", "#step-thanks"]) {
    $(id).classList.add("hidden");
  }
  setStatus(uploadStatus, "");
  setStatus($("#findStatus"), "");
  setStatus($("#replaceStatus"), "");
  setStep(1);
  $("#step-upload").scrollIntoView({ behavior: "smooth", block: "start" });
});
