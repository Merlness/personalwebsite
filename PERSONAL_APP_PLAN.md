---
tags: [personal-website, capture-pwa, personal-api, planning]
created: 2026-07-16
---

# Personal App Plan: real-time, one sign-in, and a new look

Turn merlmartin.com and the capture PWA into a real personal app. Sign in once with Google and never type a PIN again. Say "change my workout to X" and it changes now, not on the next scheduled routine. All keys live server-side. Then refresh the look with three candidate designs Merl picks from.

Craftsmanship level (Merl, 2026-07-16): this is a personal project. No TDD relay. Tests for the major things (auth, agent dispatch, pure logic), keep existing CI green, skip ceremony. Claude builds it and launches it.

## What exists today (audited 2026-07-16)

- **Public site**: Go SSG (templ + Tailwind) built in CI, deployed to GitHub Pages at merlmartin.com. Pages: home, about, blog, portfolio, enki (support / privacy / delete-account), capture. Light mode is white + ink; dark mode is #0a0a0a with a cyan #00E5FF accent, plus marigold #FF9800 / #FF6F00, pink-hot #E91E63, purple-deep #4A148C. Inter + Merriweather.
- **Capture PWA** (static/capture): tabs for Capture, Tasks, Workout, LinkedIn, Cards. Writes markdown straight into `Merlness/life-organizer` via the GitHub Contents API **from the browser**, using a GitHub PAT plus a Gemini key (capture classification, LinkedIn drafts) held in a PIN-encrypted vault in localStorage (PBKDF2 + AES-GCM). Offline read cache and write queue. Tested JS modules run in CI.
- **capture-api proxy** (cmd/capture-api, internal/captureapi, fly.capture-api.toml): written 2026-07-09 to move the GitHub token server-side. The client already supports `apiBase`. **Never deployed**: no such Fly app exists. Fly account is authed as merl@bennusystems.com; existing apps are Enki's, in org `alex-jensen-277`.
- **Processing loop**: scheduled Claude routines pull life-organizer and process inbox/captures later. Nothing is real-time except the file writes the PWA does itself.

So today the PIN exists only to protect browser-held secrets, and "real-time" stops at raw file writes. Both problems have the same fix: a small server that holds the keys and the brains.

## Target architecture

```
merlmartin.com (GitHub Pages, static, free)
  public site + PWA shell.  Zero secrets in the client.
        |
        |  fetch with session cookie (same-site subdomain)
        v
api.merlmartin.com  =  merl-personal-api (Go on Fly, scale-to-zero, phx)
  /auth/*     Google OAuth, allowlist of Merl's emails, long-lived HttpOnly cookie
  /files/*    GitHub Contents ops pinned to Merlness/life-organizer (today's proxy, renamed)
  /agent      Anthropic API (Go SDK) with tools over life-organizer -> real-time edits
  /capture    dumb fast-path save (no LLM), + server-side classification
  Secrets (Fly): GITHUB_TOKEN, ANTHROPIC_API_KEY, GOOGLE_CLIENT_ID/SECRET, SESSION_KEY
        |
        v
Merlness/life-organizer (GitHub repo stays the database)
  tasks.md, inbox.md, pulse/workout-*.md, drafts/cards/inbox, contexts/
  Scheduled routines keep running for digests; no longer the only write path.
```

## Decisions locked

1. **Auth is Google OAuth, single user.** Allowlist: mmartin777@gmail.com and merl@bennusystems.com. Successful sign-in issues a signed HttpOnly Secure cookie, ~180 days. Why: "sign in once like OAuth," no client-held secrets, same pattern Merl already shipped for Enki Android. The PIN vault gets deleted, not improved.
2. **Every key moves server-side** (GitHub PAT, Anthropic, Google client secret). Why: PWA JavaScript is public; browser-held keys are extractable; Fly secrets already proven with Enki.
3. **API lives at api.merlmartin.com.** Why: same registrable domain as the site, so cookies are same-site and survive Safari/Chrome third-party-cookie blocking. Needs one DNS CNAME plus `fly certs add`.
4. **Public site stays on GitHub Pages.** Why: it is static content, Pages is free and already wired. App-ness lives in the API and the PWA, not in server-rendered pages.
5. **Gemini goes away; Claude does it server-side.** Capture classification on Haiku (fractions of a cent), LinkedIn drafts on Sonnet with Merl's voice rules in the system prompt. Why: one brain, one bill, no second exposed key.
6. **Fast paths stay dumb.** A plain capture save is a direct file write, instant and free, still offline-queueable. The agent is for commands, edits, drafts, and questions. Why: never make a $0 operation cost tokens or latency.
7. **Request-scoped work only** on the API. Scale-to-zero kills background goroutines (Enki incident, 2026-07-14). Anything long-running later gets a durable pattern, not a goroutine.

