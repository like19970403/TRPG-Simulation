package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

type scenarioResp struct {
	ID                 string          `json:"id"`
	Status             string          `json:"status"`
	Title              string          `json:"title"`
	ValidationWarnings json.RawMessage `json:"validationWarnings"`
}

func registerAndLogin(t *testing.T, tsURL, prefix string) string {
	t.Helper()
	email := uniqueEmail(prefix)
	body, _ := json.Marshal(map[string]string{
		"username": prefix,
		"email":    email,
		"password": "TestPass123!",
	})
	http.Post(tsURL+"/api/v1/users", "application/json", bytes.NewReader(body))

	loginBody, _ := json.Marshal(map[string]string{
		"email":    email,
		"password": "TestPass123!",
	})
	res, _ := http.Post(tsURL+"/api/v1/auth/login", "application/json", bytes.NewReader(loginBody))
	var tok struct {
		AccessToken string `json:"accessToken"`
	}
	json.NewDecoder(res.Body).Decode(&tok)
	res.Body.Close()
	return tok.AccessToken
}

func authReq(method, url, token string, body []byte) (*http.Response, error) {
	req, _ := http.NewRequest(method, url, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	return http.DefaultClient.Do(req)
}

func TestScenarioCRUD(t *testing.T) {
	dbURL := testDBURL(t)
	pool := setupPool(t, dbURL)
	ts := setupServer(t, pool)

	token := registerAndLogin(t, ts.URL, "sc_crud")

	content := map[string]any{
		"title":       "Test",
		"start_scene": "s1",
		"scenes": []map[string]any{
			{"id": "s1", "name": "S1", "content": "Start", "transitions": []map[string]string{{"target": "s2", "trigger": "gm_decision"}}},
			{"id": "s2", "name": "S2", "content": "End"},
		},
		"items":     []any{},
		"npcs":      []any{},
		"variables": []any{},
	}
	contentJSON, _ := json.Marshal(content)

	// Create
	createBody, _ := json.Marshal(map[string]any{
		"title":       "Integ Scenario",
		"description": "Test",
		"content":     json.RawMessage(contentJSON),
	})
	res, err := authReq("POST", ts.URL+"/api/v1/scenarios", token, createBody)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d", res.StatusCode)
	}
	var created scenarioResp
	json.NewDecoder(res.Body).Decode(&created)
	res.Body.Close()

	if created.ID == "" || created.Status != "draft" {
		t.Fatalf("unexpected create result: %+v", created)
	}

	// Update
	updateBody, _ := json.Marshal(map[string]any{
		"title":       "Updated",
		"description": "Updated",
		"content":     json.RawMessage(contentJSON),
	})
	res2, _ := authReq("PUT", fmt.Sprintf("%s/api/v1/scenarios/%s", ts.URL, created.ID), token, updateBody)
	if res2.StatusCode != http.StatusOK {
		t.Fatalf("update: expected 200, got %d", res2.StatusCode)
	}
	res2.Body.Close()

	// Publish
	res3, _ := authReq("POST", fmt.Sprintf("%s/api/v1/scenarios/%s/publish", ts.URL, created.ID), token, nil)
	if res3.StatusCode != http.StatusOK {
		t.Fatalf("publish: expected 200, got %d", res3.StatusCode)
	}
	var published scenarioResp
	json.NewDecoder(res3.Body).Decode(&published)
	res3.Body.Close()
	if published.Status != "published" {
		t.Fatalf("publish: expected published, got %s", published.Status)
	}

	// Archive
	res4, _ := authReq("POST", fmt.Sprintf("%s/api/v1/scenarios/%s/archive", ts.URL, created.ID), token, nil)
	if res4.StatusCode != http.StatusOK {
		t.Fatalf("archive: expected 200, got %d", res4.StatusCode)
	}
	res4.Body.Close()
}
