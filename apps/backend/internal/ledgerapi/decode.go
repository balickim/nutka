// This file strictly decodes mutation bodies and rejects identity spoofing.
// Bodies must contain one JSON object and no unknown or trailing fields.
package ledgerapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

var errInvalidRequest = errors.New("ledger request is invalid")

type settlementRequest struct {
	Settlement string `json:"settlement"`
}

type refundRequest struct {
	AmountMinor int64  `json:"amount_minor"`
	Reason      string `json:"reason"`
}

type correctionRequest struct {
	Reason string `json:"reason"`
	Note   string `json:"note,omitempty"`
}

func decodeBody(e *core.RequestEvent, destination any) error {
	if e.Request.Body == nil || e.Request.ContentLength == 0 {
		return errInvalidRequest
	}
	raw, err := io.ReadAll(e.Request.Body)
	if err != nil {
		return errInvalidRequest
	}
	return decodeRaw(raw, destination)
}

func decodeRaw(raw []byte, destination any) error {
	var fields map[string]json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if err := decoder.Decode(&fields); err != nil || fields == nil {
		return errInvalidRequest
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return errInvalidRequest
	}
	if hasIdentityField(fields) {
		return errForbidden
	}
	encoded, err := json.Marshal(fields)
	if err != nil {
		return errInvalidRequest
	}
	strict := json.NewDecoder(bytes.NewReader(encoded))
	strict.DisallowUnknownFields()
	if err := strict.Decode(destination); err != nil {
		return errInvalidRequest
	}
	return nil
}

func hasIdentityField(fields map[string]json.RawMessage) bool {
	for key := range fields {
		if strings.EqualFold(key, "teacher") || strings.EqualFold(key, "learner") || strings.EqualFold(key, "actor_id") || strings.EqualFold(key, "actor_role") {
			return true
		}
	}
	return false
}
