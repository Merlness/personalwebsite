// Parser, serializer, and operations for the Life Organizer tasks.md.
// Format (SOP section 2):
//   - [ ] {Priority} | Due: {YYYY-MM-DD or none} | {Description} | Added: {YYYY-MM-DD} | Source: {source}
// Pending Review items:
//   - [ ] [REVIEW] | Source: {source} | Raw: {raw} | Added: {YYYY-MM-DD}
// Archived tasks carry a trailing "| Done: {YYYY-MM-DD}".

const TASK_RE = /^- \[( |x)\] (High|Medium|Low) \| Due: (\S+) \| (.+?) \| Added: (\d{4}-\d{2}-\d{2}) \| Source: (.+?)(?: \| Done: (\d{4}-\d{2}-\d{2}))?$/;
const REVIEW_RE = /^- \[ \] \[REVIEW\] \| Source: (.+?) \| Raw: (.+?) \| Added: (\d{4}-\d{2}-\d{2})$/;
const PLACEHOLDER_RE = /^_No active tasks yet\._$/;

export function parseTasks(md) {
  const lines = md.split("\n");
  const model = { preamble: [], sections: [] };
  let section = null;
  let inComment = false;
  for (const line of lines) {
    const wasInComment = inComment || line.trimStart().startsWith("<!--");
    if (line.trimStart().startsWith("<!--") && !line.includes("-->")) inComment = true;
    if (inComment && line.includes("-->")) inComment = false;
    if (wasInComment) {
      (section ? section.lines : model.preamble).push(line);
      continue;
    }
    const heading = line.match(/^## (.+)$/);
    if (heading) {
      section = { name: heading[1], lines: [], tasks: [], reviews: [] };
      model.sections.push(section);
      continue;
    }
    if (!section) {
      model.preamble.push(line);
      continue;
    }
    section.lines.push(line);
    const t = line.match(TASK_RE);
    if (t) {
      section.tasks.push({
        done: t[1] === "x",
        priority: t[2],
        due: t[3] === "none" ? null : t[3],
        text: t[4],
        added: t[5],
        source: t[6],
        ...(t[7] ? { doneDate: t[7] } : {}),
        line,
        section: section.name,
      });
      continue;
    }
    const r = line.match(REVIEW_RE);
    if (r) section.reviews.push({ source: r[1], raw: r[2], added: r[3], line, section: section.name });
  }
  return model;
}

export function serializeTasks(model) {
  const parts = [...model.preamble];
  for (const s of model.sections) parts.push(`## ${s.name}`, ...s.lines);
  return parts.join("\n");
}

export function taskToLine(t) {
  const box = t.done ? "x" : " ";
  const base = `- [${box}] ${t.priority} | Due: ${t.due || "none"} | ${t.text} | Added: ${t.added} | Source: ${t.source}`;
  return t.doneDate ? `${base} | Done: ${t.doneDate}` : base;
}

function findSection(model, name) {
  const s = model.sections.find((x) => x.name === name);
  if (!s) throw new Error(`section not found: ${name}`);
  return s;
}

// Insert a task line at the end of a section's task block. Replaces the
// "_No active tasks yet._" placeholder when it is the only content.
function insertTaskLine(section, line) {
  const placeholderIdx = section.lines.findIndex((l) => PLACEHOLDER_RE.test(l.trim()));
  if (placeholderIdx !== -1 && section.tasks.length === 0) {
    section.lines[placeholderIdx] = line;
    return;
  }
  let last = -1;
  for (let i = 0; i < section.lines.length; i++) {
    if (TASK_RE.test(section.lines[i])) last = i;
  }
  if (last !== -1) section.lines.splice(last + 1, 0, line);
  else {
    // empty section: insert after leading blank line, keeping one blank before
    let i = 0;
    while (i < section.lines.length && section.lines[i].trim() === "") i++;
    section.lines.splice(i, 0, line, "");
  }
}

export function addTask(md, { section, priority, due, text, added, source }) {
  const model = parseTasks(md);
  const s = findSection(model, section);
  insertTaskLine(s, taskToLine({ done: false, priority, due, text, added, source }));
  return serializeTasks(model);
}

function removeLine(model, line) {
  for (const s of model.sections) {
    const i = s.lines.indexOf(line);
    if (i !== -1) {
      s.lines.splice(i, 1);
      // drop a doubled blank left behind
      if (s.lines[i] === "" && (i === 0 || s.lines[i - 1] === "")) s.lines.splice(i, 1);
      return s;
    }
  }
  throw new Error("task not found, the list may have changed: " + line);
}

export function completeTask(md, line, today) {
  const m = line.match(TASK_RE);
  if (!m) throw new Error("not a task line: " + line);
  const model = parseTasks(md);
  removeLine(model, line);
  const archive = findSection(model, "Archive");
  const parsed = parseTasks(`## X\n${line}`).sections[0].tasks[0];
  const doneLine = taskToLine({ ...parsed, done: true, doneDate: today });
  // append after the last archived task, or after the italic note
  let at = -1;
  for (let i = 0; i < archive.lines.length; i++) {
    if (TASK_RE.test(archive.lines[i])) at = i;
    else if (at === -1 && archive.lines[i].trim().startsWith("_")) at = i;
  }
  archive.lines.splice(at + 1, 0, "", doneLine);
  const out = serializeTasks(model).replace(/\n{3,}/g, "\n\n");
  return out;
}

export function updateTask(md, line, changes) {
  const model = parseTasks(md);
  const owner = model.sections.find((s) => s.lines.includes(line));
  if (!owner) throw new Error("task not found, the list may have changed: " + line);
  const parsed = parseTasks(`## X\n${line}`).sections[0].tasks[0];
  const updated = { ...parsed, ...changes };
  if (!changes.section || changes.section === owner.name) {
    owner.lines[owner.lines.indexOf(line)] = taskToLine(updated); // in place, keep position
    return serializeTasks(model);
  }
  removeLine(model, line);
  insertTaskLine(findSection(model, changes.section), taskToLine(updated));
  return serializeTasks(model).replace(/\n{3,}/g, "\n\n");
}

export function deleteTask(md, line) {
  const model = parseTasks(md);
  removeLine(model, line);
  return serializeTasks(model).replace(/\n{3,}/g, "\n\n");
}

const ORDER = { High: 0, Medium: 1, Low: 2 };

export function effectivePriority(task, today) {
  let auto = "Low";
  if (task.due) {
    const days = Math.round((Date.parse(task.due) - Date.parse(today)) / 86400000);
    if (days <= 3) auto = "High";
    else if (days <= 7) auto = "Medium";
  }
  return ORDER[task.priority] < ORDER[auto] ? task.priority : auto;
}
