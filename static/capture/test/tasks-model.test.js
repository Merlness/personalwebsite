import { test } from "node:test";
import assert from "node:assert/strict";
import {
  parseTasks,
  serializeTasks,
  taskToLine,
  addTask,
  completeTask,
  updateTask,
  deleteTask,
  effectivePriority,
} from "../js/tasks-model.js";

const SAMPLE = `# Life Organizer — Tasks

Last updated: 2026-06-09 (daily sync note)

Priority auto-escalates by due date: within 3 days = High, within 7 days = Medium, beyond 7 days = Low (unless manually raised).

---

## Business

- [ ] Low | Due: none | Reach out to Monica re Bennu Systems | Added: 2026-05-13 | Source: Keep ("Monica Follow-up")
- [ ] High | Due: 2026-06-04 | Look for the Arizona Builders Alliance | Added: 2026-06-04 | Source: Keep ("Sales Organizer")

## Personal

- [ ] High | Due: 2026-05-12 | Fruit basket for friend | Added: 2026-05-11 | Source: Keep (Urgent: Fruit Basket for Friend)

## Financial

_No active tasks yet._

## Pending Review

- [ ] [REVIEW] | Source: Keep ("Azbo") | Raw: Title only, no body | Added: 2026-05-28

---

## Archive

_Completed tasks live here. Never deleted._

<!--
Task format:
- [ ] {Priority} | Due: {YYYY-MM-DD or none} | {Task description} | Added: {YYYY-MM-DD} | Source: {Keep note ID or "manual"}
-->
`;

test("parse extracts sections in order", () => {
  const model = parseTasks(SAMPLE);
  assert.deepEqual(
    model.sections.map((s) => s.name),
    ["Business", "Personal", "Financial", "Pending Review", "Archive"]
  );
});

test("parse extracts task fields", () => {
  const model = parseTasks(SAMPLE);
  const biz = model.sections[0].tasks;
  assert.equal(biz.length, 2);
  assert.deepEqual(biz[0], {
    done: false,
    priority: "Low",
    due: null,
    text: "Reach out to Monica re Bennu Systems",
    added: "2026-05-13",
    source: 'Keep ("Monica Follow-up")',
    line: '- [ ] Low | Due: none | Reach out to Monica re Bennu Systems | Added: 2026-05-13 | Source: Keep ("Monica Follow-up")',
    section: "Business",
  });
  assert.equal(biz[1].due, "2026-06-04");
  assert.equal(biz[1].priority, "High");
});

test("parse extracts pending review items as reviews", () => {
  const model = parseTasks(SAMPLE);
  const pending = model.sections.find((s) => s.name === "Pending Review");
  assert.equal(pending.tasks.length, 0);
  assert.equal(pending.reviews.length, 1);
  assert.equal(pending.reviews[0].raw, "Title only, no body");
  assert.equal(pending.reviews[0].source, 'Keep ("Azbo")');
});

test("serialize round-trips unchanged input byte for byte", () => {
  assert.equal(serializeTasks(parseTasks(SAMPLE)), SAMPLE);
});

test("task-shaped lines inside HTML comments are not parsed as tasks", () => {
  const md = `## Archive

_Completed tasks live here._

<!--
Example:
- [ ] High | Due: 2026-05-13 | Follow up with client on proposal | Added: 2026-05-11 | Source: Keep
-->
`;
  const archive = parseTasks(md).sections[0];
  assert.equal(archive.tasks.length, 0);
  assert.equal(serializeTasks(parseTasks(md)), md);
});

test("taskToLine formats per SOP", () => {
  const line = taskToLine({
    done: false,
    priority: "Medium",
    due: "2026-06-15",
    text: "Send Alex his one-sheet",
    added: "2026-06-10",
    source: "capture-pwa",
  });
  assert.equal(
    line,
    "- [ ] Medium | Due: 2026-06-15 | Send Alex his one-sheet | Added: 2026-06-10 | Source: capture-pwa"
  );
});

test("taskToLine uses none for missing due", () => {
  const line = taskToLine({
    done: false, priority: "Low", due: null, text: "X", added: "2026-06-10", source: "manual",
  });
  assert.ok(line.includes("Due: none"));
});

test("addTask appends to the end of the section task block", () => {
  const out = addTask(SAMPLE, {
    section: "Business",
    priority: "Low",
    due: null,
    text: "Email lawyer for referrals",
    added: "2026-06-10",
    source: "capture-pwa",
  });
  const lines = out.split("\n");
  const idx = lines.findIndex((l) => l.includes("Email lawyer for referrals"));
  assert.ok(idx > -1);
  assert.ok(lines[idx - 1].includes("Arizona Builders Alliance"), "appended after last Business task");
  // still parses and the new task is in Business
  const model = parseTasks(out);
  assert.equal(model.sections[0].tasks.length, 3);
});

test("addTask replaces the empty-section placeholder", () => {
  const out = addTask(SAMPLE, {
    section: "Financial",
    priority: "Low",
    due: null,
    text: "Set up business savings",
    added: "2026-06-10",
    source: "manual",
  });
  assert.ok(!out.includes("_No active tasks yet._"));
  const fin = parseTasks(out).sections.find((s) => s.name === "Financial");
  assert.equal(fin.tasks.length, 1);
});

