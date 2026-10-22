//go:build integration

package integration_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/GetLivreru/catalog-service/internal/app/config"
	"github.com/GetLivreru/catalog-service/internal/app/entity"
	pcategory "github.com/GetLivreru/catalog-service/internal/app/repository/category"
	rcpostgres "github.com/GetLivreru/catalog-service/internal/app/repository/conn/postgres"
	pproduct "github.com/GetLivreru/catalog-service/internal/app/repository/product"
	"github.com/gofrs/uuid"
)

func TestRepositories(t *testing.T) {
	if os.Getenv("APP_REPOSITORY_POSTGRES_NAME") == "" {
		t.Skip("set APP_REPOSITORY_POSTGRES_* for a dedicated test database")
	}
	ctx := context.Background()
	config.Load()
	client, err := rcpostgres.NewClient(ctx, config.Root.Repository.Postgres)
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(err)
	t.Cleanup(func() { must(client.GetRawBunDB().Close()) })
	oldVer, newVer, err := client.Migrate(ctx)
	must(err)
	t.Logf("Database migrated old_version=%d new_version=%d", oldVer, newVer)
	if newVer != 2 {
		t.Fatalf("migration version = %d", newVer)
	}
	_, repeatVersion, err := client.Migrate(ctx)
	must(err)
	if repeatVersion != 2 {
		t.Fatal("migration is not idempotent")
	}
	cats := pcategory.NewRepoFromPostgres(client)
	products := pproduct.NewRepoFromPostgres(client)
	now := time.Now().UTC().Truncate(time.Microsecond)
	cat := entity.Category{GUID: uuid.Must(uuid.NewV4()), Name: "Электроника", CreatedAt: now, UpdatedAt: now}
	prod := entity.Product{GUID: uuid.Must(uuid.NewV4()), CategoryGUID: cat.GUID, Name: "Пылесос Dyson V15", Price: 4999999, CreatedAt: now, UpdatedAt: now}
	t.Cleanup(func() { must(products.Delete(ctx, prod.GUID)); must(cats.Delete(ctx, cat.GUID)) })
	must(cats.Create(ctx, cat))
	t.Logf("Category created guid=%s", cat.GUID)
	found, err := cats.GetByGUIDs(ctx, []uuid.UUID{cat.GUID, uuid.Must(uuid.NewV4())})
	must(err)
	if len(found) != 1 || found[0].Name != cat.Name || found[0].ID == 0 {
		t.Fatalf("unexpected categories: %+v", found)
	}
	t.Logf("Category found name=%s", found[0].Name)
	originalID := found[0].ID
	cat.Name = "Бытовая техника"
	cat.ID = 999999
	cat.CreatedAt = now.Add(time.Hour)
	cat.UpdatedAt = now.Add(time.Second)
	must(cats.Update(ctx, cat))
	t.Log("Category updated")
	found, err = cats.GetByGUIDs(ctx, []uuid.UUID{cat.GUID})
	must(err)
	if len(found) != 1 || found[0].ID != originalID || !found[0].CreatedAt.Equal(now) || found[0].Name != cat.Name {
		t.Fatal("category update changed immutable fields or missed name")
	}
	all, err := cats.List(ctx, nil)
	must(err)
	if len(all) == 0 {
		t.Fatal("empty category list")
	}
	t.Logf("All categories count=%d", len(all))
	filtered, err := cats.List(ctx, &cat.Name)
	must(err)
	if len(filtered) == 0 {
		t.Fatal("missing category name match")
	}
	t.Logf("Filtered categories count=%d", len(filtered))
	desc := "Мощный пылесос"
	prod.Description = &desc
	must(products.Create(ctx, prod))
	t.Logf("Product created guid=%s", prod.GUID)
	if err := cats.Delete(ctx, cat.GUID); err == nil {
		t.Fatal("ON DELETE RESTRICT failed")
	}
	items, err := products.List(ctx, nil, &cat.GUID)
	must(err)
	if len(items) != 1 || items[0].Description == nil || *items[0].Description != desc {
		t.Fatalf("unexpected products: %+v", items)
	}
	t.Logf("Products in category count=%d", len(items))
	productID := items[0].ID
	prod.Name = "Обновлённый пылесос"
	prod.Description = nil
	prod.Price = 42
	prod.ID = 999999
	prod.CreatedAt = now.Add(time.Hour)
	prod.UpdatedAt = now.Add(time.Second)
	must(products.Update(ctx, prod))
	items, err = products.GetByGUIDs(ctx, []uuid.UUID{prod.GUID, uuid.Must(uuid.NewV4())})
	must(err)
	if len(items) != 1 || items[0].ID != productID || !items[0].CreatedAt.Equal(now) || items[0].Description != nil || items[0].Price != 42 || items[0].Name != prod.Name {
		t.Fatal("product update mismatch")
	}
	for _, filter := range []struct {
		name     *string
		category *uuid.UUID
		want     int
	}{
		{nil, nil, 1}, {&prod.Name, nil, 1}, {nil, &cat.GUID, 1}, {&prod.Name, &cat.GUID, 1},
	} {
		rows, err := products.List(ctx, filter.name, filter.category)
		must(err)
		if len(rows) < filter.want {
			t.Fatal("product filters missed record")
		}
	}
	missing := uuid.Must(uuid.NewV4())
	absent := "missing-" + missing.String()
	items, err = products.List(ctx, &absent, &cat.GUID)
	must(err)
	if len(items) != 0 {
		t.Fatal("product name filter ignored")
	}
	items, err = products.List(ctx, &prod.Name, &missing)
	must(err)
	if len(items) != 0 {
		t.Fatal("product category filter ignored")
	}
	filtered, err = cats.List(ctx, &absent)
	must(err)
	if len(filtered) != 0 {
		t.Fatal("category name filter ignored")
	}
	for _, ids := range [][]uuid.UUID{nil, {}, {missing}} {
		found, err = cats.GetByGUIDs(ctx, ids)
		must(err)
		if len(found) != 0 {
			t.Fatal("expected empty categories")
		}
		items, err = products.GetByGUIDs(ctx, ids)
		must(err)
		if len(items) != 0 {
			t.Fatal("expected empty products")
		}
	}
	if !errors.Is(cats.Update(ctx, entity.Category{GUID: missing}), entity.ErrNotFound) {
		t.Fatal("missing category update")
	}
	if !errors.Is(products.Update(ctx, entity.Product{GUID: missing}), entity.ErrNotFound) {
		t.Fatal("missing product update")
	}
	must(products.Delete(ctx, prod.GUID))
	must(products.Delete(ctx, prod.GUID))
	must(cats.Delete(ctx, cat.GUID))
	must(cats.Delete(ctx, cat.GUID))
	t.Log("Cleanup complete")
}
