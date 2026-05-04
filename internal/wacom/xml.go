package wacom

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// wacomPrefPath is the path to PrefUtil.exe, the Wacom preference import utility.
//
// Task 1 Process Monitor investigation (02-02 Task 1) revealed that applying Wacom
// preferences requires calling PrefUtil.exe /import <path>. PrefUtil uses a COM mechanism
// (CLSID {ff48dba4-60ef-4201-aa87-54103eef594e} InprocServer32) to notify WTabletServicePro —
// no direct file write to a WTabletServicePro system path exists. The modified
// .Export.wacomxs is written to %TEMP% then PrefUtil is called as a subprocess.
const wacomPrefPath = `C:\Program Files\Tablet\Wacom\PrefUtil.exe`

// needsServiceRestart is set based on Task 1 Investigation B findings.
// false: WTabletServicePro is notified by PrefUtil via COM — no restart required.
const needsServiceRestart = false

// maxBaselineSize is a safety limit; a legitimate Wacom preference file is < 500 KB.
// Reject files larger than 10 MB to guard against DoS (T-02-02-05).
const maxBaselineSize = 10 * 1024 * 1024 // 10 MB

// maxCoord is the upper bound for any screen coordinate value (T-02-02-01).
// 16384 px exceeds the largest practical display resolution.
const maxCoord = 16384

// SetMapping clones the baseline .Export.wacomxs, updates ALL
// InputScreenAreaArray/ArrayElement/ScreenArea entries with AreaType=1 and the
// provided coordinates (physical pixels), writes the result to a temp path, then
// imports via PrefUtil.
//
// On any failure, returns a structured error map. Never exits (D-12).
func SetMapping(logger *slog.Logger, x, y, w, h int) map[string]interface{} {
	// Validate coordinates (T-02-02-01): must be non-negative and within plausible screen bounds.
	if x < 0 || y < 0 || w <= 0 || h <= 0 {
		return errMap("x and y must be >= 0; width and height must be > 0", "ERR_INVALID_PARAMS")
	}
	if x > maxCoord || y > maxCoord || w > maxCoord || h > maxCoord {
		return errMap(
			fmt.Sprintf("coordinate out of range (max %d): x=%d y=%d width=%d height=%d", maxCoord, x, y, w, h),
			"ERR_INVALID_PARAMS",
		)
	}

	baselinePath := filepath.Join(os.Getenv("LOCALAPPDATA"), "WacomBridge", "baseline.Export.wacomxs")
	data, err := readBaseline(baselinePath)
	if err != nil {
		logger.Error("baseline not found", slog.String("path", baselinePath), slog.String("error", err.Error()))
		return errMap("baseline file not found: "+baselinePath, "ERR_BASELINE_NOT_FOUND")
	}

	modified, nodeCount, err := applyScreenAreaCoords(data, x, y, w, h)
	if err != nil {
		logger.Error("xml parse failed", slog.String("error", err.Error()))
		return errMap("failed to parse baseline XML: "+err.Error(), "ERR_XML_PARSE")
	}
	if nodeCount == 0 {
		logger.Error("no ScreenArea nodes found", slog.String("path", baselinePath))
		return errMap("XPath '//InputScreenAreaArray/ArrayElement/ScreenArea' returned no nodes", "ERR_NO_SCREEN_AREA_NODES")
	}

	// Write to ProgramData so WTabletServicePro (runs as SYSTEM) can read the file.
	// %TEMP% is per-user and inaccessible to the service account.
	tempDir := filepath.Join(os.Getenv("ProgramData"), "WacomBridge")
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		return errMap("failed to create temp dir: "+err.Error(), "ERR_XML_WRITE")
	}
	tempPath := filepath.Join(tempDir, "wacom-mapping-temp.Export.wacomxs")
	if err := os.WriteFile(tempPath, modified, 0o644); err != nil {
		logger.Error("write temp failed", slog.String("path", tempPath), slog.String("error", err.Error()))
		return errMap("failed to write temp file: "+err.Error(), "ERR_XML_WRITE")
	}
	// Grant SYSTEM read access — WTabletServicePro runs as SYSTEM and reads the file via COM.
	// Go's os.WriteFile sets per-user ACL only; icacls extends it without replacing existing entries.
	if out, err := exec.Command("icacls", tempPath, "/grant", "SYSTEM:(R)").CombinedOutput(); err != nil {
		logger.Warn("icacls failed", slog.String("output", string(out)), slog.String("error", err.Error()))
	}

	logger.Info("set_mapping XML written",
		slog.Int("x", x), slog.Int("y", y),
		slog.Int("w", w), slog.Int("h", h),
		slog.Int("nodes_updated", nodeCount),
		slog.String("temp_path", tempPath))

	if err := runPrefUtil(tempPath); err != nil {
		logger.Error("PrefUtil import failed", slog.String("error", err.Error()))
		return errMap("PrefUtil import failed: "+err.Error(), "ERR_XML_WRITE")
	}

	logger.Info("set_mapping applied via PrefUtil",
		slog.Int("x", x), slog.Int("y", y), slog.Int("w", w), slog.Int("h", h))

	if needsServiceRestart {
		if errM := RestartWacomService(logger); errM != nil {
			return errM
		}
	}

	return map[string]interface{}{"ok": true}
}

