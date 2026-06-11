// Thin GitHub Contents API client with optimistic-concurrency retry.
// fetch is injectable for tests.

export function utf8ToB64(s) {
  const bytes = new TextEncoder().encode(s);
  let bin = "";
  for (const b of bytes) bin += String.fromCharCode(b);
  return btoa(bin);
}

export function b64ToUtf8(s) {
  const bin = atob(s.replace(/\n/g, ""));
  const bytes = Uint8Array.from(bin, (c) => c.charCodeAt(0));
  return new TextDecoder().decode(bytes);
}

export class GitHubClient {
  constructor(cfg, fetchFn = globalThis.fetch) {
    this.cfg = cfg;
    // Bind to the global: native fetch throws "Illegal invocation" when
    // called with any other receiver (like this client via this.fetch()).
    this.fetch = fetchFn.bind(globalThis);
  }

  async request(path, opts = {}) {
    return this.fetch("https://api.github.com" + path, {
      ...opts,
      headers: {
        Authorization: "Bearer " + this.cfg.token,
        Accept: "application/vnd.github+json",
        "X-GitHub-Api-Version": "2022-11-28",
        ...(opts.headers || {}),
      },
    });
  }

  contentsPath(path) {
    return `/repos/${this.cfg.owner}/${this.cfg.repo}/contents/${path}`;
  }

  async getFile(path) {
    const res = await this.request(`${this.contentsPath(path)}?ref=${encodeURIComponent(this.cfg.branch)}`);
    if (!res.ok) throw new Error(`read ${path} failed (${res.status})`);
    const body = await res.json();
    return { content: b64ToUtf8(body.content), sha: body.sha };
  }

  async putFile(path, content, sha, message) {
    return this.request(this.contentsPath(path), {
      method: "PUT",
      body: JSON.stringify({ message, content: utf8ToB64(content), sha, branch: this.cfg.branch }),
    });
  }

  // create a NEW file (no sha): content is already base64 (binary-safe)
  async createFile(path, base64Content, message) {
    const res = await this.request(this.contentsPath(path), {
      method: "PUT",
      body: JSON.stringify({ message, content: base64Content, branch: this.cfg.branch }),
    });
    if (!res.ok) throw new Error(`create ${path} failed (${res.status})`);
  }

  async listDir(path) {
    const res = await this.request(`${this.contentsPath(path)}?ref=${encodeURIComponent(this.cfg.branch)}`);
    if (res.status === 404) return [];
    if (!res.ok) throw new Error(`list ${path} failed (${res.status})`);
    const body = await res.json();
    return body.map((e) => ({ name: e.name, sha: e.sha, type: e.type }));
  }

  async deleteFile(path, sha, message) {
    const res = await this.request(this.contentsPath(path), {
      method: "DELETE",
      body: JSON.stringify({ message, sha, branch: this.cfg.branch }),
    });
    if (!res.ok) throw new Error(`delete ${path} failed (${res.status})`);
  }

  // get -> transform -> put. All writes through this client are serialized
  // (concurrent callers queue up), and sha conflicts are retried with a
  // growing backoff because the Contents API can serve a stale read right
  // after a write.
  async mutateFile(path, transform, message, opts = {}) {
    const run = () => this.#mutateNow(path, transform, message, opts);
    const turn = (this.#writeChain || Promise.resolve()).then(run, run);
    this.#writeChain = turn.catch(() => {}); // a failure must not block later writes
    return turn;
  }

  #writeChain = null;

  async #mutateNow(path, transform, message, opts) {
    const retries = opts.retries ?? 4;
    const sleep = opts.sleep ?? ((ms) => new Promise((r) => setTimeout(r, ms)));
    for (let attempt = 0; attempt < retries; attempt++) {
      const file = await this.getFile(path);
      const updated = transform(file.content);
      const res = await this.putFile(path, updated, file.sha, message);
      // resolve with what was written: a refetch right after a write can
      // return stale content, so callers must render from this instead
      if (res.ok) return updated;
      if (res.status !== 409 && res.status !== 422) throw new Error(`write ${path} failed (${res.status})`);
      if (attempt < retries - 1) await sleep(500 * (attempt + 1));
    }
    throw new Error(`write ${path} failed: conflict, try again`);
  }
}
