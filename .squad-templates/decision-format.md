# Decision Inbox Format

> Canonical format for all files written to `.squad/decisions/inbox/`.
> Every agent, the Coordinator, and Scribe must follow this format.

## File Naming

```
.squad/decisions/inbox/{author}-{brief-slug}.md
```

- `{timestamp}`: ISO 8601 format (e.g., `2026-02-24T18-30-00Z`) to ensure uniqueness and chronological ordering.
- `{author}`: lowercase agent name (e.g., `ripley`, `parker`) or `copilot-directive` for Coordinator-captured directives
- `{brief-slug}`: 2-4 word kebab-case summary (e.g., `jwt-auth-strategy`, `no-orm-decision`)

## Required Fields

Every inbox file MUST contain exactly one decision block in this format:

```markdown
### {ISO-date}: {Title}

**By:** {Author name} ({role or "via Copilot" for directives})
**Status:** {status}
**What:** {1-3 sentences: the decision, verbatim or lightly paraphrased}
**Why:** {1-3 sentences: rationale, constraints, or trade-offs}
```

### Field Definitions

| Field        | Required | Format                          | Description                                                                 |
| ------------ | -------- | ------------------------------- | --------------------------------------------------------------------------- |
| `{ISO-date}` | Yes      | `YYYY-MM-DD`                    | Date the decision was made. Captured via system clock, never hardcoded.     |
| `{Title}`    | Yes      | Free text, ≤80 chars            | Concise summary of what was decided. Start with a noun or verb.             |
| `**By:**`    | Yes      | `{Name} ({Role})`               | Agent's cast name and role. For directives: `{User} (via Copilot)`.        |
| `**Status:**`| Yes      | See status values below          | Current state of the decision.                                             |
| `**What:**`  | Yes      | 1-3 sentences                    | The decision itself. Be specific — name files, patterns, or constraints.   |
| `**Why:**`   | Yes      | 1-3 sentences                    | Reasoning. Include trade-offs, alternatives considered, or user directives. |

### Status Values

| Value                    | When to use                                                        |
| ------------------------ | ------------------------------------------------------------------ |
| `Approved`               | Reviewed and accepted by Lead or Reviewer                          |
| `Decided`                | Made by the agent with confidence; no review needed                |
| `Observation`            | Noticed something worth recording; no action required              |
| `Action required — {x}`  | Needs follow-up work (specify what)                                |
| `Completed`              | Work associated with the decision is done                          |
| `User request`           | Captured from a user directive                                     |

## Example: Agent Decision

Filename: `.squad/decisions/inbox/ripley-jwt-auth-strategy.md`

```markdown
### 2026-02-24: Use JWT with short-lived tokens for API auth

**By:** Ripley (Lead)
**Status:** Decided
**What:** API authentication will use JWT with 15-minute access tokens and 7-day refresh tokens. Tokens are validated by middleware, not per-handler.
**Why:** Short-lived tokens limit blast radius of token theft. Middleware validation keeps handlers clean and consistent. Refresh flow is standard OAuth2.
```

## Example: Coordinator Directive

Filename: `.squad/decisions/inbox/copilot-directive-2026-02-24T18-30-00Z.md`

```markdown
### 2026-02-24: Always use functional components

**By:** Brady (via Copilot)
**Status:** User request
**What:** All React components must be functional components with hooks. No class components.
**Why:** User request — captured for team memory.
```

## Parsing Contract

Scribe and any downstream scripts can rely on these invariants:

1. Each inbox file contains **exactly one** decision block
2. The block starts with `### ` (H3 heading)
3. The four fields (`**By:**`, `**Status:**`, `**What:**`, `**Why:**`) appear in that order, each on its own line
4. No other H3 headings exist in the file
5. After merging, `decisions.md` is a sequence of `---`-separated blocks, each following this same format
