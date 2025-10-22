package opa

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Validator struct {
	server string
}

func NewValidator(server string) *Validator {
	return &Validator{server: server}
}

func (v *Validator) EvalTierPolicy(ctx context.Context, tier int, appName string, zones *[]string) (map[string]interface{}, error) {
	input := map[string]interface{}{
		"name": appName,
	}
	if zones != nil {
		input["zones"] = zones
	}
	return v.evalPolicy(ctx, tier, input)
}

func (v *Validator) evalPolicy(ctx context.Context, tier int, input interface{}) (map[string]interface{}, error) {
	requestBody := map[string]interface{}{
		"input": input,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/v1/data/tier%d", v.server, tier)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result["result"].(map[string]interface{}), nil
}

func (v *Validator) IsValid(result map[string]interface{}) bool {
	return result["valid"].(bool)
}

func (v *Validator) GetRequiredZones(result map[string]interface{}) []string {
	zonesList := []string{}
	for _, zone := range result["required_zones"].([]interface{}) {
		zonesList = append(zonesList, zone.(string))
	}
	return zonesList
}

func (v *Validator) GetFailures(result map[string]interface{}) []string {
	failures, ok := result["failures"]
	if !ok {
		return []string{}
	}

	// Handle both array and set cases (sets come back as arrays from OPA)
	failuresSlice, ok := failures.([]interface{})
	if !ok {
		// Handle the case where it might be a single value
		if failureStr, ok := failures.(string); ok {
			return []string{failureStr}
		}
		return []string{}
	}

	failuresList := make([]string, 0, len(failuresSlice))
	for _, failure := range failuresSlice {
		if failureStr, ok := failure.(string); ok {
			failuresList = append(failuresList, failureStr)
		}
	}
	return failuresList
}
