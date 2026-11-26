package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestWAFRuleSetsSchemaImplementation(t *testing.T) {
	ctx := context.Background()

	sr := &fwresource.SchemaResponse{}
	WafRuleSetsResource().Schema(ctx, fwresource.SchemaRequest{}, sr)
	if sr.Diagnostics.HasError() {
		t.Fatalf("building schema: %v", sr.Diagnostics)
	}

	if diags := sr.Schema.ValidateImplementation(ctx); diags.HasError() {
		t.Errorf("invalid schema implementation: %v", diags)
	}

	if sr.Schema.Version != 1 {
		t.Errorf("schema version = %d, want 1 (rule_sets map -> rule_set blocks)", sr.Schema.Version)
	}

	// Same contract as azion_waf_rule_set: optional fields are Computed, or
	// omitting one diffs forever.
	block, ok := sr.Schema.Blocks["rule_set"].(schema.SetNestedBlock)
	if !ok {
		t.Fatalf("rule_set must be a set nested block, got %T", sr.Schema.Blocks["rule_set"])
	}
	for _, name := range []string{"rule_id", "path", "operator", "active"} {
		attribute, ok := block.NestedObject.Attributes[name]
		if !ok {
			t.Errorf("%s: missing", name)
			continue
		}
		if !attribute.IsOptional() || !attribute.IsComputed() {
			t.Errorf("%s: must be Optional and Computed", name)
		}
	}
}

func genericCondition(match string) WafExceptionConditionWrapperModel {
	return WafExceptionConditionWrapperModel{Condition: &WafExceptionConditionModel{
		Match:         types.StringValue(match),
		Name:          types.StringNull(),
		Value:         types.StringNull(),
		ConditionType: types.StringValue("generic"),
	}}
}

func wafItem(id int64, path string, active bool) WafRuleSetsItemModel {
	return WafRuleSetsItemModel{
		ExceptionID: types.Int64Value(id),
		RuleID:      types.Int64Value(0),
		Path:        types.StringValue(path),
		Conditions:  []WafExceptionConditionWrapperModel{genericCondition("any_url")},
		Operator:    types.StringValue("regex"),
		Active:      types.BoolValue(active),
	}
}

// plannedItem is what the plan holds for an exception that only declares its
// conditions: everything else the API supplies is unknown.
func plannedItem() WafRuleSetsItemModel {
	return WafRuleSetsItemModel{
		ExceptionID: types.Int64Unknown(),
		RuleID:      types.Int64Unknown(),
		Path:        types.StringUnknown(),
		Conditions:  []WafExceptionConditionWrapperModel{genericCondition("any_url")},
		Operator:    types.StringUnknown(),
		Active:      types.BoolValue(true),
	}
}

