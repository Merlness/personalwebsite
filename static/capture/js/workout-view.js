// Read models for the workout tab: today's phone card from
// pulse/today-workout.md and a week-ahead schedule derived from
// pulse/workout-ledger.md plus pulse/workout-program.md.

export function extractPhoneCard(md) {
  const active = md.match(/^Active workout selected: (.+)$/m);
  const card = md.match(/^## Phone Card\n([\s\S]*?)(?=^## |\n*$(?![\s\S]))/m);
  return {
    activeDay: active ? active[1].trim() : null,
    card: card ? card[1].trim() : null,
  };
}

const ENTRY_RE = /^## (\d{4}-\d{2}-\d{2}) \| (.+)$/;

export function parseLedgerDays(md) {
  const out = [];
  for (const line of md.split("\n")) {
    const m = line.match(ENTRY_RE);
    if (!m) continue;
    const dayMatch = m[2].match(/Day (\d)/i);
    out.push({ date: m[1], name: m[2].trim(), day: dayMatch ? Number(dayMatch[1]) : null });
  }
  return out;
}

export function parseProgramDays(md) {
  const out = [];
  for (const line of md.split("\n")) {
    const m = line.match(/^## (Optional )?Day (\d): (.+)$/);
    if (m) out.push({ day: Number(m[2]), name: m[3].trim(), optional: Boolean(m[1]) });
  }
  return out;
}

function addDays(iso, n) {
  const d = new Date(iso + "T00:00:00Z");
  d.setUTCDate(d.getUTCDate() + n);
  return d.toISOString().slice(0, 10);
}

function mondayOf(iso) {
  const d = new Date(iso + "T00:00:00Z");
  const dow = (d.getUTCDay() + 6) % 7; // Mon=0
  return addDays(iso, -dow);
}

// Cadence: 4 required program days per week plus 1 optional day. Completed
// days come from the ledger; remaining required days are suggested one per
// day starting today, clamped to the current week (Mon-Sun).
export function weekAhead({ ledgerMd, programMd, today }) {
  const program = parseProgramDays(programMd);
  const weekStart = mondayOf(today);
  const weekEnd = addDays(weekStart, 6);

  const doneByDay = new Map();
  for (const e of parseLedgerDays(ledgerMd)) {
    if (e.day && e.date >= weekStart && e.date <= weekEnd && !doneByDay.has(e.day)) {
      doneByDay.set(e.day, e.date);
    }
  }

  const plan = [];
  let cursor = today;
  // if today's slot is already used by a completed session, start tomorrow
  if ([...doneByDay.values()].includes(today)) cursor = addDays(today, 1);
  for (const p of program) {
    if (doneByDay.has(p.day)) {
      plan.push({ date: doneByDay.get(p.day), day: p.day, name: p.name, status: "done" });
      continue;
    }
    const date = cursor <= weekEnd ? cursor : weekEnd;
    plan.push({ date, day: p.day, name: p.name, status: p.optional ? "optional" : "planned" });
    cursor = addDays(date, 1);
  }
  plan.sort((a, b) => (a.date < b.date ? -1 : a.date > b.date ? 1 : a.day - b.day));
  return plan;
}
