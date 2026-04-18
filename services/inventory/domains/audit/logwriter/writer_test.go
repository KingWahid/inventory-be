package logwriter

import "testing"

func TestMarshalSnap_nil(t *testing.T) {
	t.Parallel()
	b, err := marshalSnap(nil)
	if err != nil || len(b) != 0 {
		t.Fatalf("nil snap: %#v err=%v", b, err)
	}
}

func TestMarshalSnap_map(t *testing.T) {
	t.Parallel()
	b, err := marshalSnap(map[string]any{"a": 1})
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"a":1}` {
		t.Fatalf("got %s", string(b))
	}
}
