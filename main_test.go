package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"myblog_last_new/internal/repository"
	"myblog_last_new/internal/response"
	"myblog_last_new/internal/router"
	"myblog_last_new/internal/security"
	"myblog_last_new/pkg/models"
	"myblog_last_new/pkg/storage"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestAddUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("DELETE FROM users")
	if err != nil {
		t.Fatalf("failed to clear users table: %v", err)
	}

	mux := http.NewServeMux()
	router.RegisterRoutes(mux, db)

	token := loginAsOwner(t, mux)

	user := models.User{
		Name:     "New User",
		Email:    "new@example.com",
		Account:  "newuser",
		Password: "newpass123",
		Nickname: "newbie",
		Birthday: "2001-01-01",
	}

	body, _ := json.Marshal(user)
	req, _ := http.NewRequest(http.MethodPost, "/users", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Fatalf("handler returned wrong status code: got %v want %v body=%s", status, http.StatusCreated, rr.Body.String())
	}

	var createdResp response.APIResponse
	if err := json.NewDecoder(rr.Body).Decode(&createdResp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}

	data, ok := createdResp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected response data shape: %#v", createdResp.Data)
	}

	if gotName, _ := data["name"].(string); gotName != user.Name {
		t.Fatalf("handler returned unexpected body: got name %v want %v", gotName, user.Name)
	}
	if gotEmail, _ := data["email"].(string); gotEmail != user.Email {
		t.Fatalf("handler returned unexpected body: got email %v want %v", gotEmail, user.Email)
	}

	if password, exists := data["password"]; exists && password != "" {
		t.Fatalf("password should not be returned in response: %#v", password)
	}

	storedUser, err := repository.NewUserRepository(db).GetByLogin(user.Account)
	if err != nil {
		t.Fatalf("failed to query created user: %v", err)
	}

	if storedUser.Password == user.Password {
		t.Fatalf("password should be hashed before storage")
	}

	if !security.CheckPassword(storedUser.Password, user.Password) {
		t.Fatalf("stored password hash does not match plain-text password")
	}
	if storedUser.Email != user.Email {
		t.Fatalf("stored email mismatch: got %q want %q", storedUser.Email, user.Email)
	}
}

func TestRegisterUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("DELETE FROM users")
	if err != nil {
		t.Fatalf("failed to clear users table: %v", err)
	}

	mux := http.NewServeMux()
	router.RegisterRoutes(mux, db)

	reqBody, _ := json.Marshal(models.RegisterRequest{
		Email:    "reader@example.com",
		Account:  "reader",
		Name:     "Reader",
		Password: "reader-pass",
		Nickname: "bookworm",
	})
	req, _ := http.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("register returned wrong status: got %d want %d body=%s", rr.Code, http.StatusCreated, rr.Body.String())
	}

	var registerResp response.APIResponse
	if err := json.NewDecoder(rr.Body).Decode(&registerResp); err != nil {
		t.Fatalf("could not decode register response: %v", err)
	}

	data, ok := registerResp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected register response data shape: %#v", registerResp.Data)
	}

	if token, _ := data["token"].(string); token == "" {
		t.Fatalf("register response did not include a token: %#v", data)
	}

	userData, ok := data["user"].(map[string]interface{})
	if !ok {
		t.Fatalf("register response did not include a user: %#v", data)
	}
	if got, _ := userData["email"].(string); got != "reader@example.com" {
		t.Fatalf("unexpected registered email: got %q", got)
	}
	if got, _ := userData["account"].(string); got != "reader" {
		t.Fatalf("unexpected registered account: got %q", got)
	}

	storedUser, err := repository.NewUserRepository(db).GetByLogin("reader")
	if err != nil {
		t.Fatalf("failed to query registered user: %v", err)
	}
	if storedUser.Password == "reader-pass" {
		t.Fatalf("password should be hashed before storage")
	}
	if !security.CheckPassword(storedUser.Password, "reader-pass") {
		t.Fatalf("stored password hash does not match registered password")
	}
}

