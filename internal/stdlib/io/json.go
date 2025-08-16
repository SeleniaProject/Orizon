package io

import (
	"encoding/json"
)

// JSONEncode encodes v into JSON bytes using encoding/json with sane defaults.
func JSONEncode(v any) ([]byte, error) { return json.Marshal(v) }

// JSONEncodeIndent encodes v with indent for human readability.
func JSONEncodeIndent(v any, prefix, indent string) ([]byte, error) {
	return json.MarshalIndent(v, prefix, indent)
}

// JSONDecode decodes data into v.
func JSONDecode(data []byte, v any) error { return json.Unmarshal(data, v) }
