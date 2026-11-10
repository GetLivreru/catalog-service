package scategory

import (
	"context"
	"time"

	"github.com/gofrs/uuid"

	"github.com/GetLivreru/catalog-service/internal/app/entity"
	"github.com/GetLivreru/catalog-service/internal/app/repository"
	"github.com/GetLivreru/catalog-service/internal/app/service"
)

type srv struct {
	repoCategory repository.Category
	repoProduct  repository.Product
}

func NewService(repoCategory repository.Category, repoProduct repository.Product) service.Category {
	return &srv{repoCategory: repoCategory, repoProduct: repoProduct}
}

func (s *srv) GetByGUIDs(ctx context.Context, guids []uuid.UUID) ([]entity.Category, error) {
	return s.repoCategory.GetByGUIDs(ctx, guids)
}

func (s *srv) Create(ctx context.Context, req entity.RequestCategoryCreate) (entity.Category, error) {
	var item entity.Category
	err := s.repoCategory.InsideTx(ctx, func(ctx context.Context) error {
		existing, err := s.repoCategory.List(ctx, &req.Name)
		if err != nil {
			return err
		}
		if len(existing) > 0 {
			return entity.ErrAlreadyExists
		}
		now := time.Now()
		item = entity.Category{GUID: uuid.Must(uuid.NewV4()), Name: req.Name, CreatedAt: now, UpdatedAt: now}
		if err := s.repoCategory.Create(ctx, item); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return entity.Category{}, err
	}
	return item, nil
}

func (s *srv) Update(ctx context.Context, guid uuid.UUID, req entity.RequestCategoryUpdate) (entity.Category, error) {
	var item entity.Category
	err := s.repoCategory.InsideTx(ctx, func(ctx context.Context) error {
		items, err := s.GetByGUIDs(ctx, []uuid.UUID{guid})
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return entity.ErrNotFound
		}
		item = items[0]
		if req.Name != "" {
			existing, err := s.repoCategory.List(ctx, &req.Name)
			if err != nil {
				return err
			}
			for _, other := range existing {
				if other.GUID != guid {
					return entity.ErrAlreadyExists
				}
			}
			item.Name = req.Name
		}
		item.UpdatedAt = time.Now()
		if err := s.repoCategory.Update(ctx, item); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return entity.Category{}, err
	}
	return item, nil
}

func (s *srv) Delete(ctx context.Context, guid uuid.UUID) error {
	return s.repoCategory.InsideTx(ctx, func(ctx context.Context) error {
		items, err := s.GetByGUIDs(ctx, []uuid.UUID{guid})
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return entity.ErrNotFound
		}
		products, err := s.repoProduct.List(ctx, nil, &guid, nil, nil)
		if err != nil {
			return err
		}
		if len(products) > 0 {
			return entity.ErrCategoryHasProducts
		}
		return s.repoCategory.Delete(ctx, guid)
	})
}

func (s *srv) List(ctx context.Context) ([]entity.Category, error) {
	return s.repoCategory.List(ctx, nil)
}
