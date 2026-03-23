package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"myblog_last_new/internal/response"
	"myblog_last_new/internal/router"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBlogsReadFromArticleCategories(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	clearContentTables(t, db)

	folderResult, err := db.Exec(`
		INSERT INTO categories (name, type, sort_order)
		VALUES (?, 'folder', 1)
	`, "Go")
	if err != nil {
		t.Fatalf("failed to create parent category: %v", err)
	}

	parentID64, err := folderResult.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read parent category id: %v", err)
	}
	parentID := int(parentID64)

	articleResult, err := db.Exec(`
		INSERT INTO categories (name, type, description, url, parent_id, sort_order)
		VALUES (?, 'article', ?, ?, ?, 1)
	`, "How Go Skills Help", "A short article", "/posts/go-skills", parentID)
	if err != nil {
		t.Fatalf("failed to create article category: %v", err)
	}

	articleID64, err := articleResult.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read article id: %v", err)
	}
	articleID := int(articleID64)

	// Keep the legacy blogs table empty to prove the API now reads from categories.
	if _, err := db.Exec("DELETE FROM blogs"); err != nil {
		t.Fatalf("failed to clear blogs table: %v", err)
	}

	mux := http.NewServeMux()
	router.RegisterRoutes(mux, db)

	req, _ := http.NewRequest(http.MethodGet, "/blogs?category_id="+itoa(parentID), nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var listResp response.APIResponse
	if err := json.NewDecoder(rr.Body).Decode(&listResp); err != nil {
		t.Fatalf("failed to decode /blogs response: %v", err)
	}

	items, ok := listResp.Data.([]interface{})
	if !ok || len(items) != 1 {
		t.Fatalf("expected 1 blog item, got %#v", listResp.Data)
	}

	item, ok := items[0].(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected list item shape: %#v", items[0])
	}

	if got := int(item["id"].(float64)); got != articleID {
		t.Fatalf("expected article id %d, got %d", articleID, got)
	}
	if got := item["title"].(string); got != "How Go Skills Help" {
		t.Fatalf("unexpected title %q", got)
	}
	if got := int(item["category_id"].(float64)); got != parentID {
		t.Fatalf("expected category_id %d, got %d", parentID, got)
	}
	if got := item["category_name"].(string); got != "Go" {
		t.Fatalf("expected category_name %q, got %q", "Go", got)
	}

	detailReq, _ := http.NewRequest(http.MethodGet, "/blogs/"+itoa(articleID), nil)
	detailRR := httptest.NewRecorder()
	mux.ServeHTTP(detailRR, detailReq)

	if detailRR.Code != http.StatusOK {
		t.Fatalf("expected detail status %d, got %d body=%s", http.StatusOK, detailRR.Code, detailRR.Body.String())
	}

	var detailResp response.APIResponse
	if err := json.NewDecoder(detailRR.Body).Decode(&detailResp); err != nil {
		t.Fatalf("failed to decode /blogs/{id} response: %v", err)
	}

	detail, ok := detailResp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected detail response shape: %#v", detailResp.Data)
	}

	if got := int(detail["id"].(float64)); got != articleID {
		t.Fatalf("expected detail id %d, got %d", articleID, got)
	}
	if got := detail["url"].(string); got != "/posts/go-skills" {
		t.Fatalf("unexpected detail url %q", got)
	}
}

func clearContentTables(t *testing.T, db *sql.DB) {
	t.Helper()

	queries := []string{
		"DELETE FROM comments",
		"DELETE FROM blogs",
		"DELETE FROM categories",
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			t.Fatalf("failed to execute cleanup query %q: %v", query, err)
		}
	}
}

func itoa(v int) string {
	return fmt.Sprintf("%d", v)
}