test("addTask to unknown section throws", () => {
  assert.throws(() => addTask(SAMPLE, { section: "Nope", priority: "Low", due: null, text: "x", added: "2026-06-10", source: "m" }));
});

test("completeTask moves the task to Archive with Done date", () => {
  const target = parseTasks(SAMPLE).sections[1].tasks[0]; // fruit basket
  const out = completeTask(SAMPLE, target.line, "2026-06-10");
  assert.ok(!out.includes("- [ ] High | Due: 2026-05-12 | Fruit basket"), "removed from Personal");
  const archIdx = out.indexOf("## Archive");
  const doneIdx = out.indexOf("- [x] High | Due: 2026-05-12 | Fruit basket for friend | Added: 2026-05-11 | Source: Keep (Urgent: Fruit Basket for Friend) | Done: 2026-06-10");
  assert.ok(doneIdx > archIdx, "archived under Archive with Done date");
  // still round-trip parseable
  assert.equal(serializeTasks(parseTasks(out)), out);
});

test("completeTask on a missing line throws", () => {
  assert.throws(() => completeTask(SAMPLE, "- [ ] Low | Due: none | not there | Added: 2026-01-01 | Source: x", "2026-06-10"));
});

test("updateTask rewrites fields in place", () => {
  const target = parseTasks(SAMPLE).sections[0].tasks[0];
  const out = updateTask(SAMPLE, target.line, { priority: "High", due: "2026-06-12", text: "Reach out to Monica re a la carte model" });
  const model = parseTasks(out);
  const t = model.sections[0].tasks[0];
  assert.equal(t.priority, "High");
  assert.equal(t.due, "2026-06-12");
  assert.equal(t.text, "Reach out to Monica re a la carte model");
  assert.equal(t.added, "2026-05-13", "added date preserved");
  assert.equal(model.sections[0].tasks.length, 2, "no duplicate created");
});

test("updateTask can move a task to another section", () => {
  const target = parseTasks(SAMPLE).sections[0].tasks[0];
  const out = updateTask(SAMPLE, target.line, { section: "Personal" });
  const model = parseTasks(out);
  assert.equal(model.sections[0].tasks.length, 1);
  assert.equal(model.sections[1].tasks.length, 2);
});

test("completeTask never inserts inside the Archive comment block", () => {
  // mirrors the real tasks.md: the comment holds a REAL-looking example line
  const md = `## Personal

- [ ] High | Due: 2026-05-12 | Fruit basket for friend | Added: 2026-05-11 | Source: Keep

## Archive

_Completed tasks live here. Never deleted._

<!--
Example:
- [ ] High | Due: 2026-05-13 | Follow up with client on proposal | Added: 2026-05-11 | Source: Keep
-->
`;
  const target = parseTasks(md).sections[0].tasks[0];
  const out = completeTask(md, target.line, "2026-06-10");
  const doneIdx = out.indexOf("- [x] High | Due: 2026-05-12 | Fruit basket");
  assert.ok(doneIdx !== -1 && doneIdx < out.indexOf("<!--"), "archived line lands before the comment block");
  assert.equal(parseTasks(out).sections.find((s) => s.name === "Archive").tasks.length, 1, "visible to the parser");
});

test("deleteTask removes the line entirely and stays parseable", () => {
  const target = parseTasks(SAMPLE).sections[0].tasks[0];
  const out = deleteTask(SAMPLE, target.line);
  assert.ok(!out.includes("Reach out to Monica"));
  const model = parseTasks(out);
  assert.equal(model.sections[0].tasks.length, 1);
  assert.ok(!/\n{3,}/.test(out), "no triple blank lines left behind");
  assert.equal(serializeTasks(model), out);
});

test("deleteTask on a missing line throws", () => {
  assert.throws(() => deleteTask(SAMPLE, "- [ ] Low | Due: none | ghost | Added: 2026-01-01 | Source: x"));
});

// SOP table: 0-3 days or overdue = High, 4-7 = Medium, 8+ or none = Low. Manual raise wins.
test("effectivePriority follows the SOP due-date table", () => {
  const today = "2026-06-10";
  assert.equal(effectivePriority({ due: "2026-06-01", priority: "Low" }, today), "High", "overdue");
  assert.equal(effectivePriority({ due: "2026-06-10", priority: "Low" }, today), "High", "due today");
  assert.equal(effectivePriority({ due: "2026-06-13", priority: "Low" }, today), "High", "3 days out");
  assert.equal(effectivePriority({ due: "2026-06-14", priority: "Low" }, today), "Medium", "4 days out");
  assert.equal(effectivePriority({ due: "2026-06-17", priority: "Low" }, today), "Medium", "7 days out");
  assert.equal(effectivePriority({ due: "2026-06-18", priority: "Low" }, today), "Low", "8 days out");
  assert.equal(effectivePriority({ due: null, priority: "Low" }, today), "Low", "no due date");
});

test("effectivePriority lets a manual raise win", () => {
  const today = "2026-06-10";
  assert.equal(effectivePriority({ due: null, priority: "High" }, today), "High");
  assert.equal(effectivePriority({ due: "2026-06-30", priority: "Medium" }, today), "Medium");
});
