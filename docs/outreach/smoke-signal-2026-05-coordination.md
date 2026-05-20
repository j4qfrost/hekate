# Draft: Smoke Signal coordination thread (M-1)

Pre-publication draft for the M-1 outreach mandated by `docs/adr/0002-smoke-signal-relationship.md`. Edit before sending; **the maintainer must post this themselves** — Claude Code does not have credentials for the Smoke Signal discourse or GitHub org.

- **Target forum (primary):** https://discourse.smokesignal.events/
- **Target forum (fallback):** GitHub Discussions on https://github.com/SmokeSignal-Events
- **Originally due:** 2026-05-06 (M-1 window opened 2026-04-29). **Today is 2026-05-19 → 13 days overdue.** Please post; the `app.hekate.event` and `app.hekate.rsvp` lexicons remain unfrozen until this resolves.

---

## Subject

Coordination: venue-first event lexicon proposal (hekate) and `event`/`rsvp` overlap with Smoke Signal

## Body

Hi Smoke Signal maintainers — I'm building **hekate** (https://github.com/j4qfrost/hekate), a venue-first AT Protocol event coordinator. Before publishing the `app.hekate.event` and `app.hekate.rsvp` lexicons I want to coordinate with you, since those primitives overlap with what Smoke Signal already ships.

**hekate's model.** Five records under `app.hekate.*`:

- `venue` — a place (PostGIS-indexed; capacity, booking policy, contact, amenities)
- `recurrence` — RRULE + duration → derives slots
- `slot` — a bookable time-window at a venue, authored by the venue owner
- `event` — an organizer's claim on a slot (the slot's `claimedBy` is filled by the indexer, not the client, with W3 "earliest createdAt wins" semantics on multi-claims)
- `rsvp` — going / maybe / declined per event

Today the lexicons are at `lexicons/app/hekate/` and the spec is at `docs/SPEC.md`. The Go indexer + Go CLI are the v0.1 reference client; a SvelteKit web client is M3 and off the critical path. Roadmap: `docs/ROADMAP.md`.

**Where we overlap with Smoke Signal.** The `event` and `rsvp` primitives are conceptually close to Smoke Signal's `events.smokesignal.calendar.event` (and RSVP). The `venue`, `recurrence`, and `slot` records are novel: Smoke Signal currently denormalises venue info into the event, and hekate makes it a first-class record so an organizer's event is a *claim* against an authoritative slot rather than a self-contained calendar entry.

**The three options we are weighing (`docs/adr/0002-smoke-signal-relationship.md`):**

- **A. Coexist with divergent specs.** hekate ships its own `event` + `rsvp`; Smoke Signal ships theirs; indexers ignore the other side's records. Easy to do; loses interop.
- **B. hekate as venue-only.** hekate ships `venue`/`slot`/`recurrence`; events and RSVPs are delegated to Smoke Signal's existing lexicons, *if* Smoke Signal accepts an extension field for a `slot` AT-URI reference. Maximum interop; depends on you.
- **C. Joint `community.lexicon.event.*`** that both projects implement. Best long-term outcome; slowest; needs real agreement.

**What I'm asking.**

1. Is option B viable from your side? Specifically: would Smoke Signal accept an optional `slot` field (AT-URI to an external venue-first slot) on your event record? This is a one-line addition with no breakage for existing users — Smoke Signal indexers ignore the field; hekate indexers use it to enforce the W3 claim semantics.
2. If B is not viable: are you open to option C (joint namespace), and if so, would you prefer to bootstrap that off Smoke Signal's existing event schema or off a fresh draft?
3. If neither B nor C: I will document **A (coexist)** as the recorded decision, with a public note that coordination was attempted.

I'd like to set a two-week window for response — **closing 2026-06-02**. I'll write the recorded decision into ADR 0002 the day after the window closes (default to A on silence).

Happy to hop on a call, jump to GitHub Discussions, or work async in this thread — whichever you prefer.

---

## Notes for the maintainer (do not include in posted thread)

- After posting, link the thread URL in `docs/adr/0002-smoke-signal-relationship.md` under Status — replace `Pending` with: `Awaiting response — thread opened 2026-05-19, window closes 2026-06-02, <thread-URL>`.
- Posting on 2026-05-19 puts the response window close at **2026-06-02** — *after* the M0 lexicon-freeze date (2026-05-29). The `event`/`rsvp` lexicons therefore cannot freeze on the M0 schedule; M0 slips for those two records. The `venue`/`slot`/`recurrence` lexicons are NOT blocked by this ADR (see ADR 0002 Consequences) and can still freeze on time. Do not ship `event`/`rsvp` from a non-test PDS without this resolution, per the hekate `CLAUDE.md`.
- If Smoke Signal responds with substantive feedback on the venue/slot/recurrence model, capture it in `docs/adr/0002-smoke-signal-relationship.md` regardless of whether it changes the decision.
