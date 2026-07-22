# Coding Agent Instructions

## Project source of truth

- Read `features.md` before planning or implementing project work.
- Treat `features.md` as the living feature and technology tracker.
- Read `guide.md` for system boundaries, request flows, and architectural context before architecture-sensitive work.
- Before changing a service, read that service's `guide.md` when one exists.
- When the user provides another tutorial transcript, add only the newly confirmed requirements and technical decisions to `features.md`; preserve existing information and avoid duplicates.
- Do not invent unspecified behavior. Record unresolved details under **Details not decided yet** until the tutorial or user resolves them.
- Implement features incrementally and keep service boundaries explicit.
- Mark a feature as implemented (`[x]`) only after its code has been verified with relevant tests or checks.
- Keep secrets out of source control and provide sanitized `.env.example` files when configuration is introduced.

## Working practices

- Inspect the repository before making architecture-sensitive changes.
- Keep changes scoped to the requested feature.
- Update relevant documentation and tests with implementation changes.
- In architecture or service guides, use a compact flow or ownership diagram when it materially clarifies a multi-component interaction or state transition.
- Prefer commands that work from the repository root.
