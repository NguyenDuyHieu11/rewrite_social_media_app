package httputil

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

type ErrorBody struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func WriteJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// WriteError writes a standard error envelope with the given status.
//
//	code    — stable machine token (e.g. "invalid_credentials"). Clients
//	          branch on this.
//	message — human readable explanation. Safe to display, not to parse.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, ErrorBody{Error: message, Code: code})
}

// DecodeJSON strictly decodes a JSON request body into dst.
//
// Strictness rules:
//   - Body capped at maxBytes (defaults to 1 MiB if maxBytes <= 0). Prevents
//     a malicious client from forcing the server to allocate gigabytes.
//   - Unknown JSON fields are rejected. Saves you from "I sent userName but
//     the server expects username" silent failures.
//   - Body must contain exactly one JSON value. Trailing junk is rejected.
//
// The returned error is suitable to surface to the client via WriteError
// with status 400.
func DecodeJSON(r *http.Request, dst any, maxBytes int64) error {
	if maxBytes <= 0 {
		maxBytes = 1 << 20 // 1 MiB
	}
	r.Body = http.MaxBytesReader(nil, r.Body, maxBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	// A second Decode must report EOF; anything else means there was a
	// trailing token after the first JSON value.
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}
