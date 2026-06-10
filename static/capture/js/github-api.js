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
    this.fetch = fetchFn;
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

  // get -> transform -> put; on sha conflict refetch and re-apply once.
  async mutateFile(path, transform, message) {
    for (let attempt = 0; attempt < 2; attempt++) {
      const file = await this.getFile(path);
      const updated = transform(file.content);
      const res = await this.putFile(path, updated, file.sha, message);
      if (res.ok) return;
      if (res.status !== 409 && res.status !== 422) throw new Error(`write ${path} failed (${res.status})`);
    }
    throw new Error(`write ${path} failed: conflict, try again`);
  }
}
