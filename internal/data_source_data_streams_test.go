package provider

import (
	"context"
	"testing"

	fwdatasource "github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func dataStreamsDataSourceSchema(t *testing.T) fwdatasource.SchemaResponse {
	t.Helper()

	sr := &fwdatasource.SchemaResponse{}
	dataSourceAzionDataStreams().Schema(context.Background(), fwdatasource.SchemaRequest{}, sr)

	if sr.Diagnostics.HasError() {
		t.Fatalf("building schema: %v", sr.Diagnostics)
	}

	return *sr
}

func dataStreamTemplatesDataSourceSchema(t *testing.T) fwdatasource.SchemaResponse {
	t.Helper()

	sr := &fwdatasource.SchemaResponse{}
	dataSourceAzionDataStreamTemplates().Schema(context.Background(), fwdatasource.SchemaRequest{}, sr)

	if sr.Diagnostics.HasError() {
		t.Fatalf("building schema: %v", sr.Diagnostics)
	}

	return *sr
}

func TestDataStreamsDataSourceSchemaImplementation(t *testing.T) {
	ctx := context.Background()

	if diags := dataStreamsDataSourceSchema(t).Schema.ValidateImplementation(ctx); diags.HasError() {
		t.Errorf("azion_data_streams: invalid schema implementation: %v", diags)
	}
	if diags := dataStreamTemplatesDataSourceSchema(t).Schema.ValidateImplementation(ctx); diags.HasError() {
		t.Errorf("azion_data_stream_templates: invalid schema implementation: %v", diags)
	}
}

// emptyState builds a state value that is null throughout, the way the
// framework hands one to a data source's Read.
func emptyState(ctx context.Context, t *testing.T, schema fwdatasource.SchemaResponse) tfsdk.State {
	t.Helper()

	return tfsdk.State{
		Schema: schema.Schema,
		Raw:    tftypes.NewValue(schema.Schema.Type().TerraformType(ctx), nil),
	}
}

// resultsIsNull reports whether the `results` attribute of a state value came
// out null rather than an empty collection.
func resultsIsNull(t *testing.T, state tfsdk.State) bool {
	t.Helper()

	var object map[string]tftypes.Value
	if err := state.Raw.As(&object); err != nil {
		t.Fatalf("reading state: %v", err)
	}

	results, ok := object["results"]
	if !ok {
		t.Fatal("state has no results attribute")
	}
	return results.IsNull()
}

// An account with no data streams must yield `results = []`, not `results =
// null`. A null collection cannot be iterated, so `for s in
// data.azion_data_streams.all.results` fails outright with "Iteration over null
// value" — which is what a nil Go slice produces.
func TestDataStreamsDataSourceEmptyResultsIsAList(t *testing.T) {
	ctx := context.Background()
	schema := dataStreamsDataSourceSchema(t)

	t.Run("nil slice would be null", func(t *testing.T) {
		state := emptyState(ctx, t, schema)
		model := DataStreamsDataSourceModel{
			ID:      types.StringValue("Get All Data Streams"),
			Counter: types.Int64Value(0),
			Results: nil,
		}
		if diags := state.Set(ctx, &model); diags.HasError() {
			t.Fatalf("setting state: %v", diags)
		}

		if !resultsIsNull(t, state) {
			t.Skip("the framework no longer maps a nil slice to null; the guard below is what matters")
		}
	})

	t.Run("empty slice is an empty list", func(t *testing.T) {
		state := emptyState(ctx, t, schema)
		model := DataStreamsDataSourceModel{
			ID:      types.StringValue("Get All Data Streams"),
			Counter: types.Int64Value(0),
			Results: []DataStreamResults{},
		}
		if diags := state.Set(ctx, &model); diags.HasError() {
			t.Fatalf("setting state: %v", diags)
		}

		if resultsIsNull(t, state) {
			t.Error("results came out null; an empty account must produce an empty list")
		}
	})
}

// Same guarantee for the templates listing.
func TestDataStreamTemplatesDataSourceEmptyResultsIsAList(t *testing.T) {
	ctx := context.Background()
	state := emptyState(ctx, t, dataStreamTemplatesDataSourceSchema(t))

	model := DataStreamTemplatesDataSourceModel{
		ID:      types.StringValue("Get All Data Stream Templates"),
		Counter: types.Int64Value(0),
		Results: []DataStreamTemplateResults{},
	}
	if diags := state.Set(ctx, &model); diags.HasError() {
		t.Fatalf("setting state: %v", diags)
	}

	if resultsIsNull(t, state) {
		t.Error("results came out null; an empty account must produce an empty list")
	}
}

// `counter` is a plain Int64 on the model, so leaving it unset stores null and
// renders as `null` rather than `0` for an account with no streams.
func TestDataStreamsDataSourceCounterIsZeroNotNull(t *testing.T) {
	ctx := context.Background()
	state := emptyState(ctx, t, dataStreamsDataSourceSchema(t))

	model := DataStreamsDataSourceModel{
		ID:      types.StringValue("Get All Data Streams"),
		Counter: types.Int64Value(0),
		Results: []DataStreamResults{},
	}
	if diags := state.Set(ctx, &model); diags.HasError() {
		t.Fatalf("setting state: %v", diags)
	}

	var object map[string]tftypes.Value
	if err := state.Raw.As(&object); err != nil {
		t.Fatalf("reading state: %v", err)
	}
	if object["counter"].IsNull() {
		t.Error("counter came out null; it must be 0 when the API reports no streams")
	}
}
