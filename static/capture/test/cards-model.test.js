import { test } from "node:test";
import assert from "node:assert/strict";
import { makeCardId, isoLocal, buildCardNote, parseCardNote } from "../js/cards-model.js";

const NOW = new Date(2026, 5, 10, 23, 31, 5); // 2026-06-10 23:31:05 local

test("makeCardId stamps to the second", () => {
  assert.equal(makeCardId(NOW), "card-20260610-233105");
});

test("isoLocal carries the machine's utc offset", () => {
  const off = -NOW.getTimezoneOffset();
  const sign = off < 0 ? "-" : "+";
  const abs = Math.abs(off);
  const p = (n) => String(n).padStart(2, "0");
  assert.equal(isoLocal(NOW), `2026-06-10T23:31:05${sign}${p(Math.floor(abs / 60))}:${p(abs % 60)}`);
});

test("buildCardNote writes frontmatter plus the note body", () => {
  const md = buildCardNote({
    id: "card-20260610-233105",
    capturedAt: "2026-06-10T23:31:05-07:00",
    noteText: "Met at AZ Tech Week. Say hi for me.",
    photoCount: 2,
  });
  assert.equal(md, `---
id: card-20260610-233105
captured: 2026-06-10T23:31:05-07:00
source: capture-pwa
photos: 2
status: captured
---

Met at AZ Tech Week. Say hi for me.
`);
});

test("buildCardNote with no note keeps just the frontmatter", () => {
  const md = buildCardNote({ id: "card-x", capturedAt: "t", noteText: "", photoCount: 1 });
  assert.ok(md.endsWith("status: captured\n---\n"));
  assert.ok(!/\n{3,}/.test(md));
});

test("a note-only capture is valid with zero photos", () => {
  const md = buildCardNote({ id: "card-x", capturedAt: "t", noteText: "Reach out to Amanda from Izzy Turf", photoCount: 0 });
  assert.ok(md.includes("photos: 0"));
  assert.ok(md.includes("Reach out to Amanda"));
});

test("parseCardNote round-trips what buildCardNote writes", () => {
  const md = buildCardNote({
    id: "card-20260610-233105",
    capturedAt: "2026-06-10T23:31:05-07:00",
    noteText: "Met Amanda.\nBring up the proposal.",
    photoCount: 3,
  });
  assert.deepEqual(parseCardNote(md), {
    id: "card-20260610-233105",
    captured: "2026-06-10T23:31:05-07:00",
    source: "capture-pwa",
    photos: 3,
    status: "captured",
    note: "Met Amanda.\nBring up the proposal.",
  });
});

test("parseCardNote throws on a file without frontmatter", () => {
  assert.throws(() => parseCardNote("just some text"));
});
