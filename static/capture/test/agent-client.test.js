import { test } from "node:test";
import assert from "node:assert/strict";
import { askAgent, streamAgent, trimHistory, splitFrames, parseFrame } from "../js/agent-client.js";

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

// ---------- streaming ----------

test("splitFrames returns whole frames and keeps the unfinished tail", () => {
  const { frames, rest } = splitFrames('data: {"a":1}\n\ndata: {"b":2}\n\ndata: {"c"');
  assert.deepEqual(frames, ['data: {"a":1}', 'data: {"b":2}']);
  assert.equal(rest, 'data: {"c"');
});

test("parseFrame reads the event name and joins multi-line data", () => {
  assert.deepEqual(parseFrame('event: done\ndata: {"reply":"hi"}'), { event: "done", data: '{"reply":"hi"}' });
  assert.deepEqual(parseFrame("data: one\ndata: two"), { event: "message", data: "one\ntwo" });
  assert.deepEqual(parseFrame("data: x\r"), { event: "message", data: "x" });
});

// Delivers an SSE body in caller-chosen chunks, so a frame split across two
// network reads is exercised the way it happens on a phone.
function sseFetch(chunks) {
  return async () => ({
    ok: true,
    status: 200,
    headers: { get: (k) => (k.toLowerCase() === "content-type" ? "text/event-stream" : null) },
    body: {
      getReader() {
        const enc = new TextEncoder();
        let i = 0;
        return {
          read: async () =>
            i < chunks.length ? { value: enc.encode(chunks[i++]), done: false } : { value: undefined, done: true },
        };
      },
    },
  });
}

test("streamAgent reports text, steps, and writes, then resolves with the done payload", async () => {
  const fetchFn = sseFetch([
    'data: {"type":"text","text":"Looking at "}\n\ndata: {"type":"te',
    'xt","text":"your workout."}\n\n',
    'data: {"type":"step","text":"reading pulse/today-workout.md"}\n\n',
    'data: {"type":"written","path":"pulse/today-workout.md","content":"upper"}\n\n',
    'event: done\ndata: {"reply":"Swapped it.","written":[{"path":"pulse/today-workout.md","content":"upper"}]}\n\n',
  ]);
  const text = [], steps = [], writes = [];
  const res = await streamAgent(
    { apiBase: "https://x.test", token: "t", history: [], message: "swap today to upper body" },
    { onText: (t) => text.push(t), onStep: (s) => steps.push(s), onWritten: (w) => writes.push(w) },
    fetchFn,
  );

  assert.equal(text.join(""), "Looking at your workout.");
  assert.deepEqual(steps, ["reading pulse/today-workout.md"]);
  assert.deepEqual(writes, [{ path: "pulse/today-workout.md", content: "upper" }]);
  assert.equal(res.reply, "Swapped it.");
  assert.equal(res.written[0].path, "pulse/today-workout.md");
});

test("streamAgent rejects on an error frame", async () => {
  const fetchFn = sseFetch(['data: {"type":"error","text":"agent failed, try again"}\n\n']);
  await assert.rejects(
    streamAgent({ apiBase: "https://x.test", token: "t", history: [], message: "hi" }, {}, fetchFn),
    /agent failed/,
  );
});

test("streamAgent falls back to JSON when the server does not stream", async () => {
  const fetchFn = async () => ({
    ok: true,
    status: 200,
    headers: { get: () => "application/json" },
    json: async () => ({ reply: "done", written: [] }),
  });
  const res = await streamAgent({ apiBase: "https://x.test", token: "t", history: [], message: "hi" }, {}, fetchFn);
  assert.equal(res.reply, "done");
});

test("streamAgent surfaces the server error message", async () => {
  const fetchFn = fakeFetch([{ status: 503, body: { message: "agent not configured" } }]);
  await assert.rejects(
    streamAgent({ apiBase: "https://x.test", token: "t", history: [], message: "hi" }, {}, fetchFn),
    /agent not configured/,
  );
});
