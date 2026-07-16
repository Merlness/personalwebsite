// Parses a "#setup=<base64 JSON>" URL fragment into settings, so the server
// setup command can hand the phone a one-tap link instead of making Merl
// type an API URL and a 48-character token. The fragment never leaves the
// browser (fragments are not sent in HTTP requests).
export function parseSetupFragment(hash) {
  const m = /^#setup=([A-Za-z0-9+/_=-]+)$/.exec(hash || "");
  if (!m) return null;
  try {
    const b64 = m[1].replace(/-/g, "+").replace(/_/g, "/");
    const parsed = JSON.parse(atob(b64));
    const apiBase = typeof parsed.apiBase === "string" ? parsed.apiBase.trim().replace(/\/+$/, "") : "";
    const token = typeof parsed.token === "string" ? parsed.token.trim() : "";
    if (!/^https:\/\//.test(apiBase) || !token) return null;
    return { apiBase, token };
  } catch {
    return null;
  }
}