func TestWAFRuleSetsItemNeedsUpdate(t *testing.T) {
	current := wafItem(1, "/api", true)

	cases := []struct {
		name    string
		planned func() WafRuleSetsItemModel
		want    bool
	}{
		{"identical", func() WafRuleSetsItemModel { return wafItem(1, "/api", true) }, false},
		{"undeclared fields are unknown", plannedItem, false},
		{"declared active matches", func() WafRuleSetsItemModel {
			p := plannedItem()
			return withActive(p, true)
		}, false},
		{"declared path differs", func() WafRuleSetsItemModel {
			p := plannedItem()
			p.Path = types.StringValue("/other")
			return p
		}, true},
		{"condition match differs", func() WafRuleSetsItemModel {
			p := plannedItem()
			p.Conditions = []WafExceptionConditionWrapperModel{genericCondition("raw_body")}
			return p
		}, true},
		{"condition added", func() WafRuleSetsItemModel {
			p := plannedItem()
			p.Conditions = append(p.Conditions, genericCondition("raw_body"))
			return p
		}, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := wafRuleSetsItemNeedsUpdate(tc.planned(), current); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}

	// active defaults to true, so an exception disabled in Console is re-enabled.
	disabled := wafItem(1, "/api", false)
	if !wafRuleSetsItemNeedsUpdate(plannedItem(), disabled) {
		t.Error("an exception disabled outside Terraform must be re-enabled")
	}
}

func withActive(item WafRuleSetsItemModel, active bool) WafRuleSetsItemModel {
	item.Active = types.BoolValue(active)
	return item
}

func TestPlanWAFRuleSetsOps(t *testing.T) {
	current := map[string]WafRuleSetsItemModel{
		"kept":    wafItem(1, "/a", true),
		"changed": wafItem(2, "/b", true),
		"console": wafItem(3, "/c", true),
	}
	changed := plannedItem()
	changed.Path = types.StringValue("/new")
	planned := map[string]WafRuleSetsItemModel{
		"kept":    plannedItem(),
		"changed": changed,
		"new":     plannedItem(),
	}

	got := planWafRuleSetsOps(planned, current)
	want := wafRuleSetsOps{
		Deletes: []string{"console"},
		Updates: []string{"changed"},
		Creates: []string{"new"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func apiRule(id int64, name string) azionapi.WAFRule {
	rule := azionapi.WAFRule{Id: &id, Name: name}
	generic := azionapi.NewWAFExceptionGenericCondition("any_url")
	rule.Conditions = []azionapi.WAFExceptionCondition{azionapi.WAFExceptionGenericConditionAsWAFExceptionCondition(generic)}
	return rule
}

func TestBuildWAFRuleSetsStateKeysDuplicates(t *testing.T) {
	remote := []azionapi.WAFRule{apiRule(7, "dup"), apiRule(3, "dup"), apiRule(5, "solo")}

	t.Run("without prior state the lowest ID keeps the name", func(t *testing.T) {
		got, duplicates := buildWafRuleSetsState(remote, nil)
		assertKeyID(t, got, "dup", 3)
		assertKeyID(t, got, "dup#7", 7)
		assertKeyID(t, got, "solo", 5)
		if !reflect.DeepEqual(duplicates, []string{"dup#7"}) {
			t.Errorf("duplicates = %v", duplicates)
		}
	})

	t.Run("the managed exception keeps the name", func(t *testing.T) {
		prior := map[string]WafRuleSetsItemModel{"dup": wafItem(7, "/", true)}
		got, duplicates := buildWafRuleSetsState(remote, prior)
		assertKeyID(t, got, "dup", 7)
		assertKeyID(t, got, "dup#3", 3)
		if !reflect.DeepEqual(duplicates, []string{"dup#3"}) {
			t.Errorf("duplicates = %v", duplicates)
		}
	})

	t.Run("no exceptions is an empty map, not null", func(t *testing.T) {
		got, _ := buildWafRuleSetsState(nil, nil)
		if got == nil || len(got) != 0 {
			t.Errorf("got %v", got)
		}
	})
}

func assertKeyID(t *testing.T, m map[string]WafRuleSetsItemModel, key string, id int64) {
	t.Helper()
	item, ok := m[key]
	if !ok {
		t.Errorf("missing key %q in %v", key, keysOf(m))
		return
	}
	if item.ExceptionID.ValueInt64() != id {
		t.Errorf("%q: exception_id = %d, want %d", key, item.ExceptionID.ValueInt64(), id)
	}
}

func keysOf(m map[string]WafRuleSetsItemModel) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func TestIsWAFRuleSetsDuplicateKey(t *testing.T) {
	for key, want := range map[string]bool{
		"dup#7":      true,
		"a#b#12":     true,
		"name":       false,
		"name#":      false,
		"#5":         false,
		"tag#prod":   false,
		"c#sharp#v2": false,
	} {
		if got := isWafRuleSetsDuplicateKey(key); got != want {
			t.Errorf("%q: got %v, want %v", key, got, want)
		}
	}
}

// PUT replaces the exception, so a field left to the API must be sent with its
// current value, not omitted.
func TestBuildWAFRuleSetsRequestKeepsUndeclaredFields(t *testing.T) {
	prior := wafItem(1, "/api", true)
	prior.RuleID = types.Int64Value(1010)

	request := buildWafRuleSetsRequest("x", plannedItem(), &prior)

	if request.GetPath() != "/api" || request.GetRuleId() != 1010 || request.GetOperator() != "regex" {
		t.Errorf("undeclared fields not carried over: path=%q rule_id=%d operator=%q",
			request.GetPath(), request.GetRuleId(), request.GetOperator())
	}

	created := buildWafRuleSetsRequest("x", plannedItem(), nil)
	if created.HasPath() || created.HasRuleId() || created.HasOperator() {
		t.Error("a create must leave undeclared fields to the API")
	}
}

// fakeWAFAPI is an in-memory WAF exceptions endpoint. It serves one exception
// per page so pagination is exercised.
type fakeWAFAPI struct {
	mu     sync.Mutex
	nextID int64
	rules  map[int64]map[string]any
	calls  []string
}

func newFakeWAFAPI(t *testing.T) (*fakeWAFAPI, *wafRuleSetsResource) {
	t.Helper()
	fake := &fakeWAFAPI{nextID: 100, rules: map[int64]map[string]any{}}

	server := httptest.NewServer(http.HandlerFunc(fake.serve))
	t.Cleanup(server.Close)

	cfg := azionapi.NewConfiguration()
	cfg.Servers[0].URL = server.URL
	return fake, &wafRuleSetsResource{client: &apiClient{api: azionapi.NewAPIClient(cfg)}}
}

func (f *fakeWAFAPI) add(name, path string) int64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextID++
	f.rules[f.nextID] = map[string]any{
		"id": f.nextID, "name": name, "path": path, "rule_id": 0, "operator": "regex", "active": true,
		"conditions":  []any{map[string]any{"match": "any_url"}},
		"last_editor": "console", "last_modified": "2026-01-01T00:00:00Z",
	}
	return f.nextID
}

func (f *fakeWAFAPI) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/") // workspace/wafs/{id}/exceptions[/{id}]
	if len(parts) < 4 || parts[3] != "exceptions" {
		http.NotFound(w, r)
		return
	}
	f.calls = append(f.calls, r.Method)
	w.Header().Set("Content-Type", "application/json")

	if len(parts) == 4 && r.Method == http.MethodGet {
		ids := make([]int64, 0, len(f.rules))
		for id := range f.rules {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		results := []any{}
		if page >= 1 && page <= len(ids) {
			results = append(results, f.rules[ids[page-1]])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"count": len(ids), "total_pages": len(ids), "page": page, "results": results})
		return
	}

	if len(parts) == 4 && r.Method == http.MethodPost {
		f.nextID++
		f.store(f.nextID, r)
		_ = json.NewEncoder(w).Encode(map[string]any{"data": f.rules[f.nextID]})
		return
	}

	id, _ := strconv.ParseInt(parts[4], 10, 64)
	if _, ok := f.rules[id]; !ok {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errors":[{"detail":"not found"}]}`))
		return
	}
	switch r.Method {
	case http.MethodPut:
		f.store(id, r)
		_ = json.NewEncoder(w).Encode(map[string]any{"data": f.rules[id]})
	case http.MethodDelete:
		delete(f.rules, id)
		_, _ = w.Write([]byte(`{"state":"executed"}`))
	}
}

func (f *fakeWAFAPI) store(id int64, r *http.Request) {
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)
	body["id"] = id
	body["last_editor"] = "terraform"
	body["last_modified"] = "2026-02-02T00:00:00Z"
	f.rules[id] = body
}

func (f *fakeWAFAPI) snapshot() map[string]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[string]string{}
	for _, rule := range f.rules {
		out[fmt.Sprint(rule["name"])] = fmt.Sprint(rule["path"])
	}
	return out
}

