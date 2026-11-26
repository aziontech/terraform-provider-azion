package provider

import (
	"context"
	"sort"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// Terraform converts the value of a nested attribute (`x = { ... }`) to the
// schema's object type, and that conversion silently drops arguments the schema
// does not define. A misspelled argument, or one written at the wrong nesting
// level, is ignored instead of reported. Blocks (`x { ... }`) are validated by
// Terraform itself and fail with "Unsupported argument".
//
// Every nested object a practitioner can write must therefore be a block. Only
// Computed-only nested attributes (results the provider fills in) are allowed.
func TestResourceNestedObjectsAreBlocks(t *testing.T) {
	ctx := context.Background()
	p := &azionProvider{}

	for _, newResource := range p.Resources(ctx) {
		r := newResource()

		metadata := &fwresource.MetadataResponse{}
		r.Metadata(ctx, fwresource.MetadataRequest{ProviderTypeName: "azion"}, metadata)

		schemaResponse := &fwresource.SchemaResponse{}
		r.Schema(ctx, fwresource.SchemaRequest{}, schemaResponse)
		if schemaResponse.Diagnostics.HasError() {
			t.Errorf("%s: building schema: %v", metadata.TypeName, schemaResponse.Diagnostics)
			continue
		}

		var offending []string
		collectWritableNestedAttributes("", schemaResponse.Schema.Attributes, schemaResponse.Schema.Blocks, &offending)
		sort.Strings(offending)
		for _, name := range offending {
			t.Errorf("%s: %s is a nested attribute that can be configured; use a block so unknown arguments are rejected", metadata.TypeName, name)
		}
	}
}

func collectWritableNestedAttributes(prefix string, attributes map[string]schema.Attribute, blocks map[string]schema.Block, offending *[]string) {
	for name, attribute := range attributes {
		nested, ok := attribute.(schema.NestedAttribute)
		if !ok {
			continue
		}
		if attribute.IsRequired() || attribute.IsOptional() {
			*offending = append(*offending, prefix+name)
		}
		nestedAttributes := map[string]schema.Attribute{}
		for attrName, a := range nested.GetNestedObject().GetAttributes() {
			if resourceAttribute, ok := a.(schema.Attribute); ok {
				nestedAttributes[attrName] = resourceAttribute
			}
		}
		collectWritableNestedAttributes(prefix+name+".", nestedAttributes, nil, offending)
	}

	for name, block := range blocks {
		object := block.GetNestedObject()
		nestedAttributes := map[string]schema.Attribute{}
		for attrName, attribute := range object.GetAttributes() {
			if a, ok := attribute.(schema.Attribute); ok {
				nestedAttributes[attrName] = a
			}
		}
		nestedBlocks := map[string]schema.Block{}
		for blockName, nestedBlock := range object.GetBlocks() {
			if b, ok := nestedBlock.(schema.Block); ok {
				nestedBlocks[blockName] = b
			}
		}
		collectWritableNestedAttributes(prefix+name+".", nestedAttributes, nestedBlocks, offending)
	}
}
