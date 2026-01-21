package portfolio

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFilesystemService_GetCategories(t *testing.T) {
	// 1. Create temp dirs
	tmpDir := t.TempDir()

	landscapeDir := filepath.Join(tmpDir, "Landscape")
	peopleDir := filepath.Join(tmpDir, "People")

	if err := os.Mkdir(landscapeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(peopleDir, 0755); err != nil {
		t.Fatal(err)
	}

	// 2. Add dummy image files
	createFile := func(path string) {
		if err := os.WriteFile(path, []byte("dummy image content"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	createFile(filepath.Join(landscapeDir, "mountains.jpg"))
	createFile(filepath.Join(landscapeDir, "lake.png"))
	createFile(filepath.Join(landscapeDir, "z_last.jpg"))
	createFile(filepath.Join(landscapeDir, "a_first.jpg"))
	createFile(filepath.Join(peopleDir, "portrait.jpg"))

	// 3. Init service
	svc := NewFilesystemService(tmpDir, "")

	// 4. Assert correct category names and image paths
	cats, err := svc.GetCategories()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(cats) != 2 {
		t.Errorf("expected 2 categories, got %d", len(cats))
	}

	// Helper to find category
	findCat := func(name string) *Category {
		for _, c := range cats {
			if c.Name == name {
				// We return a pointer to the category in the slice
				// Note: if cats is reallocated this might be risky, but here it's fine for reading
				return &c
			}
		}
		return nil
	}

	// Check Landscape
	lCat := findCat("Landscape")
	if lCat == nil {
		t.Error("expected Landscape category, not found")
	} else {
		if len(lCat.Images) != 4 {
			t.Errorf("expected 4 images in Landscape, got %d", len(lCat.Images))
		}

		// Verify exact sort order: a_first, lake, mountains, z_last
		expectedOrder := []string{
			filepath.Join("Landscape", "a_first.jpg"),
			filepath.Join("Landscape", "lake.png"),
			filepath.Join("Landscape", "mountains.jpg"),
			filepath.Join("Landscape", "z_last.jpg"),
		}

		for i, img := range lCat.Images {
			if i >= len(expectedOrder) {
				break
			}
			if img != expectedOrder[i] {
				t.Errorf("Index %d: expected %s, got %s", i, expectedOrder[i], img)
			}
		}

		// Check CoverImage (should be the last alphabetical one: z_last.jpg)
		// Note: os.ReadDir returns sorted by name.
		// a_first.jpg, lake.png, mountains.jpg, z_last.jpg
		expectedCover := filepath.Join("Landscape", "z_last.jpg")
		if lCat.CoverImage != expectedCover {
			t.Errorf("expected cover image %s, got %s", expectedCover, lCat.CoverImage)
		}
	}

	// Check People
	pCat := findCat("People")
	if pCat == nil {
		t.Error("expected People category, not found")
	} else {
		if len(pCat.Images) != 1 {
			t.Errorf("expected 1 image in People, got %d", len(pCat.Images))
		}
		if len(pCat.Images) > 0 && pCat.Images[0] != filepath.Join("People", "portrait.jpg") {
			t.Errorf("expected People/portrait.jpg, got %s", pCat.Images[0])
		}
	}
}

func TestFilesystemService_GetCategory(t *testing.T) {
	tmpDir := t.TempDir()
	landscapeDir := filepath.Join(tmpDir, "Landscape")
	if err := os.Mkdir(landscapeDir, 0755); err != nil {
		t.Fatal(err)
	}

	createFile := func(path string) {
		if err := os.WriteFile(path, []byte("dummy image content"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	createFile(filepath.Join(landscapeDir, "mountains.jpg"))

	svc := NewFilesystemService(tmpDir, "")

	// Test existing category
	cat, err := svc.GetCategory("Landscape")
	if err != nil {
		t.Fatalf("expected no error for existing category, got %v", err)
	}
	if cat.Name != "Landscape" {
		t.Errorf("expected category name Landscape, got %s", cat.Name)
	}
	if len(cat.Images) != 1 {
		t.Errorf("expected 1 image, got %d", len(cat.Images))
	}

	// Test non-existent category
	_, err = svc.GetCategory("NonExistent")
	if err == nil {
		t.Error("expected error for non-existent category, got nil")
	}
	if err != ErrCategoryNotFound {
		t.Errorf("expected ErrCategoryNotFound, got %v", err)
	}
}