// The FR-993 scenario: one managed exception is changed in Console and another
// is created there. A refresh must surface both, and apply must restore the
// declared set.
func TestWAFRuleSetsRollsBackConsoleChanges(t *testing.T) {
	ctx := context.Background()
	fake, r := newFakeWAFAPI(t)
	const wafID = 42

	managedID := fake.add("managed", "/api")
	prior := map[string]WafRuleSetsItemModel{"managed": wafItem(managedID, "/api", true)}

	// Changes made in Console.
	fake.mu.Lock()
	fake.rules[managedID]["path"] = "/changed-in-console"
	fake.mu.Unlock()
	fake.add("console-only", "/anything")
	fake.add("managed", "/duplicate")

	// Refresh.
	remote, _, err := r.listAllExceptions(ctx, wafID)
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if len(remote) != 3 {
		t.Fatalf("listed %d exceptions across pages, want 3", len(remote))
	}
	refreshed, _ := buildWafRuleSetsState(remote, prior)

	// Configuration: only "managed", with its original path.
	planned := plannedItem()
	planned.Path = types.StringValue("/api")
	config := map[string]WafRuleSetsItemModel{"managed": planned}

	ops := planWafRuleSetsOps(config, refreshed)
	if len(ops.Deletes) != 2 || len(ops.Updates) != 1 || len(ops.Creates) != 0 {
		t.Fatalf("plan = %+v, want 2 deletes and 1 update", ops)
	}

	result, diags := r.reconcile(ctx, wafID, config, refreshed)
	if diags.HasError() {
		t.Fatalf("reconcile: %v", diags)
	}

	if got, want := fake.snapshot(), map[string]string{"managed": "/api"}; !reflect.DeepEqual(got, want) {
		t.Errorf("API holds %v, want %v", got, want)
	}
	assertKeyID(t, result, "managed", managedID)

	// A second plan against the converged API is empty.
	remote, _, err = r.listAllExceptions(ctx, wafID)
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	again, _ := buildWafRuleSetsState(remote, result)
	if ops := planWafRuleSetsOps(config, again); len(ops.Deletes)+len(ops.Updates)+len(ops.Creates) != 0 {
		t.Errorf("second plan not empty: %+v", ops)
	}
}

