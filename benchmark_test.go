package httprpc

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Benchmark request/response types
type BenchReq struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

type BenchRes struct {
	Message string `json:"message"`
	UserID  int    `json:"user_id"`
	Success bool   `json:"success"`
}

// httprpc handler
func benchHandler(_ context.Context, req BenchReq) (BenchRes, error) {
	return BenchRes{
		Message: "Hello, " + req.Name,
		UserID:  12345,
		Success: true,
	}, nil
}

// net/http handler equivalent (with manual JSON handling like httprpc does)
func benchNetHTTPHandler(w http.ResponseWriter, r *http.Request) {
	var req BenchReq
	if r.Body != nil {
		defer func() { _ = r.Body.Close() }()
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			if encodeErr := json.NewEncoder(w).Encode(map[string]string{"error": err.Error()}); encodeErr != nil {
				return
			}
			return
		}
	}

	res := BenchRes{
		Message: "Hello, " + req.Name,
		UserID:  12345,
		Success: true,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(res); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		if encodeErr := json.NewEncoder(w).Encode(map[string]string{"error": err.Error()}); encodeErr != nil {
			return
		}
	}
}

// Raw net/http handler (minimal work - just for baseline comparison)
func benchRawNetHTTPHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

// ============================================================================
// POST with JSON Body - The Main Comparison
// ============================================================================

// Benchmark httprpc POST with JSON
func BenchmarkHTTPRPC_POST_JSON(b *testing.B) {
	r := New()
	RegisterHandler[BenchReq, BenchRes](r.EndpointGroup, POST(benchHandler, "/api/user"))

	handler, err := r.Handler()
	if err != nil {
		b.Fatalf("failed to build handler: %v", err)
	}

	reqBody := []byte(`{"name":"John Doe","email":"john@example.com","age":30}`)

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/user", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			b.Fatalf("expected status 200, got %d", rec.Code)
		}
	}
}

// Benchmark net/http POST with manual JSON (apples-to-apples comparison)
func BenchmarkNetHTTP_POST_JSON(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/user", benchNetHTTPHandler)

	reqBody := []byte(`{"name":"John Doe","email":"john@example.com","age":30}`)

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/user", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			b.Fatalf("expected status 200, got %d", rec.Code)
		}
	}
}

// Benchmark raw net/http (baseline - minimal work)
func BenchmarkNetHTTP_POST_RAW(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/user", benchRawNetHTTPHandler)

	reqBody := []byte(`{"name":"John Doe","email":"john@example.com","age":30}`)

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/user", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			b.Fatalf("expected status 200, got %d", rec.Code)
		}
	}
}

// ============================================================================
// GET with Query Parameters
// ============================================================================

func BenchmarkHTTPRPC_GET_Query(b *testing.B) {
	r := New()
	RegisterHandler[BenchReq, BenchRes](r.EndpointGroup, GET(benchHandler, "/api/user"))

	handler, err := r.Handler()
	if err != nil {
		b.Fatalf("failed to build handler: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/user?name=John+Doe&email=john@example.com&age=30", http.NoBody)
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			b.Fatalf("expected status 200, got %d", rec.Code)
		}
	}
}

func BenchmarkNetHTTP_GET_Query(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/user", func(w http.ResponseWriter, r *http.Request) {
		req := BenchReq{
			Name:  r.URL.Query().Get("name"),
			Email: r.URL.Query().Get("email"),
			Age:   30, // simplified: would need strconv.Atoi in real code
		}

		res := BenchRes{
			Message: "Hello, " + req.Name,
			UserID:  12345,
			Success: true,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(res); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/user?name=John+Doe&email=john@example.com&age=30", http.NoBody)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			b.Fatalf("expected status 200, got %d", rec.Code)
		}
	}
}

// ============================================================================
// Overhead Measurement - Shows the actual cost of httprpc's abstraction
// ============================================================================

// This measures JUST the routing overhead (no JSON work)
func BenchmarkHTTPRPC_Overhead_Routing(b *testing.B) {
	r := New()
	emptyHandler := func(_ context.Context, _ struct{}) (struct{}, error) {
		return struct{}{}, nil
	}
	RegisterHandler[struct{}, struct{}](r.EndpointGroup, POST(emptyHandler, "/api/ping"))

	handler, err := r.Handler()
	if err != nil {
		b.Fatalf("failed to build handler: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/ping", http.NoBody)
		handler.ServeHTTP(rec, req)
	}
}

func BenchmarkNetHTTP_Overhead_Routing(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/ping", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/ping", http.NoBody)
		mux.ServeHTTP(rec, req)
	}
}

// ============================================================================
// Multiple Routes
// ============================================================================

func BenchmarkHTTPRPC_MultipleRoutes(b *testing.B) {
	r := New()

	// Register 10 different routes
	for i := range 10 {
		path := "/api/endpoint" + string(rune('0'+i))
		RegisterHandler[BenchReq, BenchRes](r.EndpointGroup, POST(benchHandler, path))
	}

	handler, err := r.Handler()
	if err != nil {
		b.Fatalf("failed to build handler: %v", err)
	}

	reqBody := []byte(`{"name":"John Doe","email":"john@example.com","age":30}`)

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/endpoint5", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			b.Fatalf("expected status 200, got %d", rec.Code)
		}
	}
}

