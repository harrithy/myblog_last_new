package repository

import (
	"myblog_last_new/pkg/models"
	"testing"
)

func TestBuildCategoryTreeNestedChildren(t *testing.T) {
	rootID := 1
	childID := 2

	categories := []models.Category{
		{ID: rootID, Name: "root"},
		{ID: childID, Name: "child", ParentID: &rootID},
		{ID: 3, Name: "grandchild", ParentID: &childID},
	}

	tree := BuildCategoryTree(categories)
	if len(tree) != 1 {
		t.Fatalf("expected 1 root, got %d", len(tree))
	}

	root := tree[0]
	if len(root.Children) != 1 {
		t.Fatalf("expected root to have 1 child, got %d", len(root.Children))
	}

	child := root.Children[0]
	if len(child.Children) != 1 {
		t.Fatalf("expected child to have 1 grandchild, got %d", len(child.Children))
	}

	if child.Children[0].Name != "grandchild" {
		t.Fatalf("unexpected grandchild node: %#v", child.Children[0])
	}
}
