package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHandleRoot_Redirect verifies root path redirects to default document.
func TestHandleRoot_Redirect(t *testing.T) {
	srv := New(Config{Port: ":8080", StaticDir: "testdata"})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	srv.handleRoot(rec, req)

	if rec.Code != http.StatusTemporaryRedirect {
		t.Errorf("expected status 307, got %d", rec.Code)
	}

	location := rec.Header().Get("Location")
	if location != "/doc/default" {
		t.Errorf("expected redirect to /doc/default, got %s", location)
	}
}

// TestHandleRoot_NotFound verifies non-root paths return 404.
func TestHandleRoot_NotFound(t *testing.T) {
	srv := New(Config{Port: ":8080", StaticDir: "testdata"})

	req := httptest.NewRequest(http.MethodGet, "/invalid", nil)
	rec := httptest.NewRecorder()

	srv.handleRoot(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

// TestExtractDocumentID_Valid verifies valid document IDs are extracted correctly.
func TestExtractDocumentID_Valid(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/ws/test-doc", "test-doc"},
		{"/ws/doc_123", "doc_123"},
		{"/ws/MyDoc-2024", "MyDoc-2024"},
		{"/ws/a", "a"},
		{"/ws/ABC123", "ABC123"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got, err := extractDocumentID(tt.path, "/ws/")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// TestExtractDocumentID_Invalid verifies invalid document IDs return errors.
func TestExtractDocumentID_Invalid(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"empty", "/ws/"},
		{"spaces only", "/ws/   "},
		{"special chars", "/ws/doc@123"},
		{"with slash", "/ws/doc/123"},
		{"with dot", "/ws/my.doc"},
		{"unicode", "/ws/test日本"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := extractDocumentID(tt.path, "/ws/")
			if err == nil {
				t.Error("expected error, got nil")
			}

			// Verify it's a ValidationError
			if _, ok := err.(*ValidationError); !ok {
				t.Errorf("expected *ValidationError, got %T", err)
			}
		})
	}
}

// TestIsValidDocumentID tests the document ID validation logic.
func TestIsValidDocumentID(t *testing.T) {
	tests := []struct {
		name  string
		id    string
		valid bool
	}{
		{
			name:  "alphanumeric lowercase",
			id:    "mytest123",
			valid: true,
		},
		{
			name:  "alphanumeric uppercase",
			id:    "MYTEST123",
			valid: true,
		},
		{
			name:  "with hyphens",
			id:    "my-test-doc",
			valid: true,
		},
		{
			name:  "with underscores",
			id:    "my_test_doc",
			valid: true,
		},
		{
			name:  "mixed valid characters",
			id:    "My_Test-Doc-123",
			valid: true,
		},
		{
			name:  "empty string",
			id:    "",
			valid: false,
		},
		{
			name:  "with spaces",
			id:    "my test",
			valid: false,
		},
		{
			name:  "with special characters",
			id:    "my@test",
			valid: false,
		},
		{
			name:  "with slashes",
			id:    "my/test",
			valid: false,
		},
		{
			name:  "with dots",
			id:    "my.test",
			valid: false,
		},
		{
			name:  "too long",
			id:    string(make([]byte, 101)), // 101 characters
			valid: false,
		},
		{
			name:  "exactly at limit",
			id:    string(make([]byte, 100)), // 100 characters
			valid: false,                     // All zero bytes, which are invalid characters
		},
		{
			name:  "unicode characters",
			id:    "test日本",
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidDocumentID(tt.id)
			if got != tt.valid {
				t.Errorf("isValidDocumentID(%q) = %v, want %v", tt.id, got, tt.valid)
			}
		})
	}
}

// TestIsValidDocumentID_EdgeCases tests edge cases for document ID validation.
func TestIsValidDocumentID_EdgeCases(t *testing.T) {
	// Test single character IDs
	validSingleChar := []string{"a", "A", "0", "-", "_"}
	for _, id := range validSingleChar {
		if !isValidDocumentID(id) {
			t.Errorf("isValidDocumentID(%q) = false, want true for single valid character", id)
		}
	}

	// Test boundary length (exactly 100 valid characters)
	longValidID := ""
	for i := 0; i < 100; i++ {
		longValidID += "a"
	}
	if !isValidDocumentID(longValidID) {
		t.Error("isValidDocumentID should accept 100-character string")
	}

	// Test just over boundary
	longInvalidID := longValidID + "a"
	if isValidDocumentID(longInvalidID) {
		t.Error("isValidDocumentID should reject 101-character string")
	}
}
