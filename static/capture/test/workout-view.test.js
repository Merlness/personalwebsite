import { test } from "node:test";
import assert from "node:assert/strict";
import { extractPhoneCard, parseLedgerDays, parseProgramDays, weekAhead } from "../js/workout-view.js";

const TODAY_WORKOUT = `# Today Workout

Last updated: 2026-06-09

## Status

Active workout selected: Day 2 - Deceleration and Structure

Calendar event: not yet created

## Phone Card

WORKOUT

Day: Day 2 - Deceleration and Structure

PLAN

Warm-up, 6-8 min:

- Elliptical, 4 min

## Quick Start

Ask any assistant.
`;

const LEDGER = `# Workout Ledger

## Logging Format

\`\`\`md
## YYYY-MM-DD | Day/Session Name
\`\`\`

## Entries

## 2026-06-08 | Day 1: Absolute Force

| Exercise | Sets x Reps | Load | RPE | Notes |
|---|---:|---|---:|---|
| Chest Press Machine | 3 x 12 | 130 lb | 8-9 | ok |

## 2026-06-03 | Basketball

Duration: 75 minutes
`;

const PROGRAM = `# Workout Program

## Day 1: Absolute Force

stuff

## Day 2: Deceleration And Structure

stuff

## Day 3: Elasticity And Twitch

stuff

## Day 4: Armor And Gas Tank

stuff

## Optional Day 5: Mobility, Yoga, Or Accessories

stuff
`;

test("extractPhoneCard returns the card block and the active day", () => {
  const { activeDay, card } = extractPhoneCard(TODAY_WORKOUT);
  assert.equal(activeDay, "Day 2 - Deceleration and Structure");
  assert.ok(card.startsWith("WORKOUT"));
  assert.ok(card.includes("Elliptical, 4 min"));
  assert.ok(!card.includes("Quick Start"), "stops at the next section");
});

test("parseLedgerDays reads dated entries, skipping the format example", () => {
  const days = parseLedgerDays(LEDGER);
  assert.deepEqual(days, [
    { date: "2026-06-08", name: "Day 1: Absolute Force", day: 1 },
    { date: "2026-06-03", name: "Basketball", day: null },
  ]);
});

test("parseProgramDays reads the day labels including optional", () => {
  const days = parseProgramDays(PROGRAM);
  assert.deepEqual(days, [
    { day: 1, name: "Absolute Force", optional: false },
    { day: 2, name: "Deceleration And Structure", optional: false },
    { day: 3, name: "Elasticity And Twitch", optional: false },
    { day: 4, name: "Armor And Gas Tank", optional: false },
    { day: 5, name: "Mobility, Yoga, Or Accessories", optional: true },
  ]);
});

// Week of Mon 2026-06-08. Day 1 done Mon. Today Wed 2026-06-10.
// Remaining required: 2, 3, 4 -> suggested today, +1, +2. Day 5 optional after.
test("weekAhead schedules remaining required days from today", () => {
  const plan = weekAhead({ ledgerMd: LEDGER, programMd: PROGRAM, today: "2026-06-10" });
  assert.deepEqual(plan, [
    { date: "2026-06-08", day: 1, name: "Absolute Force", status: "done" },
    { date: "2026-06-10", day: 2, name: "Deceleration And Structure", status: "planned" },
    { date: "2026-06-11", day: 3, name: "Elasticity And Twitch", status: "planned" },
    { date: "2026-06-12", day: 4, name: "Armor And Gas Tank", status: "planned" },
    { date: "2026-06-13", day: 5, name: "Mobility, Yoga, Or Accessories", status: "optional" },
  ]);
});

test("weekAhead marks days completed twice only once and keeps later dates", () => {
  const ledger = LEDGER + `
## 2026-06-10 | Day 2: Deceleration And Structure

notes
`;
  const plan = weekAhead({ ledgerMd: ledger, programMd: PROGRAM, today: "2026-06-10" });
  const day2 = plan.find((p) => p.day === 2);
  assert.equal(day2.status, "done");
  assert.equal(day2.date, "2026-06-10");
  const day3 = plan.find((p) => p.day === 3);
  assert.equal(day3.date, "2026-06-11", "next planned day starts tomorrow when today is done");
});

test("weekAhead ignores sessions from previous weeks", () => {
  const plan = weekAhead({ ledgerMd: LEDGER, programMd: PROGRAM, today: "2026-06-17" });
  assert.equal(plan.filter((p) => p.status === "done").length, 0);
  assert.equal(plan[0].day, 1);
  assert.equal(plan[0].date, "2026-06-17");
});

test("weekAhead clamps suggestions to the current week", () => {
  // Friday with nothing done: 4 required days but only Fri/Sat/Sun left.
  const plan = weekAhead({ ledgerMd: "# Workout Ledger\n## Entries\n", programMd: PROGRAM, today: "2026-06-12" });
  const dates = plan.map((p) => p.date);
  assert.ok(dates.every((d) => d <= "2026-06-14"), `all within week: ${dates}`);
  const day4 = plan.find((p) => p.day === 4);
  assert.equal(day4.date, "2026-06-14", "overflow days stack on the last day of the week");
});
