# SU SpecUse — Project Specification

## Project overview

SU SpecUse is a Thai-language website that helps people choose and assemble a
computer specification suited to their intended use and budget. It is a
computer-building service, not a movie website.

The website should help users select components, see the total price, and check
basic component compatibility before deciding what to buy.

## Target users

- People who want to build a computer suitable for their own needs.
- People who need help choosing compatible components and staying within a
  budget.

## Roles

- **Guest:** Browse, search, and filter active products; create and share a
  temporary build; view public member builds; register and sign in.
- **Member:** Manage their profile and password; save multiple named builds
  with descriptions; choose public or private visibility; edit or delete only
  their own builds. A build records the quantity needed and quantity already
  owned.
- **Admin:** Add, edit, and deactivate products, and maintain product data used
  for compatibility checks. The Go API must authorize admin operations.

Members may view public builds and their own private builds. Guests may view
only public builds. Deleting a member account retains its builds without
identifying the owner and makes those builds private.

## Technology and deployment

- Frontend: React with JavaScript/JSX, Create React App, React Router, and
  Fetch API using `async`/`await`.
- Styling: Tailwind CSS integrated with Create React App through CRACO, as
  approved for this project.
- Backend: Go REST API using JSON.
- Database and authentication: Supabase. The Go backend connects to Supabase;
  database credentials must never be exposed in the frontend.
- Frontend deployment: Vercel.
- Backend deployment: Render.

## Product and build data

The proposed core data model is:

- `profiles`: Supabase Auth user reference, display name, and `member` or
  `admin` role. Registration must not allow a user to assign themselves the
  admin role. Passwords are managed by Supabase Auth.
- `products`: shared product information, category, current THB price, and
  active status. Deactivation is preferred over deleting products referenced
  by saved builds.
- Category-specific product tables: typed fields for compatibility, including
  CPU socket, motherboard CPU socket and RAM type, and RAM type.
- `ram_models`: shared characteristics for a RAM model.
- `product_rams`: sellable single-module or paired-module RAM options, each
  linked to a RAM model and carrying its own current product price.
- `builds`: owner, name, description, and public/private visibility.
- `build_items`: non-RAM build requirements, quantity, and quantity already
  owned.
- `build_ram_items`: required and already-owned RAM module counts, linked to a
  RAM model. The backend determines which active single- or paired-module
  products to buy.

Prices use THB and current catalog prices. For RAM, select a combination that
first meets the required module count exactly, then has the lowest total price.
If an exact combination is unavailable, minimize excess modules and then
choose the lowest total price. RAM packaging and price ties must be handled
consistently by the backend.

The initial compatibility checks compare CPU socket with motherboard CPU
socket, and RAM type with motherboard RAM type. The backend is authoritative
for prices, compatibility, and permissions; frontend-submitted prices or
compatibility results are not trusted.

## API outline

REST endpoints use the `/api/v1` prefix and JSON:

- `GET /products`, `GET /products/:id`
- `POST /builds/validate`
- `GET /builds/public`, `GET /builds/public/:id`
- `/auth/register`, `/auth/login`
- `/me`, `/me/password`, `/me/builds`
- `/admin/products`

Guest build sharing uses query-string data and does not require a temporary
share-link API. Initial logout removes the locally stored token.

The Go backend verifies authentication and performs authorization for each
endpoint. It uses Supabase service credentials only on the backend, never in
the frontend. Admin authorization is checked by the backend against trusted
role data.

## Landing page scope

The first frontend milestone is a responsive Landing Page with:

1. Top navigation: logo, home, build a PC, public builds, and sign-in/profile.
2. Hero explaining builds by usage and budget, with a “เริ่มจัดสเปก” call to
   action.
3. User problems: uncertainty choosing parts, compatibility concerns, and
   budget control.
4. Service benefits: total-price calculation, basic compatibility checks, and
   saving/sharing builds.
5. Service details for Guest and Member features.
6. Usage steps: choose components, check compatibility, review the price, then
   save or share.
7. Example public builds. Clearly label sample content; do not invent user
   reviews or imply sample builds are real member submissions.
8. Frequently asked questions.
9. A call to action to start building or register.
10. Footer with project information and useful links.

The current landing page is a prototype. The build editor, compatibility
checks, product catalog and prices, accounts, saved builds, and sharing are not
available yet. Describe these as planned functionality; sample pathways are
informational examples only.

The next frontend milestone is an unsaved build-validation page that reads
active products from `GET /api/v1/products` and sends selected CPU,
motherboard, and optional RAM requirements to `POST /api/v1/builds/validate`.
It uses the local Backend fixture during development; it does not save builds
or represent fixture prices as live purchase prices.

The build editor, member build management, and admin product management are
separate pages and are outside the first Landing Page milestone. The initial
Landing Page may use clearly identified sample data without an API client.

## Design direction

- A sleek, premium, minimalist dark-red interface with obsidian and deep
  burgundy surfaces, a subtle crimson grid/mesh, and multi-tone ruby accents.
- Use translucent burgundy glass panels with fine crimson borders and restrained
  red glow on hover; keep body text crisp and high-contrast.
- Highlight “เริ่มจากสเปกที่ใช่” with the requested crimson gradient and use ruby
  red for primary actions and active states.
- Responsive layouts for desktop and mobile, readable typography, clear
  headings, and generous spacing.
- Thai-language user-facing text; no Lorem Ipsum.
- Do not display fabricated testimonials.

## Proposed repository layout

```text
spec.md
frontend/
  package.json
  public/
    index.html
  src/
    index.js
    App.jsx
    index.css
    pages/
      HomePage.jsx
    components/
      SiteHeader.jsx
      HeroSection.jsx
      ProblemSection.jsx
      BenefitsSection.jsx
      ServiceSection.jsx
      StepsSection.jsx
      PublicBuildsSection.jsx
      FAQSection.jsx
      CallToAction.jsx
      SiteFooter.jsx
backend/
  # Define later according to the Go structure and methods taught in the course.
```

The initial implementation should inspect the repository before creating files,
reuse existing configuration where present, and avoid Vite. Add API client,
backend implementation, and database schema management only when those
milestones are approved.

## Open decisions

- Exact typed fields and filters for GPU, storage, PSU, case, and cooler.
- Deterministic behavior when multiple RAM package combinations have the same
  total price.
- The final Supabase RLS configuration and database schema constraints.
