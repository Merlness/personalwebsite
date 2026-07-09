// UI shell for the organizer PWA. All parsing, formatting, scheduling, and
// API logic lives in the tested modules; this file only wires DOM to them.

import { parseTasks, addTask, completeTask, updateTask, deleteTask, effectivePriority } from "./tasks-model.js";
import { buildEntry, insertUnderUnprocessed, insertAllUnderUnprocessed, parseUnprocessed, markProcessed } from "./capture-entry.js";
import { extractPhoneCard, weekAhead } from "./workout-view.js";
import { parsePlanItems, buildWorkoutLog } from "./workout-log.js";
import { classifyCapture, draftLinkedInPost } from "./gemini.js";
import { makeCardId, isoLocal, buildCardNote } from "./cards-model.js";
import { GitHubClient } from "./github-api.js";

const FILES = {
  tasks: "tasks.md",
  inbox: "inbox.md",
  workoutCapture: "pulse/workout-capture.md",
  todayWorkout: "pulse/today-workout.md",
  ledger: "pulse/workout-ledger.md",
  program: "pulse/workout-program.md",
};
const CARDS_DIR = "drafts/cards/inbox";
const LS = { settings: "capture.settings", vault: "capture.vault", queue: "capture.queue", cache: "capture.filecache" };
const SECTIONS = ["Business", "Personal", "Financial"];

let token = null;
let geminiKey = null;
let gh = null;
let dest = "inbox";
// what the task sheet is doing: {mode:'add'|'edit'|'promote-inbox'|'promote-review', ...}
let sheetCtx = { mode: "add" };

