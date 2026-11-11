---
name: "SpecUse"
description: "A Thai-language PC-planning landing page in a restrained obsidian and ruby visual world."
colors:
  obsidian-ground: "#0b0202"
  obsidian-surface: "#120505"
  burgundy-glass: "rgba(35, 8, 10, 0.76)"
  ruby-primary: "#dc2626"
  ruby-light: "#f87171"
  crimson-deep: "#991b1b"
  text-primary: "#f8fafc"
  text-secondary: "#cbd5e1"
typography:
  display:
    fontFamily: "IBM Plex Sans Thai, Noto Sans Thai, Leelawadee UI, Tahoma, sans-serif"
    fontSize: "4.25rem"
    fontWeight: 700
    lineHeight: 1.12
    letterSpacing: "-0.035em"
  headline:
    fontFamily: "IBM Plex Sans Thai, Noto Sans Thai, Leelawadee UI, Tahoma, sans-serif"
    fontSize: "2.25rem"
    fontWeight: 700
    lineHeight: 1.2
  title:
    fontFamily: "IBM Plex Sans Thai, Noto Sans Thai, Leelawadee UI, Tahoma, sans-serif"
    fontSize: "1.125rem"
    fontWeight: 700
    lineHeight: 1.5
  body:
    fontFamily: "IBM Plex Sans Thai, Noto Sans Thai, Leelawadee UI, Tahoma, sans-serif"
    fontSize: "1rem"
    fontWeight: 400
    lineHeight: 2
  label:
    fontFamily: "IBM Plex Sans Thai, Noto Sans Thai, Leelawadee UI, Tahoma, sans-serif"
    fontSize: "0.75rem"
    fontWeight: 700
    lineHeight: 1.5
rounded:
  sm: "0.75rem"
  md: "1rem"
  lg: "1.5rem"
  pill: "9999px"
spacing:
  sm: "0.75rem"
  md: "1.5rem"
  lg: "2rem"
components:
  button-primary:
    backgroundColor: "{colors.ruby-primary}"
    textColor: "{colors.text-primary}"
    typography: "{typography.label}"
    rounded: "{rounded.pill}"
    padding: "1rem 1.75rem"
  button-secondary:
    backgroundColor: "{colors.obsidian-ground}"
    textColor: "{colors.text-primary}"
    typography: "{typography.label}"
    rounded: "{rounded.pill}"
    padding: "1rem 1.75rem"
  glass-panel:
    backgroundColor: "{colors.burgundy-glass}"
    textColor: "{colors.text-primary}"
    rounded: "{rounded.lg}"
    padding: "1.5rem"
---

# Design System: SpecUse

## Overview

**Creative North Star: "The Ruby Planning Desk"**

SpecUse's landing page feels like a considered workspace for planning a PC: dark, focused, and quietly technical without borrowing generic sci-fi ornament. Obsidian red-black creates the ground; burgundy translucency and fine crimson edges separate content while keeping the page calm. Thai copy and real component categories carry the story.

The page is a prototype. Its component preview and build pathways are illustrative; the editor, catalog, prices, checks, accounts, and saved builds are not represented as available functions. Red is reserved for navigation emphasis, primary actions, and the one requested headline phrase.

**Key Characteristics:**
- Obsidian and maroon fields with a quiet crimson grid.
- Crisp Thai typography using self-hosted IBM Plex Sans Thai.
- Burgundy glass panels with restrained borders and soft depth.
- Ruby actions and a phrase-only crimson headline treatment.

## Colors

The palette stays within red-black, burgundy, and ruby; light text is crisp and secondary copy remains legible.

