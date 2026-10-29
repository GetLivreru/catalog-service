package service

import (
	"context"

	"github.com/gofrs/uuid"

	"github.com/GetLivreru/catalog-service/internal/app/entity"
)

type Category interface {
	Create(context.Context, entity.RequestCategoryCreate) (entity.Category, error)
	GetByGUIDs(context.Context, []uuid.UUID) ([]entity.Category, error)
	Update(context.Context, uuid.UUID, entity.RequestCategoryUpdate) (entity.Category, error)
	Delete(context.Context, uuid.UUID) error
	List(ctx context.Context) ([]entity.Category, error)
}
type Product interface {
	Create(context.Context, entity.RequestProductCreate) (entity.Product, error)
	GetByGUIDs(context.Context, []uuid.UUID) ([]entity.Product, error)
	Update(context.Context, uuid.UUID, entity.RequestProductUpdate) (entity.Product, error)
	Delete(context.Context, uuid.UUID) error
	List(ctx context.Context, req entity.RequestProductList) ([]entity.Product, error)
}