// ResetMapping imports the unmodified baseline via PrefUtil, restoring full-screen mapping.
// The baseline is copied to a temp path before import to avoid any mutation of the original.
func ResetMapping(logger *slog.Logger) map[string]interface{} {
	baselinePath := filepath.Join(os.Getenv("LOCALAPPDATA"), "WacomBridge", "baseline.Export.wacomxs")

	data, err := readBaseline(baselinePath)
	if err != nil {
		logger.Error("baseline not found for reset", slog.String("path", baselinePath), slog.String("error", err.Error()))
		return errMap("baseline file not found: "+baselinePath, "ERR_BASELINE_NOT_FOUND")
	}

	// Copy baseline to temp path (PrefUtil may lock or mutate the source).
	// Write to ProgramData so WTabletServicePro (runs as SYSTEM) can read the file.
	tempDir := filepath.Join(os.Getenv("ProgramData"), "WacomBridge")
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		return errMap("failed to create temp dir: "+err.Error(), "ERR_XML_WRITE")
	}
	tempPath := filepath.Join(tempDir, "wacom-reset-temp.Export.wacomxs")
	if err := os.WriteFile(tempPath, data, 0o644); err != nil {
		logger.Error("write reset temp failed", slog.String("path", tempPath), slog.String("error", err.Error()))
		return errMap("failed to write temp file: "+err.Error(), "ERR_XML_WRITE")
	}
	if out, err := exec.Command("icacls", tempPath, "/grant", "SYSTEM:(R)").CombinedOutput(); err != nil {
		logger.Warn("icacls failed", slog.String("output", string(out)), slog.String("error", err.Error()))
	}

	if err := runPrefUtil(tempPath); err != nil {
		logger.Error("PrefUtil import failed for reset", slog.String("error", err.Error()))
		return errMap("PrefUtil import failed: "+err.Error(), "ERR_XML_WRITE")
	}

	logger.Info("reset_mapping applied via PrefUtil", slog.String("baseline", baselinePath))

	if needsServiceRestart {
		if errM := RestartWacomService(logger); errM != nil {
			return errM
		}
	}

	return map[string]interface{}{"ok": true}
}

// readBaseline reads and size-checks the baseline file (T-02-02-05).
func readBaseline(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.Size() > maxBaselineSize {
		return nil, fmt.Errorf("baseline file exceeds 10 MB limit (%d bytes)", info.Size())
	}
	return os.ReadFile(path)
}

// runPrefUtil calls PrefUtil.exe /import <path> and waits for it to exit.
// PrefUtil applies the preference via COM — no service restart is needed.
func runPrefUtil(xmlPath string) error {
	cmd := exec.Command(wacomPrefPath, "/import", xmlPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("PrefUtil /import %q: %w", xmlPath, err)
	}
	return nil
}

// applyScreenAreaCoords parses the XML token stream, finds every ScreenArea element
// nested inside InputScreenAreaArray/ArrayElement, rewrites AreaType=1 and the four
// coordinate text nodes, then returns the full modified XML bytes together with the
// count of ScreenArea nodes that were updated.
//
// All tokens outside the rewritten subtrees are emitted unchanged, preserving all
// other tablet settings (D-05 / clone-and-modify invariant).
func applyScreenAreaCoords(data []byte, x, y, w, h int) ([]byte, int, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false

	var out bytes.Buffer
	out.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")

	enc := xml.NewEncoder(&out)
	enc.Indent("", "  ")

	// pathStack tracks element names as we descend so we know our position.
	var pathStack []string
	nodeCount := 0

	// targetPath is the ancestor path that must be present for a ScreenArea element
	// to be in scope.  We check only the tail elements that matter.
	const (
		needGrandparent = "InputScreenAreaArray"
		needParent      = "ArrayElement"
	)

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, 0, fmt.Errorf("XML decode: %w", err)
		}

		switch t := tok.(type) {
		case xml.StartElement:
			name := t.Name.Local

			// Check if this is a ScreenArea inside InputScreenAreaArray/ArrayElement.
			if name == "ScreenArea" && len(pathStack) >= 2 &&
				pathStack[len(pathStack)-1] == needParent &&
				pathStack[len(pathStack)-2] == needGrandparent {

				nodeCount++
				// Rewrite the entire ScreenArea subtree.
				rewritten, err := rewriteScreenArea(t, dec, x, y, w, h)
				if err != nil {
					return nil, 0, fmt.Errorf("rewrite ScreenArea node %d: %w", nodeCount, err)
				}
				// Flush the encoder to sync the byte buffer before raw write.
				if err := enc.Flush(); err != nil {
					return nil, 0, err
				}
				out.WriteString(rewritten)
				// pathStack is NOT pushed here; rewriteScreenArea consumed </ScreenArea>.
				continue
			}

			pathStack = append(pathStack, name)
			if err := enc.EncodeToken(t); err != nil {
				return nil, 0, err
			}

		case xml.EndElement:
			if len(pathStack) > 0 {
				pathStack = pathStack[:len(pathStack)-1]
			}
			if err := enc.EncodeToken(t); err != nil {
				return nil, 0, err
			}

		default:
			if err := enc.EncodeToken(t); err != nil {
				return nil, 0, err
			}
		}
	}

	if err := enc.Flush(); err != nil {
		return nil, 0, err
	}

	return out.Bytes(), nodeCount, nil
}