### Primary
- **Ruby Primary** (#dc2626): Main calls to action and active controls.
- **Ruby Light** (#f87171): Focus outlines, icon accents, and hover emphasis.
- **Crimson Deep** (#991b1b): Dark end of the headline accent and deeper red details.

### Neutral
- **Obsidian Ground** (#0b0202): Main page background.
- **Obsidian Surface** (#120505): Alternate section background and dark controls.
- **Burgundy Glass** (rgba(35, 8, 10, 0.76)): Translucent panel surface.
- **Primary Text** (#f8fafc): Headings and high-emphasis content.
- **Secondary Text** (#cbd5e1): Body copy on dark surfaces.

### Named Rules
**The Ruby Is Intentional Rule.** Keep saturated red for primary actions, active details, and the exact hero phrase “เริ่มจากสเปกที่ใช่”; do not spread the gradient to other headings.

## Typography

**Display Font:** IBM Plex Sans Thai (with Noto Sans Thai, Leelawadee UI, Tahoma, sans-serif fallbacks)  
**Body Font:** IBM Plex Sans Thai (with the same fallback stack)

**Character:** Bold Thai display forms give the landing page a clear voice; regular body text keeps longer Thai explanations approachable. The primary font weights are self-hosted at 400 and 700.

### Hierarchy
- **Display** (700, 4.25rem desktop / 2.6rem compact, 1.12 line-height): Two-line hero statement.
- **Headline** (700, 2.25rem, 1.2 line-height): Section headings.
- **Title** (700, 1.125rem, 1.5 line-height): Card and list-item headings.
- **Body** (400, 1rem, 2 line-height): Explanatory copy.
- **Label** (700, 0.75rem, 1.5 line-height): Chips and concise metadata.

## Layout

The page uses a centered content frame up to 80rem wide, with horizontal padding that grows from 1.25rem on small screens to 2rem from the `sm` breakpoint. The hero changes from a stacked mobile reading order to a two-column desktop composition at `lg`; sample pathways use one column on mobile and three at `lg`. Section spacing is generous and consistent, with compact controls kept inside larger editorial sections. The sticky header preserves access to navigation while scrolling.

## Elevation & Depth

Depth is hybrid: dark tonal fields do most of the work, while translucent panels use backdrop blur, a fine crimson border, and a broad dark shadow. Hover adds only a soft red glow and a subtle edge shift; it does not turn every panel into a brightly illuminated card.

### Shadow Vocabulary
- **Panel:** `0 24px 70px rgba(0, 0, 0, 0.34)` with a faint inset top edge; use for glass panels.
- **Card Hover:** `0 22px 58px rgba(0, 0, 0, 0.34), 0 0 32px rgba(220, 38, 38, 0.09)`; use only for interactive hover emphasis.
- **Primary Action:** A soft, offset ruby shadow gives the main CTA separation from the dark ground.

## Shapes

Rounded silhouettes distinguish control scale: 0.75rem for compact rows, 1rem for small surfaces, 1.5rem for cards, and pill shapes for primary/secondary CTA buttons and status chips. Panel outlines are thin and crimson-tinted; component preview rows use quieter neutral-red outlines.

## Components

### Buttons
- **Shape:** Pill (9999px).
- **Primary:** Ruby fill, white bold Thai label, generous horizontal padding, and a soft offset ruby shadow.
- **Hover / Focus:** Hover shifts slightly upward and brightens the ruby; keyboard focus uses a visible 3px ruby-light outline with offset.
- **Secondary:** Obsidian fill, restrained light border, white label; hover strengthens the crimson edge.

### Chips
- **Style:** Compact pill with a translucent dark or ruby-tinted fill and a fine border.
- **State:** Ruby text and dot indicate illustrative/sample status; chips are labels, not implied controls.

### Cards / Containers
- **Corner Style:** Large panels use 1.5rem corners; nested preview rows use 0.75rem.
- **Background:** Burgundy glass against alternating obsidian and maroon sections.
- **Shadow Strategy:** Use panel depth at rest and reserve red glow for hover.
- **Border:** Fine crimson hairline, subtly stronger on hover.
- **Internal Padding:** 1.5rem default, increasing on larger screens.

### Navigation
- **Style:** Compact sticky dark header with a branded mark and clear Thai links on desktop.
- **State:** Ruby primary action; mobile navigation uses a labeled menu button below `lg`.
- **Focus:** All links and controls retain the visible keyboard focus outline.

### Component Preview
The hero's component rows are an illustrative visual, not a live build editor. Keep the sample status explicit and use real component labels rather than fabricated product data.

## Do's and Don'ts

### Do:
- **Do** keep “เริ่มจากสเปกที่ใช่” as the only gradient text.
- **Do** label sample pathways and the component preview as illustrative.
- **Do** describe planned functions as unavailable until they are implemented.
- **Do** preserve the mobile-first reading order and accessible focus states.

### Don't:
- **Don't** imply a live product catalog, final prices, compatibility engine, accounts, saved builds, or sharing.
- **Don't** invent product recommendations, popularity, member submissions, reviews, or performance claims.
- **Don't** add unrelated accent hues or spread decorative glow across every surface.
