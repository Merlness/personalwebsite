package portfolio

import (
	"os"
	"path/filepath"
	"testing"
)

func createTempFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("dummy image content"), 0644); err != nil {
		t.Fatal(err)
	}
}

func findCategory(categories []Category, name string) *Category {
	for _, category := range categories {
		if category.Name == name {
			return &category
		}
	}
	return nil
}

func TestFilesystemService_GetCategories_ReturnsTwoCategories(t *testing.T) {
	tmpDir := t.TempDir()
	landscapeDir := filepath.Join(tmpDir, "Landscape")
	peopleDir := filepath.Join(tmpDir, "People")

	if err := os.Mkdir(landscapeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(peopleDir, 0755); err != nil {
		t.Fatal(err)
	}

	createTempFile(t, filepath.Join(landscapeDir, "mountains.jpg"))
	createTempFile(t, filepath.Join(peopleDir, "portrait.jpg"))

	svc := NewFilesystemService(tmpDir, "")

	cats, err := svc.GetCategories()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(cats) != 2 {
		t.Errorf("expected 2 categories, got %d", len(cats))
	}
}

func TestFilesystemService_GetCategories_LandscapeHasFourImages(t *testing.T) {
	tmpDir := t.TempDir()
	landscapeDir := filepath.Join(tmpDir, "Landscape")
	peopleDir := filepath.Join(tmpDir, "People")

	if err := os.Mkdir(landscapeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(peopleDir, 0755); err != nil {
		t.Fatal(err)
	}

	createTempFile(t, filepath.Join(landscapeDir, "mountains.jpg"))
	createTempFile(t, filepath.Join(landscapeDir, "lake.png"))
	createTempFile(t, filepath.Join(landscapeDir, "z_last.jpg"))
	createTempFile(t, filepath.Join(landscapeDir, "a_first.jpg"))
	createTempFile(t, filepath.Join(peopleDir, "portrait.jpg"))

	svc := NewFilesystemService(tmpDir, "")

	cats, err := svc.GetCategories()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	lCat := findCategory(cats, "Landscape")
	if lCat == nil {
		t.Fatal("expected Landscape category, not found")
	}

	if len(lCat.Images) != 4 {
		t.Errorf("expected 4 images in Landscape, got %d", len(lCat.Images))
	}
}

func TestFilesystemService_GetCategories_LandscapeImagesSortedByPath(t *testing.T) {
	tmpDir := t.TempDir()
	landscapeDir := filepath.Join(tmpDir, "Landscape")
	peopleDir := filepath.Join(tmpDir, "People")

	if err := os.Mkdir(landscapeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(peopleDir, 0755); err != nil {
		t.Fatal(err)
	}

	createTempFile(t, filepath.Join(landscapeDir, "mountains.jpg"))
	createTempFile(t, filepath.Join(landscapeDir, "lake.png"))
	createTempFile(t, filepath.Join(landscapeDir, "z_last.jpg"))
	createTempFile(t, filepath.Join(landscapeDir, "a_first.jpg"))
	createTempFile(t, filepath.Join(peopleDir, "portrait.jpg"))

	svc := NewFilesystemService(tmpDir, "")

	cats, err := svc.GetCategories()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	lCat := findCategory(cats, "Landscape")
	if lCat == nil {
		t.Fatal("expected Landscape category, not found")
	}

	expectedPaths := []string{
		filepath.Join("Landscape", "a_first"),
		filepath.Join("Landscape", "lake"),
		filepath.Join("Landscape", "mountains"),
		filepath.Join("Landscape", "z_last"),
	}

	for idx, image := range lCat.Images {
		if idx >= len(expectedPaths) {
			break
		}
		if image.Path != expectedPaths[idx] {
			t.Errorf("index %d: expected path %s, got %s", idx, expectedPaths[idx], image.Path)
		}
	}
}

func TestFilesystemService_GetCategories_LandscapeImagesHaveCorrectExtensions(t *testing.T) {
	tmpDir := t.TempDir()
	landscapeDir := filepath.Join(tmpDir, "Landscape")
	peopleDir := filepath.Join(tmpDir, "People")

	if err := os.Mkdir(landscapeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(peopleDir, 0755); err != nil {
		t.Fatal(err)
	}

	createTempFile(t, filepath.Join(landscapeDir, "mountains.jpg"))
	createTempFile(t, filepath.Join(landscapeDir, "lake.png"))
	createTempFile(t, filepath.Join(landscapeDir, "z_last.jpg"))
	createTempFile(t, filepath.Join(landscapeDir, "a_first.jpg"))
	createTempFile(t, filepath.Join(peopleDir, "portrait.jpg"))

	svc := NewFilesystemService(tmpDir, "")

	cats, err := svc.GetCategories()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	lCat := findCategory(cats, "Landscape")
	if lCat == nil {
		t.Fatal("expected Landscape category, not found")
	}

	expectedExts := []string{".jpg", ".png", ".jpg", ".jpg"}

	for idx, image := range lCat.Images {
		if idx >= len(expectedExts) {
			break
		}
		if image.Ext != expectedExts[idx] {
			t.Errorf("index %d: expected ext %s, got %s", idx, expectedExts[idx], image.Ext)
		}
	}
}

func TestFilesystemService_GetCategories_LandscapeCoverImageIsLastAlphabetically(t *testing.T) {
	tmpDir := t.TempDir()
	landscapeDir := filepath.Join(tmpDir, "Landscape")
	peopleDir := filepath.Join(tmpDir, "People")

	if err := os.Mkdir(landscapeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(peopleDir, 0755); err != nil {
		t.Fatal(err)
	}

	createTempFile(t, filepath.Join(landscapeDir, "mountains.jpg"))
	createTempFile(t, filepath.Join(landscapeDir, "lake.png"))
	createTempFile(t, filepath.Join(landscapeDir, "z_last.jpg"))
	createTempFile(t, filepath.Join(landscapeDir, "a_first.jpg"))
	createTempFile(t, filepath.Join(peopleDir, "portrait.jpg"))

	svc := NewFilesystemService(tmpDir, "")

	cats, err := svc.GetCategories()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	lCat := findCategory(cats, "Landscape")
	if lCat == nil {
		t.Fatal("expected Landscape category, not found")
	}

	expectedCoverPath := filepath.Join("Landscape", "z_last")
	if lCat.CoverImage.Path != expectedCoverPath {
		t.Errorf("expected cover image path %s, got %s", expectedCoverPath, lCat.CoverImage.Path)
	}

	if lCat.CoverImage.Ext != ".jpg" {
		t.Errorf("expected cover image ext .jpg, got %s", lCat.CoverImage.Ext)
	}
}

func TestFilesystemService_GetCategories_PeopleHasOneImage(t *testing.T) {
	tmpDir := t.TempDir()
	landscapeDir := filepath.Join(tmpDir, "Landscape")
	peopleDir := filepath.Join(tmpDir, "People")

	if err := os.Mkdir(landscapeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(peopleDir, 0755); err != nil {
		t.Fatal(err)
	}

	createTempFile(t, filepath.Join(landscapeDir, "mountains.jpg"))
	createTempFile(t, filepath.Join(peopleDir, "portrait.jpg"))

	svc := NewFilesystemService(tmpDir, "")

	cats, err := svc.GetCategories()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	pCat := findCategory(cats, "People")
	if pCat == nil {
		t.Fatal("expected People category, not found")
	}

	if len(pCat.Images) != 1 {
		t.Errorf("expected 1 image in People, got %d", len(pCat.Images))
	}

	expectedPath := filepath.Join("People", "portrait")
	if pCat.Images[0].Path != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, pCat.Images[0].Path)
	}

	if pCat.Images[0].Ext != ".jpg" {
		t.Errorf("expected ext .jpg, got %s", pCat.Images[0].Ext)
	}
}

func TestFilesystemService_GetCategory_ReturnsExistingCategory(t *testing.T) {
	tmpDir := t.TempDir()
	landscapeDir := filepath.Join(tmpDir, "Landscape")
	if err := os.Mkdir(landscapeDir, 0755); err != nil {
		t.Fatal(err)
	}

	createTempFile(t, filepath.Join(landscapeDir, "mountains.jpg"))

	svc := NewFilesystemService(tmpDir, "")

	cat, err := svc.GetCategory("Landscape")
	if err != nil {
		t.Fatalf("expected no error for existing category, got %v", err)
	}

	if cat.Name != "Landscape" {
		t.Errorf("expected category name Landscape, got %s", cat.Name)
	}

	if len(cat.Images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(cat.Images))
	}

	expectedPath := filepath.Join("Landscape", "mountains")
	if cat.Images[0].Path != expectedPath {
		t.Errorf("expected image path %s, got %s", expectedPath, cat.Images[0].Path)
	}

	if cat.Images[0].Ext != ".jpg" {
		t.Errorf("expected image ext .jpg, got %s", cat.Images[0].Ext)
	}
}

func TestFilesystemService_GetCategory_ReturnsErrorForNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewFilesystemService(tmpDir, "")

	_, err := svc.GetCategory("NonExistent")
	if err == nil {
		t.Error("expected error for non-existent category, got nil")
	}
	if err != ErrCategoryNotFound {
		t.Errorf("expected ErrCategoryNotFound, got %v", err)
	}
}

func TestGroupCategories_SplitsPortfolioAndAdventures(t *testing.T) {
	categories := []Category{
		{Name: "Landscape", Group: "portfolio"},
		{Name: "People", Group: "portfolio"},
		{Name: "Alaska", Group: "adventure"},
		{Name: "Wildlife", Group: "portfolio"},
	}

	portfolioCategories, adventureCategories := GroupCategories(categories)

	if len(portfolioCategories) != 3 {
		t.Fatalf("expected 3 portfolio categories, got %d", len(portfolioCategories))
	}
	if portfolioCategories[0].Name != "Landscape" {
		t.Errorf("expected first portfolio category Landscape, got %s", portfolioCategories[0].Name)
	}
	if portfolioCategories[1].Name != "People" {
		t.Errorf("expected second portfolio category People, got %s", portfolioCategories[1].Name)
	}
	if portfolioCategories[2].Name != "Wildlife" {
		t.Errorf("expected third portfolio category Wildlife, got %s", portfolioCategories[2].Name)
	}

	if len(adventureCategories) != 1 {
		t.Fatalf("expected 1 adventure category, got %d", len(adventureCategories))
	}
	if adventureCategories[0].Name != "Alaska" {
		t.Errorf("expected adventure category Alaska, got %s", adventureCategories[0].Name)
	}
}

func TestGroupCategories_EmptyInput(t *testing.T) {
	var categories []Category

	portfolioCategories, adventureCategories := GroupCategories(categories)

	if len(portfolioCategories) != 0 {
		t.Errorf("expected 0 portfolio categories, got %d", len(portfolioCategories))
	}
	if len(adventureCategories) != 0 {
		t.Errorf("expected 0 adventure categories, got %d", len(adventureCategories))
	}
}

func TestGroupCategories_AllPortfolio(t *testing.T) {
	categories := []Category{
		{Name: "Landscape", Group: "portfolio"},
		{Name: "Structures", Group: "portfolio"},
	}

	portfolioCategories, adventureCategories := GroupCategories(categories)

	if len(portfolioCategories) != 2 {
		t.Fatalf("expected 2 portfolio categories, got %d", len(portfolioCategories))
	}
	if len(adventureCategories) != 0 {
		t.Errorf("expected 0 adventure categories, got %d", len(adventureCategories))
	}
}

func TestGroupCategories_AllAdventures(t *testing.T) {
	categories := []Category{
		{Name: "Alaska", Group: "adventure"},
	}

	portfolioCategories, adventureCategories := GroupCategories(categories)

	if len(portfolioCategories) != 0 {
		t.Errorf("expected 0 portfolio categories, got %d", len(portfolioCategories))
	}
	if len(adventureCategories) != 1 {
		t.Fatalf("expected 1 adventure category, got %d", len(adventureCategories))
	}
	if adventureCategories[0].Name != "Alaska" {
		t.Errorf("expected adventure category Alaska, got %s", adventureCategories[0].Name)
	}
}

func TestGetCategories_AlaskaHasAdventureGroup(t *testing.T) {
	tmpDir := t.TempDir()
	landscapeDir := filepath.Join(tmpDir, "Landscape")
	alaskaDir := filepath.Join(tmpDir, "Alaska")

	if err := os.Mkdir(landscapeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(alaskaDir, 0755); err != nil {
		t.Fatal(err)
	}

	createTempFile(t, filepath.Join(landscapeDir, "mountains.jpg"))
	createTempFile(t, filepath.Join(alaskaDir, "glacier.jpg"))

	svc := NewFilesystemService(tmpDir, "")

	cats, err := svc.GetCategories()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	alaskaCat := findCategory(cats, "Alaska")
	if alaskaCat == nil {
		t.Fatal("expected Alaska category, not found")
	}
	if alaskaCat.Group != "adventure" {
		t.Errorf("expected Alaska group 'adventure', got '%s'", alaskaCat.Group)
	}

	landscapeCat := findCategory(cats, "Landscape")
	if landscapeCat == nil {
		t.Fatal("expected Landscape category, not found")
	}
	if landscapeCat.Group != "portfolio" {
		t.Errorf("expected Landscape group 'portfolio', got '%s'", landscapeCat.Group)
	}
}
