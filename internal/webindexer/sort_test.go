package webindexer

import (
	"testing"
)

func TestOrderByName(t *testing.T) {
	items := []TemplateItem{
		{Name: "banana"},
		{Name: "apple"},
		{Name: "cherry"},
	}
	expectedOrder := []string{"apple", "banana", "cherry"}
	orderByName(&items)

	for i, item := range items {
		if item.Name != expectedOrder[i] {
			t.Errorf("expected %s, got %s", expectedOrder[i], item.Name)
		}
	}
}

func TestOrderByLastModified(t *testing.T) {
	items := []TemplateItem{
		{Name: "banana", LastModified: "2020-01-03"},
		{Name: "apple", LastModified: "2020-01-01"},
		{Name: "cherry", LastModified: "2020-01-02"},
	}
	expectedOrder := []string{"banana", "cherry", "apple"} // descending order
	orderByLastModified(&items)

	for i, item := range items {
		if item.Name != expectedOrder[i] {
			t.Errorf("expected %s, got %s", expectedOrder[i], item.Name)
		}
	}
}

func TestOrderByNaturalName(t *testing.T) {
	items := []TemplateItem{
		{Name: "item10"},
		{Name: "item2"},
		{Name: "item1"},
	}
	expectedOrder := []string{"item1", "item2", "item10"}
	orderByNaturalName(&items)

	for i, item := range items {
		if item.Name != expectedOrder[i] {
			t.Errorf("expected %s, got %s", expectedOrder[i], item.Name)
		}
	}
}

func TestOrderDirsFirst(t *testing.T) {
	items := []TemplateItem{
		{Name: "file.txt", IsDir: false},
		{Name: "folder", IsDir: true},
		{Name: "another_folder", IsDir: true},
	}
	expectedOrderIsDir := []bool{true, true, false}
	orderDirsFirst(&items)

	for i, item := range items {
		if item.IsDir != expectedOrderIsDir[i] {
			t.Errorf("expected %t, got %t for %s", expectedOrderIsDir[i], item.IsDir, item.Name)
		}
	}
}
