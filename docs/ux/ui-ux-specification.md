# Marionette UI/UX Specification

- **Version**: 0.1
- **Status**: Proposed for implementation
- **Date**: 2026-09-25
- **Language**: English (`en-US`)
- **References**: FR-20..29, NFR-08..10, ADR-0003, ADR-0007

## 1. Product model

Marionette is a quiet operations console for a small number of devices or
services. The operator should answer three questions within seconds:

1. What is healthy, failing, or unknown?
2. What can I safely trigger now?
3. What changed and what was the last result?

The dashboard is an operational workspace, management is a configuration tool,
and card detail is a diagnostic view. The UI favors scanning, explicit states,
and safe feedback over decoration.

## 2. Information architecture

### 2.1 Shared app shell

Every primary screen uses the same shell:

- brand and current section in the top-left;
- primary navigation: **Overview**, **Manage cards**;
- right-side connection state, refresh, and contextual action;
- centered content with a maximum width of 1200 px;
- mobile navigation collapses into a compact top bar without hiding the
  current section or the main action.

The active navigation item is visually and textually identifiable. Navigation
is not available only as a link buried inside a card.

### 2.2 Screens

| Route             | Screen      | Purpose                                           |
| ----------------- | ----------- | ------------------------------------------------- |
| `/`               | Overview    | Scan all cards and trigger primary operations     |
| `/cards/:id`      | Card detail | Current state, last result, runs, and transitions |
| `/cards/new/edit` | New card    | Empty creation form                               |
| `/cards/:id/edit` | Edit card   | Existing card configuration                       |

## 3. Product language and vocabulary

All visible UI copy is English. Technical values such as commands, arguments,
environment variable names, and regular expressions remain unchanged.

| Internal concept     | UI label       |
| -------------------- | -------------- |
| Action card          | Action card    |
| Primary action       | Primary action |
| Status action        | Status check   |
| Unknown              | Unknown        |
| OK                   | Healthy        |
| Fail                 | Problem        |
| Running              | Running        |
| Accepted             | Queued         |
| Refresh              | Refresh        |
| Run primary action   | Run action     |
| Manual status action | Check status   |

Internal values such as `unknown`, `ok`, `fail`, `accepted`, and `running` are
never exposed as raw labels.

## 4. State and feedback model

The UI separates two axes:

- **Request state**: `idle`, `loading`, `accepted`, `running`, `success`, or
  `error` for the current interaction.
- **Device state**: `unknown`, `ok`, or `fail` for the last known projection,
  including the last check time.

Rules:

- Clicking a primary action immediately shows **Queued**, never a false
  success.
- While work is pending, controls for that card are disabled and the card
  shows **Running**.
- After acceptance, status reads update the card until a new check is visible
  or the configured fast-polling window expires.
- An enqueue failure is an operation error, not automatically a device
  failure.
- A status read failure keeps the card visible and marks its data as unknown;
  it must not hide the entire dashboard.
- Every operation defines loading, empty, success, error, and destructive
  states.

Status always includes text, an icon, and a semantic color:

- **Unknown**: question mark, neutral gray;
- **Healthy**: check mark, green;
- **Problem**: warning mark, red/terracotta;
- **Running**: spinner or clock, blue;
- **Queued**: queue/arrow icon, amber.

Color is never the only carrier of meaning.

## 5. Overview screen

### 5.1 Content priority

1. Page title, card count, and API connection state.
2. A page-level error banner only when the card collection cannot load.
3. A responsive grid of action cards.
4. An empty state with the direct action **Create your first card**.

### 5.2 Action card anatomy

Every card has the same hierarchy:

- icon and status badge;
- card name;
- description limited to three lines;
- `Last checked ...` or `Not checked yet`;
- primary **Run action** button;
- secondary **Check status** button when a status action exists;
- overflow menu with **View details** and **Edit**.

Commands and arguments are not the main card content. They belong in detail or
management views.

### 5.3 Responsive layout

- Desktop: three columns, up to four on very wide screens.
- Tablet: two columns.
- 320–767 px: one column, full-width action controls, no overlap.
- Card ordering is stable across refreshes.

## 6. Card detail

The detail header contains the card name, icon, current state, and last check.
The body contains:

- **Summary**: current state, state duration, and last check;
- **Actions**: run primary action and manually check status;
- **Recent runs**: time, outcome, duration, and expandable output;
- **Status history**: chronological transition timeline with duration;
- **Configuration**: link to edit, not inline configuration in diagnostics.

## 7. Card editing

### 7.1 List

