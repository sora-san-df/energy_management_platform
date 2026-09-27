package anomaly

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func do(t *testing.T, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	NewHandler(newMemoryService()).Register(mux)
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, target, nil)
	} else {
		req = httptest.NewRequest(method, target, strings.NewReader(body))
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestHandlerList(t *testing.T) {
	rec := do(t, http.MethodGet, "/api/anomalies?severity=HIGH", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body struct {
		Items []Anomaly `json:"items"`
		Total int       `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Total != 2 || body.Items[0].MeterID != "M-109" {
		t.Fatalf("unexpected body %+v", body)
	}
}

func TestHandlerListLowercaseFilters(t *testing.T) {
	rec := do(t, http.MethodGet, "/api/anomalies?type=data_quality", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "M-112") {
		t.Fatalf("unexpected response %d %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerListInvalidFilter(t *testing.T) {
	if rec := do(t, http.MethodGet, "/api/anomalies?severity=EXTREME", ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestHandlerGet(t *testing.T) {
	rec := do(t, http.MethodGet, "/api/anomalies/3", "")
	var a Anomaly
	if err := json.Unmarshal(rec.Body.Bytes(), &a); err != nil || rec.Code != http.StatusOK || a.MeterID != "M-109" || a.Priority != 1 {
		t.Fatalf("unexpected response %d %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerGetErrors(t *testing.T) {
	if rec := do(t, http.MethodGet, "/api/anomalies/abc", ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
	if rec := do(t, http.MethodGet, "/api/anomalies/99", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestHandlerUpdateStatus(t *testing.T) {
	rec := do(t, http.MethodPatch, "/api/anomalies/3", `{"status":"RESOLVED"}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"status":"RESOLVED"`) {
		t.Fatalf("unexpected response %d %s", rec.Code, rec.Body.String())
	}
	if rec := do(t, http.MethodPatch, "/api/anomalies/3", `{"status":"LOST"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
	if rec := do(t, http.MethodPatch, "/api/anomalies/3", `not-json`); rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}
