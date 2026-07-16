import { test } from "node:test";
import assert from "node:assert/strict";
import { askAgent, trimHistory } from "../js/agent-client.js";

function fakeFetch(handlers) {
  const calls = [];
  const fn = async (url, opts = {}) => {
    calls.push({ url, opts });
    const h = handlers.shift();
    if (!h) throw new Error("unexpected fetch: " + url);
    return {
      ok: h.status >= 200 && h.status < 300,
      status: h.status,
      json: async () => h.body,
    };
  };
  fn.calls = calls;
  return fn;
}

test("trimHistory keeps only the newest turns", () => {
  const history = Array.from({ length: 14 }, (_, i) => ({ role: "user", content: String(i) }));
  const trimmed = trimHistory(history);
  assert.equal(trimmed.length, 10);
  assert.equal(trimmed[0].content, "4");
  assert.equal(trimmed[9].content, "13");
});

test("askAgent posts message plus trimmed history with the bearer token", async () => {
  const fetchFn = fakeFetch([{ status: 200, body: { reply: "done", written: [{ path: "tasks.md", content: "x" }] } }]);
  const history = Array.from({ length: 12 }, (_, i) => ({ role: "user", content: String(i) }));

  const res = await askAgent(
    { apiBase: "https://merl-personal-api.fly.dev/", token: "tok", history, message: "add a task" },
    fetchFn,
  );

  assert.equal(res.reply, "done");
  assert.equal(res.written[0].path, "tasks.md");
  const call = fetchFn.calls[0];
  assert.equal(call.url, "https://merl-personal-api.fly.dev/agent");
  assert.equal(call.opts.headers.Authorization, "Bearer tok");
  const body = JSON.parse(call.opts.body);
  assert.equal(body.message, "add a task");
  assert.equal(body.history.length, 10);
});

test("askAgent surfaces the server error message", async () => {
  const fetchFn = fakeFetch([{ status: 503, body: { message: "agent not configured" } }]);
  await assert.rejects(
    askAgent({ apiBase: "https://x.test", token: "t", history: [], message: "hi" }, fetchFn),
    /agent not configured/,
  );
});

test("askAgent requires an apiBase", async () => {
  await assert.rejects(
    askAgent({ apiBase: "", token: "t", history: [], message: "hi" }, fakeFetch([])),
    /API server/,
  );
});

test("askAgent binds fetch to the global receiver", async () => {
  function strictFetch() {
    if (this !== globalThis) throw new TypeError("Illegal invocation");
    return { ok: true, status: 200, json: async () => ({ reply: "ok", written: [] }) };
  }
  const res = await askAgent({ apiBase: "https://x.test", token: "t", history: [], message: "hi" }, strictFetch);
  assert.equal(res.reply, "ok");
});