The Dashboard is the card list and the primary page action is **New card**.
Card Detail provides **Edit card** and **Delete card** actions. Editing is a
standalone workflow and does not combine a card list with a configuration form.

### 7.2 Form sections

1. **Card identity**: name, description, icon.
2. **Primary action**: command, arguments, working directory, environment,
   timeout, and output rule.
3. **Status check**: enable/disable switch and the same action editor.
4. **Automatic checks**: standard interval, fast interval, and fast window.
5. **Save bar**: sticky save/cancel actions.

The ID is optional on creation and immutable during editing. Arguments and
environment variables are repeatable rows, not an opaque JSON editor.

### 7.3 Validation and unsaved changes

- Validation appears beside the relevant field and in a summary at the top.
- Server errors keep the form open and preserve all entered values.
- Leaving a dirty form asks **Discard unsaved changes?**.
- Delete requires a dialog naming the card and an explicit **Delete card**
  destructive action.
- After save, show **Card saved** and remain in the edit context; navigation
  away is explicit.

## 8. Design system

### 8.1 Visual direction

The direction is **quiet botanical operations**: an airy sage canvas, soft
green surfaces, dark green-gray ink, and one confident natural-green action
accent. The base palette is intentionally restrained and close to monochrome;
blue, amber, and red appear only when they communicate a distinct operational
state. Avoid purple gradients, glass panels, decorative orbs, marketing hero
sections, and nested cards.

### 8.2 Shared tokens

Tokens are defined once and consumed by all views:

```css
:root {
  --color-canvas: #f2f7f3;
  --color-surface: #fbfdfb;
  --color-surface-raised: #ffffff;
  --color-ink: #1d3028;
  --color-muted: #60746b;
  --color-border: #d6e3da;
  --color-border-strong: #b7cbbd;
  --color-accent: #3e8064;
  --color-accent-strong: #2e604b;
  --color-accent-soft: #e5f0e8;
  --color-success: #2f805f;
  --color-info: #3e7184;
  --color-info-soft: #e7f0f3;
  --color-warning: #a87528;
  --color-warning-soft: #f6eedc;
  --color-danger: #b2504b;
  --color-danger-soft: #f7e7e5;
  --radius-sm: 6px;
  --radius-md: 10px;
  --space-1: 4px;
  --space-2: 8px;
  --space-3: 12px;
  --space-4: 16px;
  --space-6: 24px;
  --space-8: 32px;
}
```

Views use semantic tokens rather than local hex values. PrimeVue theme tokens
are aligned with these values.

### 8.3 Typography and density

- Use one expressive display family and one readable UI sans-serif family;
  typography does not change between screens.
- Body text is at least 16 px; supporting text is at least 13 px.
- Card headings never use hero-scale type.
- Spacing follows a 4/8 px rhythm.
- Cards are framed work surfaces with a light border; cards are not nested.

### 8.4 Shared components

Build these shared patterns before polishing individual views:

- `AppShell`, `PageHeader`, `ConnectionStatus`;
- `StatusBadge`, `RequestState`, `InlineError`, `EmptyState`;
- `ActionCard`, `ActionControls`;
- `ActionEditor`, `FormSection`, `UnsavedChangesDialog`;
- `RunTable`, `StatusTimeline`.

## 9. Accessibility and responsive behavior

- All controls are keyboard reachable in logical order.
- Focus rings remain visible and are never removed by CSS.
- Icon-only controls have `aria-label` and a tooltip; text buttons use clear
  action labels.
- Status changes and errors use appropriate live regions without spamming
  assistive technology.
- Contrast is checked against WCAG 2.2 AA.
- At 320 px, controls stack, forms become one column, and tables use a mobile
  alternative rather than forcing page-level horizontal scrolling.

## 10. UX verification scenarios

Before closing a UX block, verify:

1. Empty overview offers creation of the first card.
2. Overview is readable at 320 px, tablet, and desktop widths.
3. Primary action shows **Queued** and **Running**, never a false success.
4. One card's status failure does not hide other cards.
5. Validation and server errors preserve form values.
6. Unsaved changes cannot be discarded accidentally.
7. Deletion requires an explicit confirmation.
8. Keyboard navigation covers overview, dialogs, and forms.
9. States remain understandable without color perception.
10. Detail view presents runs and status history in diagnostic order.

## 11. Implementation sequence

- **UX-01**: tokens, PrimeVue theme, and shared `AppShell`;
- **UX-02**: overview and action/status feedback;
- **UX-03**: management and form system;
- **UX-04**: card detail, runs, and status timeline;
- **UX-05**: accessibility, responsive audit, and visual regression checks.

Each UX block requires its own scope, test plan, and `Schváleno` state before
implementation.
