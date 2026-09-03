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
	expectedOrder := []string{"apple", "cherry", "banana"} // ascending order
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

func assertOrder(t *testing.T, items []TemplateItem, expected []string) {
	t.Helper()
	for i, item := range items {
		if item.Name != expected[i] {
			t.Errorf("expected %v, got %s at index %d", expected, item.Name, i)
			return
		}
	}
}

func TestSort_NameDesc(t *testing.T) {
	items := []TemplateItem{
		{Name: "banana"},
		{Name: "apple"},
		{Name: "cherry"},
	}
	idx := &Indexer{Cfg: Config{SortBy: "name", Order: "desc"}}
	idx.sort(&items)
	assertOrder(t, items, []string{"cherry", "banana", "apple"})
}

func TestSort_NaturalNameDesc(t *testing.T) {
	items := []TemplateItem{
		{Name: "item10"},
		{Name: "item2"},
		{Name: "item1"},
	}
	idx := &Indexer{Cfg: Config{SortBy: "natural_name", Order: "desc"}}
	idx.sort(&items)
	assertOrder(t, items, []string{"item10", "item2", "item1"})
}

func TestSort_LastModifiedDesc(t *testing.T) {
	items := []TemplateItem{
		{Name: "banana", LastModified: "2020-01-03"},
		{Name: "apple", LastModified: "2020-01-01"},
		{Name: "cherry", LastModified: "2020-01-02"},
	}
	idx := &Indexer{Cfg: Config{SortBy: "last_modified", Order: "desc"}}
	idx.sort(&items)
	assertOrder(t, items, []string{"banana", "cherry", "apple"})
}

func TestSort_LastModifiedAsc(t *testing.T) {
	items := []TemplateItem{
		{Name: "banana", LastModified: "2020-01-03"},
		{Name: "apple", LastModified: "2020-01-01"},
		{Name: "cherry", LastModified: "2020-01-02"},
	}
	idx := &Indexer{Cfg: Config{SortBy: "last_modified", Order: "asc"}}
	idx.sort(&items)
	assertOrder(t, items, []string{"apple", "cherry", "banana"})
}

func TestSort_DirsFirstWithLastModifiedDesc(t *testing.T) {
	items := []TemplateItem{
		{Name: "file-b.txt", LastModified: "2020-01-03", IsDir: false},
		{Name: "dir-a", LastModified: "2020-01-01", IsDir: true},
		{Name: "file-a.txt", LastModified: "2020-01-02", IsDir: false},
		{Name: "dir-b", LastModified: "2020-01-04", IsDir: true},
	}
	idx := &Indexer{Cfg: Config{SortBy: "last_modified", Order: "desc", DirsFirst: true}}
	idx.sort(&items)

	// Dirs must come first, each group ordered newest-first.
	assertOrder(t, items, []string{"dir-b", "dir-a", "file-b.txt", "file-a.txt"})
	expectedIsDir := []bool{true, true, false, false}
	for i, item := range items {
		if item.IsDir != expectedIsDir[i] {
			t.Errorf("expected IsDir=%t, got %t for %s", expectedIsDir[i], item.IsDir, item.Name)
		}
	}
}

func TestSort_DirsFirstWithNameAsc(t *testing.T) {
	items := []TemplateItem{
		{Name: "zeta.txt", IsDir: false},
		{Name: "alpha-dir", IsDir: true},
		{Name: "beta.txt", IsDir: false},
		{Name: "omega-dir", IsDir: true},
	}
	idx := &Indexer{Cfg: Config{SortBy: "name", Order: "asc", DirsFirst: true}}
	idx.sort(&items)

	assertOrder(t, items, []string{"alpha-dir", "omega-dir", "beta.txt", "zeta.txt"})
}
