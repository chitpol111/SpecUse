---
version: 1
slug: "frontend-src-pages-profilepage-jsx"
primary_target: "frontend/src/pages/ProfilePage.jsx"
related_targets: ["frontend/src/App.jsx","frontend/src/components/SiteHeader.jsx"]
---

## Surface

Scope: Member profile and password management. Visitor mode: Operate.

## Audience and action

Members view their account, update their display name, and change their password.
Guests are directed to sign in. Email and role are read-only.

## Constraints

- Follow the user's profile reference without commerce metrics or unsupported fields.
- Use the existing obsidian, burgundy, ruby, and Thai typography system.
- Backend remains authoritative; show loading, success, and actionable error states.
- Preserve keyboard and mobile access; no sample account data in the shipped page.

## Direction contract

THESIS: An account overview keeps identity and security distinct but on one page.
OWN-WORLD: Obsidian ground, quiet burgundy surfaces, restrained ruby actions.
STORY: Members see their account details, then edit their name or password inline.
FIRST VIEWPORT: Header, account heading, profile summary and detail rows on the left,
with password controls alongside on desktop and below the profile on mobile.
FORM: User-provided profile reference adapted directly to the existing visual world.
FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance
