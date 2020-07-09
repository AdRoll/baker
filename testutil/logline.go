package testutil

import "github.com/AdRoll/baker"

// NewLogLineFromMap populates an baker.LogLine with the fields in m.
func NewLogLineFromMap(m map[baker.FieldIndex]string) baker.Record {
	ll := &baker.LogLine{}
	for fidx, v := range m {
		if v != "" {
			ll.Set(fidx, []byte(v))
		}
	}
	return ll
}