const $ = (id) => document.getElementById(id);
const todayISO = () => {
  const d = new Date(), p = (n) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`;
};

// ---------- settings & vault ----------
function getSettings() {
  try { return JSON.parse(localStorage.getItem(LS.settings)) || {}; } catch { return {}; }
}
function setSettings(s) { localStorage.setItem(LS.settings, JSON.stringify(s)); }

const enc = new TextEncoder(), dec = new TextDecoder();
const b64 = (buf) => btoa(String.fromCharCode(...new Uint8Array(buf)));
const unb64 = (s) => Uint8Array.from(atob(s), (c) => c.charCodeAt(0));

async function deriveKey(pin, salt) {
  const base = await crypto.subtle.importKey("raw", enc.encode(pin), "PBKDF2", false, ["deriveKey"]);
  return crypto.subtle.deriveKey({ name: "PBKDF2", salt, iterations: 310000, hash: "SHA-256" }, base, { name: "AES-GCM", length: 256 }, false, ["encrypt", "decrypt"]);
}
async function storeKeys(pin, keys) {
  const salt = crypto.getRandomValues(new Uint8Array(16));
  const iv = crypto.getRandomValues(new Uint8Array(12));
  const key = await deriveKey(pin, salt);
  const ct = await crypto.subtle.encrypt({ name: "AES-GCM", iv }, key, enc.encode(JSON.stringify(keys)));
  localStorage.setItem(LS.vault, JSON.stringify({ salt: b64(salt), iv: b64(iv), ct: b64(ct) }));
}
// vault v1 stored a bare GitHub token string; v2 stores JSON {gh, gemini}
async function unlockKeys(pin) {
  const v = JSON.parse(localStorage.getItem(LS.vault));
  const key = await deriveKey(pin, unb64(v.salt));
  const plain = dec.decode(await crypto.subtle.decrypt({ name: "AES-GCM", iv: unb64(v.iv) }, key, unb64(v.ct)));
  try {
    const parsed = JSON.parse(plain);
    if (parsed && typeof parsed === "object" && parsed.gh) return parsed;
  } catch {}
  return { gh: plain, gemini: null };
}

function makeClient() {
  const { owner, repo, branch, apiBase } = getSettings();
  gh = new GitHubClient({ owner, repo, branch: branch || "main", apiBase: apiBase || "", token });
}

// ---------- offline file cache (read-only fallback) ----------
function cacheGet() {
  try { return JSON.parse(localStorage.getItem(LS.cache)) || {}; } catch { return {}; }
}
function cachePut(path, content) {
  const c = cacheGet();
  c[path] = { content, at: Date.now() };
  localStorage.setItem(LS.cache, JSON.stringify(c));
}
// For a short window after we write a file, trust our own written copy:
// the Contents API can serve stale reads and visually undo the action.
const recentWrites = {};
function noteWritten(path, content) {
  recentWrites[path] = { content, at: Date.now() };
  cachePut(path, content);
}

async function readFile(path) {
  const w = recentWrites[path];
  if (w && Date.now() - w.at < 60000) return { content: w.content, offline: false };
  try {
    const f = await gh.getFile(path);
    cachePut(path, f.content);
    return { content: f.content, offline: false };
  } catch (e) {
    const c = cacheGet()[path];
    if (c) return { content: c.content, offline: true };
    throw e;
  }
}

// ---------- status ----------
let statusTimer;
function setStatus(msg, cls = "") {
  clearTimeout(statusTimer);
  const el = $("status");
  el.textContent = msg;
  el.className = "status " + cls;
  if (msg) statusTimer = setTimeout(() => { el.textContent = ""; el.className = "status"; }, 4000);
}

// ---------- tabs ----------
const TABS = ["tasks", "workout", "linkedin", "cards", "capture"];
function showTab(name) {
  for (const t of TABS) {
    $(`tab-${t}`).classList.toggle("hidden", t !== name);
    $(`nav-${t}`).classList.toggle("active", t === name);
  }
  $("fab").classList.toggle("hidden", name !== "tasks");
  if (name === "tasks") loadTasks();
  if (name === "workout") loadWorkout();
  if (name === "linkedin") loadDrafts();
}
for (const t of TABS) $(`nav-${t}`).onclick = () => showTab(t);

// ---------- linkedin tab ----------
let liFiles = [];
$("liPhotos").onchange = () => {
  liFiles = [...liFiles, ...$("liPhotos").files];
  $("liPhotos").value = "";
  renderThumbs();
};
function renderThumbs() {
  $("liThumbs").innerHTML = "";
  liFiles.forEach((f, i) => {
    const wrap = el("div", "thumb-wrap");
    const img = document.createElement("img");
    img.src = URL.createObjectURL(f);
    wrap.appendChild(img);
    const x = el("button", "thumb-x", "✕");
    x.onclick = () => { liFiles.splice(i, 1); renderThumbs(); };
    wrap.appendChild(x);
    $("liThumbs").appendChild(wrap);
  });
}

// compress to <=1600px JPEG and return raw base64 (no data: prefix)
function compressPhoto(file) {
  return new Promise((resolve, reject) => {
    const img = new Image();
    img.onload = () => {
      const scale = Math.min(1, 1600 / Math.max(img.width, img.height));
      const canvas = document.createElement("canvas");
      canvas.width = Math.round(img.width * scale);
      canvas.height = Math.round(img.height * scale);
      canvas.getContext("2d").drawImage(img, 0, 0, canvas.width, canvas.height);
      resolve(canvas.toDataURL("image/jpeg", 0.85).split(",")[1]);
    };
    img.onerror = reject;
    img.src = URL.createObjectURL(file);
  });
}

$("liSave").onclick = async () => {
  const noteText = $("liNote").value.trim();
  if (!noteText) { setStatus("Say or type the post idea first", "err"); return; }
  $("liSave").disabled = true;
  setStatus("Uploading…");
  try {
    const p = (n) => String(n).padStart(2, "0");
    const d = new Date();
    const stamp = `${d.getFullYear()}${p(d.getMonth() + 1)}${p(d.getDate())}-${p(d.getHours())}${p(d.getMinutes())}`;
    const dir = `drafts/linkedin/inbox/post-${stamp}`;
    const noteMd = `# LinkedIn post idea\n\nCaptured: ${todayISO()} ${p(d.getHours())}:${p(d.getMinutes())} | Source: capture-pwa\nPhotos: ${liFiles.length}\n\n${noteText}\n`;
    await gh.createFile(`${dir}/note.md`, utf8b64(noteMd), `linkedin: post idea ${stamp}`);
    for (let i = 0; i < liFiles.length; i++) {
      setStatus(`Uploading photo ${i + 1}/${liFiles.length}…`);
      await gh.createFile(`${dir}/photo-${i + 1}.jpg`, await compressPhoto(liFiles[i]), `linkedin: photo ${i + 1} for ${stamp}`);
    }
    $("liNote").value = ""; liFiles = []; $("liThumbs").innerHTML = ""; $("liPhotos").value = "";
    if (geminiKey) {
      setStatus("Drafting now…");
      let refs = "";
      try { refs = (await readFile("drafts/linkedin/reference-posts.md")).content; } catch {}
      const draft = await draftLinkedInPost(noteText, refs, { apiKey: geminiKey });
      if (draft) {
        await gh.createFile(`drafts/linkedin/ready/post-${stamp}.md`,
          utf8b64(`Rough draft (instant). The evening run refines it if still here.\n\n---\n\n${draft}\n`),
          `linkedin: instant rough draft ${stamp}`);
        setStatus("Rough draft ready below", "ok");
        loadDrafts();
      } else {
        setStatus("Gemini unavailable. Idea saved; the 9pm run writes the draft.", "err");
      }
    } else {
      setStatus("Sent to drafting. The evening run writes the post.", "ok");
    }
  } catch (e) {
    setStatus(e.message, "err");
  } finally {
    $("liSave").disabled = false;
  }
};

