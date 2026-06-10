import { test } from "node:test";
import assert from "node:assert/strict";
import { GitHubClient, utf8ToB64, b64ToUtf8 } from "../js/github-api.js";

const CFG = { owner: "Merlness", repo: "life-organizer", branch: "main", token: "tok" };

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

test("utf8 base64 round-trips multibyte text", () => {
  const s = "café — naïve ✓";
  assert.equal(b64ToUtf8(utf8ToB64(s)), s);
});

// Native window.fetch throws "Illegal invocation" unless called with the
// global as its receiver. Emulate that strictness to pin the binding.
test("client invokes fetch with the global as receiver, never itself", async () => {
  function strictFetch() {
    if (this !== globalThis) throw new TypeError("Failed to execute 'fetch' on 'Window': Illegal invocation");
    return { ok: true, status: 200, json: async () => ({ content: utf8ToB64("x"), sha: "s" }) };
  }
  const gh = new GitHubClient(CFG, strictFetch);
  const file = await gh.getFile("inbox.md");
  assert.equal(file.content, "x");
});

test("getFile decodes content and returns sha", async () => {
  const fetch = fakeFetch([
    { status: 200, body: { content: utf8ToB64("hello\n"), sha: "abc" } },
  ]);
  const gh = new GitHubClient(CFG, fetch);
  const file = await gh.getFile("inbox.md");
  assert.equal(file.content, "hello\n");
  assert.equal(file.sha, "abc");
  assert.ok(fetch.calls[0].url.includes("/repos/Merlness/life-organizer/contents/inbox.md?ref=main"));
  assert.equal(fetch.calls[0].opts.headers.Authorization, "Bearer tok");
});

test("getFile throws with status on failure", async () => {
  const gh = new GitHubClient(CFG, fakeFetch([{ status: 404, body: {} }]));
  await assert.rejects(() => gh.getFile("nope.md"), /404/);
});

test("mutateFile gets, transforms, puts with sha, and resolves with the written content", async () => {
  const fetch = fakeFetch([
    { status: 200, body: { content: utf8ToB64("a\n"), sha: "s1" } },
    { status: 200, body: {} },
  ]);
  const gh = new GitHubClient(CFG, fetch);
  const written = await gh.mutateFile("tasks.md", (c) => c + "b\n", "msg");
  assert.equal(written, "a\nb\n", "caller can render the written state without a stale refetch");
  const put = fetch.calls[1];
  const body = JSON.parse(put.opts.body);
  assert.equal(put.opts.method, "PUT");
  assert.equal(b64ToUtf8(body.content), "a\nb\n");
  assert.equal(body.sha, "s1");
  assert.equal(body.message, "msg");
  assert.equal(body.branch, "main");
});

test("mutateFile rides out repeated conflicts with backoff", async () => {
  const fetch = fakeFetch([
    { status: 200, body: { content: utf8ToB64("v1\n"), sha: "s1" } },
    { status: 409, body: {} },
    { status: 200, body: { content: utf8ToB64("v1\n"), sha: "s1" } }, // stale read
    { status: 409, body: {} },
    { status: 200, body: { content: utf8ToB64("v2\n"), sha: "s2" } },
    { status: 200, body: {} },
  ]);
  const sleeps = [];
  const gh = new GitHubClient(CFG, fetch);
  await gh.mutateFile("inbox.md", (c) => c + "x\n", "msg", { sleep: async (ms) => sleeps.push(ms) });
  assert.equal(fetch.calls.length, 6);
  assert.equal(sleeps.length, 2, "slept between conflict retries");
  assert.ok(sleeps[1] > sleeps[0], "backoff grows");
});

test("concurrent mutateFile calls are serialized, never interleaved", async () => {
  const order = [];
  const fetch = async (url, opts = {}) => {
    const method = opts.method || "GET";
    order.push(method);
    if (method === "GET") return { ok: true, status: 200, json: async () => ({ content: utf8ToB64("x\n"), sha: "s" }) };
    return { ok: true, status: 200, json: async () => ({}) };
  };
  const gh = new GitHubClient(CFG, fetch);
  await Promise.all([
    gh.mutateFile("inbox.md", (c) => c + "a", "m1"),
    gh.mutateFile("inbox.md", (c) => c + "b", "m2"),
  ]);
  assert.deepEqual(order, ["GET", "PUT", "GET", "PUT"], "second write waits for the first");
});

test("a failed mutateFile does not block later writes", async () => {
  const fetch = fakeFetch([
    { status: 200, body: { content: utf8ToB64("v\n"), sha: "s" } },
    { status: 500, body: {} },
    { status: 200, body: { content: utf8ToB64("v\n"), sha: "s" } },
    { status: 200, body: {} },
  ]);
  const gh = new GitHubClient(CFG, fetch);
  await assert.rejects(() => gh.mutateFile("inbox.md", (c) => c, "m1"), /500/);
  await gh.mutateFile("inbox.md", (c) => c, "m2"); // must not hang or reject
});

test("mutateFile re-applies the transform to fresh content after a conflict", async () => {
  const fetch = fakeFetch([
    { status: 200, body: { content: utf8ToB64("v1\n"), sha: "s1" } },
    { status: 409, body: {} },
    { status: 200, body: { content: utf8ToB64("v2\n"), sha: "s2" } },
    { status: 200, body: {} },
  ]);
  const gh = new GitHubClient(CFG, fetch);
  await gh.mutateFile("tasks.md", (c) => c + "x\n", "msg", { sleep: async () => {} });
  const secondPut = JSON.parse(fetch.calls[3].opts.body);
  assert.equal(b64ToUtf8(secondPut.content), "v2\nx\n", "transform re-applied to fresh content");
  assert.equal(secondPut.sha, "s2");
});

test("mutateFile gives up after exhausting retries and throws", async () => {
  const fetch = fakeFetch([
    { status: 200, body: { content: utf8ToB64("v1\n"), sha: "s1" } },
    { status: 409, body: {} },
    { status: 200, body: { content: utf8ToB64("v2\n"), sha: "s2" } },
    { status: 409, body: {} },
  ]);
  const gh = new GitHubClient(CFG, fetch);
  await assert.rejects(
    () => gh.mutateFile("tasks.md", (c) => c, "msg", { retries: 2, sleep: async () => {} }),
    /conflict/
  );
});

test("mutateFile surfaces transform errors without writing", async () => {
  const fetch = fakeFetch([
    { status: 200, body: { content: utf8ToB64("v1\n"), sha: "s1" } },
  ]);
  const gh = new GitHubClient(CFG, fetch);
  await assert.rejects(() => gh.mutateFile("tasks.md", () => { throw new Error("task not found"); }, "m"), /task not found/);
  assert.equal(fetch.calls.length, 1, "no PUT attempted");
});
