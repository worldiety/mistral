package miel

import (
	"encoding/json"
	"testing"
)

func TestUnmarshalUUID(t *testing.T) {
	type obj struct {
		ID UUID `json:"id"`
	}

	r := NewUUID()
	buf, err := json.Marshal(obj{ID: r})
	if err != nil {
		t.Fatal(err)
	}

	var o obj
	if err := json.Unmarshal(buf, &o); err != nil {
		t.Fatal(err)
	}

	if x := o.ID; x != r {
		t.Fatal(x)
	}
}