// rewriteScreenArea reads all tokens from the ScreenArea subtree (the decoder is
// positioned just after the opening <ScreenArea> start element), modifies the five
// target fields, and returns the complete rewritten element as an XML string.
//
// The target fields are text content of:
//   - AreaType                      → "1"  (custom region)
//   - ScreenOutputArea/Origin/X     → strconv.Itoa(x)
//   - ScreenOutputArea/Origin/Y     → strconv.Itoa(y)
//   - ScreenOutputArea/Extent/X     → strconv.Itoa(w)
//   - ScreenOutputArea/Extent/Y     → strconv.Itoa(h)
//
// All other tokens are emitted unchanged.
func rewriteScreenArea(start xml.StartElement, dec *xml.Decoder, x, y, w, h int) (string, error) {
	// Buffer the inner content; we'll wrap it in the outer element manually.
	var inner bytes.Buffer
	enc := xml.NewEncoder(&inner)
	enc.Indent("", "  ")

	// elementPath tracks the local names of open elements within the ScreenArea body.
	var elemPath []string
	depth := 1 // we entered <ScreenArea>; exit when depth reaches 0

	for depth > 0 {
		tok, err := dec.Token()
		if err != nil {
			return "", fmt.Errorf("reading ScreenArea body: %w", err)
		}
		tok = xml.CopyToken(tok)

		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			elemPath = append(elemPath, t.Name.Local)
			if err := enc.EncodeToken(t); err != nil {
				return "", err
			}

		case xml.EndElement:
			depth--
			if depth == 0 {
				// This is </ScreenArea> — stop consuming.
				break
			}
			if len(elemPath) > 0 {
				elemPath = elemPath[:len(elemPath)-1]
			}
			if err := enc.EncodeToken(t); err != nil {
				return "", err
			}

		case xml.CharData:
			text := string(bytes.TrimSpace(t))
			// Only rewrite non-whitespace text nodes.
			if len(text) > 0 {
				pk := strings.Join(elemPath, "/")
				switch pk {
				case "AreaType":
					text = "1"
				case "ScreenOutputArea/Origin/X":
					text = strconv.Itoa(x)
				case "ScreenOutputArea/Origin/Y":
					text = strconv.Itoa(y)
				case "ScreenOutputArea/Extent/X":
					text = strconv.Itoa(w)
				case "ScreenOutputArea/Extent/Y":
					text = strconv.Itoa(h)
				}
				if err := enc.EncodeToken(xml.CharData(text)); err != nil {
					return "", err
				}
			} else if len(t) > 0 {
				// Preserve whitespace-only tokens as-is.
				if err := enc.EncodeToken(t); err != nil {
					return "", err
				}
			}

		default:
			if err := enc.EncodeToken(tok); err != nil {
				return "", err
			}
		}
	}

	if err := enc.Flush(); err != nil {
		return "", err
	}

	// Build the opening tag with preserved attributes.
	var tag strings.Builder
	tag.WriteString("<")
	tag.WriteString(start.Name.Local)
	for _, a := range start.Attr {
		tag.WriteString(fmt.Sprintf(` %s="%s"`, a.Name.Local, a.Value))
	}
	tag.WriteString(">")

	return "\n" + tag.String() + inner.String() + "</" + start.Name.Local + ">", nil
}

// errMap returns a structured error response (D-11).
func errMap(msg, code string) map[string]interface{} {
	return map[string]interface{}{"error": msg, "code": code}
}
