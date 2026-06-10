import { test } from "node:test";
import assert from "node:assert/strict";
import { buildEntry, insertUnderUnprocessed, insertAllUnderUnprocessed, parseUnprocessed } from "../js/capture-entry.js";

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

test("insertAllUnderUnprocessed lands every entry in one pass, oldest first", () => {
  const md = "# Inbox\n\n## Unprocessed\n\n## Processed\n";
  const out = insertAllUnderUnprocessed(md, [
    "- [ ] id: a | Raw: first captured",
    "- [ ] id: b | Raw: second captured",
  ]);
  assert.ok(out.includes("id: a") && out.includes("id: b"));
  assert.ok(out.indexOf("id: a") < out.indexOf("id: b"), "queue order preserved top-down");
  assert.ok(!/\n{3,}/.test(out));
});

test("insertAllUnderUnprocessed with empty list returns input unchanged", () => {
  const md = "# Inbox\n\n## Unprocessed\n\n## Processed\n";
  assert.equal(insertAllUnderUnprocessed(md, []), md);
});

test("parseUnprocessed reads entries under the Unprocessed heading only", () => {
  const md = `# Inbox

## Unprocessed

- [ ] id: 20260610-1 | Captured: 2026-06-10 14:30 | Source: capture-pwa | Raw: call the landlord
- [ ] id: 20260610-2 | Captured: 2026-06-10 15:00 | Source: codex-mobile | Raw: buy coffee

<!--
- [ ] id: example | Captured: 2026-06-02 14:30 | Source: codex-mobile | Raw: commented example
-->

## Processed

- [x] id: old-1 | Processed: 2026-06-03 | Result: Added | Raw: done thing
`;
  const items = parseUnprocessed(md);
  assert.equal(items.length, 2);
  assert.deepEqual(items[0], { id: "20260610-1", captured: "2026-06-10 14:30", source: "capture-pwa", raw: "call the landlord" });
  assert.equal(items[1].raw, "buy coffee");
});

test("parseUnprocessed returns empty for a fresh inbox", () => {
  assert.deepEqual(parseUnprocessed("# Inbox\n\n## Unprocessed\n\n## Processed\n"), []);
});

test("insertUnderUnprocessed never produces triple blank lines", () => {
  const md = "# Inbox\n\n## Unprocessed\n\n\n- [ ] id: old | Raw: existing\n";
  const out = insertUnderUnprocessed(md, "- [ ] id: new | Raw: fresh");
  assert.ok(!/\n{3,}/.test(out));
});
