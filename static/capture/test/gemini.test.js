import { test } from "node:test";
import assert from "node:assert/strict";
import { classifyCapture } from "../js/gemini.js";

const OPTS = { apiKey: "k", today: "2026-06-10" };

function geminiFetch(jsonText, status = 200) {
  const calls = [];
  const fn = async (url, opts) => {
    calls.push({ url, opts });
    return {
      ok: status === 200,
      status,
      json: async () => ({ candidates: [{ content: { parts: [{ text: jsonText }] } }] }),
    };
  };
  fn.calls = calls;
  return fn;
}

test("a clear commitment becomes a task with bucket and priority", async () => {
  const fetch = geminiFetch(JSON.stringify({ action: "task", section: "Business", priority: "Medium", due: "2026-06-13", text: "Email lawyer for referrals" }));
  const r = await classifyCapture("email my lawyer asking for referrals by Friday", { ...OPTS, fetchFn: fetch });
  assert.deepEqual(r, { action: "task", section: "Business", priority: "Medium", due: "2026-06-13", text: "Email lawyer for referrals" });
  const req = JSON.parse(fetch.calls[0].opts.body);
  assert.ok(JSON.stringify(req).includes("Business"), "prompt carries the routing rules");
  assert.ok(fetch.calls[0].url.includes("key=k"));
});

test("an idea stays in the inbox", async () => {
  const fetch = geminiFetch(JSON.stringify({ action: "inbox", reason: "idea" }));
  const r = await classifyCapture("maybe I should create an estimation app for construction", { ...OPTS, fetchFn: fetch });
  assert.equal(r.action, "inbox");
});

test("invalid section falls back to inbox instead of corrupting tasks.md", async () => {
  const fetch = geminiFetch(JSON.stringify({ action: "task", section: "Work", priority: "High", due: null, text: "x" }));
  const r = await classifyCapture("anything", { ...OPTS, fetchFn: fetch });
  assert.equal(r.action, "inbox");
});

test("invalid priority or malformed due date falls back to inbox", async () => {
  const badPri = await classifyCapture("a", { ...OPTS, fetchFn: geminiFetch(JSON.stringify({ action: "task", section: "Personal", priority: "Urgent", due: null, text: "x" })) });
  assert.equal(badPri.action, "inbox");
  const badDue = await classifyCapture("a", { ...OPTS, fetchFn: geminiFetch(JSON.stringify({ action: "task", section: "Personal", priority: "Low", due: "Friday", text: "x" })) });
  assert.equal(badDue.action, "inbox");
});

test("non-JSON response falls back to inbox", async () => {
  const r = await classifyCapture("a", { ...OPTS, fetchFn: geminiFetch("Sure! Here's my analysis...") });
  assert.equal(r.action, "inbox");
});

test("API errors fall back to inbox, never throw", async () => {
  const r = await classifyCapture("a", { ...OPTS, fetchFn: geminiFetch("{}", 429) });
  assert.equal(r.action, "inbox");
  const r2 = await classifyCapture("a", { ...OPTS, fetchFn: async () => { throw new Error("network down"); } });
  assert.equal(r2.action, "inbox");
});

test("empty task text falls back to inbox", async () => {
  const r = await classifyCapture("a", { ...OPTS, fetchFn: geminiFetch(JSON.stringify({ action: "task", section: "Personal", priority: "Low", due: null, text: "  " })) });
  assert.equal(r.action, "inbox");
});

import { draftLinkedInPost } from "../js/gemini.js";

test("draftLinkedInPost returns the drafted text and feeds references into the prompt", async () => {
  const fetch = geminiFetch("First time pitching.\n\nGreat night at the event.");
  const text = await draftLinkedInPost("won money at SBA pitch", "REF POSTS HERE", { apiKey: "k", fetchFn: fetch });
  assert.equal(text, "First time pitching.\n\nGreat night at the event.");
  const req = JSON.stringify(JSON.parse(fetch.calls[0].opts.body));
  assert.ok(req.includes("REF POSTS HERE") && req.includes("won money at SBA pitch"));
  assert.ok(req.toLowerCase().includes("em dash"), "voice rules included");
});

test("draftLinkedInPost returns null on API error or empty text", async () => {
  assert.equal(await draftLinkedInPost("x", "", { apiKey: "k", fetchFn: geminiFetch("", 429) }), null);
  assert.equal(await draftLinkedInPost("x", "", { apiKey: "k", fetchFn: geminiFetch("   ") }), null);
  assert.equal(await draftLinkedInPost("x", "", { apiKey: "k", fetchFn: async () => { throw new Error("net"); } }), null);
});
