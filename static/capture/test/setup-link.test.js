import { test } from "node:test";
import assert from "node:assert";
import { parseSetupFragment } from "../js/setup-link.js";

const frag = (obj) => "#setup=" + Buffer.from(JSON.stringify(obj)).toString("base64");

test("parses a valid setup fragment", () => {
  const got = parseSetupFragment(frag({ apiBase: "https://merl-personal-api.fly.dev", token: "abc123" }));
  assert.deepStrictEqual(got, { apiBase: "https://merl-personal-api.fly.dev", token: "abc123" });
});

test("strips trailing slashes from the api base", () => {
  const got = parseSetupFragment(frag({ apiBase: "https://x.fly.dev//", token: "t" }));
  assert.strictEqual(got.apiBase, "https://x.fly.dev");
});

test("accepts base64url variants", () => {
  const b64 = Buffer.from(JSON.stringify({ apiBase: "https://x.fly.dev", token: "t" }))
    .toString("base64").replace(/\+/g, "-").replace(/\//g, "_");
  assert.ok(parseSetupFragment("#setup=" + b64));
});

test("rejects non-https api bases", () => {
  assert.strictEqual(parseSetupFragment(frag({ apiBase: "http://evil.test", token: "t" })), null);
});

test("rejects missing token, garbage, and unrelated fragments", () => {
  assert.strictEqual(parseSetupFragment(frag({ apiBase: "https://x.fly.dev" })), null);
  assert.strictEqual(parseSetupFragment("#setup=%%%not-base64%%%"), null);
  assert.strictEqual(parseSetupFragment("#setup=aGVsbG8="), null); // valid base64, not our JSON
  assert.strictEqual(parseSetupFragment("#other"), null);
  assert.strictEqual(parseSetupFragment(""), null);
  assert.strictEqual(parseSetupFragment(null), null);
});
