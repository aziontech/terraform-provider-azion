package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// resourceSchemas returns every registered resource schema keyed by type name.
func resourceSchemas(t *testing.T) map[string]schema.Schema {
	t.Helper()
	ctx := context.Background()
	out := map[string]schema.Schema{}
	for _, newResource := range New("test").Resources(ctx) {
		res := newResource()

		mdResp := &resource.MetadataResponse{}
		res.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "azion"}, mdResp)

		schemaResp := &resource.SchemaResponse{}
		res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
		if schemaResp.Diagnostics.HasError() {
			t.Fatalf("%s: schema errors: %v", mdResp.TypeName, schemaResp.Diagnostics)
		}
		out[mdResp.TypeName] = schemaResp.Schema
	}
	return out
}

// TestResourceSchemasUseNestedBlocks asserts that no resource schema contains a nested
// attribute. Terraform decodes nested attributes as a single value and silently discards
// object keys that are not in the schema, so a user typo inside `foo = { ... }` is accepted
// and dropped before the provider can see it. Nested blocks are decoded through the HCL body
// schema, which rejects unknown arguments.
//
// See AGENTS.md, "Resource Schemas Use Nested Blocks".
func TestResourceSchemasUseNestedBlocks(t *testing.T) {
	for name, s := range resourceSchemas(t) {
		for attrName, attr := range s.Attributes {
			checkNotNestedAttribute(t, name, attrName, attr)
		}
		for blockName, block := range s.Blocks {
			walkBlock(t, name, blockName, block, func(path string, attr schema.Attribute) {
				checkNotNestedAttribute(t, name, path, attr)
			})
		}
	}
}

func checkNotNestedAttribute(t *testing.T, res, path string, attr schema.Attribute) {
	t.Helper()
	switch attr.(type) {
	case schema.SingleNestedAttribute, schema.ListNestedAttribute,
		schema.SetNestedAttribute, schema.MapNestedAttribute:
		t.Errorf("%s: %q is a %T; resource schemas must use nested blocks so that unknown "+
			"arguments are rejected instead of silently dropped", res, path, attr)
	}
}

// TestNoRequiredAttributesUnderOptionalSingleBlock guards the SingleNestedBlock pitfall: the
// framework enforces a block's Required attributes even when the block itself is absent from
// the configuration, which makes an optional block impossible to omit. Such attributes must be
// Optional plus an AlsoRequires validator on the parent block.
//
// Attributes inside list/set block elements are exempt: they are only validated for elements
// the user actually wrote.
func TestNoRequiredAttributesUnderOptionalSingleBlock(t *testing.T) {
	for name, s := range resourceSchemas(t) {
		for blockName, block := range s.Blocks {
			walkSingleBlocks(t, name, blockName, block, true)
		}
	}
}

func walkSingleBlocks(t *testing.T, res, path string, block schema.Block, parentMandatory bool) {
	t.Helper()

	attrs, blocks := nestedOf(block)

	mandatory := true
	if sb, ok := block.(schema.SingleNestedBlock); ok {
		mandatory = parentMandatory && hasIsRequired(sb.Validators)
		if !mandatory {
			for attrName, attr := range attrs {
				if attr.IsRequired() {
					t.Errorf("%s: %q is Required but sits under optional block %q; the framework "+
						"enforces it even when the block is omitted. Make it Optional and add "+
						"objectvalidator.AlsoRequires on %q instead",
						res, path+"."+attrName, path, path)
				}
			}
		}
	}

	for childName, child := range blocks {
		walkSingleBlocks(t, res, path+"."+childName, child, mandatory)
	}
}

// hasIsRequired reports whether the validators include objectvalidator.IsRequired().
func hasIsRequired(validators []validator.Object) bool {
	want := objectvalidator.IsRequired().Description(context.Background())
	for _, v := range validators {
		if v.Description(context.Background()) == want {
			return true
		}
	}
	return false
}

// nestedOf returns the attributes and blocks nested inside a block, regardless of nesting mode.
func nestedOf(block schema.Block) (map[string]schema.Attribute, map[string]schema.Block) {
	switch b := block.(type) {
	case schema.SingleNestedBlock:
		return b.Attributes, b.Blocks
	case schema.ListNestedBlock:
		return b.NestedObject.Attributes, b.NestedObject.Blocks
	case schema.SetNestedBlock:
		return b.NestedObject.Attributes, b.NestedObject.Blocks
	default:
		return nil, nil
	}
}

func walkBlock(t *testing.T, res, path string, block schema.Block, visit func(string, schema.Attribute)) {
	t.Helper()
	attrs, blocks := nestedOf(block)
	if attrs == nil && blocks == nil {
		t.Errorf("%s: %q has unhandled block type %T", res, path, block)
		return
	}
	for attrName, attr := range attrs {
		visit(path+"."+attrName, attr)
	}
	for childName, child := range blocks {
		walkBlock(t, res, path+"."+childName, child, visit)
	}
}

// TestResourceSchemasHaveNoDanglingBlockPaths is a sanity check that every resource exposes at
// least one nested block, i.e. the migration covered it. Resources that are intentionally flat
// are listed as exceptions.
func TestResourceSchemasHaveNoDanglingBlockPaths(t *testing.T) {
	flat := map[string]bool{
		"azion_application_rule_engine_order": true,
		"azion_firewall_rule_engine_order":    true,
	}
	var missing []string
	for name, s := range resourceSchemas(t) {
		if len(s.Blocks) == 0 && !flat[name] {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		t.Errorf("resources without nested blocks (add to the exception list if intentionally flat): %s",
			strings.Join(missing, ", "))
	}
}
