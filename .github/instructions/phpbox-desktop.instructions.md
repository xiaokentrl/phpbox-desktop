---
applyTo: "**/*.{go,ts,vue,md,js,json,yml,yaml}"
---

# phpbox Desktop Development Instructions

## Mission
Build the desktop app as a developer-focused local environment manager, not a generic operations dashboard.

## Engineering truth
- The bash `phpbox` engine is the authoritative source of functionality.
- The GUI is a thin presentation layer and command aggregator.
- File-based interfaces are the contract: vhost files, `.env`, `extensions.env`, `backups`, and `offline` data are all real state.

## UI fidelity
- Match the approved prototype and implementation spec exactly.
- Preserve DOM structure, spacing, color tokens, motion timing, and text semantics.
- Do not make aesthetic changes that drift away from the approved design without explicit alignment.

## Implementation rules
- Verify CLI signatures and environment assumptions before changing behavior.
- When writing Go bindings or engine logic, keep business logic in the bash/engine side and keep bindings thin.
- Keep the frontend as a view of real state, not a source of truth.
- Do not create duplicate state in the UI that cannot be derived from files or CLI output.

## Validation
- Run the smallest relevant validation for the file or task you changed.
- Prefer project validation commands such as `vue-tsc`, `vite build`, `go test`, and targeted Go checks.
- Report the exact command and result, not vague assurances.

## Safety and quality bar
- Do not hide errors or pretend a failed operation succeeded.
- Do not broaden the change beyond the task at hand.
- Do not refactor unrelated code when a minimal fix is enough.
- Do not claim completion without evidence.
