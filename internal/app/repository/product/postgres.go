package pproduct

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/uptrace/bun"

	"github.com/GetLivreru/catalog-service/internal/app/entity"
	"github.com/GetLivreru/catalog-service/internal/app/repository"
	rcpostgres "github.com/GetLivreru/catalog-service/internal/app/repository/conn/postgres"
)

type (
	repoPg struct{ *_DB }
	_DB    = rcpostgres.Client
)

func NewRepoFromPostgres(client *rcpostgres.Client) repository.Product {
	return &repoPg{_DB: client}
}

func (r *repoPg) Create(ctx context.Context, product entity.Product) error {
	_, err := r.NewInsert().Model(&product).Exec(ctx)
	return err
}

func (r *repoPg) GetByGUIDs(ctx context.Context, guids []uuid.UUID) ([]entity.Product, error) {
	result := make([]entity.Product, 0)
	if len(guids) == 0 {
		return result, nil
	}
	err := r.NewSelect().Model(&result).Where("guid IN (?)", bun.In(guids)).Scan(ctx)
	return result, err
}

func (r *repoPg) Update(ctx context.Context, product entity.Product) error {
	res, err := r.NewUpdate().Model(&product).WherePK().ExcludeColumn("id", "created_at").Exec(ctx)
	return rcpostgres.UpdateErr(res, err)
}

func (r *repoPg) Delete(ctx context.Context, guid uuid.UUID) error {
	_, err := r.NewDelete().Model((*entity.Product)(nil)).Where("guid = ?", guid).Exec(ctx)
	return rcpostgres.DeleteErr(err)
}

func (r *repoPg) List(ctx context.Context, name *string, categoryGUID *uuid.UUID, minPrice, maxPrice *int64) ([]entity.Product, error) {
	result := make([]entity.Product, 0)
	query := r.NewSelect().Model(&result)
	if name != nil {
		query = query.Where("name = ?", *name)
	}
	if categoryGUID != nil {
		query = query.Where("category_guid = ?", *categoryGUID)
	}
	if minPrice != nil {
		query = query.Where("price >= ?", *minPrice)
	}
	if maxPrice != nil {
		query = query.Where("price <= ?", *maxPrice)
	}
	err := query.Scan(ctx)
	return result, err
}
