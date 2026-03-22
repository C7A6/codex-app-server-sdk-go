# AGENTS.md

## CORE PROTOCOLS (STRICT ADHERENCE REQUIRED)

* **PROJECT LAYOUT**: This project STRICTLY FOLLOWING "go standard layout"
* **ROADMAP VERSIONING**: The version of the Roadmap MUST strictly synchronize with the output of `codex --version`.
* **DOCUMENTATION LANGUAGE**: All generated documentation and communication MUST be authored exclusively in **ENGLISH**.
* **TASK EXECUTION LIMIT**: You MUST process exactly **ONE (1) TODO item** from `ROADMAP.md` per turn. Do NOT attempt to consolidate multiple tasks into a single turn unless explicitly authorized by the user.
* **TASK TRACKING**: All new action items MUST be recorded in the **ROADMAP.md** file under the appropriate version section.
* **POST-TASK COMPLETION PROTOCOL**: Immediately upon completing a task, you MUST:
    1. Update the status in `ROADMAP.md`.
    2. Execute Git commits.
    3. **DO NOT** bundle all changes into a single commit. Divide changes into logical groups and use **Conventional Commits** (e.g., `feat:`, `fix:`, `docs:`) for each.

## ROADMAP.md Template

```markdown
# ROADMAP

* The version of the SDK follows the version of the Codex CLI for consistency.

## ${VERSION}

- [ ] TODO
- [ ] ...
```
