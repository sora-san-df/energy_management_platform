package meter

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func serve(t *testing.T, target string) *httptest.ResponseRecorder {
	t.Helper()
	svc, _ := newTestService()
	mux := http.NewServeMux()
	NewHandler(svc).Register(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	return rec
}

func TestHandlerList(t *testing.T) {
	rec := serve(t, "/api/meters?status=critical&sort=variation&order=desc")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Items []Overview `json:"items"`
		Total int        `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Total != 1 || body.Items[0].MeterID != "M-109" {
		t.Fatalf("unexpected body %+v", body)
	}
}

func TestHandlerListInvalidFilter(t *testing.T) {
	if rec := serve(t, "/api/meters?status=weird"); rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestHandlerGet(t *testing.T) {
	rec := serve(t, "/api/meters/M-109")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var d Detail
	if err := json.Unmarshal(rec.Body.Bytes(), &d); err != nil || d.MeterID != "M-109" {
		t.Fatalf("unexpected body %s", rec.Body.String())
	}
}

func TestHandlerGetNotFound(t *testing.T) {
	if rec := serve(t, "/api/meters/M-999"); rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestHandlerReadings(t *testing.T) {
	rec := serve(t, "/api/meters/M-109/readings?granularity=day")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var res ReadingsResult
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil || len(res.Points) != 3 {
		t.Fatalf("unexpected body %s", rec.Body.String())
	}
}

func TestHandlerReadingsRange(t *testing.T) {
	rec := serve(t, "/api/meters/M-109/readings?from=2026-09-02T00:00:00Z&to=2026-09-02T23:00:00Z")
	var res ReadingsResult
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil || len(res.Points) != 24 {
		t.Fatalf("unexpected body %d %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerReadingsInvalidDate(t *testing.T) {
	if rec := serve(t, "/api/meters/M-109/readings?from=yesterday"); rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}