## Phases

### P1. Deploy the API and agent v1 (real-time proof, no auth changes yet)

**STATUS 2026-07-16: SHIPPED, pending Merl's three secrets.** Deployed at https://merl-personal-api.fly.dev (commits `0e21f70` + `91a631b`, pushed; Pages redeploys the PWA with the Ask tab). Notes: Merl's personal Fly org trial has ended, so the app lives in the Bennu org (`alex-jensen-277`) for now, move later with `fly apps move merl-personal-api --org personal` after adding a card. Fly deprecated `phx` for new resources; the app runs in `lax`. The service boots degraded (healthz + 503) until GITHUB_TOKEN, APP_TOKEN, ANTHROPIC_API_KEY are set. A leftover local capture-api dev process from Jul 9 still listens on port 8199 on Merl's Mac; kill it whenever.
- Create the Fly app (name `merl-personal-api`, Merl's personal org, phx, scale-to-zero) from the existing proxy code. Rename configs from capture-api.
- Merl sets secrets: fine-grained GitHub PAT (life-organizer, contents read/write) and ANTHROPIC_API_KEY.
- Flip the PWA's `apiBase` to the deployed URL so the browser stops talking to api.github.com. Vault now only guards the app token (interim).
- Add `POST /agent`: anthropic-sdk-go, model Haiku default, tools `list_dir` / `read_file` / `write_file` pinned to life-organizer paths (tasks.md, inbox.md, pulse/, drafts/), system prompt with Merl profile, Phoenix date, file map, voice rules (no em dashes).
- Minimal omnibox bar in the PWA (type a command, see the reply, UI refetches touched files).
- Tests: proxy path pinning (exists), agent tool dispatch, write-path allowlist.
- **Acceptance: from the PWA, "swap today's workout to upper body" edits pulse/today-workout.md and the workout tab shows it, in one interaction, in seconds.**

### P2. Google sign-in, kill the PIN

**STATUS 2026-08-14: code shipped, waiting on Merl's Google OAuth client and the DNS record.** `internal/auth` (login, callback, logout, `/auth/me`, signed 180-day cookie, email allowlist) is committed and tested. The API authorizes a request by *either* a valid session or the app token, and CORS now sends `Access-Control-Allow-Credentials`. The PWA boots session-first: a live session starts the app with no prompt and no credential on the device; with no session it shows one "Sign in with Google" button. The PIN vault survives only as the fallback for the case where the API has no Google client configured, which is today. Once steps 3 and 4 below are done, the PIN stops appearing and the vault code can be deleted.
- `/auth/login` -> Google -> `/auth/callback`: verify ID token, check email allowlist, set signed session cookie (SESSION_KEY secret, 180d). `/auth/logout`. 403 for anyone else.
- Custom domain: DNS CNAME api.merlmartin.com -> the Fly app, `fly certs add`. CORS tightened to https://merlmartin.com with credentials.
- PWA: all fetches use `credentials: "include"`; on 401 show one "Sign in with Google" button. Delete the vault, PIN screens, and app-token setup. localStorage keeps only non-secret state (queue, cache, settings).
- Tests: allowlist, session verify/expiry, 401 flow.
- **Acceptance: fresh browser, one Google tap, everything works for months. Any other Google account gets a 403. The word PIN no longer appears in the codebase.**

### P3. Real-time features end to end

**STATUS 2026-08-14: SSE streaming shipped.** `/agent` streams when the client sends `Accept: text/event-stream`, emitting reply text as it is written, a step line before each tool call, and a written event per file. The Ask tab paints into one growing bubble with the current step under it. The app also refetches the active tab on focus, on window focus, and on reconnect, and dots the nav button for any tab whose files the agent changed. Still open below: server-side capture classification, the LinkedIn Sonnet draft path, and optimistic edits on the tasks and workout tabs.
- SSE streaming for /agent replies (feels instant).
- Capture saves classify server-side at write time (Haiku) instead of client-side Gemini; inbox entries land pre-tagged.
- LinkedIn tab: "draft it" runs a Sonnet tool with the write-as-merl voice rules; drafts persist until Merl deletes or marks posted (existing rule).
- Tasks and workout tabs get agent-powered edits with optimistic UI ("push leg day to Friday", "add protein powder to shopping").
- Conversation history stays client-side (last ~10 turns posted with each call), so the server stays stateless and scale-to-zero-safe.
- **Acceptance: capture, ask, edit, and draft all round-trip in seconds with the routines untouched.**

### P4. Redesign bake-off (runs any time, independent)
Three same-content mockups as artifacts, real copy from the live site, Merl picks one or mixes. Direction: personal, Latino flex, spice, zero AI-slop tells (no purple gradient hero, no glass cards, no emoji headers, no Inter-on-white sameness).

1. **Desert Editorial** (evolution, lowest risk): crema/sand light base, ink text, marigold #FF9800 kept as THE accent plus terracotta; a characterful display serif (Fraunces energy) over a clean body; current layout skeleton, better rhythm and spacing.
2. **Noche de Oro** (dark flex): espresso/charcoal warm dark (replaces the cyan-on-black), marigold gold + chili red accents, bold condensed display type, subtle papel-picado-inspired geometric dividers as SVG, restrained glow.
3. **Mercado Moderno** (boldest): poster/zine energy, big type, solid warm color blocks, thick rules and borders, hand-touch details, playful bilingual microcopy seasoning.

Then implement the winner in templ + Tailwind (public pages + PWA shell inherit the palette). Constraints: keep /enki/* URLs (App Store listing points at them), keep /capture, keep blog/portfolio content, keep dist/CNAME.
- **Acceptance: Merl says it feels like him, and nothing about it reads as generated.**

### P5. Acts-like-an-app polish
- Web Push (VAPID) so routines can ping: daily brief ready, reminder nudges. Service worker already exists.
- Icon/manifest refresh to match the new brand; install prompt polish.
- Optional: passkey as a secondary sign-in (Pixel fingerprint, MacBook Touch ID).

### P6. Cleanup
- Remove Gemini code path and client-side GitHub-direct mode. Merl revokes the old browser PAT and Gemini key. Rotate nothing else; document the runbook in README.

## Merl's manual steps (the complete list, ~10 minutes total)

1. **GitHub PAT**: create a fine-grained PAT scoped to `Merlness/life-organizer`, Contents read/write. Then `fly secrets set GITHUB_TOKEN=... -a merl-personal-api`.
2. **Anthropic key**: console.anthropic.com, set a monthly spend cap (even $10 is 10x headroom). Then `fly secrets set ANTHROPIC_API_KEY=... -a merl-personal-api`.
3. **Google OAuth client** (P2): GCP console, OAuth client ID, type Web, authorized origin `https://merlmartin.com`, redirect `https://api.merlmartin.com/auth/callback`. Then `fly secrets set GOOGLE_CLIENT_ID=... GOOGLE_CLIENT_SECRET=... -a merl-personal-api`.
4. **DNS** (P2): add CNAME `api.merlmartin.com` -> `merl-personal-api.fly.dev` at the registrar.
5. **Pick the design** (P4): choose a mockup or name the mix.

Claude cannot paste keys for you (safety rule), so steps 1-3 are literally you running one command each; everything else is Claude.

## Cost

- Fly: shared-cpu-1x 256MB, scale-to-zero, one small personal app: roughly $0-3/month. Cold start after idle is 1-2 seconds; if that ever annoys, `min_machines_running = 1` is about $2/month more.
- Claude: Haiku commands well under a cent each; Sonnet LinkedIn drafts a cent or three. Realistic personal volume: under $5/month total.

## Risks and notes

- **Scale-to-zero**: request-scoped work only (decision 7). No fire-and-forget goroutines.
- **GitHub as database** is fine at personal scale (5k requests/hour limit, single user, optimistic-concurrency retry already in the client).
- **Offline**: unchanged model. Reads fall back to cache, writes queue and flush. Session cookie means offline needs no unlock at all.
- **Timing**: Enki release day is 2026-07-17 and outranks this. Start P1 after release-day tasks settle.

Related: [[CARDS_FEATURE_PLAN]] (shipped), FRONTEND_REVIEW.md and FULL_REVIEW.md (older audits; the bake-off supersedes their cosmetic items).
