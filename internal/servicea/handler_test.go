package servicea

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel/trace/noop"
)

func TestServiceAHandler_Validation(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")

	tests := []struct {
		name           string
		body           string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Invalid CEP - 7 digits",
			body:           `{"cep": "1234567"}`,
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   "invalid zipcode",
		},
		{
			name:           "Invalid CEP - contains letters",
			body:           `{"cep": "1234567a"}`,
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   "invalid zipcode",
		},
		{
			name:           "Invalid CEP - numeric type instead of string",
			body:           `{"cep": 12345678}`,
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   "invalid zipcode",
		},
		{
			name:           "Malformed JSON",
			body:           `{cep: "12345678"`,
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   "invalid zipcode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler("http://dummy", tracer)
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.Routes().ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
			if rec.Body.String() != tt.expectedBody {
				t.Errorf("expected body %q, got %q", tt.expectedBody, rec.Body.String())
			}
		})
	}
}

func TestServiceAHandler_ForwardToServiceB(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")

	mockServiceB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"city":"São Paulo","temp_C":28.5,"temp_F":83.3,"temp_K":301.65}`))
	}))
	defer mockServiceB.Close()

	handler := NewHandler(mockServiceB.URL, tracer)
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"cep":"01001000"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	expectedResponse := `{"city":"São Paulo","temp_C":28.5,"temp_F":83.3,"temp_K":301.65}`
	if rec.Body.String() != expectedResponse {
		t.Errorf("expected body %q, got %q", expectedResponse, rec.Body.String())
	}
}
