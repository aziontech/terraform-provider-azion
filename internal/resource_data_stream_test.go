package provider

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func dataStreamSchema(t *testing.T) fwresource.SchemaResponse {
	t.Helper()

	sr := &fwresource.SchemaResponse{}
	NewDataStreamResource().Schema(context.Background(), fwresource.SchemaRequest{}, sr)

	if sr.Diagnostics.HasError() {
		t.Fatalf("building schema: %v", sr.Diagnostics)
	}

	return *sr
}

func TestDataStreamSchemaImplementation(t *testing.T) {
	ctx := context.Background()

	if diags := dataStreamSchema(t).Schema.ValidateImplementation(ctx); diags.HasError() {
		t.Errorf("invalid schema implementation: %v", diags)
	}
}

// listOfSize builds a list value of the given length. The size validators only
// look at the element count, so the element type is irrelevant.
func listOfSize(n int) types.List {
	elements := make([]attr.Value, n)
	for i := range elements {
		elements[i] = types.StringValue("element")
	}
	return types.ListValueMust(types.StringType, elements)
}

// dataStreamListBlock returns one of the list blocks under `data_stream`
// (`inputs`, `transform`, `outputs`).
func dataStreamListBlock(t *testing.T, name string) schema.ListNestedBlock {
	t.Helper()

	stream, ok := dataStreamSchema(t).Schema.Blocks["data_stream"].(schema.SingleNestedBlock)
	if !ok {
		t.Fatalf("data_stream: expected a single nested block")
	}
	block, ok := stream.Blocks[name].(schema.ListNestedBlock)
	if !ok {
		t.Fatalf("data_stream.%s: expected a list nested block", name)
	}
	return block
}

// validateList runs every list validator attached to a list block under
// `data_stream` and reports whether the value was rejected.
func validateList(t *testing.T, name string, value types.List) bool {
	t.Helper()
	ctx := context.Background()

	resp := &validator.ListResponse{}
	for _, v := range dataStreamListBlock(t, name).ListValidators() {
		v.ValidateList(ctx, validator.ListRequest{
			Path:        path.Root("data_stream").AtName(name),
			ConfigValue: value,
		}, resp)
	}
	return resp.Diagnostics.HasError()
}

// rejectsMissingObject runs the object validators of a single nested block
// against an absent block and reports whether it was rejected.
func rejectsMissingObject(t *testing.T, block schema.SingleNestedBlock) bool {
	t.Helper()
	ctx := context.Background()

	resp := &validator.ObjectResponse{}
	for _, v := range block.ObjectValidators() {
		v.ValidateObject(ctx, validator.ObjectRequest{
			ConfigValue: types.ObjectNull(block.Type().(types.ObjectType).AttrTypes),
		}, resp)
	}
	return resp.Diagnostics.HasError()
}

// validateString runs every string validator attached to an attribute.
func validateString(t *testing.T, attrPath path.Path, value string) bool {
	t.Helper()
	ctx := context.Background()

	attribute, diags := dataStreamSchema(t).Schema.AttributeAtPath(ctx, attrPath)
	if diags.HasError() {
		t.Fatalf("%s: %v", attrPath, diags)
	}

	withValidators, ok := attribute.(interface{ StringValidators() []validator.String })
	if !ok {
		t.Fatalf("%s: attribute carries no string validators", attrPath)
	}

	resp := &validator.StringResponse{}
	for _, v := range withValidators.StringValidators() {
		v.ValidateString(ctx, validator.StringRequest{
			Path:        attrPath,
			ConfigValue: types.StringValue(value),
		}, resp)
	}
	return resp.Diagnostics.HasError()
}

// requireJSONEqual compares two JSON documents by value. The SDK's generated
// marshallers build a map before encoding, so key order is not stable and
// cannot be asserted on.
func requireJSONEqual(t *testing.T, got []byte, want string) {
	t.Helper()

	var gotValue, wantValue interface{}
	if err := json.Unmarshal(got, &gotValue); err != nil {
		t.Fatalf("decoding produced JSON: %v", err)
	}
	if err := json.Unmarshal([]byte(want), &wantValue); err != nil {
		t.Fatalf("decoding expected JSON: %v", err)
	}

	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Errorf("JSON mismatch\n got: %s\nwant: %s", got, want)
	}
}

// The API rejects a missing or empty transform list (10059 "Required Field",
// 10049 "Min Length List Field"). A list block has no Required flag, and zero
// blocks reach the provider as a null list during validation and as an empty
// one during planning, so both IsRequired and the minimum size are needed.
func TestDataStreamTransformIsRequired(t *testing.T) {
	if !validateList(t, "transform", types.ListNull(types.StringType)) {
		t.Error("an absent transform list must be rejected")
	}
	if !validateList(t, "transform", listOfSize(0)) {
		t.Error("zero transform blocks must be rejected")
	}
	// A render_template entry plus sampling or filter_workloads is also
	// required, but that is a cross-entry rule the API owns; the schema only
	// enforces the size.
	for _, size := range []int{1, 2, 3} {
		if validateList(t, "transform", listOfSize(size)) {
			t.Errorf("a transform list of %d entries must be accepted", size)
		}
	}
}

