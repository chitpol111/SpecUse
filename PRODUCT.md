# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

Thai gamers and creators who want to choose PC components suited to their
intended use and budget. This also includes people who need help checking
component compatibility and staying within budget.

Guests can browse, search, and filter active products, create and share a
temporary build, view public builds, and register or sign in. Members can
manage their profile and password, save multiple named builds with
descriptions, set visibility, and edit or delete only their own builds. Admins
maintain product data used by the catalog and compatibility checks.

## Product Purpose

SpecUse helps people select and assemble a PC specification for their needs
and budget. Users can review the total price and basic component compatibility
before deciding what to buy. Success means users can make a more informed
build choice and save or share it when useful.

## Positioning

The service guides PC building by intended use and budget, with current catalog
prices, total-cost visibility, and basic compatibility checks in the same
workflow.

## Operating Context

Users choose and compare components, then may share a build with others in
communities such as LINE or Facebook. The experience must work particularly
well on smartphones; mobile-first design is a requirement.

## Capabilities and Constraints

- The initial compatibility checks compare CPU socket with motherboard CPU
  socket, and RAM type with motherboard RAM type. The backend is authoritative
  for compatibility, prices, and permissions.
- Prices use Thai Baht (฿), formatted for Thai users (for example, `25,000 ฿`).
- The user-facing experience must support Thai natively.
- Builds can record required and already-owned quantities. Members choose
  public or private visibility; guests can view only public builds.
- RAM purchasing uses active single- or paired-module products. The backend
  selects a combination that meets the required module count exactly at the
  lowest price when possible; otherwise it minimizes excess modules before
  minimizing price.
- Frontend: React, JavaScript/JSX, Create React App with CRACO, React Router,
  Tailwind CSS, and Fetch API with `async`/`await`.
- Backend: Go REST API using JSON. Supabase provides authentication and data
  storage; database credentials stay on the backend. The backend authorizes
  protected operations, including admin operations.
- Frontend deployment is Vercel; backend deployment is Render.
- The first frontend milestone is a responsive landing page. Build editing,
  member build management, and admin product management are separate pages and
  outside that milestone. Initial landing-page examples may use sample data
  without an API client.
- GPU, storage, PSU, case, and cooler fields and filters remain undecided.
  Final Supabase RLS/schema constraints and deterministic RAM tie-breaking
  behavior also remain undecided.
- Account deletion retains builds without identifying the former owner and
  makes them private.

## Brand Commitments

- Product name: SpecUse (SU SpecUse).
- User-stated preference: prioritize a sleek dark-mode interface.

## Evidence on Hand

The current landing page includes clearly labeled sample build ideas and
explicitly states they are not member builds or real user reviews. No real
testimonials or live product catalog are established in the project materials;
do not fabricate either.

## Product Principles

- Fit component choices to the user's intended work, play, and budget.
- Make total cost and the scope of compatibility checks clear before purchase.
- Keep sharing, privacy, and ownership behavior explicit.
- Treat product data and permissions as backend-authoritative.
- Represent sample content honestly; do not imply it is real user evidence.
