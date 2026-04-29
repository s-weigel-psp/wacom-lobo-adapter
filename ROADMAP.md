# Wacom Lobo Adapter — Roadmap & Spec

> Spec für Claude Code zur Umsetzung im lokalen Repository.
> Stand: April 2026

---

## Projektziel

Nutzer in einer Windows-Domäne sollen mit einem Wacom One M Tablet PDFs in einer browserbasierten Drittanwendung bearbeiten können. Das Tablet-Mapping soll automatisch auf den Bereich des Bildschirms beschränkt werden, in dem das PDF angezeigt wird — ohne dass der Nutzer manuell Wacom-Einstellungen anpassen muss.

**Eingabe**: Eine stabile DOM-id, die das HTML-Element identifiziert, in dem das PDF gerendert wird.
**Ausgabe**: Wacom-Stift bewegt sich nur innerhalb des PDF-Bereichs auf dem Bildschirm.

---

## Designentscheidung: Explicit-Sync-Modell

Statt das DOM-Element live zu tracken (technisch riskant, hoher Performance-Aufwand, Treiber-Reload-Latenz problematisch) wird ein **explizites Sync-Modell** gewählt:

- Beim Aktivieren des PDF-Modus wird das Wacom-Mapping **einmal** auf die aktuelle Element-Position gesetzt.
- Die Browser-Extension überwacht Veränderungen (Resize, Move, DPR-Wechsel) und blendet ein Banner ein: *"PDF-Bereich hat sich verändert. [Wacom-Bereich neu kalibrieren]"*.
- Der Nutzer behält die Kontrolle und triggert die Neusynchronisation per Klick.

**Begründung**: Wacom-Treiber unterstützen kein dokumentiertes Live-API. Mapping-Wechsel über `Wacom_TabletUserPrefs.exe` dauern ~1–2 Sekunden — akzeptabel für einmalige Aktivierung, nicht akzeptabel für Live-Tracking. Beobachtetes Nutzerverhalten (PDF einmal öffnen, dann darin arbeiten) macht den expliziten Ansatz funktional gleichwertig bei deutlich geringerer Komplexität.

---

## Komponenten-Übersicht

```txt
┌────────────────────────────────────────────────────────────────┐
│  Browser (Chrome/Edge)                                         │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  Browser Extension                                       │  │
│  │  ┌───────────────────────┐   ┌────────────────────────┐  │  │
│  │  │ Content Script        │   │ Background Service     │  │  │
│  │  │ - DOM-Tracking        │◄──┤ - Native Messaging     │  │  │
│  │  │ - Banner-UI           │   │ - State Management     │  │  │
│  │  │ - Koord.-Berechnung   │   └─────────┬──────────────┘  │  │
│  │  └───────────────────────┘             │                 │  │
│  └────────────────────────────────────────┼─────────────────┘  │
└───────────────────────────────────────────┼────────────────────┘
                                            │ Native Messaging
                                            │ (stdin/stdout, JSON)
                                            ▼
┌────────────────────────────────────────────────────────────────┐
│  Native Host (C# .NET 8, Windows-Service oder Single-Exe)     │
│  - JSON-Protokoll                                              │
│  - Wacom-Preference-XML generieren                             │
│  - Wacom_TabletUserPrefs.exe aufrufen                          │
│  - Logging                                                     │
└────────────────────────┬───────────────────────────────────────┘
                         │
                         ▼
┌────────────────────────────────────────────────────────────────┐
│  Wacom Tablet Driver                                           │
│  - Mapping wird auf gewünschten Bildschirmbereich gesetzt      │
└────────────────────────────────────────────────────────────────┘
```

---

## Phasen

### Phase 1 — Wacom-Mapping-Spike *(Fokus dieser Roadmap)*

**Ziel**: Verifizieren, dass Wacom-Mapping zur Laufzeit programmatisch und zuverlässig auf einen beliebigen Bildschirmbereich gesetzt werden kann.

**Risiko-Status**: HOCH — wenn diese Phase scheitert, ist das Gesamtprojekt nicht in der geplanten Form umsetzbar.

**Aufgaben**:

1. **Recherche & Setup**
   - Wacom-Treiber für Wacom One M auf Test-Rechner installieren
   - Pfad zu `Wacom_TabletUserPrefs.exe` verifizieren (typ. `C:\Program Files\Tablet\Wacom\`)
   - Pfade zu Preference-Dateien identifizieren (typ. `%ProgramData%\Tablet\Wacom\` und/oder `%LOCALAPPDATA%\Wacom\`)
   - Aktuelle Preference manuell exportieren als Baseline-Profil
   - XML-Struktur analysieren — relevante Tags identifizieren (insb. `<MapToOutput>`, Mapping-Koordinaten)

2. **PowerShell-Spike-Skript** (`spike/Set-WacomMapping.ps1`)
   - Parameter: `-X`, `-Y`, `-Width`, `-Height` (in physischen Bildschirm-Pixeln)
   - Liest Baseline-Preference-Template
   - Ersetzt Mapping-Koordinaten
   - Schreibt temporäre Preference-Datei
   - Importiert über `Wacom_TabletUserPrefs.exe /import <pfad>`
   - Verifikation: Wacom-Tablet-Eigenschaften öffnen und Mapping prüfen
   - Reset-Funktion: Baseline-Profil zurückimportieren

3. **Manuelle Tests**
   - Test 1: Mapping auf linke Bildschirmhälfte → Stift bewegt sich nur dort
   - Test 2: Mapping auf rechte Bildschirmhälfte → Stift bewegt sich nur dort
   - Test 3: Mapping auf 800x600-Region in Bildschirmmitte
   - Test 4: Wechsel zwischen drei verschiedenen Mappings hintereinander, Latenz messen
   - Test 5: Reset → Stift bewegt sich wieder über gesamten Bildschirm
   - Test 6: Multi-Monitor-Setup (falls verfügbar), Mapping auf sekundären Bildschirm

4. **Fallback-Untersuchung** (nur falls `Wacom_TabletUserPrefs.exe` nicht funktioniert)
   - Direkte XML-Manipulation der Preference-Files unter `%ProgramData%\Tablet\Wacom\`
   - Wacom-Service-Restart über `Restart-Service "TabletServiceWacom"` (Service-Namen verifizieren)
   - Dokumentieren, ob Treiber-Restart nötig ist und wie lange er dauert

5. **Dokumentation** (`spike/SPIKE-RESULTS.md`)
   - Welche Methode hat funktioniert?
   - Welche Pfade, welche Service-Namen, welche XML-Tags?
   - Latenz pro Mapping-Wechsel?
   - Welche Edge-Cases sind aufgetaucht?
   - Empfehlung für Phase 2: PowerShell beibehalten oder in C# portieren?

**Erfolgskriterium**: Skript kann auf Kommando ein beliebiges Mapping setzen, Wechsel dauert < 3 Sekunden, Stift respektiert das Mapping nach dem Wechsel.

**Geschätzter Aufwand**: 1–2 Tage (abhängig davon, wie kooperativ die Wacom-Tools sich verhalten).

**Deliverables in diesem Repo**:

- `spike/Set-WacomMapping.ps1`
- `spike/Reset-WacomMapping.ps1`
- `spike/baseline-profile.xml` (exportiertes Baseline-Profil)
- `spike/SPIKE-RESULTS.md`
- `spike/test-log.md` (Protokoll der manuellen Tests)

---

### Phase 2 — Native Messaging Host *(nach erfolgreichem Spike)*

**Ziel**: Produktionsreifer Windows-Helper, der per Native Messaging von einer Browser-Extension angesteuert werden kann.

**Tech-Stack**: C# .NET 8, Single-File-Deployment, MSI-Paketierung via WiX.

**Aufgaben**:

- Native-Messaging-Protokoll implementieren (4-Byte-Längenprefix + JSON-Body, stdin/stdout)
- Befehle: `set_mapping`, `reset_mapping`, `get_status`, `ping`
- Wacom-Logik aus PowerShell-Spike portieren
- Logging in `%LOCALAPPDATA%\WacomBridge\logs\`
- Registry-Manifest installieren unter `HKLM\SOFTWARE\Google\Chrome\NativeMessagingHosts\com.eurefirma.wacombridge` und Edge-Pendant
- WiX-Installer mit Pre/Post-Install-Hooks

**Geschätzter Aufwand**: 3–5 Tage.

---

### Phase 3 — Browser-Extension *(parallel zu Phase 2 möglich)*

**Ziel**: Chrome/Edge Manifest-V3-Extension, die das DOM-Element trackt und Sync-Befehle an den Native Host sendet.

**Aufgaben**:

- Manifest V3 mit `nativeMessaging`-Permission und URL-Allowlist
- Content-Script:
  - Polling auf `document.getElementById(TARGET_ID)`
  - Bei Fund: Banner-UI einblenden (Shadow DOM, um Styling-Konflikte zu vermeiden)
  - Sync-Button löst Koordinaten-Berechnung aus
  - `ResizeObserver` + `window.addEventListener('resize')` + Polling auf `window.screenX/Y` für Veränderungs-Erkennung
  - Bei Veränderung: Banner-Status auf "veraltet" wechseln
- Background-Service-Worker:
  - `chrome.runtime.connectNative('com.eurefirma.wacombridge')`
  - Nachrichten weiterleiten zwischen Content-Script und Native Host
- Window-Management-API (`window.getScreenDetails()`) für robuste Bildschirmkoordinaten — wenn nicht verfügbar, Fallback auf `screenX/Y + outerWidth-Differenzen`
- DPR-Berücksichtigung für Windows-DPI-Skalierung

**Geschätzter Aufwand**: 3–5 Tage.

---

### Phase 4 — Domain-Deployment

**Ziel**: Zentrale Auslieferung an alle Nutzer in der Domäne ohne manuellen Eingriff.

**Aufgaben**:

- Extension via `ExtensionInstallForcelist`-GPO-Policy zwangsinstalliert (Chrome und Edge separat)
- URL-Allowlist via `ExtensionSettings`-Policy
- Window-Management-Permission via `WindowManagementAllowedForUrls`-Policy auto-granten
- MSI-Paketierung für Native Host, Deployment via Intune oder GPO Software Installation
- Pilottest auf 1–2 Maschinen, dann Rollout-Plan

**Geschätzter Aufwand**: 2–3 Tage.

---

## Repository-Struktur (Vorschlag)

```txt
wacom-pdf-mapping/
├── README.md                    # Projekt-Übersicht
├── ROADMAP.md                   # Diese Datei
├── ARCHITECTURE.md              # Architektur-Detail (parallel als .docx vorhanden)
├── spike/                       # Phase 1 — Wacom-Spike
│   ├── Set-WacomMapping.ps1
│   ├── Reset-WacomMapping.ps1
│   ├── baseline-profile.xml
│   ├── SPIKE-RESULTS.md
│   └── test-log.md
├── native-host/                 # Phase 2 — C# Native Messaging Host
│   └── (kommt in Phase 2)
├── extension/                   # Phase 3 — Browser-Extension
│   └── (kommt in Phase 3)
├── deployment/                  # Phase 4 — GPO-/Intune-Artefakte
│   └── (kommt in Phase 4)
└── docs/
    └── (Dokumentation, Diagramme, Screenshots)
```

---

## Phase-1-Checkliste für Claude Code

```txt
[ ] Test-Rechner vorbereiten: Windows + Wacom One M + offizieller Wacom-Treiber installiert
[ ] Wacom-Tablet einmalig verbinden, Mapping in GUI manuell ausprobieren (Sanity-Check)
[ ] Aktuelles Wacom-Profil über GUI als XML exportieren → spike/baseline-profile.xml
[ ] Pfad zu Wacom_TabletUserPrefs.exe verifizieren und in PowerShell-Skript hardcoden
[ ] XML-Struktur untersuchen, Mapping-Tags identifizieren
[ ] Set-WacomMapping.ps1 schreiben mit Parametern X, Y, Width, Height
[ ] Reset-WacomMapping.ps1 schreiben (Re-Import des Baseline-Profils)
[ ] Test 1: Mapping linke Bildschirmhälfte (z.B. 0,0,960,1080)
[ ] Test 2: Mapping rechte Bildschirmhälfte (z.B. 960,0,960,1080)
[ ] Test 3: Mapping Mitte 800x600
[ ] Test 4: Drei Wechsel hintereinander, Latenz pro Wechsel messen und protokollieren
[ ] Test 5: Reset funktioniert, Tablet ist wieder voll-Bildschirm
[ ] Falls Multi-Monitor verfügbar: Test 6
[ ] SPIKE-RESULTS.md ausfüllen mit Befunden, Pfaden, Latenzen, Empfehlung
[ ] test-log.md mit Datum/Uhrzeit/Ergebnis pro Test ausfüllen
[ ] Falls Wacom_TabletUserPrefs.exe nicht funktioniert: Fallback auf direkte XML-Manipulation + Service-Restart erproben und dokumentieren
```

---

## Hinweise zur Umsetzung in Claude Code

- **DPI-Skalierung beachten**: Auf Windows mit aktivierter Skalierung (125%, 150% etc.) liefert PowerShell `[System.Windows.Forms.Screen]::PrimaryScreen.Bounds` skalierte Werte. Wacom erwartet vermutlich physische Pixel — das im Spike testen und dokumentieren.
- **Service-Name verifizieren**: Bei neueren Treiberversionen kann der Service `WTabletServicePro` oder `Wacom Professional Service` heißen. `Get-Service | Where-Object { $_.Name -like "*acom*" -or $_.Name -like "*ablet*" }` zur Identifikation.
- **Admin-Rechte**: Manche Wacom-Operationen erfordern erhöhte Rechte. Im Spike testen, ob Imports auch im User-Kontext funktionieren — falls nein, ist das ein wichtiger Befund für die Architektur (Native Host müsste dann als Service laufen, nicht im User-Kontext).
- **Backup vor jedem Test**: Aktuelles Wacom-Profil sichern, falls Skript es kaputt macht.
- **Logging**: PowerShell-Skript soll Schritte mit `Write-Host` protokollieren, damit Fehler nachvollziehbar sind.

---

## Offene Fragen für die nächste Iteration

- Soll der Native Host als Windows-Service oder pro User-Session laufen?
- Welche Logging-Detailtiefe ist für Support-Fälle nötig?
- Wie wird die Extension bei Nutzern deinstalliert/geupdatet, wenn das Frontend der Drittanwendung sich ändert?
- Soll es ein Fallback geben, wenn der Native Host nicht erreichbar ist (z.B. nicht installiert)?

Diese Fragen sind erst für Phase 2/3 relevant — Phase 1 muss zuerst Klarheit über die technische Machbarkeit liefern.
