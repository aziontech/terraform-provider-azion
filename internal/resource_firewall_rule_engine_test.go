package provider

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The API validates the argument of the network list operators as an integer, so it must be
// serialized as a JSON number. Every other operator keeps its argument as a JSON string, even
// when the value happens to be numeric.
func TestBuildFirewallCriterionArgument(t *testing.T) {
	tests := []struct {
		name     string
		operator string
		argument string
		want     string
		wantErr  bool
	}{
		{
			name:     "is_in_list sends the network list ID as a number",
			operator: "is_in_list",
			argument: "12345",
			want:     "12345",
		},
		{
			name:     "is_not_in_list sends the network list ID as a number",
			operator: "is_not_in_list",
			argument: "12345",
			want:     "12345",
		},
		{
			name:     "surrounding whitespace is tolerated",
			operator: "is_in_list",
			argument: "  12345  ",
			want:     "12345",
		},
		{
			name:     "non-numeric network list ID is rejected",
			operator: "is_in_list",
			argument: "my-network-list",
			wantErr:  true,
		},
		{
			name:     "numeric argument on a non-list operator stays a string",
			operator: "is_equal",
			argument: "8080",
			want:     `"8080"`,
		},
		{
			name:     "regex argument stays a string",
			operator: "matches",
			argument: "/admin.*",
			want:     `"/admin.*"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			arg, err := buildFirewallCriterionArgument(tt.operator, types.StringValue(tt.argument))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error for operator %q with argument %q, got none", tt.operator, tt.argument)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got, errMarshal := json.Marshal(arg)
			if errMarshal != nil {
				t.Fatalf("failed to marshal argument: %v", errMarshal)
			}
			if string(got) != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}
