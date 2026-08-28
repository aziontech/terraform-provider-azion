package provider

import (
	"net/http"
	"testing"
)

func TestIsHTTPStatusHandlesNilResponse(t *testing.T) {
	if isHTTPStatus(nil, http.StatusTooManyRequests) {
		t.Fatalf("nil response must not match HTTP status")
	}
}
