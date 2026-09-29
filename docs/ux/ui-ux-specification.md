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
- primary navigation: **Overview** (the card list is the dashboard; cards are
  managed from their detail, see §7.1) and **Devices** (only when access
  control is on and this browser is paired, §7.4);
- right-side connection state: **Live** while the status stream is open,
  **Reconnecting** while REST polling covers for it; refresh and contextual
  actions live in the page header;
- centered content with a maximum width of 1200 px;
- mobile navigation collapses into a compact top bar without hiding the
  current section or the main action;
- a quiet footer with the running version (`Marionette v1.2.1`, FR-41a)
  from `/api/health`, read again when the status stream reconnects (a
  service restart during an upgrade); it is hidden when the version cannot
  be read.

The pairing screen uses the shell without navigation, connection state or
footer: nothing behind it is available yet.

The active navigation item is visually and textually identifiable. Navigation
is not available only as a link buried inside a card.

### 2.2 Screens

| Route             | Screen      | Purpose                                           |
| ----------------- | ----------- | ------------------------------------------------- |
| `/`               | Overview    | Scan all cards and trigger primary operations     |
| `/cards/:id`      | Card detail | Current state, last result, runs, and transitions |
| `/cards/new/edit` | New card    | Empty creation form                               |
| `/cards/:id/edit` | Edit card   | Existing card configuration                       |
| `/devices`        | Devices     | Paired browsers, removal, code for a new device   |
| `/pair`           | Pair device | One-time code entry for an unpaired browser       |

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
- After acceptance, the card waits for a check newer than the one the server
  knew when it accepted the request (the 202 response carries that
  `checkedAt`, so a scheduled check that completed during the request is not
  mistaken for the result): a `status.changed` event from the shared stream
  or a REST read every 2 s. After **Run action** the wait is bounded by the
  configured fast-polling window; after **Check status** by the status
  action's timeout plus a 30 s queue margin. The wait happens only when the
  server will actually run a check: always after **Check status**, and after
  **Run action** only for a card with a status check and all three
  automatic-check values set (standard interval, fast interval, fast window),
  because that is when the server activates fast polling. The stream is
  shared by all views; while it is down, the card list is re-read over REST
  every 5 s and the statuses it carries are applied to the dashboard.
- When the wait ends, the controls are released immediately. The badge
  shows the new state, so **Updated** is not repeated in text; only an
  outcome the badge cannot show (**Result not available yet**, or the
  enqueue error) stays visible for about 4 s. A card without a status
  action, or a primary action that no check follows, shows **Accepted**
  right after the request is queued.
- When the server records the run of a primary action (`run.recorded`,
  FR-42a), whoever started it, the dashboard card reports it in the same
  line: a failed run (**Action failed · exit 2**, **Action timed out**,
  **Action canceled**) stays until the next action on the card, a newer run
  or a reload, because the badge never shows it; a successful run shows
  **Action finished** for about 4 s, and is only announced when a status
  check follows the action (the badge then shows its result). A run
  recorded while the card's own request is still in flight keeps its note
  over the request's **Accepted** or **Updated**.
- The pending phases (**Queued**, **Running**) are shown by the badge and
  the button spinner only, never by a second line of text. Every phase and
  outcome, **Updated** included, is announced through a visually hidden
  live region that takes no space.
- Action feedback never changes the size of a card or panel: a visible
  outcome replaces the quiet line in place (**Last checked ...** on a
  card, the asynchronous note in the detail Actions panel) and the line
  returns when the outcome expires.
- A snapshot never replaces a newer one: a slow REST read cannot undo a
  transition that arrived through the stream in the meantime.
- An enqueue failure is an operation error, not automatically a device
  failure.
- A status read failure keeps the card visible and marks its data as unknown:
  the badge reads **Unknown**, the note **Status could not be refreshed**
  appears next to the last check time, and both clear with the next status
  that arrives. It must not hide the entire dashboard.
- Every operation defines loading, empty, success, error, and destructive
  states.

Feedback channels:

- **Inline** (next to its subject, in place of the quiet line for action
  outcomes) for everything that belongs to one card,
  one form field or the current page: action outcomes on a card or in the
  detail, validation and server errors, load errors, a failed delete. Several
  cards can report at once without covering each other.
- **Toast** (top right; full width minus the 16 px gutter on phones; about
  4 s) only for the outcome of an operation that ends by leaving or
  replacing the page, so the notice survives the navigation: **Card saved**
  and **Card deleted**. Toasts are success notices only; an error never
  appears only in a toast.

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
- `Last checked ...` or `Not checked yet`, replaced in place by an action
  outcome the badge cannot show (§4); the line is reserved, empty, on a
  card without a status action so every card keeps its height;
- primary **Run action** button;
- secondary **Check status** button when a status action exists;
- overflow menu with **View details** and **Edit**.

