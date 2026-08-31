package utils

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"testing"
)

func TestIsReferencedByAnotherResource(t *testing.T) {
	tests := []struct {
		name string
		body string
		err  error
		want bool
	}{
		{
			name: "firewall in use by workload (code 24003)",
			body: `{"errors":[{"code":"24003","title":"Cannot Delete Firewall","detail":"To delete this firewall, you must first remove its usage in the following workloads: [1781601824].","status":"400","meta":{"workloads_using_firewall":[1781601824]}}]}`,
			want: true,
		},
		{
			name: "certificate in use (code 16000)",
			body: `{"errors":[{"code":"16000","detail":"This certificate is in use and cannot be deleted."}]}`,
			want: true,
		},
		{
			name: "legacy referenced by another resource",
			body: `{"detail":"resource is referenced by another resource"}`,
			want: true,
		},
		{
			name: "match comes from error string only",
			body: "",
			err:  errors.New("400 Bad Request: you must first remove its usage"),
			want: true,
		},
		{
			name: "unrelated 400 should not retry",
			body: `{"errors":[{"detail":"invalid field value"}]}`,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{Body: io.NopCloser(bytes.NewReader([]byte(tt.body)))}
			got := isReferencedByAnotherResource(resp, tt.err)
			if got != tt.want {
				t.Fatalf("isReferencedByAnotherResource() = %v, want %v", got, tt.want)
			}

			// Body must remain readable for downstream error reporting.
			rest, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("reading restored body: %v", err)
			}
			if string(rest) != tt.body {
				t.Fatalf("body not restored: got %q, want %q", string(rest), tt.body)
			}
		})
	}
}

func TestReadAPIErrorBody(t *testing.T) {
	tests := []struct {
		name     string
		response *http.Response
		want     string
	}{
		{
			name:     "nil response",
			response: nil,
			want:     "API request failed",
		},
		{
			name:     "nil body",
			response: &http.Response{StatusCode: 500},
			want:     "API request failed",
		},
		{
			name: "empty body",
			response: &http.Response{
				StatusCode: 502,
				Body:       io.NopCloser(bytes.NewReader(nil)),
			},
			want: "API request failed with status 502",
		},
		{
			name: "body is returned verbatim",
			response: &http.Response{
				StatusCode: 400,
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"detail":"invalid"}`))),
			},
			want: `{"detail":"invalid"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ReadAPIErrorBody(tt.response); got != tt.want {
				t.Errorf("ReadAPIErrorBody() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStatusCodeOf(t *testing.T) {
	if got := StatusCodeOf(nil); got != 0 {
		t.Errorf("StatusCodeOf(nil) = %d, want 0", got)
	}
	if got := StatusCodeOf(&http.Response{StatusCode: 404}); got != 404 {
		t.Errorf("StatusCodeOf(404) = %d, want 404", got)
	}
}

func TestRetryOn429WithoutResponse(t *testing.T) {
	// A transport error yields a nil response; the retry helper must return it
	// instead of dereferencing it.
	calls := 0
	_, response, err := RetryOn429(func() (*string, *http.Response, error) { //nolint:bodyclose // the call always returns a nil response
		calls++
		return nil, nil, errors.New("connection refused")
	}, 5)

	if calls != 1 {
		t.Errorf("apiCall was invoked %d times, want 1", calls)
	}
	if response != nil {
		t.Errorf("response = %v, want nil", response)
	}
	if err == nil || err.Error() != "connection refused" {
		t.Errorf("err = %v, want \"connection refused\"", err)
	}
}
