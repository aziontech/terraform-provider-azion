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
