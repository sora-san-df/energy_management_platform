package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestJSONWritesBodyAndHeaders(t *testing.T) {
	rec := httptest.NewRecorder()
	JSON(rec, http.StatusCreated, map[string]string{"hello": "world"})

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("content-type = %q", ct)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body["hello"] != "world" {
		t.Fatalf("unexpected body %s", rec.Body.String())
	}
}

func TestErrorEnvelope(t *testing.T) {
	rec := httptest.NewRecorder()
	Error(rec, http.StatusNotFound, "meter_not_found", "Medidor no encontrado")

	var body ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusNotFound || body.Error.Code != "meter_not_found" || body.Error.Message != "Medidor no encontrado" {
		t.Fatalf("unexpected error response %d %+v", rec.Code, body)
	}
}

func TestDecodeRejectsUnknownFields(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"x","extra":1}`))
	var dst struct {
		Name string `json:"name"`
	}
	if err := Decode(req, &dst); err == nil {
		t.Fatal("expected error for unknown field")
	}
}

func TestDecodeAcceptsEmptyBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
	var dst struct {
		Name string `json:"name"`
	}
	if err := Decode(req, &dst); err != nil {
		t.Fatalf("empty body should decode to zero value: %v", err)
	}
}

func TestDecodeRejectsOversizedBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"`+strings.Repeat("a", MaxBodyBytes)+`"}`))
	var dst struct {
		Name string `json:"name"`
	}
	if err := Decode(req, &dst); err == nil {
		t.Fatal("expected error for oversized body")
	}
}
