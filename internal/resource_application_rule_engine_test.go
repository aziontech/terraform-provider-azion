package provider

import (
	"context"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// assertBlocks fails for every dotted name that is not a block of the resource's
// schema. Nested objects are blocks so Terraform rejects unknown arguments
// instead of silently dropping them. List blocks are walked through their nested
// object, so "a.b" names block b inside list block a.
func assertBlocks(t *testing.T, r fwresource.Resource, names ...string) {
	t.Helper()

	schemaResponse := &fwresource.SchemaResponse{}
	r.Schema(context.Background(), fwresource.SchemaRequest{}, schemaResponse)
	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("building schema: %v", schemaResponse.Diagnostics)
	}
	if diags := schemaResponse.Schema.ValidateImplementation(context.Background()); diags.HasError() {
		t.Fatalf("invalid schema implementation: %v", diags)
	}

	for _, name := range names {
		if findBlock(schemaResponse.Schema.Blocks, name) == nil {
			t.Errorf("%s: must be a block, not a nested attribute", name)
		}
	}
}

func findBlock(blocks map[string]schema.Block, dotted string) schema.Block {
	for name, block := range blocks {
		if dotted == name {
			return block
		}
		prefix := name + "."
		if len(dotted) > len(prefix) && dotted[:len(prefix)] == prefix {
			nested := map[string]schema.Block{}
			for nestedName, nestedBlock := range block.GetNestedObject().GetBlocks() {
				if b, ok := nestedBlock.(schema.Block); ok {
					nested[nestedName] = b
				}
			}
			return findBlock(nested, dotted[len(prefix):])
		}
	}
	return nil
}

func TestApplicationRuleEngineNestedObjectsAreBlocks(t *testing.T) {
	assertBlocks(t, NewApplicationRulesEngineResource(),
		"results",
		"results.behaviors",
		"results.behaviors.behavior",
		"results.behaviors.behavior.attributes",
		"results.behaviors.behavior.capture_attributes",
		"results.criteria",
		"results.criteria.entries",
		"results.criteria.entries.criterion",
	)
}

// Blocks have no Required flag: the required ones carry a validator instead, and
// zero list blocks reach validators as a null list, so IsRequired rejects them.
func TestApplicationRuleEngineRequiredBlocksAreValidated(t *testing.T) {
	schemaResponse := &fwresource.SchemaResponse{}
	NewApplicationRulesEngineResource().Schema(context.Background(), fwresource.SchemaRequest{}, schemaResponse)

	for _, name := range []string{
		"results",
		"results.behaviors",
		"results.behaviors.behavior",
		"results.criteria.entries",
		"results.criteria.entries.criterion",
	} {
		block := findBlock(schemaResponse.Schema.Blocks, name)
		switch b := block.(type) {
		case schema.SingleNestedBlock:
			if len(b.Validators) == 0 {
				t.Errorf("%s: required block has no validator", name)
			}
		case schema.ListNestedBlock:
			if len(b.Validators) == 0 {
				t.Errorf("%s: required list block has no validator", name)
			}
		default:
			t.Errorf("%s: unexpected block %T", name, block)
		}
	}
}