// inputs and outputs are both stored one-per-stream by the API. Given a longer
// list it keeps a single entry and reports success, so the schema caps them
// instead of letting the discarded entries surface as an inconsistent result.
// IsRequired and the lower bound also make the blocks mandatory.
func TestDataStreamInputsAndOutputsAreCappedAtOne(t *testing.T) {
	for _, name := range []string{"inputs", "outputs"} {
		if !validateList(t, name, types.ListNull(types.StringType)) {
			t.Errorf("%s: an absent list must be rejected", name)
		}
		if validateList(t, name, listOfSize(1)) {
			t.Errorf("%s: exactly one entry must be accepted", name)
		}
		for _, size := range []int{0, 2, 3} {
			if !validateList(t, name, listOfSize(size)) {
				t.Errorf("%s: %d entries must be rejected", name, size)
			}
		}
	}
}

// Nested objects are blocks so Terraform rejects unknown arguments instead of
// dropping them. A nested attribute here would silently accept a typo.
func TestDataStreamNestedObjectsAreBlocks(t *testing.T) {
	ctx := context.Background()
	sr := dataStreamSchema(t)

	names := []string{
		"data_stream",
		"data_stream.inputs",
		"data_stream.transform",
		"data_stream.outputs",
	}
	for _, name := range names {
		if _, diags := sr.Schema.AttributeAtPath(ctx, tfPath(t, name)); !diags.HasError() {
			t.Errorf("%s: must be a block, not a nested attribute", name)
		}
	}

	stream := sr.Schema.Blocks["data_stream"].(schema.SingleNestedBlock)
	inputs := stream.Blocks["inputs"].(schema.ListNestedBlock)
	if _, ok := inputs.NestedObject.Blocks["attributes"].(schema.SingleNestedBlock); !ok {
		t.Error("data_stream.inputs.attributes: must be a single nested block")
	}
	for _, name := range []string{"sampling_attributes", "filter_workloads_attributes", "render_template_attributes"} {
		if _, ok := stream.Blocks["transform"].(schema.ListNestedBlock).NestedObject.Blocks[name].(schema.SingleNestedBlock); !ok {
			t.Errorf("data_stream.transform.%s: must be a single nested block", name)
		}
	}
	for _, name := range []string{
		"standard_attributes", "kafka_attributes", "s3_attributes", "big_query_attributes",
		"elasticsearch_attributes", "splunk_attributes", "aws_kinesis_firehose_attributes",
		"datadog_attributes", "qradar_attributes", "azure_monitor_attributes", "azure_blob_storage_attributes",
	} {
		if _, ok := stream.Blocks["outputs"].(schema.ListNestedBlock).NestedObject.Blocks[name].(schema.SingleNestedBlock); !ok {
			t.Errorf("data_stream.outputs.%s: must be a single nested block", name)
		}
	}
}

// Single blocks that used to be Required nested attributes carry IsRequired, so
// an omitted block is still a plan-time error.
func TestDataStreamRequiredSingleBlocks(t *testing.T) {
	stream := dataStreamSchema(t).Schema.Blocks["data_stream"].(schema.SingleNestedBlock)
	if !rejectsMissingObject(t, stream) {
		t.Error("data_stream: an omitted block must be rejected")
	}

	inputAttributes := stream.Blocks["inputs"].(schema.ListNestedBlock).NestedObject.Blocks["attributes"].(schema.SingleNestedBlock)
	if !rejectsMissingObject(t, inputAttributes) {
		t.Error("data_stream.inputs.attributes: an omitted block must be rejected")
	}
}