// An exception deleted outside Terraform after the last refresh is recreated
// rather than failing the apply.
func TestWAFRuleSetsUpdateRecreatesMissingException(t *testing.T) {
	ctx := context.Background()
	fake, r := newFakeWAFAPI(t)

	current := map[string]WafRuleSetsItemModel{"gone": wafItem(999, "/old", true)}
	planned := plannedItem()
	planned.Path = types.StringValue("/new")

	result, diags := r.reconcile(ctx, 42, map[string]WafRuleSetsItemModel{"gone": planned}, current)
	if diags.HasError() {
		t.Fatalf("reconcile: %v", diags)
	}
	if got := fake.snapshot(); got["gone"] != "/new" {
		t.Errorf("API holds %v", got)
	}
	if result["gone"].ExceptionID.ValueInt64() == 999 {
		t.Error("state must hold the recreated exception's ID")
	}
}

func TestWAFRuleSetsListNotFound(t *testing.T) {
	_, r := newFakeWAFAPI(t)
	r.client.api.GetConfig().Servers[0].URL += "/missing"

	_, status, err := r.listAllExceptions(context.Background(), 42)
	if err == nil || status != http.StatusNotFound {
		t.Errorf("status = %d, err = %v; want 404", status, err)
	}
}

func wafRuleSetsPlanAndState(t *testing.T, plan, state WafRuleSetsResourceModel) (fwresource.ModifyPlanRequest, *fwresource.ModifyPlanResponse) {
	t.Helper()
	ctx := context.Background()

	sr := &fwresource.SchemaResponse{}
	WafRuleSetsResource().Schema(ctx, fwresource.SchemaRequest{}, sr)
	s := sr.Schema

	tfPlan := tfsdk.Plan{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}
	tfState := tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}
	if diags := tfPlan.Set(ctx, &plan); diags.HasError() {
		t.Fatalf("setting plan: %v", diags)
	}
	if diags := tfState.Set(ctx, &state); diags.HasError() {
		t.Fatalf("setting state: %v", diags)
	}

	// The plan's declared values stand in for the configuration.
	config := tfsdk.Config{Schema: s, Raw: tfPlan.Raw}
	req := fwresource.ModifyPlanRequest{Config: config, Plan: tfPlan, State: tfState}
	return req, &fwresource.ModifyPlanResponse{Plan: tfPlan}
}

