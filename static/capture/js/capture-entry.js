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

export function insertUnderUnprocessed(content, entry) {
  const lines = content.split("\n");
  const i = lines.findIndex((l) => l.trim().toLowerCase() === "## unprocessed");
  if (i === -1) return (content.trimEnd() + "\n\n## Unprocessed\n\n" + entry + "\n").replace(/\n{3,}/g, "\n\n");
  lines.splice(i + 1, 0, "", entry);
  return lines.join("\n").replace(/\n{3,}/g, "\n\n");
}
