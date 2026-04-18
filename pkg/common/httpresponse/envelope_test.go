package httpresponse

import (
	"encoding/json"
	"testing"
)

func TestAPISuccessEnvelope_JSON(t *testing.T) {
	env := APISuccessEnvelope{
		Success: true,
		Data: map[string]string{"access_token": "x"},
		Meta:    map[string]interface{}{"request_id": "rid-1"},
	}
	b, err := json.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]json.RawMessage
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if string(got["success"]) != "true" {
		t.Fatalf("success %s", got["success"])
	}
}
