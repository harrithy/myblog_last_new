package main

import (
	"bytes"
	"encoding/json"
	"myblog_last_new/internal/router"
	"myblog_last_new/pkg/models"
	"myblog_last_new/pkg/storage"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAddUser(t *testing.T) {
	db, err := storage.ConnectDB()
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Clear the users table before testing
	_, err = db.Exec("DELETE FROM users")
	if err != nil {
		t.Fatalf("Failed to clear users table: %v", err)
	}

	// Create a user for login
	_, err = db.Exec("INSERT INTO users(name, email, nickname, birthday) VALUES(?, ?, ?, ?)", "testuser", "test@example.com", "tester", "2000-01-01")
	if err != nil {
		t.Fatalf("Failed to create user for login: %v", err)
	}

	mux := http.NewServeMux()
	router.RegisterRoutes(mux, db)

	// Login to get a token
	loginCreds := models.User{
		Account: "test@example.com",
	}
	loginBody, _ := json.Marshal(loginCreds)
	loginReq, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")

	loginRR := httptest.NewRecorder()
	mux.ServeHTTP(loginRR, loginReq)

	if status := loginRR.Code; status != http.StatusOK {
		t.Errorf("login handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var tokenMap map[string]string
	if err := json.NewDecoder(loginRR.Body).Decode(&tokenMap); err != nil {
		t.Errorf("could not decode login response: %v", err)
	}
	token := tokenMap["token"]

	// Test case
	user := models.User{
		Name:     "newuser",
		Account:  "new@example.com",
		Nickname: "newbie",
		Birthday: "2001-01-01",
	}
	body, _ := json.Marshal(user)
	req, _ := http.NewRequest("POST", "/users", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusCreated)
	}

	var createdUser models.User
	if err := json.NewDecoder(rr.Body).Decode(&createdUser); err != nil {
		t.Errorf("could not decode response: %v", err)
	}

	if createdUser.Name != user.Name {
		t.Errorf("handler returned unexpected body: got name %v want %v",
			createdUser.Name, user.Name)
	}
}