A card with a colour (FR-10a) has a 6 px stripe of that colour along its
top edge, drawn by the top border and an inset shadow so the card keeps
the size of a card without a colour. The colour only groups cards; it never
carries a state, which stays with the status badge.

Commands and arguments are not the main card content. They belong in detail or
management views.

### 5.3 Responsive layout

- Desktop: three columns, up to four on very wide screens.
- Tablet: two columns.
- 320–767 px: one column, full-width action controls, no overlap.
- Page header actions (for example **Refresh** and **New card**, **Edit card**
  and **Delete card**): button labels never wrap; up to 768 px the actions
  take their own row below the title and share its full width equally.
- Card ordering is stable across refreshes.

## 6. Card detail

The detail header contains the card name, icon, current state, and last check.
A card colour (FR-10a) is a stripe along the top edge of the **Current
status** panel, drawn the same way and in the same colour as on the
dashboard card; the header, the icon tile and the other panels are not
coloured.
The body contains:

- **Summary**: current state, state duration, and last check with its
  outcome, exit code, duration, and expandable output (FR-21a), so the
  cause of a **Problem** is visible without the API;
- **Actions**: run primary action and manually check status; both buttons
  are blocked while a check is pending (the badge shows **Queued** /
  **Running**), and recent runs and status history are refetched once a
  newer check is visible; an outcome the badge cannot show (**Result not
  available yet**, an enqueue error, or **Action accepted** when no check
  follows, §4) replaces the panel's asynchronous note in place. The page-level
  line above the summary is used only for deleting the card;
- **Recent runs**: time, outcome, duration, and expandable output. A
  `run.recorded` event for the card re-reads the list at once, also for a
  run started elsewhere, and so does a reconnect of the stream. After
  **Run action** is accepted the detail waits for the new run: the event
  normally ends the wait; as a fallback the list is re-read at the card's
  fast polling interval (2 s when the card has none), starting only after
  the action's timeout while the stream is live (at once when it drops),
  at most for the action's timeout plus the 30 s queue margin; the buttons
  do not wait for it, and a failed read shows **Run history is unavailable** while
  the wait goes on;
- **Status history**: chronological transition timeline with duration;
- **Configuration**: link to edit, not inline configuration in diagnostics.

## 7. Card editing

### 7.1 List

The Dashboard is the card list and the primary page action is **New card**.
Card Detail provides **Edit card** and **Delete card** actions. Editing is a
standalone workflow and does not combine a card list with a configuration form.

### 7.2 Form sections

1. **Card identity**: name, description, icon, colour (**None** by default).
2. **Primary action**: command line, working directory, environment,
   timeout, and output rule.
3. **Status check**: enable/disable switch and the same action editor.
4. **Automatic checks**: standard interval, fast interval, and fast window.
5. **Save bar**: sticky save/cancel actions.