// The framework checks Required on the children of an absent single nested
// block too, so a Required attribute inside an optional `*_attributes` block
// would reject every configuration that leaves the other types' blocks out.
// Those attributes are Optional and the block's AlsoRequires validator enforces
// them once the block is written.
func TestDataStreamOptionalBlocksRequireTheirFieldsWhenSet(t *testing.T) {
	ctx := context.Background()
	stream := dataStreamSchema(t).Schema.Blocks["data_stream"].(schema.SingleNestedBlock)

	for _, listName := range []string{"transform", "outputs"} {
		entry := stream.Blocks[listName].(schema.ListNestedBlock).NestedObject
		for blockName, b := range entry.Blocks {
			block := b.(schema.SingleNestedBlock)
			where := "data_stream." + listName + "." + blockName

			// AlsoRequires describes its paths as `... these are set: "[a,b]"`.
			validatorDescriptions := ""
			alsoRequired := map[string]bool{}
			for _, v := range block.ObjectValidators() {
				description := v.Description(ctx)
				validatorDescriptions += description
				if start, end := strings.Index(description, "["), strings.LastIndex(description, "]"); start >= 0 && end > start {
					for _, name := range strings.Split(description[start+1:end], ",") {
						alsoRequired[name] = true
					}
				}
			}

			for attrName, attribute := range block.Attributes {
				if attribute.IsRequired() {
					t.Errorf("%s.%s: must be Optional; enforce it with dataStreamRequiredWhenSet", where, attrName)
				}
				mandatory := strings.HasPrefix(attribute.GetDescription(), "Required.")
				enforced := alsoRequired[attrName]
				if mandatory && !enforced {
					t.Errorf("%s.%s: documented as Required but not in the block's AlsoRequires (%s)", where, attrName, validatorDescriptions)
				}
				if !mandatory && enforced {
					t.Errorf("%s.%s: enforced by AlsoRequires but its description does not start with \"Required.\"", where, attrName)
				}
			}
		}
	}
}

// The API silently rewrites an unrecognized data_source rather than rejecting
// it, so the value has to be checked before the request goes out. `http`,
// `cells_console` and `rtm_activity` were valid in earlier API versions and are
// the likeliest stale values to appear in an existing configuration.
func TestDataStreamDataSourceIsValidated(t *testing.T) {
	dataSourcePath := path.Root("data_stream").AtName("inputs").AtListIndex(0).
		AtName("attributes").AtName("data_source")

	for _, valid := range []string{"workloads", "waf", "functions_console", "activity_history"} {
		if validateString(t, dataSourcePath, valid) {
			t.Errorf("data_source %q must be accepted", valid)
		}
	}
	for _, invalid := range []string{"http", "cells_console", "rtm_activity", ""} {
		if !validateString(t, dataSourcePath, invalid) {
			t.Errorf("data_source %q must be rejected", invalid)
		}
	}
}

// The API trims log_line_separator, so an all-whitespace value is stored as "".
// Unlike `data_set` on a template — where trimming only drops a trailing
// newline from a payload that otherwise survives — trimming destroys this
// value outright, so it is rejected at plan time rather than papered over by
// keeping the configured value in state.
func TestDataStreamLogLineSeparatorRejectsWhitespace(t *testing.T) {
	separatorPath := dataStreamOutputAttrPath("standard_attributes").AtName("log_line_separator")

	accepted := []string{
		`\n`, // the two-character escape, and the API's own default
		",",
		"|",
		"---",
		"", // explicitly empty is stored as empty
		"a\nb",
	}
	for _, value := range accepted {
		if validateString(t, separatorPath, value) {
			t.Errorf("log_line_separator %q must be accepted", value)
		}
	}

	rejected := []string{
		"\n", // a literal newline: trimmed to ""
		"\t", // likewise
		"\r\n",
		"  ",  // whitespace only
		" | ", // trimmed to "|", so state would not match the configuration
	}
	for _, value := range rejected {
		if !validateString(t, separatorPath, value) {
			t.Errorf("log_line_separator %q must be rejected", value)
		}
	}
}