func TestLoginWithAccountOrEmail(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("DELETE FROM users")
	if err != nil {
		t.Fatalf("failed to clear users table: %v", err)
	}

	hashedPassword, err := security.HashPassword("reader-pass")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	if err := repository.NewUserRepository(db).Create(&models.User{
		Name:     "Reader",
		Email:    "reader@example.com",
		Account:  "reader",
		Nickname: "bookworm",
		Password: hashedPassword,
	}); err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}

	mux := http.NewServeMux()
	router.RegisterRoutes(mux, db)

	testCases := []struct {
		name string
		body models.AuthCredentials
	}{
		{
			name: "account",
			body: models.AuthCredentials{
				Account:  "reader",
				Password: "reader-pass",
			},
		},
		{
			name: "email",
			body: models.AuthCredentials{
				Email:    "reader@example.com",
				Password: "reader-pass",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.body)
			req, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("login returned wrong status: got %d want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
			}

			var loginResp response.APIResponse
			if err := json.NewDecoder(rr.Body).Decode(&loginResp); err != nil {
				t.Fatalf("could not decode login response: %v", err)
			}

			data, ok := loginResp.Data.(map[string]interface{})
			if !ok {
				t.Fatalf("unexpected login response data shape: %#v", loginResp.Data)
			}

			if token, _ := data["token"].(string); token == "" {
				t.Fatalf("login response did not include a token: %#v", data)
			}
		})
	}
}

func loginAsOwner(t *testing.T, mux *http.ServeMux) string {
	t.Helper()

	loginCreds := models.AuthCredentials{
		Account:  os.Getenv("OWNER_ACCOUNT"),
		Password: os.Getenv("OWNER_PASSWORD"),
	}
	loginBody, _ := json.Marshal(loginCreds)
	loginReq, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")

	loginRR := httptest.NewRecorder()
	mux.ServeHTTP(loginRR, loginReq)

	if status := loginRR.Code; status != http.StatusOK {
		t.Fatalf("login handler returned wrong status code: got %v want %v body=%s", status, http.StatusOK, loginRR.Body.String())
	}

	var loginResp response.APIResponse
	if err := json.NewDecoder(loginRR.Body).Decode(&loginResp); err != nil {
		t.Fatalf("could not decode login response: %v", err)
	}

	data, ok := loginResp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected login response data shape: %#v", loginResp.Data)
	}

	token, ok := data["token"].(string)
	if !ok || token == "" {
		t.Fatalf("login response did not include a token: %#v", data)
	}

	return token
}

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dbUser := firstNonEmpty(os.Getenv("TEST_DB_USER"), os.Getenv("DB_USER"), "root")
	dbPassword := firstNonEmpty(os.Getenv("TEST_DB_PASSWORD"), os.Getenv("DB_PASSWORD"), "525300")
	dbHost := firstNonEmpty(os.Getenv("TEST_DB_HOST"), os.Getenv("DB_HOST"), "localhost")
	dbPort := firstNonEmpty(os.Getenv("TEST_DB_PORT"), os.Getenv("DB_PORT"), "3306")
	dbName := firstNonEmpty(os.Getenv("TEST_DB_NAME"), "blog_test")

	adminDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/?charset=utf8mb4&parseTime=True&loc=Local", dbUser, dbPassword, dbHost, dbPort)
	adminDB, err := sql.Open("mysql", adminDSN)
	if err != nil {
		t.Skipf("skipping database-backed test: %v", err)
	}
	defer adminDB.Close()

	if err := adminDB.Ping(); err != nil {
		t.Skipf("skipping database-backed test: %v", err)
	}

	escapedDBName := strings.ReplaceAll(dbName, "`", "``")
	if _, err := adminDB.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", escapedDBName)); err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	t.Setenv("DB_USER", dbUser)
	t.Setenv("DB_PASSWORD", dbPassword)
	t.Setenv("DB_HOST", dbHost)
	t.Setenv("DB_PORT", dbPort)
	t.Setenv("DB_NAME", dbName)
	t.Setenv("JWT_SECRET", "test-jwt-secret")
	t.Setenv("OWNER_ACCOUNT", "harrio")
	t.Setenv("OWNER_PASSWORD", "owner-pass")
	t.Setenv("OWNER_NAME", "harrio")
	t.Setenv("OWNER_EMAIL", "harrithy@github.com")

	db, err := storage.ConnectDB()
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	if err := storage.InitDB(db); err != nil {
		db.Close()
		t.Fatalf("failed to initialize test database: %v", err)
	}

	return db
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
