---
version: 1
slug: "frontend-src-pages-buildpage-jsx"
primary_target: "frontend/src/pages/BuildPage.jsx"
related_targets: ["frontend/src/api/specUseApi.js","frontend/src/components/SiteHeader.jsx","frontend/src/App.jsx","frontend/src/index.css"]
---

## Surface

Scope: A responsive component catalog, live build summary, and unsaved validation flow that calls the Go API.
Visitor mode: Operate.

## Audience and action

Thai PC planners browse components by category, add products to a draft, check compatibility, and review a line-item list with quantities, discount status, and backend-computed additional cost.

## Proof and content

The backend is authoritative for compatibility and price calculations. The current backend source is a local fixture; visibly identify fixture data and clarify that the page neither saves a build nor initiates a purchase.

## Constraints

- Read products from `GET /api/v1/products`; preserve server product UUIDs, typed specs, and `ram_model_id`.
- Send only contract fields to `POST /api/v1/builds/validate`; never submit prices or client-computed compatibility.
- Cancel stale validation requests when selections change; the backend remains authoritative for compatibility and additional cost.
- Show loading, empty, retry/error, validating, incompatible, and compatible states; show a category-specific confirmation when a product is added.
- A draft can be incomplete. Validation requires a selected CPU and motherboard; RAM is optional.
- Discount, saving, sharing, and purchase capabilities are not present in the current API; make their unavailable state explicit rather than implying those actions work.
- The fixture currently supports CPU, motherboard, and RAM only. Other component categories stay visible but unavailable until the catalog and validation API support them.
- Inherit the SpecUse obsidian/ruby visual world, responsive patterns, Thai typography, and accessible focus treatment.
