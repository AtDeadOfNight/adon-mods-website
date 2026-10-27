package compat

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strconv"
)

const utf8BOM = "\xEF\xBB\xBF"

var errTrailingJSON = errors.New("unexpected data after JSON value")

// Bun's text() drops one leading byte order mark, as the WHATWG UTF-8 decoder does.
func WithoutBOM(data []byte) []byte {
	return bytes.TrimPrefix(data, []byte(utf8BOM))
}

// Numbers stay json.Number so that one out of float64 range only affects the field that holds it.
func DecodeJSON(data []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(WithoutBOM(data)))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return nil, errTrailingJSON
	}
	return value, nil
}

// Field is value?.[key]?.[…] in JavaScript: nil wherever a step is not an object.
func Field(value any, keys ...string) any {
	for _, key := range keys {
		object, _ := value.(map[string]any)
		value = object[key]
	}
	return value
}

// Each reads one field of every object in list; like a Zod array schema, one bad element fails the list.
func Each[T any](list any, key string, read func(any) (T, bool)) ([]T, bool) {
	elements, ok := list.([]any)
	if !ok {
		return nil, false
	}
	values := make([]T, 0, len(elements))
	for _, element := range elements {
		object, isObject := element.(map[string]any)
		value, ok := read(object[key])
		if !isObject || !ok {
			return nil, false
		}
		values = append(values, value)
	}
	return values, true
}

// Number is what JSON.parse makes of a JSON number: out of range becomes ±Infinity or 0, which Zod 3's number() accepts.
func Number(value any) (float64, bool) {
	number, ok := value.(json.Number)
	if !ok {
		return 0, false
	}
	parsed, err := strconv.ParseFloat(string(number), 64)
	if err != nil && !errors.Is(err, strconv.ErrRange) {
		return 0, false
	}
	return parsed, true
}
