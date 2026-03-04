package integration_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

func TestAuthFlow(t *testing.T) {
	dbURL := testDBURL(t)
	pool := setupPool(t, dbURL)
	ts := setupServer(t, pool)

	email := uniqueEmail("auth")
	password := "IntegTestPass123!"

	// Register
	regBody, _ := json.Marshal(map[string]string{
		"username": "integ_auth",
		"email":    email,
		"password": password,
	})
	res, err := http.Post(ts.URL+"/api/v1/users", "application/json", bytes.NewReader(regBody))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d", res.StatusCode)
	}

	// Login
	loginBody, _ := json.Marshal(map[string]string{
		"email":    email,
		"password": password,
	})
	res2, err := http.Post(ts.URL+"/api/v1/auth/login", "application/json", bytes.NewReader(loginBody))
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	defer res2.Body.Close()
	if res2.StatusCode != http.StatusOK {
		t.Fatalf("login: expected 200, got %d", res2.StatusCode)
	}

	var tokenResp struct {
		AccessToken string `json:"accessToken"`
	}
	json.NewDecoder(res2.Body).Decode(&tokenResp)
	if tokenResp.AccessToken == "" {
		t.Fatal("login: empty access token")
	}

	// Authenticated request (health check)
	req, _ := http.NewRequest("GET", ts.URL+"/api/health", nil)
	req.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)
	res3, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("health: %v", err)
	}
	defer res3.Body.Close()
	if res3.StatusCode != http.StatusOK {
		t.Fatalf("health: expected 200, got %d", res3.StatusCode)
	}

	// Logout
	logoutReq, _ := http.NewRequest("POST", ts.URL+"/api/v1/auth/logout", nil)
	logoutReq.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)
	res4, err := http.DefaultClient.Do(logoutReq)
	if err != nil {
		t.Fatalf("logout: %v", err)
	}
	defer res4.Body.Close()
	if res4.StatusCode != http.StatusOK {
		t.Fatalf("logout: expected 200, got %d", res4.StatusCode)
	}
}
