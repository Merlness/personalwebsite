// Instant capture triage via the Gemini API (free tier). Classifies a
// capture as a clear task (filed immediately) or inbox (ideas, context,
// anything ambiguous - left for the scheduled Claude run). Any failure or
// doubt falls back to inbox: captures must never be lost or misfiled.

const MODEL = "gemini-2.0-flash";
const SECTIONS = ["Business", "Personal", "Financial"];
const PRIORITIES = ["High", "Medium", "Low"];
const DATE_RE = /^\d{4}-\d{2}-\d{2}$/;

const RULES = `You triage one voice/text capture for Merl Martin's Life Organizer.

Decide: is this a SINGLE CLEAR CONCRETE COMMITMENT Merl owns (a task), or anything else?

Respond with ONLY a JSON object, no markdown:
- A clear task: {"action":"task","section":"Business|Personal|Financial","priority":"High|Medium|Low","due":"YYYY-MM-DD or null","text":"<concise imperative task, cleaned up from the rambling capture>"}
- Anything else: {"action":"inbox","reason":"<one word: idea|shopping|relationship|opportunity|unclear|multiple>"}

Rules:
- Business: work, clients, deliverables, business development. Personal: friends, family, errands, social, health. Financial: bills, taxes, investments, banking.
- Priority from due date: due within 3 days = High, within 7 = Medium, else Low. No due date = Low unless the capture says urgent.
- NEVER invent a due date. Only set "due" if the capture names a specific day; resolve relative days ("Friday", "tomorrow") against today's date, which is given.
- Ideas, brainstorms, "maybe I should...", shopping lists, notes about people, opportunities, or captures containing MULTIPLE tasks: all go to inbox.
- When in doubt, inbox.`;

export async function classifyCapture(text, { apiKey, today, fetchFn = globalThis.fetch } = {}) {
  const fallback = { action: "inbox", reason: "unclassified" };
  try {
    const fetchBound = fetchFn.bind(globalThis);
    const res = await fetchBound(
      `https://generativelanguage.googleapis.com/v1beta/models/${MODEL}:generateContent?key=${encodeURIComponent(apiKey)}`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          contents: [{ parts: [{ text: `${RULES}\n\nToday is ${today}.\n\nCapture: ${text}` }] }],
          generationConfig: { responseMimeType: "application/json", temperature: 0 },
        }),
      }
    );
    if (!res.ok) return fallback;
    const body = await res.json();
    const raw = body?.candidates?.[0]?.content?.parts?.[0]?.text;
    const parsed = JSON.parse(raw);
    if (parsed.action === "inbox") return { action: "inbox", reason: String(parsed.reason || "unclear") };
    if (parsed.action !== "task") return fallback;
    const t = String(parsed.text || "").trim();
    if (!t) return fallback;
    if (!SECTIONS.includes(parsed.section)) return fallback;
    if (!PRIORITIES.includes(parsed.priority)) return fallback;
    const due = parsed.due == null ? null : String(parsed.due);
    if (due !== null && !DATE_RE.test(due)) return fallback;
    return { action: "task", section: parsed.section, priority: parsed.priority, due, text: t };
  } catch {
    return fallback;
  }
}
