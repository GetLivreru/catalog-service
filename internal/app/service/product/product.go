package sproduct

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

func NewService(repoProduct repository.Product, repoCategory repository.Category) service.Product {
	return &srv{repoCategory: repoCategory, repoProduct: repoProduct}
}

func (s *srv) GetByGUIDs(ctx context.Context, guids []uuid.UUID) ([]entity.Product, error) {
	return s.repoProduct.GetByGUIDs(ctx, guids)
}

func (s *srv) Create(ctx context.Context, req entity.RequestProductCreate) (entity.Product, error) {
	existing, err := s.repoProduct.List(ctx, &req.Name, nil)
	if err != nil {
		return entity.Product{}, err
	}
	if len(existing) > 0 {
		return entity.Product{}, entity.ErrAlreadyExists
	}
	if err := s.checkCategory(ctx, req.CategoryGUID); err != nil {
		return entity.Product{}, err
	}
	now := time.Now()
	item := entity.Product{GUID: uuid.Must(uuid.NewV4()), Name: req.Name, Description: req.Description, Price: req.Price, CategoryGUID: req.CategoryGUID, CreatedAt: now, UpdatedAt: now}
	if err := s.repoProduct.Create(ctx, item); err != nil {
		return entity.Product{}, err
	}
	return item, nil
}

func (s *srv) Update(ctx context.Context, guid uuid.UUID, req entity.RequestProductUpdate) (entity.Product, error) {
	items, err := s.GetByGUIDs(ctx, []uuid.UUID{guid})
	if err != nil {
		return entity.Product{}, err
	}
	if len(items) == 0 {
		return entity.Product{}, entity.ErrNotFound
	}
	item := items[0]
	if req.Name != "" {
		existing, err := s.repoProduct.List(ctx, &req.Name, nil)
		if err != nil {
			return entity.Product{}, err
		}
		for _, other := range existing {
			if other.GUID != guid {
				return entity.Product{}, entity.ErrAlreadyExists
			}
		}
		item.Name = req.Name
	}
	if req.CategoryGUID != uuid.Nil {
		if err := s.checkCategory(ctx, req.CategoryGUID); err != nil {
			return entity.Product{}, err
		}
		item.CategoryGUID = req.CategoryGUID
	}
	if req.Price > 0 {
		item.Price = req.Price
	}
	if req.Description != nil {
		item.Description = req.Description
	}
	item.UpdatedAt = time.Now()
	if err := s.repoProduct.Update(ctx, item); err != nil {
		return entity.Product{}, err
	}
	return item, nil
}

func (s *srv) Delete(ctx context.Context, guid uuid.UUID) error {
	items, err := s.GetByGUIDs(ctx, []uuid.UUID{guid})
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return entity.ErrNotFound
	}
	return s.repoProduct.Delete(ctx, guid)
}

func (s *srv) List(ctx context.Context, req entity.RequestProductList) ([]entity.Product, error) {
	return s.repoProduct.List(ctx, nil, req.CategoryGUID)
}

func (s *srv) checkCategory(ctx context.Context, guid uuid.UUID) error {
	items, err := s.repoCategory.GetByGUIDs(ctx, []uuid.UUID{guid})
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return entity.ErrNotFound
	}
	return nil
}