func BenchmarkNetHTTP_MultipleRoutes(b *testing.B) {
	mux := http.NewServeMux()

	// Register 10 different routes
	for i := range 10 {
		path := "/api/endpoint" + string(rune('0'+i))
		mux.HandleFunc(path, benchNetHTTPHandler)
	}

	reqBody := []byte(`{"name":"John Doe","email":"john@example.com","age":30}`)

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/endpoint5", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			b.Fatalf("expected status 200, got %d", rec.Code)
		}
	}
}

// ============================================================================
// With Middleware
// ============================================================================

func BenchmarkHTTPRPC_WithMiddleware(b *testing.B) {
	r := New()

	// Add a simple middleware
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("X-Custom-Header", "value")
			next.ServeHTTP(w, req)
		})
	})

	RegisterHandler[BenchReq, BenchRes](r.EndpointGroup, POST(benchHandler, "/api/user"))

	handler, err := r.Handler()
	if err != nil {
		b.Fatalf("failed to build handler: %v", err)
	}

	reqBody := []byte(`{"name":"John Doe","email":"john@example.com","age":30}`)

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/user", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			b.Fatalf("expected status 200, got %d", rec.Code)
		}
	}
}

func BenchmarkNetHTTP_WithMiddleware(b *testing.B) {
	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("X-Custom-Header", "value")
			next.ServeHTTP(w, req)
		})
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/user", benchNetHTTPHandler)
	handler := middleware(mux)

	reqBody := []byte(`{"name":"John Doe","email":"john@example.com","age":30}`)

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/user", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			b.Fatalf("expected status 200, got %d", rec.Code)
		}
	}
}

// ============================================================================
// Error Handling
// ============================================================================

func BenchmarkHTTPRPC_ErrorHandling(b *testing.B) {
	r := New()
	RegisterHandler[BenchReq, BenchRes](r.EndpointGroup, POST(benchHandler, "/api/user"))

	handler, err := r.Handler()
	if err != nil {
		b.Fatalf("failed to build handler: %v", err)
	}

	// Invalid JSON to trigger error
	reqBody := []byte(`{"name":"John Doe","email":`)

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/user", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			b.Fatalf("expected status 400, got %d", rec.Code)
		}
	}
}

func BenchmarkNetHTTP_ErrorHandling(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/user", benchNetHTTPHandler)

	// Invalid JSON to trigger error
	reqBody := []byte(`{"name":"John Doe","email":`)

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/user", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			b.Fatalf("expected status 400, got %d", rec.Code)
		}
	}
}

// ============================================================================
// Large Payload
// ============================================================================

type LargeReq struct {
	Items []BenchReq `json:"items"`
}

type LargeRes struct {
	Results []BenchRes `json:"results"`
	Total   int        `json:"total"`
}

func largeBenchHandler(_ context.Context, req LargeReq) (LargeRes, error) {
	results := make([]BenchRes, len(req.Items))
	for i, item := range req.Items {
		results[i] = BenchRes{
			Message: "Hello, " + item.Name,
			UserID:  i,
			Success: true,
		}
	}
	return LargeRes{
		Results: results,
		Total:   len(results),
	}, nil
}

func BenchmarkHTTPRPC_LargePayload(b *testing.B) {
	r := New()
	RegisterHandler[LargeReq, LargeRes](r.EndpointGroup, POST(largeBenchHandler, "/api/batch"))

	handler, err := r.Handler()
	if err != nil {
		b.Fatalf("failed to build handler: %v", err)
	}

	// Create a payload with 100 items
	items := make([]BenchReq, 100)
	for i := range items {
		items[i] = BenchReq{
			Name:  "User " + string(rune('0'+i%10)),
			Email: "user@example.com",
			Age:   20 + i%50,
		}
	}
	reqBody, err := json.Marshal(LargeReq{Items: items})
	if err != nil {
		b.Fatalf("marshal request: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/batch", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			b.Fatalf("expected status 200, got %d", rec.Code)
		}
	}
}

func BenchmarkNetHTTP_LargePayload(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/batch", func(w http.ResponseWriter, r *http.Request) {
		var req LargeReq
		defer func() { _ = r.Body.Close() }()
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		results := make([]BenchRes, len(req.Items))
		for i, item := range req.Items {
			results[i] = BenchRes{
				Message: "Hello, " + item.Name,
				UserID:  i,
				Success: true,
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if encodeErr := json.NewEncoder(w).Encode(LargeRes{
			Results: results,
			Total:   len(results),
		}); encodeErr != nil {
			return
		}
	})

	// Create a payload with 100 items
	items := make([]BenchReq, 100)
	for i := range items {
		items[i] = BenchReq{
			Name:  "User " + string(rune('0'+i%10)),
			Email: "user@example.com",
			Age:   20 + i%50,
		}
	}
	reqBody, err := json.Marshal(LargeReq{Items: items})
	if err != nil {
		b.Fatalf("marshal request: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/batch", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			b.Fatalf("expected status 200, got %d", rec.Code)
		}
	}
}