function utf8b64(s) {
  const bytes = new TextEncoder().encode(s);
  let bin = "";
  for (const b of bytes) bin += String.fromCharCode(b);
  return btoa(bin);
}

async function loadDrafts() {
  if (!token) return;
  const root = $("liDrafts");
  root.innerHTML = "";
  try {
    const entries = (await gh.listDir("drafts/linkedin/ready")).filter((e) => e.type === "file" && e.name.endsWith(".md"));
    const pending = (await gh.listDir("drafts/linkedin/inbox")).filter((e) => e.type === "dir");
    if (pending.length) root.appendChild(el("div", "muted pad", `${pending.length} idea${pending.length > 1 ? "s" : ""} waiting for the drafting run`));
    if (entries.length) root.appendChild(el("div", "section-h", `Ready to post (${entries.length})`));
    for (const e of entries) {
      const { content } = await gh.getFile(`drafts/linkedin/ready/${e.name}`);
      const card = el("div", "draft-card", content);
      const actions = el("div", "draft-actions");
      actions.appendChild(actionBtn("Copy", async () => {
        await navigator.clipboard.writeText(content);
        setStatus("Copied. Paste into LinkedIn and attach your photos.", "ok");
      }));
      actions.appendChild(actionBtn("Mark posted", async () => {
        if (!confirm("Remove this draft? (Stays in git history.)")) return;
        try {
          const fresh = (await gh.listDir("drafts/linkedin/ready")).find((x) => x.name === e.name);
          if (fresh) await gh.deleteFile(`drafts/linkedin/ready/${e.name}`, fresh.sha, "linkedin: posted");
          loadDrafts();
        } catch (err) { setStatus(err.message, "err"); }
      }));
      card.appendChild(actions);
      root.appendChild(card);
    }
    if (!entries.length && !pending.length) root.appendChild(el("div", "muted pad", "No drafts yet. Send a post idea above; the evening run writes it in your voice."));
  } catch (e) {
    root.appendChild(el("div", "muted pad", `Could not load drafts (${e.message})`));
  }
}

// ---------- cards tab ----------
let cardFiles = [];
$("cardPhotos").onchange = () => {
  cardFiles = [...cardFiles, ...$("cardPhotos").files];
  $("cardPhotos").value = "";
  renderCardThumbs();
};
function renderCardThumbs() {
  $("cardThumbs").innerHTML = "";
  cardFiles.forEach((f, i) => {
    const wrap = el("div", "thumb-wrap");
    const img = document.createElement("img");
    img.src = URL.createObjectURL(f);
    wrap.appendChild(img);
    const x = el("button", "thumb-x", "✕");
    x.onclick = () => { cardFiles.splice(i, 1); renderCardThumbs(); };
    wrap.appendChild(x);
    $("cardThumbs").appendChild(wrap);
  });
}

$("cardSave").onclick = async () => {
  const noteText = $("cardNote").value.trim();
  if (!noteText && !cardFiles.length) { setStatus("Add card photos or a note first", "err"); return; }
  $("cardSave").disabled = true;
  try {
    const now = new Date();
    const id = makeCardId(now);
    const dir = `${CARDS_DIR}/${id}`;
    // photos go up first; note.md lands last as the completion marker, so the
    // processing run never acts on a folder until it is whole
    for (let i = 0; i < cardFiles.length; i++) {
      setStatus(`Uploading card ${i + 1}/${cardFiles.length}…`);
      await gh.createFile(`${dir}/photo-${i + 1}.jpg`, await compressPhoto(cardFiles[i]), `cards: photo ${i + 1} for ${id}`);
    }
    setStatus("Saving…");
    const noteMd = buildCardNote({ id, capturedAt: isoLocal(now), noteText, photoCount: cardFiles.length });
    await gh.createFile(`${dir}/note.md`, utf8b64(noteMd), `cards: capture ${id}`);
    // a future instant-extraction step slots in here: write extracted.json
    // into the same folder and flip status (see CARDS_FEATURE_PLAN.md)
    $("cardNote").value = ""; cardFiles = []; renderCardThumbs(); $("cardPhotos").value = "";
    setStatus("Saved. The processing run files the contacts.", "ok");
  } catch (e) {
    if (cardFiles.length) {
      setStatus(`Upload failed (${e.message}). Photos stay attached; retry when you're back online.`, "err");
    } else {
      setQueue([...getQueue(), { dest: "cards", text: noteText, ts: Date.now() }]);
      $("cardNote").value = "";
      setStatus(`Offline or error (${e.message}). Queued on this phone.`, "err");
    }
  } finally {
    $("cardSave").disabled = false;
  }
};

// ---------- tasks tab ----------
const PRI_ORDER = { High: 0, Medium: 1, Low: 2 };

let inboxItems = []; // last known unprocessed captures, for re-renders