// buildDataStreamRequest is the whole write path. Each output entry must carry
// its discriminator on the variant wrapper and the endpoint payload underneath
// it — `{"type": ..., "attributes": {...}}` with no `type` inside attributes.
func TestBuildDataStreamRequestOutputWireFormat(t *testing.T) {
	tests := []struct {
		name   string
		output DataStreamOutputModel
		want   string
	}{
		{
			name: "standard carries only the fields that were set",
			output: DataStreamOutputModel{
				Type: types.StringValue("standard"),
				StandardAttributes: &DataStreamStandardOutputModel{
					URL:     types.StringValue("https://logs.example.com/ingest"),
					Headers: map[string]types.String{"Authorization": types.StringValue("Bearer t")},
					MaxSize: types.Int64Value(1000000),
				},
			},
			want: `{"type":"standard","attributes":{"url":"https://logs.example.com/ingest","max_size":1000000,"headers":{"Authorization":"Bearer t"}}}`,
		},
		{
			name: "kafka",
			output: DataStreamOutputModel{
				Type: types.StringValue("kafka"),
				KafkaAttributes: &DataStreamKafkaOutputModel{
					BootstrapServers: types.StringValue("kafka:9092"),
					KafkaTopic:       types.StringValue("logs"),
					UseTLS:           types.BoolValue(true),
				},
			},
			want: `{"type":"kafka","attributes":{"bootstrap_servers":"kafka:9092","kafka_topic":"logs","use_tls":true}}`,
		},
		{
			name: "s3 omits an unset object_key_prefix",
			output: DataStreamOutputModel{
				Type: types.StringValue("s3"),
				S3Attributes: &DataStreamS3OutputModel{
					AccessKey:   types.StringValue("AK"),
					SecretKey:   types.StringValue("SK"),
					Region:      types.StringValue("us-east-1"),
					BucketName:  types.StringValue("bucket"),
					ContentType: types.StringValue("plain/text"),
					HostURL:     types.StringValue("https://s3.example.com"),
				},
			},
			want: `{"type":"s3","attributes":{"access_key":"AK","secret_key":"SK","region":"us-east-1","bucket_name":"bucket","content_type":"plain/text","host_url":"https://s3.example.com"}}`,
		},
		{
			name: "qradar",
			output: DataStreamOutputModel{
				Type:             types.StringValue("qradar"),
				QRadarAttributes: &DataStreamQRadarOutputModel{URL: types.StringValue("https://qradar.example.com")},
			},
			want: `{"type":"qradar","attributes":{"url":"https://qradar.example.com"}}`,
		},
		{
			name: "azure_monitor",
			output: DataStreamOutputModel{
				Type: types.StringValue("azure_monitor"),
				AzureMonitorAttributes: &DataStreamAzureMonitorOutputModel{
					LogType:            types.StringValue("AzionLogs"),
					SharedKey:          types.StringValue("key"),
					WorkspaceID:        types.StringValue("wid"),
					TimeGeneratedField: types.StringValue("timestamp"),
				},
			},
			want: `{"type":"azure_monitor","attributes":{"log_type":"AzionLogs","shared_key":"key","time_generated_field":"timestamp","workspace_id":"wid"}}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request, err := buildDataStreamOutputRequest(0, test.output.Type.ValueString(), test.output)
			if err != nil {
				t.Fatalf("building output: %v", err)
			}

			got, err := json.Marshal(request)
			if err != nil {
				t.Fatalf("marshalling output: %v", err)
			}

			requireJSONEqual(t, got, test.want)
		})
	}
}

// The whole request body, to pin the `[]` (never `null`) marshalling of the
// required lists and the type+attributes nesting of inputs and transforms.
func TestBuildDataStreamRequestBody(t *testing.T) {
	model := &dataStreamResourceResults{
		Name:   types.StringValue("stream"),
		Active: types.BoolValue(false),
		Inputs: []DataStreamInputModel{{
			Type:       types.StringValue("raw_logs"),
			Attributes: &DataStreamInputAttributesModel{DataSource: types.StringValue("workloads")},
		}},
		Transform: []DataStreamTransformModel{
			{
				Type:               types.StringValue("sampling"),
				SamplingAttributes: &DataStreamSamplingAttributesModel{Rate: types.Int64Value(100)},
			},
			{
				Type:                     types.StringValue("render_template"),
				RenderTemplateAttributes: &DataStreamRenderTemplateAttributesModel{Template: types.Int64Value(2)},
			},
		},
		Outputs: []DataStreamOutputModel{{
			Type: types.StringValue("qradar"),
			QRadarAttributes: &DataStreamQRadarOutputModel{
				URL: types.StringValue("https://qradar.example.com"),
			},
		}},
	}

	request, err := buildDataStreamRequest(model)
	if err != nil {
		t.Fatalf("building request: %v", err)
	}

	got, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshalling request: %v", err)
	}

	const want = `{"name":"stream","active":false,` +
		`"inputs":[{"type":"raw_logs","attributes":{"data_source":"workloads"}}],` +
		`"transform":[{"type":"sampling","attributes":{"rate":100}},{"type":"render_template","attributes":{"template":2}}],` +
		`"outputs":[{"type":"qradar","attributes":{"url":"https://qradar.example.com"}}]}`

	requireJSONEqual(t, got, want)
}

func TestBuildDataStreamOutputRequestErrors(t *testing.T) {
	tests := []struct {
		name       string
		outputType string
		output     DataStreamOutputModel
	}{
		{
			name:       "missing attributes block for the declared type",
			outputType: "kafka",
			output:     DataStreamOutputModel{Type: types.StringValue("kafka")},
		},
		{
			name:       "unsupported endpoint type",
			outputType: "carrier_pigeon",
			output:     DataStreamOutputModel{Type: types.StringValue("carrier_pigeon")},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := buildDataStreamOutputRequest(0, test.outputType, test.output); err == nil {
				t.Error("expected an error")
			}
		})
	}
}

// The API stores the transform pipeline in its own canonical order regardless
// of the order it was submitted in, so the response has to be mapped back onto
// the configured order or every apply fails with "Provider produced
// inconsistent result after apply".
func TestOrderTransformsLikePrior(t *testing.T) {
	sampling := DataStreamTransformModel{
		Type:               types.StringValue("sampling"),
		SamplingAttributes: &DataStreamSamplingAttributesModel{Rate: types.Int64Value(50)},
	}
	renderTemplate := DataStreamTransformModel{
		Type:                     types.StringValue("render_template"),
		RenderTemplateAttributes: &DataStreamRenderTemplateAttributesModel{Template: types.Int64Value(2)},
	}
	filterWorkloads := DataStreamTransformModel{Type: types.StringValue("filter_workloads")}

	// What the API always returns for a sampling + render_template pipeline.
	apiOrder := []DataStreamTransformModel{sampling, renderTemplate}

	tests := []struct {
		name  string
		prior []DataStreamTransformModel
		want  []string
	}{
		{
			name:  "configured order wins when the types match",
			prior: []DataStreamTransformModel{renderTemplate, sampling},
			want:  []string{"render_template", "sampling"},
		},
		{
			name:  "already in the API's order",
			prior: []DataStreamTransformModel{sampling, renderTemplate},
			want:  []string{"sampling", "render_template"},
		},
		{
			name:  "no prior state, as on import: the API's order is kept",
			prior: nil,
			want:  []string{"sampling", "render_template"},
		},
		{
			name:  "a different set of types is real drift and must not be masked",
			prior: []DataStreamTransformModel{filterWorkloads, renderTemplate},
			want:  []string{"sampling", "render_template"},
		},
		{
			name:  "a shorter prior list is real drift too",
			prior: []DataStreamTransformModel{sampling},
			want:  []string{"sampling", "render_template"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := orderTransformsLikePrior(test.prior, apiOrder)

			if len(got) != len(test.want) {
				t.Fatalf("got %d entries, want %d", len(got), len(test.want))
			}
			for i, want := range test.want {
				if got[i].Type.ValueString() != want {
					t.Errorf("entry %d: got %q, want %q", i, got[i].Type.ValueString(), want)
				}
			}
		})
	}
}

// Reordering must carry each entry's attributes with it, not just its type.
func TestOrderTransformsLikePriorKeepsAttributes(t *testing.T) {
	apiOrder := []DataStreamTransformModel{
		{
			Type:               types.StringValue("sampling"),
			SamplingAttributes: &DataStreamSamplingAttributesModel{Rate: types.Int64Value(50)},
		},
		{
			Type:                     types.StringValue("render_template"),
			RenderTemplateAttributes: &DataStreamRenderTemplateAttributesModel{Template: types.Int64Value(7)},
		},
	}
	prior := []DataStreamTransformModel{
		{Type: types.StringValue("render_template")},
		{Type: types.StringValue("sampling")},
	}

	got := orderTransformsLikePrior(prior, apiOrder)

	if got[0].RenderTemplateAttributes == nil || got[0].RenderTemplateAttributes.Template.ValueInt64() != 7 {
		t.Errorf("render_template attributes lost: %+v", got[0])
	}
	if got[1].SamplingAttributes == nil || got[1].SamplingAttributes.Rate.ValueInt64() != 50 {
		t.Errorf("sampling attributes lost: %+v", got[1])
	}
}

const dataStreamResponse = `{
  "id": 42,
  "name": "stream",
  "last_editor": "user@example.com",
  "last_modified": "2019-08-24T14:15:22Z",
  "created": "2019-08-24T14:15:22Z",
  "active": true,
  "product_version": "1.0",
  "inputs": [{"type": "raw_logs", "attributes": {"data_source": "workloads"}}],
  "transform": [
    {"type": "sampling", "attributes": {"rate": 50}},
    {"type": "render_template", "attributes": {"template": 2}}
  ],
  "outputs": [{"type": "s3", "attributes": {
    "access_key": "masked", "secret_key": "masked", "region": "us-east-1",
    "bucket_name": "bucket", "content_type": "plain/text",
    "host_url": "https://s3.example.com", "object_key_prefix": "prefix"
  }}]
}`

// Reading a stream back has to survive the response nesting each output's
// discriminator inside the variant (`<Endpoint>Attributes`) rather than on a
// shared wrapper.
func TestPopulateDataStreamFromResponse(t *testing.T) {
	var stream azionapi.DataStream
	if err := json.Unmarshal([]byte(dataStreamResponse), &stream); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	model := &dataStreamResourceResults{}
	populateDataStreamFromResponse(model, stream)

	if model.ID.ValueInt64() != 42 {
		t.Errorf("id: got %d, want 42", model.ID.ValueInt64())
	}
	if model.Name.ValueString() != "stream" {
		t.Errorf("name: got %q", model.Name.ValueString())
	}
	if !model.Active.ValueBool() {
		t.Error("active: got false, want true")
	}

	if len(model.Inputs) != 1 || model.Inputs[0].Attributes.DataSource.ValueString() != "workloads" {
		t.Errorf("inputs: %+v", model.Inputs)
	}

	if len(model.Transform) != 2 {
		t.Fatalf("transform: got %d entries, want 2", len(model.Transform))
	}
	if model.Transform[0].SamplingAttributes.Rate.ValueInt64() != 50 {
		t.Errorf("sampling rate: %+v", model.Transform[0].SamplingAttributes)
	}
	if model.Transform[1].RenderTemplateAttributes.Template.ValueInt64() != 2 {
		t.Errorf("render_template: %+v", model.Transform[1].RenderTemplateAttributes)
	}

	if len(model.Outputs) != 1 {
		t.Fatalf("outputs: got %d entries, want 1", len(model.Outputs))
	}
	output := model.Outputs[0]
	if output.Type.ValueString() != "s3" {
		t.Errorf("output type: got %q, want s3", output.Type.ValueString())
	}
	if output.S3Attributes == nil {
		t.Fatal("s3_attributes not populated")
	}
	if output.S3Attributes.BucketName.ValueString() != "bucket" {
		t.Errorf("bucket_name: %q", output.S3Attributes.BucketName.ValueString())
	}
	if output.S3Attributes.ObjectKeyPrefix.ValueString() != "prefix" {
		t.Errorf("object_key_prefix: %q", output.S3Attributes.ObjectKeyPrefix.ValueString())
	}
}

// Credentials come back masked. The configured value has to win, or every plan
// after the first shows a diff on secrets that never actually changed.
func TestPopulateDataStreamFromResponseKeepsConfiguredSecrets(t *testing.T) {
	var stream azionapi.DataStream
	if err := json.Unmarshal([]byte(dataStreamResponse), &stream); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	model := &dataStreamResourceResults{
		Outputs: []DataStreamOutputModel{{
			Type: types.StringValue("s3"),
			S3Attributes: &DataStreamS3OutputModel{
				AccessKey: types.StringValue("AKIAIOSFODNN7EXAMPLE"),
				SecretKey: types.StringValue("real-secret"),
			},
		}},
	}
	populateDataStreamFromResponse(model, stream)

	attrs := model.Outputs[0].S3Attributes
	if attrs.AccessKey.ValueString() != "AKIAIOSFODNN7EXAMPLE" {
		t.Errorf("access_key: got %q, want the configured value", attrs.AccessKey.ValueString())
	}
	if attrs.SecretKey.ValueString() != "real-secret" {
		t.Errorf("secret_key: got %q, want the configured value", attrs.SecretKey.ValueString())
	}

	// With no prior state, as on import, the masked value is all there is.
	imported := &dataStreamResourceResults{}
	populateDataStreamFromResponse(imported, stream)
	if imported.Outputs[0].S3Attributes.SecretKey.ValueString() != "masked" {
		t.Errorf("import: got %q, want the API value",
			imported.Outputs[0].S3Attributes.SecretKey.ValueString())
	}
}

// Every endpoint type the SDK can actually decode has to round-trip. The three
// `{url, api_key}` types — elasticsearch, splunk, datadog — are deliberately
// absent: oneOf(Output) has no discriminator, so a payload of any of the three
// matches all three and the SDK refuses to decode it. Add them here once the
// spec grows a discriminator.
func TestPopulateDataStreamOutputsPerEndpointType(t *testing.T) {
	tests := []struct {
		outputType string
		attributes string
		check      func(*testing.T, DataStreamOutputModel)
	}{
		{
			outputType: "standard",
			attributes: `{"url":"https://a","log_line_separator":"\\n","payload_format":"$dataset","max_size":1000000,"headers":{"k":"v"}}`,
			check: func(t *testing.T, o DataStreamOutputModel) {
				if o.StandardAttributes.URL.ValueString() != "https://a" {
					t.Errorf("url: %q", o.StandardAttributes.URL.ValueString())
				}
				if o.StandardAttributes.LogLineSeparator.ValueString() != `\n` {
					t.Errorf("log_line_separator: %q, want the two-character escape",
						o.StandardAttributes.LogLineSeparator.ValueString())
				}
				if o.StandardAttributes.MaxSize.ValueInt64() != 1000000 {
					t.Errorf("max_size: %d", o.StandardAttributes.MaxSize.ValueInt64())
				}
			},
		},
		{
			outputType: "kafka",
			attributes: `{"bootstrap_servers":"k:9092","kafka_topic":"t","use_tls":true}`,
			check: func(t *testing.T, o DataStreamOutputModel) {
				if !o.KafkaAttributes.UseTLS.ValueBool() {
					t.Error("use_tls: got false")
				}
			},
		},
		{
			outputType: "s3",
			attributes: `{"access_key":"a","secret_key":"s","region":"r","bucket_name":"b","content_type":"plain/text","host_url":"h","object_key_prefix":null}`,
			check: func(t *testing.T, o DataStreamOutputModel) {
				if !o.S3Attributes.ObjectKeyPrefix.IsNull() {
					t.Errorf("object_key_prefix: got %q, want null",
						o.S3Attributes.ObjectKeyPrefix.ValueString())
				}
			},
		},
		{
			outputType: "big_query",
			attributes: `{"dataset_id":"d","project_id":"p","table_id":"t","service_account_key":"k"}`,
			check: func(t *testing.T, o DataStreamOutputModel) {
				if o.BigQueryAttributes.TableID.ValueString() != "t" {
					t.Errorf("table_id: %q", o.BigQueryAttributes.TableID.ValueString())
				}
			},
		},
		{
			outputType: "aws_kinesis_firehose",
			attributes: `{"access_key":"a","stream_name":"s","region":"r","secret_key":"k"}`,
			check: func(t *testing.T, o DataStreamOutputModel) {
				if o.AWSKinesisFirehoseAttrs.StreamName.ValueString() != "s" {
					t.Errorf("stream_name: %q", o.AWSKinesisFirehoseAttrs.StreamName.ValueString())
				}
			},
		},
		{
			outputType: "qradar",
			attributes: `{"url":"https://q"}`,
			check: func(t *testing.T, o DataStreamOutputModel) {
				if o.QRadarAttributes.URL.ValueString() != "https://q" {
					t.Errorf("url: %q", o.QRadarAttributes.URL.ValueString())
				}
			},
		},
		{
			outputType: "azure_monitor",
			attributes: `{"log_type":"l","shared_key":"s","time_generated_field":"tgf","workspace_id":"w"}`,
			check: func(t *testing.T, o DataStreamOutputModel) {
				if o.AzureMonitorAttributes.TimeGeneratedField.ValueString() != "tgf" {
					t.Errorf("time_generated_field: %q",
						o.AzureMonitorAttributes.TimeGeneratedField.ValueString())
				}
			},
		},
		{
			outputType: "azure_blob_storage",
			attributes: `{"storage_account":"s","container_name":"c","blob_sas_token":"b"}`,
			check: func(t *testing.T, o DataStreamOutputModel) {
				if o.AzureBlobStorageAttributes.ContainerName.ValueString() != "c" {
					t.Errorf("container_name: %q", o.AzureBlobStorageAttributes.ContainerName.ValueString())
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.outputType, func(t *testing.T) {
			body := `{"id":1,"name":"n","last_editor":"e","active":true,"product_version":"1.0",
				"created":"2019-08-24T14:15:22Z","last_modified":"2019-08-24T14:15:22Z",
				"inputs":[{"type":"raw_logs","attributes":{"data_source":"workloads"}}],
				"transform":[{"type":"sampling","attributes":{"rate":100}}],
				"outputs":[{"type":"` + test.outputType + `","attributes":` + test.attributes + `}]}`

			var stream azionapi.DataStream
			if err := json.Unmarshal([]byte(body), &stream); err != nil {
				t.Fatalf("decoding %s: %v", test.outputType, err)
			}

			model := &dataStreamResourceResults{}
			populateDataStreamFromResponse(model, stream)

			if len(model.Outputs) != 1 {
				t.Fatalf("got %d outputs, want 1", len(model.Outputs))
			}
			if model.Outputs[0].Type.ValueString() != test.outputType {
				t.Errorf("type: got %q, want %q", model.Outputs[0].Type.ValueString(), test.outputType)
			}
			test.check(t, model.Outputs[0])
		})
	}
}

// dataStreamOutputEndpoint has to unwrap every variant of the response oneOf.
// A type missing from its switch would silently produce an output entry with no
// attributes at all.
func TestDataStreamOutputEndpointCoversEveryVariant(t *testing.T) {
	variants := map[string]azionapi.Output{
		"standard": {HttpPostEndpointAttributes: &azionapi.HttpPostEndpointAttributes{
			Type: "standard", Attributes: azionapi.HttpPostEndpoint{}}},
		"kafka": {KafkaEndpointAttributes: &azionapi.KafkaEndpointAttributes{
			Type: "kafka", Attributes: azionapi.KafkaEndpoint{}}},
		"s3": {S3EndpointAttributes: &azionapi.S3EndpointAttributes{
			Type: "s3", Attributes: azionapi.S3Endpoint{}}},
		"big_query": {BigQueryEndpointAttributes: &azionapi.BigQueryEndpointAttributes{
			Type: "big_query", Attributes: azionapi.BigQueryEndpoint{}}},
		"elasticsearch": {ElasticsearchEndpointAttributes: &azionapi.ElasticsearchEndpointAttributes{
			Type: "elasticsearch", Attributes: azionapi.ElasticsearchEndpoint{}}},
		"splunk": {SplunkEndpointAttributes: &azionapi.SplunkEndpointAttributes{
			Type: "splunk", Attributes: azionapi.SplunkEndpoint{}}},
		"datadog": {DatadogEndpointAttributes: &azionapi.DatadogEndpointAttributes{
			Type: "datadog", Attributes: azionapi.DatadogEndpoint{}}},
		"aws_kinesis_firehose": {AWSKinesisFirehoseEndpointAttributes: &azionapi.AWSKinesisFirehoseEndpointAttributes{
			Type: "aws_kinesis_firehose", Attributes: azionapi.AWSKinesisFirehoseEndpoint{}}},
		"qradar": {QRadarEndpointAttributes: &azionapi.QRadarEndpointAttributes{
			Type: "qradar", Attributes: azionapi.QRadarEndpoint{}}},
		"azure_monitor": {AzureMonitorEndpointAttributes: &azionapi.AzureMonitorEndpointAttributes{
			Type: "azure_monitor", Attributes: azionapi.AzureMonitorEndpoint{}}},
		"azure_blob_storage": {AzureBlobStorageEndpointAttributes: &azionapi.AzureBlobStorageEndpointAttributes{
			Type: "azure_blob_storage", Attributes: azionapi.AzureBlobStorageEndpoint{}}},
	}

	for want, output := range variants {
		got, endpoint := dataStreamOutputEndpoint(output)
		if got != want {
			t.Errorf("%s: got type %q", want, got)
		}
		if endpoint == nil {
			t.Errorf("%s: endpoint not unwrapped", want)
		}
	}
}

func TestBuildDataStreamTransformRequestErrors(t *testing.T) {
	tests := []struct {
		name      string
		transform DataStreamTransformModel
	}{
		{
			name:      "sampling without its attributes",
			transform: DataStreamTransformModel{Type: types.StringValue("sampling")},
		},
		{
			name:      "filter_workloads without its attributes",
			transform: DataStreamTransformModel{Type: types.StringValue("filter_workloads")},
		},
		{
			name:      "render_template without its attributes",
			transform: DataStreamTransformModel{Type: types.StringValue("render_template")},
		},
		{
			name:      "unsupported transform type",
			transform: DataStreamTransformModel{Type: types.StringValue("reticulate_splines")},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := buildDataStreamTransformRequests([]DataStreamTransformModel{test.transform})
			if err == nil {
				t.Error("expected an error")
			}
		})
	}
}

func TestBuildDataStreamInputRequestRequiresAttributes(t *testing.T) {
	_, err := buildDataStreamInputRequests([]DataStreamInputModel{{Type: types.StringValue("raw_logs")}})
	if err == nil {
		t.Error("expected an error for an input with no attributes block")
	}
}

// The filter_workloads transform round-trips []int64 through a list of numbers.
func TestBuildDataStreamFilterWorkloadsTransform(t *testing.T) {
	transforms, err := buildDataStreamTransformRequests([]DataStreamTransformModel{{
		Type: types.StringValue("filter_workloads"),
		FilterWorkloadsAttributes: &DataStreamFilterWorkloadsAttributesModel{
			Workloads: []types.Int64{types.Int64Value(1234), types.Int64Value(5678)},
		},
	}})
	if err != nil {
		t.Fatalf("building transform: %v", err)
	}

	got, err := json.Marshal(transforms)
	if err != nil {
		t.Fatalf("marshalling transform: %v", err)
	}

	requireJSONEqual(t, got, `[{"type":"filter_workloads","attributes":{"workloads":[1234,5678]}}]`)
}

// path.Path helper for the nested output attributes, kept separate so the
// schema tests above stay readable.
func dataStreamOutputAttrPath(name string) path.Path {
	return path.Root("data_stream").AtName("outputs").AtListIndex(0).AtName(name)
}

// The optional endpoint fields the API defaults must stay Optional + Computed,
// or omitting one makes every plan show a diff against the API's default.
func TestDataStreamStandardOptionalFieldsAreComputed(t *testing.T) {
	ctx := context.Background()
	sr := dataStreamSchema(t)

	for _, name := range []string{"log_line_separator", "payload_format", "max_size"} {
		attribute, diags := sr.Schema.AttributeAtPath(ctx,
			dataStreamOutputAttrPath("standard_attributes").AtName(name))
		if diags.HasError() {
			t.Errorf("%s: %v", name, diags)
			continue
		}

		if !attribute.IsOptional() || !attribute.IsComputed() {
			t.Errorf("%s: expected Optional + Computed, got optional=%v computed=%v",
				name, attribute.IsOptional(), attribute.IsComputed())
		}
	}
}