func wafRuleSetsModel(items map[string]WafRuleSetsItemModel) WafRuleSetsResourceModel {
	return WafRuleSetsResourceModel{
		ID:       types.StringValue("42"),
		WafID:    types.Int64Value(42),
		RuleSets: wafRuleSetsList(items),
	}
}

// On any change the framework marks every undeclared computed attribute
// unknown, in every exception. ModifyPlan must narrow that back to the
// exceptions that actually change.
func TestWAFRuleSetsModifyPlanOnlyShowsChangedExceptions(t *testing.T) {
	ctx := context.Background()

	state := wafRuleSetsModel(map[string]WafRuleSetsItemModel{
		"changed":   wafItem(1, "/api", true),
		"untouched": wafItem(2, "/other", true),
	})
	changed := plannedItem()
	changed.Path = types.StringValue("/new")
	plan := wafRuleSetsModel(map[string]WafRuleSetsItemModel{
		"changed":   changed,
		"untouched": plannedItem(),
	})

	req, resp := wafRuleSetsPlanAndState(t, plan, state)
	WafRuleSetsResource().(*wafRuleSetsResource).ModifyPlan(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ModifyPlan: %v", resp.Diagnostics)
	}

	var got WafRuleSetsResourceModel
	if diags := resp.Plan.Get(ctx, &got); diags.HasError() {
		t.Fatalf("reading plan: %v", diags)
	}

	gotByName, stateByName := wafRuleSetsByName(got.RuleSets), wafRuleSetsByName(state.RuleSets)
	if !reflect.DeepEqual(gotByName["untouched"], stateByName["untouched"]) {
		t.Errorf("untouched exception should plan as its state, got %+v", gotByName["untouched"])
	}
	if gotByName["changed"].ExceptionID.ValueInt64() != 1 {
		t.Error("changed exception should keep its exception_id")
	}
	if !gotByName["changed"].RuleID.IsUnknown() {
		t.Error("changed exception's undeclared rule_id should be unknown")
	}
}

func TestWAFRuleSetsModifyPlanEmptyWhenNothingChanges(t *testing.T) {
	ctx := context.Background()

	state := wafRuleSetsModel(map[string]WafRuleSetsItemModel{
		"a": wafItem(1, "/api", true),
	})
	plan := wafRuleSetsModel(map[string]WafRuleSetsItemModel{
		"a": plannedItem(),
	})

	req, resp := wafRuleSetsPlanAndState(t, plan, state)
	WafRuleSetsResource().(*wafRuleSetsResource).ModifyPlan(ctx, req, resp)

	var got WafRuleSetsResourceModel
	if diags := resp.Plan.Get(ctx, &got); diags.HasError() {
		t.Fatalf("reading plan: %v", diags)
	}
	if !reflect.DeepEqual(got, state) {
		t.Errorf("plan should equal state, got %+v", got)
	}
}

// Zero rule_set blocks reach the provider as a null set, which the model reads
// as a nil slice. That must mean "no exceptions": apply deletes all of them.
func TestWAFRuleSetsNullRuleSetDeletesEverything(t *testing.T) {
	ctx := context.Background()
	fake, r := newFakeWAFAPI(t)
	a, b := fake.add("a", "/a"), fake.add("b", "/b")
	current := map[string]WafRuleSetsItemModel{"a": wafItem(a, "/a", true), "b": wafItem(b, "/b", true)}

	var plan WafRuleSetsResourceModel // RuleSets nil, as decoded from a null set
	result, diags := r.reconcile(ctx, 42, wafRuleSetsByName(plan.RuleSets), current)
	if diags.HasError() {
		t.Fatalf("reconcile: %v", diags)
	}
	if got := fake.snapshot(); len(got) != 0 || len(result) != 0 {
		t.Errorf("API holds %v, state %v; want both empty", got, result)
	}
}