async function loadTasks() {
  if (!token) return;
  $("taskList").innerHTML = '<div class="muted pad">Loading…</div>';
  try {
    const [tasks, inbox] = await Promise.all([readFile(FILES.tasks), readFile(FILES.inbox)]);
    inboxItems = parseUnprocessed(inbox.content);
    renderTasks(tasks.content, tasks.offline);
  } catch (e) {
    $("taskList").innerHTML = `<div class="muted pad">Could not load tasks (${e.message})</div>`;
  }
}

// Re-render from content we just wrote: a refetch right after a write can
// return a stale version and visually undo the action.
function applyWritten(written) {
  noteWritten(FILES.tasks, written);
  renderTasks(written, false);
}

function renderTasks(md, offline) {
  const model = parseTasks(md);
  const today = todayISO();
  const root = $("taskList");
  root.innerHTML = "";
  if (offline) root.appendChild(el("div", "offline-note", "Offline copy, actions disabled"));

  for (const name of SECTIONS) {
    const section = model.sections.find((s) => s.name === name);
    if (!section) continue;
    const open = section.tasks.filter((t) => !t.done);
    const h = el("div", "section-h", `${name} (${open.length})`);
    root.appendChild(h);
    if (!open.length) { root.appendChild(el("div", "muted pad", "Nothing here")); continue; }
    const sorted = [...open].sort((a, b) => {
      const pa = PRI_ORDER[effectivePriority(a, today)] - PRI_ORDER[effectivePriority(b, today)];
      if (pa) return pa;
      return (a.due || "9999") < (b.due || "9999") ? -1 : 1;
    });
    for (const t of sorted) root.appendChild(taskRow(t, today, offline));
  }

  const pending = model.sections.find((s) => s.name === "Pending Review");
  if (pending && pending.reviews.length) {
    root.appendChild(el("div", "section-h", `Pending Review (${pending.reviews.length})`));
    for (const r of pending.reviews) {
      const row = el("div", "task-row review");
      row.appendChild(el("div", "task-text", r.raw));
      const meta = el("div", "task-meta");
      meta.appendChild(el("span", "due", `from ${r.source}`));
      meta.appendChild(actionBtn("→ Task", () => openTaskSheet(null, { mode: "promote-review", line: r.line, raw: r.raw })));
      meta.appendChild(actionBtn("Dismiss", async () => {
        if (!confirm("Remove this from Pending Review?")) return;
        try {
          const written = await gh.mutateFile(FILES.tasks, (x) => deleteTask(x, r.line), "task: dismiss pending review item");
          applyWritten(written);
        } catch (e) { setStatus(e.message, "err"); }
      }));
      row.appendChild(meta);
      root.appendChild(row);
    }
  }

  if (inboxItems.length) {
    root.appendChild(el("div", "section-h", `Inbox (${inboxItems.length} unprocessed)`));
    for (const c of inboxItems) {
      const row = el("div", "task-row review");
      row.appendChild(el("div", "task-text", c.raw));
      const meta = el("div", "task-meta");
      meta.appendChild(el("span", "due", `captured ${c.captured}`));
      meta.appendChild(actionBtn("→ Task", () => openTaskSheet(null, { mode: "promote-inbox", id: c.id, raw: c.raw })));
      meta.appendChild(actionBtn("Dismiss", async () => {
        if (!confirm("Dismiss this capture?")) return;
        try {
          const written = await gh.mutateFile(FILES.inbox, (x) => markProcessed(x, c.id, "Dismissed from phone", todayISO()), "capture: dismiss from phone");
          noteWritten(FILES.inbox, written);
          inboxItems = parseUnprocessed(written);
          loadTasks();
        } catch (e) { setStatus(e.message, "err"); }
      }));
      row.appendChild(meta);
      root.appendChild(row);
    }
    root.appendChild(el("div", "muted pad", "Promote a capture into a task, or leave it for the scheduled triage run."));
  }
}

function actionBtn(label, onclick) {
  const b = el("button", "mini-btn", label);
  b.onclick = (ev) => { ev.stopPropagation(); onclick(); };
  return b;
}

function taskRow(t, today, offline) {
  const row = el("div", "task-row");
  const box = document.createElement("button");
  box.className = "checkbox";
  box.setAttribute("aria-label", "Complete task");
  box.onclick = async (ev) => {
    ev.stopPropagation();
    if (offline) { setStatus("Offline, cannot complete tasks", "err"); return; }
    box.disabled = true;
    row.classList.add("fading");
    try {
      const written = await gh.mutateFile(FILES.tasks, (c) => completeTask(c, t.line, todayISO()), `task: complete "${t.text.slice(0, 50)}"`);
      setStatus("Completed, archived", "ok");
      applyWritten(written);
    } catch (e) {
      row.classList.remove("fading");
      box.disabled = false;
      setStatus(e.message, "err");
    }
  };
  row.appendChild(box);

  const body = el("div", "task-body");
  body.appendChild(el("div", "task-text", t.text));
  const pri = effectivePriority(t, today);
  const meta = el("div", "task-meta");
  const chip = el("span", `chip ${pri.toLowerCase()}`, pri);
  meta.appendChild(chip);
  if (t.due) {
    const overdue = t.due < today;
    meta.appendChild(el("span", overdue ? "due overdue" : "due", `due ${t.due}${overdue ? " (overdue)" : ""}`));
  }
  body.appendChild(meta);
  body.onclick = () => { if (!offline) openTaskSheet(t); };
  row.appendChild(body);
  return row;
}

