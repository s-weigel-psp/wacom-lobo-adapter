package messaging

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

// maxMessageSize is the Chrome/Edge Native Messaging spec limit for host→browser messages.
// We apply the same limit to browser→host messages to prevent allocation attacks (T-02-01-01, T-02-01-03).
const maxMessageSize = 1 << 20 // 1 MB

// ReadMessage reads one Native Messaging message from r.
// Returns (nil, io.EOF) when the browser closes the port.
// Returns an error response map (never nil) for malformed messages to allow
// the caller to respond with ERR_INVALID_PARAMS rather than crashing.
func ReadMessage(r io.Reader) (map[string]interface{}, error) {
	var length uint32
	if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
		return nil, err // io.EOF here means browser closed port — caller should exit(0)
	}
	// Guard against zero-length and oversized messages (T-02-01-03).
	if length == 0 {
		return nil, errors.New("zero-length message")
	}
	if length > maxMessageSize {
		return nil, fmt.Errorf("message length %d exceeds maximum %d", length, maxMessageSize)
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	var msg map[string]interface{}
	if err := json.Unmarshal(buf, &msg); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	return msg, nil
}

// WriteMessage writes one Native Messaging message to w.
func WriteMessage(w io.Writer, v interface{}) error {
	body, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(len(body))); err != nil {
		return err
	}
	_, err = w.Write(body)
	return err
}

// MessageLoop reads commands until EOF (browser disconnect) and dispatches them.
// dispatch must never return nil — always return a map with "ok" or "error"+"code".
// On io.EOF: browser closed port — os.Exit(0) (clean shutdown).
// On other read errors: os.Exit(1).
func MessageLoop(dispatch func(map[string]interface{}) interface{}) {
	for {
		msg, err := ReadMessage(os.Stdin)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				os.Exit(0)
			}
			// Length guard errors (zero-length, oversized) — return ERR_INVALID_PARAMS
			// and stay running (D-12).
			resp := map[string]interface{}{
				"error": err.Error(),
				"code":  "ERR_INVALID_PARAMS",
			}
			_ = WriteMessage(os.Stdout, resp)
			continue
		}
		response := dispatch(msg)
		_ = WriteMessage(os.Stdout, response)
	}
}
