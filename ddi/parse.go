package ddi

import (
	"bytes"
	"errors"
)

// Parse reads XML or JSON, chosen by the first byte after a byte order mark
// and whitespace. JSON gives no warnings.
func Parse(data []byte) (*Node, []string, error) {
	s := bytes.TrimLeft(bytes.TrimPrefix(data, []byte("\xef\xbb\xbf")), xmlSpace)
	switch {
	case bytes.HasPrefix(s, []byte("<")):
		return ParseXML(data)
	case bytes.HasPrefix(s, []byte("{")):
		root, err := ParseJSON(data)
		return root, nil, err
	}
	return nil, nil, errors.New("input is neither XML nor JSON")
}