function el(tag, cls, text) {
  const e = document.createElement(tag);
  if (cls) e.className = cls;
  if (text !== undefined) e.textContent = text;
  return e;
}

// ---------- task add/edit sheet ----------
function openTaskSheet(task, ctx) {
  sheetCtx = ctx || (task ? { mode: "edit", line: task.line } : { mode: "add" });
  const titles = { add: "New task", edit: "Edit task", "promote-inbox": "Promote capture", "promote-review": "Promote to task" };
  $("sheetTitle").textContent = titles[sheetCtx.mode];
  $("tText").value = task ? task.text : (sheetCtx.raw || "");
  $("tSection").value = task ? task.section : "Business";
  $("tPriority").value = task ? task.priority : "Low";
  $("tDue").value = task && task.due ? task.due : "";
  $("sheetErr").textContent = "";
  $("tDelete").classList.toggle("hidden", sheetCtx.mode !== "edit");
  $("taskSheet").classList.remove("hidden");
}
$("tDelete").onclick = async () => {
  if (sheetCtx.mode !== "edit" || !confirm("Delete this task permanently? (It stays in git history.)")) return;
  $("tDelete").disabled = true;
  try {
    const written = await gh.mutateFile(FILES.tasks, (c) => deleteTask(c, sheetCtx.line), "task: delete");
    $("taskSheet").classList.add("hidden");
    setStatus("Deleted", "ok");
    applyWritten(written);
  } catch (e) {
    $("sheetErr").textContent = e.message;
  } finally {
    $("tDelete").disabled = false;
  }
};
$("fab").onclick = () => openTaskSheet(null);
$("tCancel").onclick = () => $("taskSheet").classList.add("hidden");
$("tSave").onclick = async () => {
  const text = $("tText").value.trim();
  if (!text) { $("sheetErr").textContent = "Task text is required"; return; }
  const fields = {
    section: $("tSection").value,
    priority: $("tPriority").value,
    due: $("tDue").value || null,
    text,
  };
  $("tSave").disabled = true;
  try {
    const addFields = { ...fields, added: todayISO(), source: "capture-pwa" };
    let written;
    if (sheetCtx.mode === "edit") {
      written = await gh.mutateFile(FILES.tasks, (c) => updateTask(c, sheetCtx.line, fields), `task: edit "${text.slice(0, 50)}"`);
    } else if (sheetCtx.mode === "promote-review") {
      written = await gh.mutateFile(FILES.tasks, (c) => addTask(deleteTask(c, sheetCtx.line), addFields), `task: promote from pending review`);
    } else {
      written = await gh.mutateFile(FILES.tasks, (c) => addTask(c, addFields), `task: add "${text.slice(0, 50)}"`);
    }
    if (sheetCtx.mode === "promote-inbox") {
      const inboxWritten = await gh.mutateFile(FILES.inbox, (c) => markProcessed(c, sheetCtx.id, `Promoted to ${fields.section}`, todayISO()), "capture: promote to task");
      noteWritten(FILES.inbox, inboxWritten);
      inboxItems = parseUnprocessed(inboxWritten);
    }
    $("taskSheet").classList.add("hidden");
    setStatus("Saved", "ok");
    applyWritten(written);
  } catch (e) {
    $("sheetErr").textContent = e.message;
  } finally {
    $("tSave").disabled = false;
  }
};

