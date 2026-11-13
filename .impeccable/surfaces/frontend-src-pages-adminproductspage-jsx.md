---
version: 1
slug: "frontend-src-pages-adminproductspage-jsx"
primary_target: "frontend/src/pages/AdminProductsPage.jsx"
related_targets: ["frontend/src/App.jsx","frontend/src/components/SiteHeader.jsx","frontend/src/api/authApi.js"]
---

## Surface

Scope: Admin product catalog maintenance. Visitor mode: Operate.

## Audience and action

An authorized admin searches, filters, creates, edits, and deactivates catalog products.
The backend remains the authority for role checks, validation, prices, and persistence.

## Constraints

- Follow the assigned split list-and-form layout, adapting its palette to SpecUse.
- Use only fields/endpoints in the admin API contract; never allow role changes.
- Keep category immutable on edit; deactivation is reversible through active status.
- Show loading, empty, error, disabled, confirmation, and success states.
- Keep the table readable on desktop and convert rows to a usable mobile list.

## Direction contract

THESIS: Searchable catalog rows and an adjacent editor keep maintenance in view.
OWN-WORLD: SpecUse obsidian ground, burgundy surfaces, ruby actions, Thai type.
STORY: Admin narrows the catalog, edits typed data, and can deactivate without deletion.
FIRST VIEWPORT: Header and title lead; filters and rows occupy the left, the editor
the right; on narrow screens the list and form stack with filters kept visible.
FORM: Assigned split layout, code-led implementation; seed:e5a7f431dbf09f61.
FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance
