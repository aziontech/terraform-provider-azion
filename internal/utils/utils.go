package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aziontech/azionapi-go-sdk/edgeapplications"
	"github.com/aziontech/azionapi-go-sdk/edgefunctionsinstance_edgefirewall"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func SliceStringTypeToList(slice []types.String) types.List {
	if len(slice) == 0 {
		return types.ListValueMust(types.StringType, nil)
	}
	strs := []attr.Value{}
	for _, value := range slice {
		strs = append(strs, value)
	}
	return types.ListValueMust(types.StringType, strs)
}

func SliceIntTypeToList(slice []types.Int64) types.List {
	if len(slice) == 0 {
		return types.ListValueMust(types.Int64Type, nil)
	}
	integers := []attr.Value{}
	for _, value := range slice {
		integers = append(integers, value)
	}
	return types.ListValueMust(types.Int64Type, integers)
}

func SliceStringTypeToSet(slice []types.String) types.Set {
	if len(slice) == 0 {
		return types.SetNull(types.StringType)
	}
	strings := []attr.Value{}
	for _, value := range slice {
		strings = append(strings, value)
	}
	return types.SetValueMust(types.StringType, strings)
}

func SliceStringTypeToSetOrNull(slice []types.String) types.Set {
	if len(slice) == 0 {
		return types.SetValueMust(types.StringType, nil)
	}
	strings := []attr.Value{}
	for _, value := range slice {
		strings = append(strings, value)
	}
	return types.SetValueMust(types.StringType, strings)
}

func ConvertStringToInterface(jsonArgs string) (interface{}, error) {
	var data map[string]interface{}
	err := json.Unmarshal([]byte(jsonArgs), &data)
	if err != nil {
		fmt.Println("Error:", err)
		return nil, err
	}
	return data, err
}

func UnmarshallJsonArgs(jsonArgs string) (edgeapplications.ApplicationCreateInstanceRequestArgs, error) {
	var data edgeapplications.ApplicationCreateInstanceRequestArgs
	args := make(map[string]interface{})
	err := json.Unmarshal([]byte(jsonArgs), &data)
	if err != nil {
		fmt.Println("Error:", err)
		data.MapmapOfStringAny = &args
		return data, nil
	}
	return data, nil
}

func UnmarshallJsonArgsFirewall(jsonArgs string) (edgefunctionsinstance_edgefirewall.EdgeFunctionsInstanceJsonArgs, error) {
	var data edgefunctionsinstance_edgefirewall.EdgeFunctionsInstanceJsonArgs
	args := make(map[string]interface{})
	err := json.Unmarshal([]byte(jsonArgs), &data)
	if err != nil {
		fmt.Println("Error:", err)
		data.MapmapOfStringAny = &args
		return data, nil
	}
	return data, nil
}

func ConvertInterfaceToString(jsonArgs interface{}) (string, error) {
	jsonArgsStr, err := json.Marshal(jsonArgs)
	if err != nil {
		fmt.Println("Error:", err)
		return "{}", nil
	}

	return string(jsonArgsStr), err
}

func ConvertInterfaceToFloat64List(listInt interface{}) []types.Float64 {
	iListInt := listInt.([]interface{})
	var integers []types.Float64
	for _, v := range iListInt {
		if _, ok := v.(float64); !ok {
			return nil
		}
		integers = append(integers, types.Float64Value(v.(float64)))
	}
	return integers
}

func ConvertFloat64ToInterface(sliceInt []types.Float64) (interface{}, error) {
	var integers []float64
	for _, v := range sliceInt {
		integers = append(integers, v.ValueFloat64())
	}
	return integers, nil
}

func AtoiNoError(strToConv string, resp *resource.ReadResponse) int32 {
	intReturn, err := strconv.ParseInt(strToConv, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error ",
			"Could not convert String to Int",
		)
		return 0
	}
	if intReturn > math.MaxInt32 || intReturn < math.MinInt32 {
		resp.Diagnostics.AddError(
			"Value Overflow error",
			"Converted value exceeds the range of int32",
		)
		return 0
	}
	return int32(intReturn)
}

// CheckInt64toInt32Security check parse direct int64 to int32.
func CheckInt64toInt32Security(n int64) (int32, error) {
	if n < math.MinInt32 || n > math.MaxInt32 {
		return 0, errors.New("Overflow")
	}
	return int32(n), nil
}

func ExceedsValidRange(resp any, vl any) {
	summary := "Error: value exceeds the valid range"
	detail := "n32 exceeds int32 limits"
	if vl != nil {
		detail = fmt.Sprintf("n32 %v exceeds int32 limits", vl)
	}
	switch v := resp.(type) {
	case *resource.DeleteResponse:
		v.Diagnostics.AddError(summary, detail)
		return
	case *resource.CreateResponse:
		v.Diagnostics.AddError(summary, detail)
		return
	case *resource.SchemaResponse:
		v.Diagnostics.AddError(summary, detail)
		return
	case *resource.UpdateResponse:
		v.Diagnostics.AddError(summary, detail)
		return
	case *resource.ConfigureResponse:
		v.Diagnostics.AddError(summary, detail)
		return
	case *resource.ModifyPlanResponse:
		v.Diagnostics.AddError(summary, detail)
		return
	case *resource.ImportStateResponse:
		v.Diagnostics.AddError(summary, detail)
		return
	case *resource.UpgradeStateResponse:
		v.Diagnostics.AddError(summary, detail)
		return
	}
}

func SleepAfter429(response *http.Response) error {
	timeToSleep := response.Header.Get("retry-after")
	num, err := strconv.Atoi(timeToSleep)
	if err != nil {
		return err
	}
	time.Sleep((time.Duration(num) + 1) * time.Second)
	return nil
}

// RetryOn429 retries an API call if the response status is 429.
func RetryOn429[T any](apiCall func() (T, *http.Response, error), maxRetries int) (T, *http.Response, error) {
	var result T
	var response *http.Response
	var err error

	for i := 0; i < maxRetries; i++ {
		// Call the API function
		result, response, err = apiCall()

		// If no error and not a 429, return successfully
		if err == nil && response.StatusCode != http.StatusTooManyRequests {
			return result, response, nil
		}

		// If error is not 429, return immediately
		if response.StatusCode != http.StatusTooManyRequests {
			return result, response, err
		}

		// Sleep before retrying
		if sleepErr := SleepAfter429(response); sleepErr != nil {
			return result, response, sleepErr
		}
	}

	return result, response, errors.New("max retries exceeded for API request")
}

// referencedByAnotherResourceMsg is the API error fragment returned when a
// resource cannot be deleted because another resource still references it.
const referencedByAnotherResourceMsg = "referenced by another resource"

// RetryOn429Delete retries a delete API call when the response is a 429, a 500,
// or a 400 whose error indicates the resource is still referenced by another
// resource (API eventual-consistency lag after the referencing resource is
// destroyed). It sleeps 10s before the first retry, increasing by 1s on each
// subsequent retry.
//
// The apiCall's first return value (the response body) may be nil for delete
// endpoints that do not return a body.
func RetryOn429Delete[T any](apiCall func() (T, *http.Response, error), maxRetries int) (T, *http.Response, error) {
	var result T
	var response *http.Response
	var err error

	for i := 0; i < maxRetries; i++ {
		// Call the API function.
		result, response, err = apiCall()

		// Success.
		if err == nil {
			return result, response, nil
		}

		// No response to inspect, return immediately.
		if response == nil {
			return result, response, err
		}

		// Only retry on a 429, a 500, or a 400 caused by the resource still being referenced.
		if response.StatusCode != http.StatusTooManyRequests &&
			response.StatusCode != http.StatusInternalServerError &&
			!(response.StatusCode == http.StatusBadRequest && isReferencedByAnotherResource(response, err)) {
			return result, response, err
		}

		// Sleep 10s before the first retry, increasing by 1s each subsequent retry.
		time.Sleep(time.Duration(10+i) * time.Second)
	}

	return result, response, errors.New("max retries exceeded for API request")
}

// isReferencedByAnotherResource reports whether a delete failure was caused by
// the resource still being referenced by another resource. It inspects both the
// error and the response body, restoring the body so the caller can still read it.
func isReferencedByAnotherResource(response *http.Response, err error) bool {
	if err != nil && strings.Contains(strings.ToLower(err.Error()), referencedByAnotherResourceMsg) {
		return true
	}
	if response == nil || response.Body == nil {
		return false
	}
	bodyBytes, readErr := io.ReadAll(response.Body)
	if readErr != nil {
		return false
	}
	// Restore the body so callers can still read it for error reporting.
	response.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	return strings.Contains(strings.ToLower(string(bodyBytes)), referencedByAnotherResourceMsg)
}