// ---------- workout tab ----------
async function loadWorkout() {
  if (!token) return;
  $("workoutCard").innerHTML = '<div class="muted pad">Loading…</div>';
  $("weekList").innerHTML = "";
  try {
    const [tw, ledger, program] = await Promise.all([
      readFile(FILES.todayWorkout), readFile(FILES.ledger), readFile(FILES.program),
    ]);
    const offline = tw.offline || ledger.offline || program.offline;
    const { activeDay, card } = extractPhoneCard(tw.content);
    const root = $("workoutCard");
    root.innerHTML = "";
    if (offline) root.appendChild(el("div", "offline-note", "Offline copy"));
    root.appendChild(el("div", "section-h", activeDay ? `Up next: ${activeDay}` : "Today"));
    if (card) renderWorkoutCard(root, activeDay, card);
    else root.appendChild(el("div", "card-body", "No card generated yet. Ask me to prep your workout card."));

    const week = weekAhead({ ledgerMd: ledger.content, programMd: program.content, today: todayISO() });
    const wl = $("weekList");
    wl.appendChild(el("div", "section-h", "This week"));
    for (const w of week) {
      const row = el("div", `week-row ${w.status}`);
      const d = new Date(w.date + "T00:00:00");
      row.appendChild(el("span", "week-date", d.toLocaleDateString("en-US", { weekday: "short", month: "numeric", day: "numeric" })));
      row.appendChild(el("span", "week-name", `Day ${w.day}: ${w.name}`));
      row.appendChild(el("span", `badge ${w.status}`, w.status === "done" ? "✓ done" : w.status));
      wl.appendChild(row);
    }
    wl.appendChild(el("div", "muted pad", "Planned dates are suggestions from your 4+1 cadence, not commitments."));
  } catch (e) {
    $("workoutCard").innerHTML = `<div class="muted pad">Could not load workout (${e.message})</div>`;
  }
}

// ---------- interactive workout card ----------
const WS_KEY = "capture.workoutState";
function getWorkoutState(day) {
  try {
    const s = JSON.parse(localStorage.getItem(WS_KEY));
    if (s && s.day === day) return s;
  } catch {}
  return { day, checked: {}, fields: {} };
}
function putWorkoutState(s) { localStorage.setItem(WS_KEY, JSON.stringify(s)); }

function renderWorkoutCard(root, activeDay, card) {
  const state = getWorkoutState(activeDay);
  const items = parsePlanItems(card);

  // context above the plan (goal, hip check, timing)
  const head = card.split(/^PLAN$/m)[0].trim();
  if (head) root.appendChild(el("div", "card-body", head));

  // adjustment selector
  const adjRow = el("div", "adj-row");
  adjRow.appendChild(el("span", "muted", "Today I'm at:"));
  for (const lvl of ["100", "80", "70", "50"]) {
    const b = el("button", "adj-chip" + ((state.fields.adjustment || "100") === lvl ? " active" : ""), lvl + "%");
    b.onclick = () => {
      state.fields.adjustment = lvl;
      putWorkoutState(state);
      loadWorkout();
    };
    adjRow.appendChild(b);
  }
  root.appendChild(adjRow);
  const ADJ_HINTS = {
    100: "Run the planned session.",
    80: "Keep the main pattern; drop one accessory set or cut load 5-10%.",
    70: "Cap effort at RPE 7, cut sets ~25%, skip the finisher.",
    50: "Recovery only: easy cardio, light accessories, mobility.",
  };
  root.appendChild(el("div", "muted pad", ADJ_HINTS[state.fields.adjustment || "100"]));

  // checkable plan
  let lastGroup = null;
  for (const it of items) {
    if (it.group !== lastGroup) { root.appendChild(el("div", "wo-group", it.group)); lastGroup = it.group; }
    const row = el("div", "task-row wo-item" + (state.checked[it.text] ? " done" : ""));
    const box = el("button", "checkbox" + (state.checked[it.text] ? " checked" : ""));
    box.onclick = () => {
      state.checked[it.text] = !state.checked[it.text];
      putWorkoutState(state);
      loadWorkout();
    };
    row.appendChild(box);
    row.appendChild(el("div", "task-text", it.text));
    root.appendChild(row);
  }

  // actuals form
  root.appendChild(el("div", "wo-group", "Actuals"));
  const form = el("div", "wo-form");
  const field = (id, label, type = "text") => {
    const wrap = el("div", "wo-field");
    wrap.appendChild(el("label", "", label));
    const inp = document.createElement(type === "textarea" ? "textarea" : "input");
    inp.id = id;
    if (type !== "textarea") inp.type = "text";
    inp.value = state.fields[id] || "";
    inp.oninput = () => { state.fields[id] = inp.value; putWorkoutState(state); };
    wrap.appendChild(inp);
    return wrap;
  };
  const row3 = el("div", "wo-3");
  row3.appendChild(field("woReadiness", "Readiness %"));
  row3.appendChild(field("woHipBefore", "Hip before /10"));
  row3.appendChild(field("woHipAfter", "Hip after /10"));
  form.appendChild(row3);
  form.appendChild(field("woNotes", "Notes (weights, reps, how it felt)", "textarea"));
  const logBtn = el("button", "wo-log", "Log workout");
  logBtn.onclick = async () => {
    const completed = items.filter((i) => state.checked[i.text]).map((i) => i.text);
    const skipped = items.filter((i) => !state.checked[i.text]).map((i) => i.text);
    if (!completed.length && !state.fields.woNotes) { setStatus("Check off what you did or add a note first", "err"); return; }
    const raw = buildWorkoutLog({
      day: activeDay || "unknown day",
      readiness: state.fields.woReadiness || "",
      hipBefore: state.fields.woHipBefore || "",
      hipAfter: state.fields.woHipAfter || "",
      adjustment: state.fields.adjustment || "100",
      completed, skipped,
      notes: state.fields.woNotes || "",
    });
    logBtn.disabled = true;
    try {
      const written = await gh.mutateFile(FILES.workoutCapture, (c) => insertUnderUnprocessed(c, buildEntry(raw, new Date())), "workout: log from phone");
      noteWritten(FILES.workoutCapture, written);
      localStorage.removeItem(WS_KEY);
      setStatus("Workout logged. The scheduled run will update your ledger.", "ok");
      loadWorkout();
    } catch (e) {
      setStatus(e.message, "err");
    } finally {
      logBtn.disabled = false;
    }
  };
  form.appendChild(logBtn);
  root.appendChild(form);
}

