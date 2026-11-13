---
version: 1
slug: "frontend-src-pages-authpage-jsx"
primary_target: "frontend/src/pages/AuthPage.jsx"
related_targets: ["frontend/src/App.jsx","frontend/src/components/SiteHeader.jsx"]
---

## Surface

Scope: Login and registration form previews with a shared sign-in entry point.
Visitor mode: Operate.

## Audience and action

Visitors navigate to login or registration, enter their details, and see an
honest unavailable message until the backend account contract is ready.

## Constraints

- Preserve the shared dark-red design and existing navigation.
- Use a simple centered form inspired by the user's screenshots.
- Login uses email and password; registration also asks for a display name.
- Forms do not send credentials, authenticate, or claim success yet.
- Keep forms usable with keyboard and at mobile widths.

## Direction contract

THESIS: A quiet account form puts the next action ahead of decoration.
OWN-WORLD: Obsidian, burgundy edges, ruby actions, and the site's Thai type.
STORY: The visitor can move between forms and understand that accounts await the backend.
FIRST VIEWPORT: Shared header with sign-in at the right; a narrow centered form below,
with a heading, stacked labeled inputs, primary action, and a link to the other form.
FORM: User-provided login and registration references; direct code-led implementation.
FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance
