# phpbox Desktop Copilot Instructions

## Project facts
- This repo is a Wails v3 + Go desktop app with a Vue 3 frontend.
- The real source of truth for mutations is the bash `phpbox` CLI, not the GUI.
- UI is a view layer and aggregator; it must not create GUI-only parallel state.
- The front-end must align with the final UI prototype and the frontend implementation spec.

## Hard rules
1. Fact-first: verify bash CLI signatures, file formats, and naming rules before implementing behavior.
2. GUI never replaces CLI: all state changes must be routed through the real CLI transaction flow.
3. File-as-interface: treat vhost, `.env`, `extensions.env`, `backups/`, and `offline/` as real files/directories, not hidden GUI state.
4. No fake success: do not hide errors, suppress logs, or mask failures with misleading success text.
5. UI fidelity: do not “improve” design by changing token values, DOM structure, spacing, rounding, motion, or language text without matching the approved prototype.
6. Minimum fix: do not broaden scope or refactor unrelated code.
7. Validate: run the smallest meaningful verification command after the change, and report the result.

## Priority order
1. bash / CLI real behavior
2. real file and filesystem structure
3. prototype and UI implementation spec
4. maintainability or code elegance
5. optional design improvements

If there is a conflict, follow the higher-priority source.

## Task workflow
1. Verify the real facts from the CLI or actual files.
2. Compare the target against the UI prototype/spec.
3. Implement the smallest correct change.
4. Validate with the narrowest relevant command.
5. Report the change, the fact source, and the verification evidence.

## Forbidden patterns
- Inventing CLI flags, file layouts, or env variables.
- Creating GUI-only state that is not backed by a real file or CLI flow.
- “Fixing” by rewriting unrelated code.
- Claiming success without verification.
- Hiding failures behind generic messages.

## Scope of work
- Prefer real project files under `frontend/src`, `internal`, `cmd`, and `build`.
- Respect the project structure and architecture described in `AGENTS.md`.
- Maintain the separation between GUI read-only parsing and CLI-driven mutation.