The ID is optional on creation and immutable during editing. The command and
its arguments are one **Command line** field, as typed in a terminal
(`/usr/bin/ping -c 1 -W 2 192.168.1.10`); below it a live preview shows the
command and each argument as the process receives them, and a line that
needs a shell (unquoted `|`, `&`, `;`, `<`, `>`, `(`, `)`, `` ` ``, `$`), an
unclosed quote, or a trailing backslash is explained beside the field and
cannot be saved. Quotes and backslashes follow
[ADR-0012](../architecture/decisions/0012-single-line-command-editor.md); a
stored action is shown as one line with quotes added where needed; an
action with a multi-line argument (written through the API) is shown in a
text area that keeps the line break.
Environment variables are repeatable rows, not an opaque JSON editor.

### 7.3 Validation and unsaved changes

- Validation appears beside the relevant field and in a summary at the top.
- Server errors keep the form open and preserve all entered values.
- Leaving a dirty form asks **Discard unsaved changes?**.
- Delete requires a dialog naming the card and an explicit **Delete card**
  destructive action.
- After save, show **Card saved** as a toast and remain in the edit context;
  navigation away is explicit.
- After a confirmed delete, return to the overview and show **Card deleted**
  as a toast naming the card; a failed delete keeps the detail open with the
  error inline.

### 7.4 Access: pairing and devices

Only paired browsers use Marionette (FR-50..FR-56, ADR-0011). Pairing
happens once per browser; there is no password and no sign-in screen after
that.

- **Pair this device** (`/pair`): every navigation of an unpaired browser,
  and any API answer 401, leads here with the intended path in `next`; after
  pairing the browser continues there (paths inside the app only).
  - Fields: **Pairing code** (typed in any case, grouped as `XXXX-XXXX`,
    filled in from `?code=`) and **Device name** (suggested from the browser
    and system, e.g. "Chrome on Android"; required, up to 64 characters).
  - While no device is paired at all, the screen shows the command that
    reads the code from the service log; otherwise it points to **Devices →
    Pair a new device** on a paired device.
  - A wrong, used or expired code shows one message: **The pairing code is
    invalid or has expired.** Success shows the toast **Device paired**.
- **Devices** (`/devices`): the paired browsers, most recently used first,
  each with name, paired and last used time, and **This device** on the
  current one; the lede states the expiry in days.
  - **Pair a new device** shows the code in large monospace type, the time
    left, and a link that fills the code in (with **Copy link** when the
    browser allows the clipboard, which plain HTTP does not). When the new
    device pairs, the code disappears and the page says so inline; an
    expired code offers **New code**.
  - **Rename** (pencil icon, accessible name "Rename <name>") replaces the
    name with a **Device name** field (up to 64 characters) and **Save** /
    **Cancel**; Enter saves, Escape cancels. A rejected name is shown under
    the field; a saved one updates the list and the page says
    **"<name>" saved.** inline. Any paired device can rename any device;
    names need not be unique (FR-57).
  - **Remove device** asks for confirmation naming the device. Removing
    another device reports inline; removing this device signs it out, opens
    the pairing screen and shows the toast **This device was removed**.
  - With access control off (`MARIONETTE_AUTH=off`) the page explains that
    instead of listing devices.

## 8. Design system

### 8.1 Visual direction

The direction is **quiet botanical operations**: an airy sage canvas, soft
green surfaces, dark green-gray ink, and one confident natural-green action
accent. The base palette is intentionally restrained and close to monochrome;
blue, amber, and red appear only when they communicate a distinct operational
state. Avoid purple gradients, glass panels, decorative orbs, marketing hero
sections, and nested cards.

### 8.2 Shared tokens

Tokens are defined once in `web/src/styles/tokens.css` and consumed by all
views (values as implemented in blocks 0019/0020 and confirmed in 0033):

```css
:root {
  --color-canvas: #eef2f1;
  --color-surface: #f8faf9;
  --color-surface-raised: #ffffff;
  --color-ink: #26332f;
  --color-muted: #68746f;
  --color-border: #d7e0dd;
  --color-border-strong: #b9c9c3;
  --color-accent: #4b8969;
  --color-accent-strong: #397254;
  --color-accent-soft: #e6f1e8;
  --color-success: #4f8c68;
  --color-info: #557f8a;
  --color-info-soft: #e9f2f3;
  --color-warning: #ad7e3f;
  --color-warning-soft: #f8f0e2;
  --color-danger: #b45d56;
  --color-danger-soft: #f7e7e5;
  --color-focus: #2f7181;
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

Layout utilities (`grid`, `col-12`, `md:col-4`, `md:col-6`, `lg:col-4`,
`flex`, `flex-column`, `flex-wrap`, `align-items-center`,
`justify-content-between`, `gap-2`, `gap-3`, `p-2`, `mt-4`) are the
project's own classes in `web/src/styles/layout.css` with PrimeFlex-compatible
values and breakpoints (768 px, 992 px); a new utility is added there only when
a view needs it (ADR-0010).

Card colours (FR-10a) are the tokens `--card-color-black`, `-blue`,
`-teal`, `-purple`, `-pink`, `-orange` and `-yellow`; the card stores the
name, `web/src/ui/cardColors.ts` lists the options. Red and green are left
out because they mean **Problem** and **Healthy**.

Views use semantic tokens rather than local hex values. PrimeVue components
take the same palette from `web/src/theme/preset.ts` (`definePreset(Aura, …)`:
primary and surface scales, form field, text, content and overlay tokens), so
no component style is overridden with `!important`. The app ships one light
scheme; the OS dark mode is not followed.

### 8.3 Typography and density

- Use one expressive display family and one readable UI sans-serif family;
  typography does not change between screens.
- Body text is at least 16 px; supporting text is at least 13 px.
- Card headings never use hero-scale type.
- Spacing follows a 4/8 px rhythm.
- Cards are framed work surfaces with a light border; cards are not nested.

### 8.4 Shared components

Shared patterns live in `web/src/components` and are used by every view
(block 0033):

- `AppShell`, `PageHeader`, `ConnectionStatus`;
- `StatusBadge`, `RequestState`, `InlineError`, `EmptyState`; page-level
  notices go through the shared PrimeVue `Toast` in `App.vue` and the
  `useNotify` composable (§4 feedback channels);
- `ActionCard`, `ActionControls`;
- `ActionEditor`, `FormSection`, `CardIdentityFields`, `PollingFields`,
  `SaveBar`; the unsaved-changes dialog is the shared `ConfirmDialog` driven
  by the `useUnsavedChangesGuard` composable;
- `RunTable`, `StatusTimeline`, `StatusSummary`, `DetailPanel`,
  `DetailErrorState`.

All labels, icons and tones come from `web/src/ui/vocabulary.ts` (§3, §4);
dates and durations are formatted in `web/src/ui/format.ts`.

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

Each UX block requires its own scope, test plan, and `Approved` state before
implementation.
