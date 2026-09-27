package analysis

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newHandlerFixture(t *testing.T) (*Service, *http.ServeMux) {
	t.Helper()
	svc := NewService(context.Background(), newFixture().deps)
	mux := http.NewServeMux()
	NewHandler(svc).Register(mux)
	return svc, mux
}

func send(mux *http.ServeMux, method, target, body string) *httptest.ResponseRecorder {
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

func TestHandlerAnalyzeAndPoll(t *testing.T) {
	svc, mux := newHandlerFixture(t)

	rec := send(mux, http.MethodPost, "/api/ai/analyze", `{}`)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var run Run
	if err := json.Unmarshal(rec.Body.Bytes(), &run); err != nil || run.ID == 0 {
		t.Fatalf("unexpected body %s", rec.Body.String())
	}
	if loc := rec.Header().Get("Location"); loc == "" {
		t.Fatal("expected Location header pointing to the analysis")
	}
	svc.Wait()

	rec = send(mux, http.MethodGet, "/api/ai/analysis/1", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"status":"COMPLETED"`) {
		t.Fatalf("unexpected poll response %d %s", rec.Code, rec.Body.String())
	}

	rec = send(mux, http.MethodGet, "/api/ai/analysis/latest", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"id":1`) {
		t.Fatalf("unexpected latest response %d %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerAnalyzeWithoutBody(t *testing.T) {
	svc, mux := newHandlerFixture(t)
	rec := send(mux, http.MethodPost, "/api/ai/analyze", "")
	svc.Wait()
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestHandlerAnalyzeScoped(t *testing.T) {
	svc, mux := newHandlerFixture(t)
	rec := send(mux, http.MethodPost, "/api/ai/analyze", `{"meter_ids":["M-109"]}`)
	svc.Wait()
	if rec.Code != http.StatusAccepted || !strings.Contains(rec.Body.String(), `"scope":["M-109"]`) {
		t.Fatalf("unexpected response %d %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerAnalyzeErrors(t *testing.T) {
	_, mux := newHandlerFixture(t)
	if rec := send(mux, http.MethodPost, "/api/ai/analyze", `{"meter_ids":["M-999"]}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown meter status = %d", rec.Code)
	}
	if rec := send(mux, http.MethodPost, "/api/ai/analyze", `{bad json`); rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid json status = %d", rec.Code)
	}
}

func TestHandlerAnalysisNotFound(t *testing.T) {
	_, mux := newHandlerFixture(t)
	if rec := send(mux, http.MethodGet, "/api/ai/analysis/42", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
	if rec := send(mux, http.MethodGet, "/api/ai/analysis/latest", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("latest status = %d", rec.Code)
	}
	if rec := send(mux, http.MethodGet, "/api/ai/analysis/abc", ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid id status = %d", rec.Code)
	}
}
