// Pure helpers for business card captures: folder ids plus note.md building
// and parsing for drafts/cards/inbox in the Life Organizer repo.
// v1 always writes status: captured. A later instant-extraction step writes
// an extracted.json beside the note and flips status to extracted; the
// processing run must accept both shapes.

const p = (n) => String(n).padStart(2, "0");

export function makeCardId(now) {
  return `card-${now.getFullYear()}${p(now.getMonth() + 1)}${p(now.getDate())}-${p(now.getHours())}${p(now.getMinutes())}${p(now.getSeconds())}`;
}

// Local wall-clock time with the machine's UTC offset, e.g. 2026-06-10T23:31:05-07:00
export function isoLocal(now) {
  const off = -now.getTimezoneOffset();
  const sign = off < 0 ? "-" : "+";
  const abs = Math.abs(off);
  return `${now.getFullYear()}-${p(now.getMonth() + 1)}-${p(now.getDate())}` +
    `T${p(now.getHours())}:${p(now.getMinutes())}:${p(now.getSeconds())}` +
    `${sign}${p(Math.floor(abs / 60))}:${p(abs % 60)}`;
}

export function buildCardNote({ id, capturedAt, noteText, photoCount }) {
  const note = (noteText || "").trim();
  return [
    "---",
    `id: ${id}`,
    `captured: ${capturedAt}`,
    "source: capture-pwa",
    `photos: ${photoCount}`,
    "status: captured",
    "---",
    "",
    note,
    "",
  ].join("\n").replace(/\n{3,}$/, "\n");
}

export function parseCardNote(md) {
  const m = md.match(/^---\n([\s\S]*?)\n---\n?([\s\S]*)$/);
  if (!m) throw new Error("not a card note: missing frontmatter");
  const fields = {};
  for (const line of m[1].split("\n")) {
    const i = line.indexOf(":");
    if (i > 0) fields[line.slice(0, i).trim()] = line.slice(i + 1).trim();
  }
  return {
    id: fields.id,
    captured: fields.captured,
    source: fields.source,
    photos: Number(fields.photos || 0),
    status: fields.status,
    note: m[2].trim(),
  };
}
