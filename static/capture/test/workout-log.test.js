import { test } from "node:test";
import assert from "node:assert/strict";
import { parsePlanItems, buildWorkoutLog } from "../js/workout-log.js";

const CARD = `WORKOUT

Day: Day 2 - Deceleration and Structure

PLAN

Warm-up, 6-8 min:

- Elliptical, 4 min
- Glute bridge, 1 x 12

Block A, posterior chain:

- A1. DB Romanian deadlift, 4 x 6, 3-second lowering, rest 90s. Start 40-45 lb DBs.
- A2. Cable row, 4 x 8, rest 75s

Finisher:

- Stability ball leg curl, 3 x 10-12

ADJUSTMENT

- 100% = planned session
- 80% = reduce one accessory set

ACTUALS

Readiness:

Notes:
`;

test("parsePlanItems lists exercises between PLAN and ADJUSTMENT, grouped", () => {
  const items = parsePlanItems(CARD);
  assert.deepEqual(items.map((i) => i.group), ["Warm-up, 6-8 min:", "Warm-up, 6-8 min:", "Block A, posterior chain:", "Block A, posterior chain:", "Finisher:"]);
  assert.equal(items[2].text, "A1. DB Romanian deadlift, 4 x 6, 3-second lowering, rest 90s. Start 40-45 lb DBs.");
  assert.ok(!items.some((i) => i.text.includes("100%")), "adjustment bullets excluded");
});

test("parsePlanItems handles a card with no PLAN section", () => {
  assert.deepEqual(parsePlanItems("WORKOUT\n\nno plan today"), []);
});

test("buildWorkoutLog composes a complete natural-language entry", () => {
  const raw = buildWorkoutLog({
    day: "Day 2 - Deceleration and Structure",
    readiness: "80",
    hipBefore: "2",
    hipAfter: "3",
    adjustment: "80",
    completed: ["Elliptical, 4 min", "A1. DB Romanian deadlift, 4 x 6"],
    skipped: ["Stability ball leg curl, 3 x 10-12"],
    notes: "RDLs felt clean",
  });
  assert.equal(
    raw,
    "Workout log. Day 2 - Deceleration and Structure. Ran at 80%. Readiness 80%. Hip before 2/10, after 3/10. Completed: Elliptical, 4 min; A1. DB Romanian deadlift, 4 x 6. Skipped: Stability ball leg curl, 3 x 10-12. Notes: RDLs felt clean"
  );
});

test("buildWorkoutLog omits empty fields cleanly", () => {
  const raw = buildWorkoutLog({ day: "Day 1", completed: ["Bench"], skipped: [], readiness: "", hipBefore: "", hipAfter: "", adjustment: "100", notes: "" });
  assert.equal(raw, "Workout log. Day 1. Completed: Bench.");
});
