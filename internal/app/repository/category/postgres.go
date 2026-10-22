package pcategory

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

func NewRepoFromPostgres(client *rcpostgres.Client) repository.Category {
	return &repoPg{_DB: client}
}

func (r *repoPg) Create(ctx context.Context, category entity.Category) error {
	_, err := r.NewInsert().Model(&category).Exec(ctx)
	return err
}

func (r *repoPg) GetByGUIDs(ctx context.Context, guids []uuid.UUID) ([]entity.Category, error) {
	result := make([]entity.Category, 0)
	if len(guids) == 0 {
		return result, nil
	}
	err := r.NewSelect().Model(&result).Where("guid IN (?)", bun.In(guids)).Scan(ctx)
	return result, err
}

func (r *repoPg) Update(ctx context.Context, category entity.Category) error {
	res, err := r.NewUpdate().Model(&category).WherePK().ExcludeColumn("id", "created_at").Exec(ctx)
	return rcpostgres.UpdateErr(res, err)
}

func (r *repoPg) Delete(ctx context.Context, guid uuid.UUID) error {
	_, err := r.NewDelete().Model((*entity.Category)(nil)).Where("guid = ?", guid).Exec(ctx)
	return rcpostgres.DeleteErr(err)
}

func (r *repoPg) List(ctx context.Context, name *string) ([]entity.Category, error) {
	result := make([]entity.Category, 0)
	query := r.NewSelect().Model(&result)
	if name != nil {
		query = query.Where("name = ?", *name)
	}

	err := query.Scan(ctx)
	return result, err
}
