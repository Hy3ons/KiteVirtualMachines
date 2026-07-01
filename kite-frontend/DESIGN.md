# Kite Frontend Design System

## Product Intent
Kite Frontend is a practical VM and Kubernetes management console. The UI should feel like an engineering tool: calm, dense enough for repeated work, and direct about system state.

## Tokens
- Background: `#F9F8F6`
- Surface: `#FFFFFF`
- Muted surface: `#EFE9E3`
- Border: `#D9CFC7`
- Primary: `#8B7355`
- Accent: `#C9B59C`
- Text primary: `#2C2A29`
- Text secondary: `#594E46`
- Success: `#7A9C74`
- Warning: `#D4A373`
- Error: `#C86B6B`

## Shape
- Border radius is `0px` across buttons, cards, drawers, modals, tables, and outer landing panels.
- Landing authentication controls may use an `8px` radius when the page is acting as a branded first impression rather than a dense workspace.
- Shadows are only used for elevated surfaces such as cards, tables, dropdowns, drawers, modals, and landing panels. Landing panel shadows should be soft and even on all sides.

## Typography
- Primary font stack: system UI fonts first, then Korean platform fonts: `-apple-system`, `BlinkMacSystemFont`, `Segoe UI`, `system-ui`, `Apple SD Gothic Neo`, `Malgun Gothic`, `Noto Sans KR`, `Roboto`, sans-serif.
- Landing display headings may use common serif fonts: `Georgia`, `Times New Roman`, serif.
- Numeric dashboard values use tabular figures where possible.
- Korean and English labels should stay readable without forcing single-line overflow in narrow layouts.

## Layout
- Spacing follows an 8px grid, with 4px exceptions only for tight control alignment.
- Content containers use max widths between 800px and 1400px depending on page density.
- Tables may scroll horizontally on small viewports instead of compressing columns until text breaks.
- Header and toolbar controls wrap into additional rows on narrow widths.

## Components
- Global header: brand left, documentation links and session controls right, wrapping on mobile.
- Dashboard cards: flat rectangular surfaces with light border and minimal shadow.
- Tables: constrained columns, ellipsis for long identifiers, horizontal scroll for operational data.
- Forms: vertical labels, 44px controls, full-width inputs in modals and settings pages.

## States
- Loading, empty, and failure states use AntD primitives styled through the theme.
- Interactive controls must keep visible hover, active, disabled, and keyboard focus states.
