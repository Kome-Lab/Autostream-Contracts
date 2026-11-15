package lifecycle

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"unicode/utf8"
)

// DecodeRecord is an internal conformance-data decoder, not a selected signing
// or transport wire. No lossy Unicode substitution or implicit normalization is
// permitted. Object/array/depth and byte budgets derive only from explicit Limits.
func DecodeRecord(data []byte, p ComparisonProfile) (Record, Result) {
	if r := ValidateProfile(p); r.Kind != StructurallyCoherent {
		return Record{}, r
	}
	budget := p.Limits.MaxIDBytes * p.Limits.MaxRefs * 16
	if len(data) > budget {
		return Record{}, result(RefusedUnchanged, "DOCUMENT_LIMIT_EXCEEDED", "JSON")
	}
	if !utf8.Valid(data) || !validJSONEscapes(data) {
		return Record{}, result(RefusedUnchanged, "NON_LITERAL_JSON_UNICODE", "JSON")
	}
	collectionLimit := p.Limits.MaxMembers
	for _, v := range []int{p.Limits.MaxAxes, p.Limits.MaxRefs, p.Limits.MaxObligations, p.Limits.MaxPartitions, p.Limits.MaxRanges} {
		if v > collectionLimit {
			collectionLimit = v
		}
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if r := scanJSON(d, 0, p.Limits.MaxDepth, p.Limits.MaxRefs*16, collectionLimit); r.Kind != StructurallyCoherent {
		return Record{}, r
	}
	if _, e := d.Token(); e != io.EOF {
		return Record{}, result(RefusedUnchanged, "TRAILING_JSON", "JSON")
	}
	var record Record
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&record); err != nil {
		return Record{}, result(RefusedUnchanged, "INVALID_OR_UNKNOWN_JSON_FIELD", "JSON")
	}
	return record, ValidateRecord(record, p)
}
func scanJSON(d *json.Decoder, depth, limit, objectLimit, arrayLimit int) Result {
	if depth > limit {
		return result(RefusedUnchanged, "JSON_DEPTH_EXCEEDED", "JSON")
	}
	token, err := d.Token()
	if err != nil {
		return result(RefusedUnchanged, "INVALID_JSON", "JSON")
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return coherent()
	}
	switch delim {
	case '{':
		keys := []string{}
		for d.More() {
			if len(keys) >= objectLimit {
				return result(RefusedUnchanged, "JSON_OBJECT_LIMIT_EXCEEDED", "JSON")
			}
			tok, err := d.Token()
			if err != nil {
				return result(RefusedUnchanged, "INVALID_JSON", "JSON")
			}
			key, ok := tok.(string)
			if !ok {
				return result(RefusedUnchanged, "INVALID_JSON_KEY", "JSON")
			}
			// Match encoding/json's Unicode case-fold aliases, including long-s/Kelvin.
			for _, prior := range keys {
				if strings.EqualFold(prior, key) {
					return result(RefusedUnchanged, "DUPLICATE_JSON_KEY", "JSON")
				}
			}
			keys = append(keys, key)
			if r := scanJSON(d, depth+1, limit, objectLimit, arrayLimit); r.Kind != StructurallyCoherent {
				return r
			}
		}
		if close, err := d.Token(); err != nil || close != json.Delim('}') {
			return result(RefusedUnchanged, "INVALID_JSON_OBJECT", "JSON")
		}
	case '[':
		count := 0
		for d.More() {
			if count >= arrayLimit {
				return result(RefusedUnchanged, "JSON_ARRAY_LIMIT_EXCEEDED", "JSON")
			}
			count++
			if r := scanJSON(d, depth+1, limit, objectLimit, arrayLimit); r.Kind != StructurallyCoherent {
				return r
			}
		}
		if close, err := d.Token(); err != nil || close != json.Delim(']') {
			return result(RefusedUnchanged, "INVALID_JSON_ARRAY", "JSON")
		}
	default:
		return result(RefusedUnchanged, "INVALID_JSON_DELIMITER", "JSON")
	}
	return coherent()
}
func jsonHex4(data []byte, start int) (uint16, bool) {
	if start+4 > len(data) {
		return 0, false
	}
	var n uint16
	for _, b := range data[start : start+4] {
		n <<= 4
		switch {
		case b >= '0' && b <= '9':
			n |= uint16(b - '0')
		case b >= 'a' && b <= 'f':
			n |= uint16(b - 'a' + 10)
		case b >= 'A' && b <= 'F':
			n |= uint16(b - 'A' + 10)
		default:
			return 0, false
		}
	}
	return n, true
}
func validJSONEscapes(data []byte) bool {
	for i := 0; i < len(data); i++ {
		if data[i] != '"' {
			continue
		}
		i++
		closed := false
		for i < len(data) {
			if data[i] == '"' {
				closed = true
				break
			}
			if data[i] < 0x20 {
				return false
			}
			if data[i] == '\\' {
				i++
				if i >= len(data) {
					return false
				}
				switch data[i] {
				case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
				case 'u':
					n, ok := jsonHex4(data, i+1)
					if !ok {
						return false
					}
					i += 4
					if n >= 0xd800 && n <= 0xdbff {
						if i+6 >= len(data) || data[i+1] != '\\' || data[i+2] != 'u' {
							return false
						}
						low, ok := jsonHex4(data, i+3)
						if !ok || low < 0xdc00 || low > 0xdfff {
							return false
						}
						i += 6
					} else if n >= 0xdc00 && n <= 0xdfff {
						return false
					}
				default:
					return false
				}
			}
			i++
		}
		if !closed {
			return false
		}
	}
	return true
}
