import { test } from "node:test";
import assert from "node:assert/strict";
import { assembleFinals } from "../js/transcript.js";

const r = (t, isFinal) => ({ isFinal, 0: { transcript: t }, length: 1 });

test("desktop pattern: distinct final segments concatenate", () => {
  const results = [r("workout log day two", true), r(" readiness 70", true), r(" RDLs 4 by 6", true)];
  assert.equal(assembleFinals(results), "workout log day two readiness 70 RDLs 4 by 6");
});

test("android pattern: cumulative finals collapse to the longest", () => {
  const results = [
    r("hey", true),
    r("hey I'm just", true),
    r("hey I'm just testing", true),
    r("hey I'm just testing this", true),
    r("still listening", false),
  ];
  assert.equal(assembleFinals(results), "hey I'm just testing this");
});

test("mixed pattern: cumulative then a new segment after a pause", () => {
  const results = [r("call Sam", true), r("call Sam about the proposal", true), r("and buy coffee", true)];
  assert.equal(assembleFinals(results), "call Sam about the proposal and buy coffee");
});

test("exact duplicate finals are dropped", () => {
  const results = [r("send the email", true), r("send the email", true)];
  assert.equal(assembleFinals(results), "send the email");
});

test("interim results are ignored", () => {
  const results = [r("done part", true), r("half spoken interi", false)];
  assert.equal(assembleFinals(results), "done part");
});

test("case differences still match the cumulative pattern", () => {
  const results = [r("Hey there", true), r("hey there friend", true)];
  assert.equal(assembleFinals(results), "hey there friend");
});

test("empty and whitespace finals are skipped", () => {
  const results = [r("  ", true), r("real text", true)];
  assert.equal(assembleFinals(results), "real text");
});
