// Builds inbox.md / workout-capture.md entries per the Life Organizer schema.

export function buildEntry(text, now, source = "capture-pwa") {
  const p = (n) => String(n).padStart(2, "0");
  const date = `${now.getFullYear()}-${p(now.getMonth() + 1)}-${p(now.getDate())}`;
  const time = `${p(now.getHours())}:${p(now.getMinutes())}`;
  const stamp = `${now.getFullYear()}${p(now.getMonth() + 1)}${p(now.getDate())}-${p(now.getHours())}${p(now.getMinutes())}${p(now.getSeconds())}`;
  const slug = text.toLowerCase().replace(/[^a-z0-9\s]/g, "").trim().split(/\s+/).filter(Boolean).slice(0, 3).join("-") || "note";
  const raw = text.replace(/\s*\n\s*/g, " ").trim();
  return `- [ ] id: ${stamp}-${slug} | Captured: ${date} ${time} | Source: ${source} | Raw: ${raw}`;
}

const ENTRY_RE = /^- \[ \] id: (\S+) \| Captured: (.+?) \| Source: (.+?) \| Raw: (.+)$/;

// List unprocessed captures (between "## Unprocessed" and the next heading),
// skipping anything inside HTML comments.
export function parseUnprocessed(md) {
  const out = [];
  let inSection = false, inComment = false;
  for (const line of md.split("\n")) {
    if (/^## /.test(line)) { inSection = line.trim().toLowerCase() === "## unprocessed"; continue; }
    if (line.trimStart().startsWith("<!--")) inComment = true;
    if (inComment) { if (line.includes("-->")) inComment = false; continue; }
    if (!inSection) continue;
    const m = line.match(ENTRY_RE);
    if (m) out.push({ id: m[1], captured: m[2], source: m[3], raw: m[4] });
  }
  return out;
}

// Move an unprocessed entry to the Processed section with a result note.
export function markProcessed(md, id, result, date) {
  const lines = md.split("\n");
  const idx = lines.findIndex((l) => {
    const m = l.match(ENTRY_RE);
    return m && m[1] === id;
  });
  if (idx === -1) throw new Error("capture not found in Unprocessed: " + id);
  const m = lines[idx].match(ENTRY_RE);
  lines.splice(idx, 1);
  const procLine = `- [x] id: ${m[1]} | Processed: ${date} | Result: ${result} | Raw: ${m[4]}`;
  const proc = lines.findIndex((l) => l.trim().toLowerCase() === "## processed");
  if (proc === -1) lines.push("", "## Processed", "", procLine);
  else lines.splice(proc + 1, 0, "", procLine);
  return lines.join("\n").replace(/\n{3,}/g, "\n\n");
}

// Insert several entries in one pass, preserving their order top-down.
// Reduce in reverse: the last insert lands topmost.
export function insertAllUnderUnprocessed(content, entries) {
  return [...entries].reverse().reduce((c, e) => insertUnderUnprocessed(c, e), content);
}

export function insertUnderUnprocessed(content, entry) {
  const lines = content.split("\n");
  const i = lines.findIndex((l) => l.trim().toLowerCase() === "## unprocessed");
  if (i === -1) return (content.trimEnd() + "\n\n## Unprocessed\n\n" + entry + "\n").replace(/\n{3,}/g, "\n\n");
  lines.splice(i + 1, 0, "", entry);
  return lines.join("\n").replace(/\n{3,}/g, "\n\n");
}