// ---------- capture tab ----------
function getQueue() {
  try { return JSON.parse(localStorage.getItem(LS.queue)) || []; } catch { return []; }
}
function setQueue(q) {
  localStorage.setItem(LS.queue, JSON.stringify(q));
  $("pending").innerHTML = q.length ? `${q.length} unsent capture${q.length > 1 ? "s" : ""} <button id="retryBtn">retry</button>` : "";
  const r = $("retryBtn");
  if (r) r.onclick = flushQueue;
}
async function saveCapture(destKey, text) {
  const path = destKey === "inbox" ? FILES.inbox : FILES.workoutCapture;
  const written = await gh.mutateFile(path, (c) => insertUnderUnprocessed(c, buildEntry(text, new Date())), `capture: ${destKey} note from phone`);
  noteWritten(path, written);
  if (destKey === "inbox") inboxItems = parseUnprocessed(written);
}
// Queued note-only cards each become their own folder, sent one at a time.
async function flushCardsQueue() {
  for (const item of getQueue().filter((i) => i.dest === "cards")) {
    const now = new Date(item.ts || Date.now());
    const id = makeCardId(now);
    try {
      await gh.createFile(`${CARDS_DIR}/${id}/note.md`,
        utf8b64(buildCardNote({ id, capturedAt: isoLocal(now), noteText: item.text, photoCount: 0 })),
        `cards: queued capture ${id}`);
      setQueue(getQueue().filter((q) => !(q.dest === "cards" && q.ts === item.ts)));
      setStatus("Sent queued card note", "ok");
    } catch (e) {
      setStatus(`Queue send failed (${e.message})`, "err");
      return;
    }
  }
}
// Send all queued captures for each destination as ONE write, so a backlog
// cannot race itself into conflicts.
async function flushQueue() {
  if (!token) return;
  await flushCardsQueue();
  for (const destKey of ["inbox", "workout"]) {
    const q = getQueue();
    const mine = q.filter((i) => i.dest === destKey);
    if (!mine.length) continue;
    const path = destKey === "inbox" ? FILES.inbox : FILES.workoutCapture;
    const entries = mine.map((i) => buildEntry(i.text, new Date(i.ts || Date.now())));
    try {
      const written = await gh.mutateFile(path, (c) => insertAllUnderUnprocessed(c, entries),
        `capture: ${mine.length} queued ${destKey} note${mine.length > 1 ? "s" : ""} from phone`);
      noteWritten(path, written);
      if (destKey === "inbox") inboxItems = parseUnprocessed(written);
      setQueue(getQueue().filter((i) => i.dest !== destKey));
      setStatus(`Sent ${mine.length} queued capture${mine.length > 1 ? "s" : ""}`, "ok");
    } catch (e) {
      setStatus(`Queue send failed (${e.message})`, "err");
      break;
    }
  }
}
$("saveBtn").onclick = async () => {
  const text = $("note").value.trim();
  if (!text) return;
  $("saveBtn").disabled = true;
  setStatus("Saving…");
  try {
    // with a Gemini key, clear inbox captures file themselves as tasks
    let geminiDown = false;
    if (dest === "inbox" && geminiKey) {
      const c = await classifyCapture(text, { apiKey: geminiKey, today: todayISO() });
      geminiDown = c.action === "inbox" && c.reason === "unclassified";
      if (c.action === "task") {
        const written = await gh.mutateFile(FILES.tasks,
          (x) => addTask(x, { section: c.section, priority: c.priority, due: c.due, text: c.text, added: todayISO(), source: "capture-pwa (auto)" }),
          `task: auto-file "${c.text.slice(0, 50)}"`);
        noteWritten(FILES.tasks, written);
        $("note").value = "";
        setStatus(`Filed: ${c.section} / ${c.priority}${c.due ? " / due " + c.due : ""}`, "ok");
        flushQueue();
        return;
      }
    }
    await saveCapture(dest, text);
    $("note").value = "";
    setStatus(
      geminiDown ? "Gemini unavailable. Saved to Inbox; the triage run will file it."
        : dest === "inbox" ? "Saved to Inbox for triage" : "Saved to Workout",
      geminiDown ? "err" : "ok");
    flushQueue();
  } catch (e) {
    setQueue([...getQueue(), { dest, text, ts: Date.now() }]);
    $("note").value = "";
    setStatus(`Offline or error (${e.message}). Queued on this phone.`, "err");
  } finally {
    $("saveBtn").disabled = false;
  }
};
function setDest(d) {
  dest = d;
  $("destInbox").classList.toggle("active", d === "inbox");
  $("destWorkout").classList.toggle("active", d === "workout");
}
$("destInbox").onclick = () => setDest("inbox");
$("destWorkout").onclick = () => setDest("workout");

