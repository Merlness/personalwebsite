// UI shell for the organizer PWA. All parsing, formatting, scheduling, and
// API logic lives in the tested modules; this file only wires DOM to them.

import { parseTasks, addTask, completeTask, updateTask, deleteTask, effectivePriority } from "./tasks-model.js";
import { buildEntry, insertUnderUnprocessed, insertAllUnderUnprocessed } from "./capture-entry.js";
import { extractPhoneCard, weekAhead } from "./workout-view.js";
import { GitHubClient } from "./github-api.js";

const FILES = {
  tasks: "tasks.md",
  inbox: "inbox.md",
  workoutCapture: "pulse/workout-capture.md",
  todayWorkout: "pulse/today-workout.md",
  ledger: "pulse/workout-ledger.md",
  program: "pulse/workout-program.md",
};
const LS = { settings: "capture.settings", vault: "capture.vault", queue: "capture.queue", cache: "capture.filecache" };
const SECTIONS = ["Business", "Personal", "Financial"];

let token = null;
let gh = null;
let dest = "inbox";
let editingLine = null; // task line being edited, null = adding

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
async function storeToken(pin, tok) {
  const salt = crypto.getRandomValues(new Uint8Array(16));
  const iv = crypto.getRandomValues(new Uint8Array(12));
  const key = await deriveKey(pin, salt);
  const ct = await crypto.subtle.encrypt({ name: "AES-GCM", iv }, key, enc.encode(tok));
  localStorage.setItem(LS.vault, JSON.stringify({ salt: b64(salt), iv: b64(iv), ct: b64(ct) }));
}
async function unlockToken(pin) {
  const v = JSON.parse(localStorage.getItem(LS.vault));
  const key = await deriveKey(pin, unb64(v.salt));
  return dec.decode(await crypto.subtle.decrypt({ name: "AES-GCM", iv: unb64(v.iv) }, key, unb64(v.ct)));
}

function makeClient() {
  const { owner, repo, branch } = getSettings();
  gh = new GitHubClient({ owner, repo, branch: branch || "main", token });
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
async function readFile(path) {
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
function showTab(name) {
  for (const t of ["tasks", "workout", "capture"]) {
    $(`tab-${t}`).classList.toggle("hidden", t !== name);
    $(`nav-${t}`).classList.toggle("active", t === name);
  }
  $("fab").classList.toggle("hidden", name !== "tasks");
  if (name === "tasks") loadTasks();
  if (name === "workout") loadWorkout();
}
for (const t of ["tasks", "workout", "capture"]) $(`nav-${t}`).onclick = () => showTab(t);

// ---------- tasks tab ----------
const PRI_ORDER = { High: 0, Medium: 1, Low: 2 };

async function loadTasks() {
  if (!token) return;
  $("taskList").innerHTML = '<div class="muted pad">Loading…</div>';
  try {
    const { content, offline } = await readFile(FILES.tasks);
    renderTasks(content, offline);
  } catch (e) {
    $("taskList").innerHTML = `<div class="muted pad">Could not load tasks (${e.message})</div>`;
  }
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
      row.appendChild(el("div", "task-meta", `from ${r.source}`));
      root.appendChild(row);
    }
    root.appendChild(el("div", "muted pad", 'Triage these with "run my capture intake" at a machine.'));
  }
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
      await gh.mutateFile(FILES.tasks, (c) => completeTask(c, t.line, todayISO()), `task: complete "${t.text.slice(0, 50)}"`);
      setStatus("Completed, archived", "ok");
      loadTasks();
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
function openTaskSheet(task) {
  editingLine = task ? task.line : null;
  $("sheetTitle").textContent = task ? "Edit task" : "New task";
  $("tText").value = task ? task.text : "";
  $("tSection").value = task ? task.section : "Business";
  $("tPriority").value = task ? task.priority : "Low";
  $("tDue").value = task && task.due ? task.due : "";
  $("sheetErr").textContent = "";
  $("tDelete").classList.toggle("hidden", !task);
  $("taskSheet").classList.remove("hidden");
}
$("tDelete").onclick = async () => {
  if (!editingLine || !confirm("Delete this task permanently? (It stays in git history.)")) return;
  $("tDelete").disabled = true;
  try {
    await gh.mutateFile(FILES.tasks, (c) => deleteTask(c, editingLine), "task: delete");
    $("taskSheet").classList.add("hidden");
    setStatus("Deleted", "ok");
    loadTasks();
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
    if (editingLine) {
      await gh.mutateFile(FILES.tasks, (c) => updateTask(c, editingLine, fields), `task: edit "${text.slice(0, 50)}"`);
    } else {
      await gh.mutateFile(FILES.tasks, (c) => addTask(c, { ...fields, added: todayISO(), source: "capture-pwa" }), `task: add "${text.slice(0, 50)}"`);
    }
    $("taskSheet").classList.add("hidden");
    setStatus("Saved", "ok");
    loadTasks();
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
    const pre = el("div", "card-body", card || "No card generated yet. Ask me to prep your workout card.");
    root.appendChild(pre);

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
  await gh.mutateFile(path, (c) => insertUnderUnprocessed(c, buildEntry(text, new Date())), `capture: ${destKey} note from phone`);
}
// Send all queued captures for each destination as ONE write, so a backlog
// cannot race itself into conflicts.
async function flushQueue() {
  if (!token) return;
  for (const destKey of ["inbox", "workout"]) {
    const q = getQueue();
    const mine = q.filter((i) => i.dest === destKey);
    if (!mine.length) continue;
    const path = destKey === "inbox" ? FILES.inbox : FILES.workoutCapture;
    const entries = mine.map((i) => buildEntry(i.text, new Date(i.ts || Date.now())));
    try {
      await gh.mutateFile(path, (c) => insertAllUnderUnprocessed(c, entries),
        `capture: ${mine.length} queued ${destKey} note${mine.length > 1 ? "s" : ""} from phone`);
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
    await saveCapture(dest, text);
    $("note").value = "";
    setStatus(`Saved to ${dest === "inbox" ? "Inbox" : "Workout"}`, "ok");
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
    token = await unlockToken($("pinInput").value);
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
  $("setToken").value = "";
  $("setPin").value = "";
  $("setErr").textContent = "";
  $("settingsOverlay").classList.remove("hidden");
}
$("gearBtn").onclick = showSettings;
$("setCancelBtn").onclick = () => $("settingsOverlay").classList.add("hidden");
$("setSaveBtn").onclick = async () => {
  const owner = $("setOwner").value.trim(), repo = $("setRepo").value.trim(), branch = $("setBranch").value.trim() || "main";
  const tok = $("setToken").value.trim(), pin = $("setPin").value;
  if (!owner || !repo) { $("setErr").textContent = "Owner and repository are required"; return; }
  if (tok && pin.length < 4) { $("setErr").textContent = "PIN must be at least 4 characters"; return; }
  if (!tok && !localStorage.getItem(LS.vault)) { $("setErr").textContent = "Enter a token to finish setup"; return; }
  setSettings({ owner, repo, branch });
  if (tok) { await storeToken(pin, tok); token = tok; }
  if (token) makeClient();
  $("settingsOverlay").classList.add("hidden");
  setStatus("Settings saved", "ok");
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
