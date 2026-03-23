package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreatedUses201InBodyAndStatus(t *testing.T) {
	rr := httptest.NewRecorder()

	Created(rr, map[string]string{"name": "demo"})

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected HTTP %d, got %d", http.StatusCreated, rr.Code)
	}

	var apiResp APIResponse
	if err := json.NewDecoder(rr.Body).Decode(&apiResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if apiResp.Code != http.StatusCreated {
		t.Fatalf("expected response code %d, got %d", http.StatusCreated, apiResp.Code)
	}
}
