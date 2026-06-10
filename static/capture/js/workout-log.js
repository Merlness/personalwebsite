// Interactive workout card helpers: parse the plan into checkable items and
// compose a natural-language log entry for pulse/workout-capture.md.

// Exercises are the "- " bullets between PLAN and ADJUSTMENT, grouped under
// their block labels (lines ending with ":").
export function parsePlanItems(card) {
  const lines = card.split("\n");
  const start = lines.findIndex((l) => l.trim() === "PLAN");
  if (start === -1) return [];
  const items = [];
  let group = "";
  for (let i = start + 1; i < lines.length; i++) {
    const line = lines[i].trim();
    if (line === "ADJUSTMENT" || line === "ACTUALS") break;
    if (line.endsWith(":")) { group = line; continue; }
    if (line.startsWith("- ")) items.push({ group, text: line.slice(2).trim() });
  }
  return items;
}

export function buildWorkoutLog({ day, readiness, hipBefore, hipAfter, adjustment, completed, skipped, notes }) {
  const parts = [`Workout log. ${day}.`];
  if (adjustment && adjustment !== "100") parts.push(`Ran at ${adjustment}%.`);
  if (readiness) parts.push(`Readiness ${readiness}%.`);
  if (hipBefore || hipAfter) parts.push(`Hip before ${hipBefore || "?"}/10, after ${hipAfter || "?"}/10.`);
  if (completed.length) parts.push(`Completed: ${completed.join("; ")}.`);
  if (skipped.length) parts.push(`Skipped: ${skipped.join("; ")}.`);
  if (notes) parts.push(`Notes: ${notes}`);
  return parts.join(" ");
}
