import { test } from "node:test";
import assert from "node:assert/strict";
import { buildEntry, insertUnderUnprocessed } from "../js/capture-entry.js";

const NOW = new Date(2026, 5, 10, 14, 30, 45); // 2026-06-10 14:30:45 local

test("buildEntry matches the inbox schema", () => {
  const entry = buildEntry("Call Sam about the proposal Friday", NOW);
  assert.equal(
    entry,
    "- [ ] id: 20260610-143045-call-sam-about | Captured: 2026-06-10 14:30 | Source: capture-pwa | Raw: Call Sam about the proposal Friday"
  );
});

test("buildEntry flattens newlines and trims", () => {
  const entry = buildEntry("  line one\n line two  ", NOW);
  assert.ok(entry.endsWith("Raw: line one line two"));
});

test("buildEntry slug falls back when text has no usable words", () => {
  const entry = buildEntry("!!!", NOW);
  assert.ok(entry.includes("id: 20260610-143045-note |"));
});

test("insertUnderUnprocessed puts the entry right below the heading", () => {
  const md = "# Inbox\n\n## Unprocessed\n\n- [ ] id: old | Raw: existing\n\n## Processed\n";
  const out = insertUnderUnprocessed(md, "- [ ] id: new | Raw: fresh");
  const lines = out.split("\n");
  const h = lines.indexOf("## Unprocessed");
  assert.equal(lines[h + 2], "- [ ] id: new | Raw: fresh");
  assert.ok(out.indexOf("id: new") < out.indexOf("id: old"), "newest first");
});

test("insertUnderUnprocessed creates the heading when missing", () => {
  const out = insertUnderUnprocessed("# Inbox\n", "- [ ] id: new | Raw: fresh");
  assert.ok(out.includes("## Unprocessed"));
  assert.ok(out.includes("id: new"));
});

test("insertUnderUnprocessed never produces triple blank lines", () => {
  const md = "# Inbox\n\n## Unprocessed\n\n\n- [ ] id: old | Raw: existing\n";
  const out = insertUnderUnprocessed(md, "- [ ] id: new | Raw: fresh");
  assert.ok(!/\n{3,}/.test(out));
});
