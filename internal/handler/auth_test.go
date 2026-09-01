package handler

import (
	"encoding/json"
	"testing"
)

func TestDashboardLoginResponseOmitsAPIKey(t *testing.T) {
	body, err := json.Marshal(authResponse{Token: "jwt-session", User: map[string]string{"id": "user-1"}})
	if err != nil {
		t.Fatal(err)
	}

	var response map[string]any
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatal(err)
	}
	if response["token"] != "jwt-session" {
		t.Fatalf("token = %v, want jwt-session", response["token"])
	}
	if _, ok := response["api_key"]; ok {
		t.Fatal("dashboard login response must not contain or create an API key")
	}
}
