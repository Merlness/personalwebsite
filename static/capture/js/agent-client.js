// Client for the personal-api /agent endpoint. fetch is injectable for tests.

export function trimHistory(history, limit = 10) {
  return history.slice(Math.max(0, history.length - limit));
}

export async function askAgent({ apiBase, token, history, message }, fetchFn = globalThis.fetch) {
  if (!apiBase) throw new Error("Set the API server in settings to use Ask");
  // Bind to the global: native fetch throws "Illegal invocation" otherwise.
  const doFetch = fetchFn.bind(globalThis);
  const res = await doFetch(apiBase.replace(/\/+$/, "") + "/agent", {
    method: "POST",
    headers: {
      Authorization: "Bearer " + token,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ message, history: trimHistory(history || []) }),
  });
  if (!res.ok) {
    let msg = `agent failed (${res.status})`;
    try {
      const body = await res.json();
      if (body && body.message) msg = body.message;
    } catch {}
    throw new Error(msg);
  }
  const body = await res.json();
  return { reply: body.reply || "", written: body.written || [] };
}
