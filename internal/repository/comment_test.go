package repository

import (
	"myblog_last_new/pkg/models"
	"testing"
)

func TestBuildCommentTreeNestedReplies(t *testing.T) {
	rootID := 1
	replyID := 2

	comments := []models.Comment{
		{ID: rootID, ArticleID: 100, Nickname: "root"},
		{ID: replyID, ArticleID: 100, Nickname: "reply", ParentID: &rootID},
		{ID: 3, ArticleID: 100, Nickname: "nested", ParentID: &replyID},
	}

	tree := buildCommentTree(comments, []int{rootID})
	if len(tree) != 1 {
		t.Fatalf("expected 1 root comment, got %d", len(tree))
	}
	if len(tree[0].Children) != 1 {
		t.Fatalf("expected 1 first-level reply, got %d", len(tree[0].Children))
	}
	if len(tree[0].Children[0].Children) != 1 {
		t.Fatalf("expected 1 nested reply, got %d", len(tree[0].Children[0].Children))
	}
	if tree[0].Children[0].Children[0].Nickname != "nested" {
		t.Fatalf("unexpected nested reply: %#v", tree[0].Children[0].Children[0])
	}
}
