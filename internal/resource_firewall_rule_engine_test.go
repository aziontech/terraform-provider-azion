package provider

import (
	"context"
	"encoding/json"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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

// Nested objects are blocks so Terraform rejects unknown arguments instead of
// dropping them, and the required ones carry a validator because tfplugindocs and
// Terraform treat every block as optional.
func TestFirewallRuleEngineNestedObjectsAreBlocks(t *testing.T) {
	ctx := context.Background()

	schemaResponse := &fwresource.SchemaResponse{}
	NewFirewallRuleEngineResource().Schema(ctx, fwresource.SchemaRequest{}, schemaResponse)
	if diags := schemaResponse.Schema.ValidateImplementation(ctx); diags.HasError() {
		t.Fatalf("invalid schema implementation: %v", diags)
	}

	results, ok := schemaResponse.Schema.Blocks["results"].(schema.SingleNestedBlock)
	if !ok || len(results.Validators) == 0 {
		t.Fatal("results: must be a required single block")
	}

	criteria, ok := results.Blocks["criteria"].(schema.ListNestedBlock)
	if !ok || len(criteria.Validators) == 0 {
		t.Fatal("results.criteria: must be a list block with a size validator")
	}
	entries, ok := criteria.NestedObject.Blocks["entries"].(schema.ListNestedBlock)
	if !ok || len(entries.Validators) == 0 {
		t.Fatal("results.criteria.entries: must be a list block with a size validator")
	}
	if criterion, ok := entries.NestedObject.Blocks["criterion"].(schema.SingleNestedBlock); !ok || len(criterion.Validators) == 0 {
		t.Error("results.criteria.entries.criterion: must be a required single block")
	}

	behaviors, ok := results.Blocks["behaviors"].(schema.ListNestedBlock)
	if !ok || len(behaviors.Validators) == 0 {
		t.Fatal("results.behaviors: must be a list block with a size validator")
	}
	behavior, ok := behaviors.NestedObject.Blocks["behavior"].(schema.SingleNestedBlock)
	if !ok || len(behavior.Validators) == 0 {
		t.Fatal("results.behaviors.behavior: must be a required single block")
	}
	if attributes, ok := behavior.Blocks["attributes"].(schema.SingleNestedBlock); !ok || len(attributes.Validators) != 0 {
		t.Error("results.behaviors.behavior.attributes: must be an optional single block")
	}
}