// ---------- install prompt ----------
let installEvent = null;
window.addEventListener("beforeinstallprompt", (e) => {
  e.preventDefault();
  installEvent = e;
  $("installBtn").classList.remove("hidden");
});
$("installBtn").onclick = async () => {
  if (!installEvent) return;
  installEvent.prompt();
  const choice = await installEvent.userChoice;
  if (choice.outcome === "accepted") $("installBtn").classList.add("hidden");
  installEvent = null;
};
window.addEventListener("appinstalled", () => $("installBtn").classList.add("hidden"));

// ---------- unlock & settings ----------
function showUnlock() {
  if (!localStorage.getItem(LS.vault)) { showSettings(); return; }
  $("pinOverlay").classList.remove("hidden");
  $("pinInput").focus();
}
$("pinUnlockBtn").onclick = async () => {
  try {
    const keys = await unlockKeys($("pinInput").value);
    token = keys.gh;
    geminiKey = keys.gemini || null;
    $("pinInput").value = "";
    $("pinErr").textContent = "";
    $("pinOverlay").classList.add("hidden");
    makeClient();
    flushQueue();
    showTab("tasks");
  } catch {
    $("pinErr").textContent = "Wrong PIN";
  }
};
$("pinInput").addEventListener("keydown", (e) => { if (e.key === "Enter") $("pinUnlockBtn").click(); });

function showSettings() {
  const s = getSettings();
  $("setOwner").value = s.owner || "Merlness";
  $("setRepo").value = s.repo || "life-organizer";
  $("setBranch").value = s.branch || "main";
  $("setApiBase").value = s.apiBase || "";
  $("setToken").value = "";
  $("setGemini").value = "";
  $("setPin").value = "";
  $("setErr").textContent = "";
  $("settingsOverlay").classList.remove("hidden");
}
$("gearBtn").onclick = showSettings;
$("setCancelBtn").onclick = () => $("settingsOverlay").classList.add("hidden");
$("setSaveBtn").onclick = async () => {
  const owner = $("setOwner").value.trim(), repo = $("setRepo").value.trim(), branch = $("setBranch").value.trim() || "main";
  const apiBase = $("setApiBase").value.trim().replace(/\/+$/, "");
  const tok = $("setToken").value.trim(), gem = $("setGemini").value.trim(), pin = $("setPin").value;
  if (!owner || !repo) { $("setErr").textContent = "Owner and repository are required"; return; }
  if (apiBase && !/^https?:\/\//.test(apiBase)) { $("setErr").textContent = "API server must be a full https:// URL"; return; }
  if (!tok && !localStorage.getItem(LS.vault)) { $("setErr").textContent = "Enter a token to finish setup"; return; }
  if ((tok || gem) && pin.length < 4) { $("setErr").textContent = "Enter your PIN (4+ characters) to save keys"; return; }
  setSettings({ owner, repo, branch, apiBase });
  if (tok || gem) {
    // changing one key keeps the other; unlock first if it is not in memory
    if (!token && localStorage.getItem(LS.vault)) {
      try {
        const old = await unlockKeys(pin);
        token = old.gh;
        geminiKey = old.gemini || null;
      } catch { $("setErr").textContent = "Wrong PIN"; return; }
    }
    if (tok) token = tok;
    if (gem) geminiKey = gem;
    await storeKeys(pin, { gh: token, gemini: geminiKey });
  }
  if (token) makeClient();
  $("settingsOverlay").classList.add("hidden");
  setStatus(geminiKey ? "Settings saved, instant filing on" : "Settings saved", "ok");
  if (token) { flushQueue(); showTab("tasks"); }
};

// ---------- init ----------
$("refreshBtn").onclick = () => {
  const active = ["tasks", "workout", "capture"].find((t) => !$(`tab-${t}`).classList.contains("hidden"));
  if (active === "tasks") loadTasks();
  if (active === "workout") loadWorkout();
};
setQueue(getQueue());
window.addEventListener("online", flushQueue);
if (localStorage.getItem(LS.vault)) showUnlock(); else showSettings();
if ("serviceWorker" in navigator) navigator.serviceWorker.register("sw.js");
