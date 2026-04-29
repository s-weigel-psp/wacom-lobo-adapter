<!-- GSD:project-start source:PROJECT.md -->
## Project

Wacom Lobo Adapter — A browser extension + Windows native messaging host that automatically restricts Wacom One M tablet input to the PDF rendering area in a third-party browser application. Users in a Windows domain annotate PDFs without manually adjusting Wacom settings. Four phases: (1) PowerShell spike to validate Wacom driver scripting, (2) C# .NET 8 native host, (3) Chrome/Edge MV3 extension, (4) GPO/Intune domain deployment.
<!-- GSD:project-end -->

<!-- GSD:stack-start source:STACK.md -->
## Technology Stack

Technology stack not yet documented. Will populate after codebase mapping or first phase.

**Known from spec:**
- Native Host: C# .NET 8, single-file deployment, WiX 4 MSI packaging
- Browser Extension: Chrome/Edge Manifest V3, Shadow DOM for UI, Native Messaging protocol
- Spike: PowerShell, `Wacom_TabletUserPrefs.exe` preference XML
- Deployment: GPO ADMX policies, Intune Win32 app
<!-- GSD:stack-end -->

<!-- GSD:conventions-start source:CONVENTIONS.md -->
## Conventions

Conventions not yet established. Will populate as patterns emerge during development.
<!-- GSD:conventions-end -->

<!-- GSD:architecture-start source:ARCHITECTURE.md -->
## Architecture

Three-tier system:

1. **Browser Extension** (Chrome/Edge MV3): Content script reads DOM element position → Background service worker forwards via Native Messaging → receives confirmation
2. **Native Messaging Host** (C# .NET 8 exe): Receives JSON commands over stdin/stdout, generates Wacom preference XML, calls `Wacom_TabletUserPrefs.exe /import`
3. **Wacom Driver**: Applies the imported preference XML and restricts stylus to the configured screen region

Communication uses the Native Messaging protocol (4-byte length prefix + JSON body). The extension uses Shadow DOM for banner UI to avoid CSS conflicts with the host application.
<!-- GSD:architecture-end -->

<!-- GSD:skills-start source:skills/ -->
## Project Skills

No project skills found. Add skills to `.claude/skills/` with a `SKILL.md` index file.
<!-- GSD:skills-end -->

<!-- GSD:workflow-start source:GSD defaults -->
## GSD Workflow Enforcement

Before using Edit, Write, or other file-changing tools, start work through a GSD command so planning artifacts and execution context stay in sync.

Use these entry points:
- `/gsd-quick` for small fixes, doc updates, and ad-hoc tasks
- `/gsd-debug` for investigation and bug fixing
- `/gsd-execute-phase` for planned phase work

Do not make direct repo edits outside a GSD workflow unless the user explicitly asks to bypass it.
<!-- GSD:workflow-end -->

<!-- GSD:profile-start -->
## Developer Profile

> Profile not yet configured. Run `/gsd-profile-user` to generate your developer profile.
> This section is managed by `generate-claude-profile` — do not edit manually.
<!-- GSD:profile-end -->
