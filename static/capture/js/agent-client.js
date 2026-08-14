// Client for the personal-api /agent endpoint. fetch is injectable for tests.

export function trimHistory(history, limit = 10) {
  return history.slice(Math.max(0, history.length - limit));
}

function endpoint(apiBase) {
  if (!apiBase) throw new Error("Set the API server in settings to use Ask");
  return apiBase.replace(/\/+$/, "") + "/agent";
}

async function failure(res) {
  let msg = `agent failed (${res.status})`;
  try {
    const body = await res.json();
    if (body && body.message) msg = body.message;
  } catch {}
  return new Error(msg);
}

function send(fetchFn, { apiBase, token, history, message }, accept) {
  // Bind to the global: native fetch throws "Illegal invocation" otherwise.
  const doFetch = fetchFn.bind(globalThis);
  const headers = { Authorization: "Bearer " + token, "Content-Type": "application/json" };
  if (accept) headers.Accept = accept;
  return doFetch(endpoint(apiBase), {
    method: "POST",
    headers,
    body: JSON.stringify({ message, history: trimHistory(history || []) }),
  });
}

export async function askAgent(opts, fetchFn = globalThis.fetch) {
  const res = await send(fetchFn, opts);
  if (!res.ok) throw await failure(res);
  const body = await res.json();
  return { reply: body.reply || "", written: body.written || [] };
}

// Splits a Server-Sent Events buffer into the frames that have fully arrived
// and the tail that has not. A blank line ends a frame.
export function splitFrames(buffer) {
  const parts = buffer.split("\n\n");
  return { frames: parts.slice(0, -1), rest: parts[parts.length - 1] };
}

// Reads one frame into its event name and its joined data payload.
export function parseFrame(frame) {
  let event = "message";
  const data = [];
  for (const raw of frame.split("\n")) {
    const line = raw.replace(/\r$/, "");
    if (line.startsWith("data:")) data.push(line.slice(5).replace(/^ /, ""));
    else if (line.startsWith("event:")) event = line.slice(6).trim();
  }
  return { event, data: data.join("\n") };
}

// streamAgent runs a command and reports progress while it happens:
// onText for each piece of the reply, onStep before each tool call, and
// onWritten as soon as a file changes. It resolves with the same shape
// askAgent returns, so the caller stores history the same way either path ran.
export async function streamAgent(opts, handlers = {}, fetchFn = globalThis.fetch) {
  const res = await send(fetchFn, opts, "text/event-stream");
  if (!res.ok) throw await failure(res);

  const type = res.headers && res.headers.get ? res.headers.get("Content-Type") || "" : "";
  if (!type.includes("text/event-stream") || !res.body) {
    // An older deploy, or something in front of it that will not stream.
    const body = await res.json();
    return { reply: body.reply || "", written: body.written || [] };
  }
  return consume(res.body.getReader(), handlers);
}

async function consume(reader, handlers) {
  const decoder = new TextDecoder();
  let buf = "";
  let result = { reply: "", written: [] };
  for (;;) {
    const { value, done } = await reader.read();
    if (done) break;
    buf += decoder.decode(value, { stream: true });
    const split = splitFrames(buf);
    buf = split.rest;
    for (const frame of split.frames) {
      const { event, data } = parseFrame(frame);
      if (!data) continue;
      let e;
      try { e = JSON.parse(data); } catch { continue; }
      if (event === "done") {
        result = { reply: e.reply || "", written: e.written || [] };
      } else if (e.type === "text") {
        handlers.onText?.(e.text || "");
      } else if (e.type === "step") {
        handlers.onStep?.(e.text || "");
      } else if (e.type === "written") {
        handlers.onWritten?.({ path: e.path || "", content: e.content || "" });
      } else if (e.type === "error") {
        throw new Error(e.text || "agent failed, try again");
      }
    }
  }
  return result;
}
