package store_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
	"github.com/Nerses01/Mountain_Breath/backend/internal/store"
)

// F2 (decision #95): the category admin. What the suite pins: the editor's
// read returns RAW English plus translations, updates are whole-value (a
// dropped language falls back to English again), deletion is refused by
// the schema's RESTRICT — translated, not pre-checked — and reorder is
// all-or-nothing.
func TestCategoryAdmin(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	honey := domain.Category{
		Slug: "honey", Name: "Honey", SortOrder: 10,
		Translations: map[domain.Locale]string{domain.LocaleHY: "Մեղր", domain.LocaleRU: "Мёд"},
	}
	tea := domain.Category{Slug: "tea", Name: "Tea", SortOrder: 20}
	if err := s.CreateCategory(ctx, &honey); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateCategory(ctx, &tea); err != nil {
		t.Fatal(err)
	}

	t.Run("the editor's read carries raw English and the translations", func(t *testing.T) {
		cats, err := s.AdminCategories(ctx)
		if err != nil {
			t.Fatalf("AdminCategories: %v", err)
		}
		if len(cats) != 2 || cats[0].Slug != "honey" || cats[1].Slug != "tea" {
			t.Fatalf("cats = %+v, want honey then tea by sort_order", cats)
		}
		if cats[0].Name != "Honey" {
			t.Errorf("name = %q, want the raw English", cats[0].Name)
		}
		if cats[0].Translations[domain.LocaleHY] != "Մեղր" {
			t.Errorf("translations = %v, want the Armenian name present", cats[0].Translations)
		}
	})

	t.Run("update is whole-value; a dropped language falls back", func(t *testing.T) {
		upd := domain.Category{
			ID: honey.ID, Slug: "mountain-honey", Name: "Mountain Honey", SortOrder: 10,
			Translations: map[domain.Locale]string{domain.LocaleHY: "Լեռնային մեղր"},
		}
		if err := s.UpdateCategory(ctx, &upd); err != nil {
			t.Fatalf("UpdateCategory: %v", err)
		}
		// The English row follows Name; the omitted Russian row is GONE, so
		// Russian readers fall back to English — what omission means.
		if got, _ := translationRow(t, "category_translations", "category_id", honey.ID, domain.LocaleEN); got != "Mountain Honey" {
			t.Errorf("en row = %q, want Mountain Honey", got)
		}
		if got, _ := translationRow(t, "category_translations", "category_id", honey.ID, domain.LocaleHY); got != "Լեռնային մեղր" {
			t.Errorf("hy row = %q", got)
		}
		if _, ok := translationRow(t, "category_translations", "category_id", honey.ID, domain.LocaleRU); ok {
			t.Error("ru row survived a whole-value update that omitted it")
		}
	})

	t.Run("a slug collision on update is ErrSlugTaken", func(t *testing.T) {
		upd := domain.Category{ID: tea.ID, Slug: "mountain-honey", Name: "Tea", SortOrder: 20}
		if err := s.UpdateCategory(ctx, &upd); !errors.Is(err, domain.ErrSlugTaken) {
			t.Errorf("err = %v, want ErrSlugTaken", err)
		}
	})

	t.Run("updating a ghost is ErrNotFound", func(t *testing.T) {
		upd := domain.Category{ID: 99999, Slug: "ghost", Name: "Ghost"}
		if err := s.UpdateCategory(ctx, &upd); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("reorder rewrites by position, all-or-nothing", func(t *testing.T) {
		if err := s.ReorderCategories(ctx, []int64{tea.ID, honey.ID}); err != nil {
			t.Fatalf("ReorderCategories: %v", err)
		}
		cats, err := s.AdminCategories(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if cats[0].ID != tea.ID || cats[1].ID != honey.ID {
			t.Errorf("order after reorder = %v then %v, want tea first", cats[0].Slug, cats[1].Slug)
		}

		// One unknown id must abort the WHOLE reorder — the admin never saw
		// the half-applied ordering that would otherwise commit.
		if err := s.ReorderCategories(ctx, []int64{honey.ID, 99999}); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
		cats, err = s.AdminCategories(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if cats[0].ID != tea.ID {
			t.Error("a failed reorder still moved rows")
		}
	})

	t.Run("delete: refused with products, allowed when empty", func(t *testing.T) {
		// Hang one product off honey — the RESTRICT constraint's trigger.
		var productID int64
		if err := testPool.QueryRow(ctx, `
			INSERT INTO products (category_id, slug, name)
			VALUES ($1, 'wild-honey', 'Wild Honey') RETURNING id`,
			honey.ID).Scan(&productID); err != nil {
			t.Fatal(err)
		}

		if err := s.DeleteCategory(ctx, honey.ID); !errors.Is(err, domain.ErrCategoryInUse) {
			t.Errorf("err = %v, want ErrCategoryInUse", err)
		}

		// Empty categories go, translations riding along via CASCADE.
		if err := s.DeleteCategory(ctx, tea.ID); err != nil {
			t.Fatalf("deleting empty category: %v", err)
		}
		if err := s.DeleteCategory(ctx, tea.ID); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("second delete: err = %v, want ErrNotFound", err)
		}
	})
}
