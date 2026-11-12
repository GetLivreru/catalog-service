package sproduct

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/GetLivreru/catalog-service/internal/app/entity"
	"github.com/GetLivreru/catalog-service/internal/app/repository/mocks"
	"github.com/GetLivreru/catalog-service/internal/pkg/testutil"
)

// Each table entry gets fresh mocks: SetupTest alone does not run for suite subtests.
type productSuite struct {
	suite.Suite
	srv          *srv
	productRepo  *mocks.MockProduct
	categoryRepo *mocks.MockCategory
	ctx          context.Context
	txCtx        context.Context
}

type transactionContextKey struct{}

func (s *productSuite) SetupTest() {
	s.ctx = context.Background()
	s.txCtx = context.WithValue(s.ctx, transactionContextKey{}, "transaction")
	s.productRepo = mocks.NewMockProduct(s.T())
	s.categoryRepo = mocks.NewMockCategory(s.T())
	s.srv = NewService(s.productRepo, s.categoryRepo).(*srv)
}

func (s *productSuite) SetupSubTest() { s.SetupTest() }

// Execute the callback with a different context, so expectations also verify
// that every repository operation receives the transaction context.
func (s *productSuite) expectTransaction() *mock.Call {
	return s.productRepo.EXPECT().InsideTx(s.ctx, mock.Anything).
		RunAndReturn(func(_ context.Context, fn func(context.Context) error) error {
			return fn(s.txCtx)
		}).Once()
}

func productFixture() entity.Product {
	return entity.Product{
		GUID: uuid.FromStringOrNil("10000000-0000-4000-8000-000000000001"),
		Name: "Original product", Description: testutil.PtrString("Original description"), Price: 1000,
		CategoryGUID: uuid.FromStringOrNil("20000000-0000-4000-8000-000000000001"),
		CreatedAt:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:    time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
	}
}

type createProductSuite struct{ productSuite }

func TestCreateProductSuite(t *testing.T) { suite.Run(t, new(createProductSuite)) }

func (s *createProductSuite) TestCreate() {
	type args struct{ req entity.RequestProductCreate }
	type want struct{ err error }
	fixture := productFixture()
	request := entity.RequestProductCreate{
		Name: "New product", Description: testutil.PtrString("New description"), Price: 2500, CategoryGUID: fixture.CategoryGUID,
	}
	repoErr := errors.New("repository unavailable")
	cases := []struct {
		name    string
		args    args
		want    want
		prepare func(entity.RequestProductCreate)
	}{
		{name: "success", args: args{request}, prepare: func(req entity.RequestProductCreate) {
			tx := s.expectTransaction()
			list := s.productRepo.EXPECT().List(s.txCtx, &req.Name, (*uuid.UUID)(nil), (*int64)(nil), (*int64)(nil)).Return(nil, nil).Once()
			category := s.categoryRepo.EXPECT().GetByGUIDs(s.txCtx, []uuid.UUID{req.CategoryGUID}).
				Return([]entity.Category{{GUID: req.CategoryGUID}}, nil).Once()
			create := s.productRepo.EXPECT().Create(s.txCtx, mock.MatchedBy(func(item entity.Product) bool {
				return item.Name == req.Name && item.Price == req.Price && item.CategoryGUID == req.CategoryGUID &&
					item.Description != nil && *item.Description == *req.Description && item.GUID != uuid.Nil &&
					!item.CreatedAt.IsZero() && item.UpdatedAt.Equal(item.CreatedAt)
			})).Return(nil).Once()
			mock.InOrder(tx, list, category, create)
		}},
		{name: "already exists", args: args{request}, want: want{entity.ErrAlreadyExists}, prepare: func(req entity.RequestProductCreate) {
			tx := s.expectTransaction()
			list := s.productRepo.EXPECT().List(s.txCtx, &req.Name, (*uuid.UUID)(nil), (*int64)(nil), (*int64)(nil)).Return([]entity.Product{fixture}, nil).Once()
			mock.InOrder(tx, list)
		}},
		{name: "category not found", args: args{request}, want: want{entity.ErrNotFound}, prepare: func(req entity.RequestProductCreate) {
			tx := s.expectTransaction()
			list := s.productRepo.EXPECT().List(s.txCtx, &req.Name, (*uuid.UUID)(nil), (*int64)(nil), (*int64)(nil)).Return(nil, nil).Once()
			category := s.categoryRepo.EXPECT().GetByGUIDs(s.txCtx, []uuid.UUID{req.CategoryGUID}).Return(nil, nil).Once()
			mock.InOrder(tx, list, category)
		}},
		{name: "list error", args: args{request}, want: want{repoErr}, prepare: func(req entity.RequestProductCreate) {
			tx := s.expectTransaction()
			list := s.productRepo.EXPECT().List(s.txCtx, &req.Name, (*uuid.UUID)(nil), (*int64)(nil), (*int64)(nil)).Return(nil, repoErr).Once()
			mock.InOrder(tx, list)
		}},
		{name: "category lookup error", args: args{request}, want: want{repoErr}, prepare: func(req entity.RequestProductCreate) {
			tx := s.expectTransaction()
			list := s.productRepo.EXPECT().List(s.txCtx, &req.Name, (*uuid.UUID)(nil), (*int64)(nil), (*int64)(nil)).Return(nil, nil).Once()
			category := s.categoryRepo.EXPECT().GetByGUIDs(s.txCtx, []uuid.UUID{req.CategoryGUID}).Return(nil, repoErr).Once()
			mock.InOrder(tx, list, category)
		}},
		{name: "create error", args: args{request}, want: want{repoErr}, prepare: func(req entity.RequestProductCreate) {
			tx := s.expectTransaction()
			list := s.productRepo.EXPECT().List(s.txCtx, &req.Name, (*uuid.UUID)(nil), (*int64)(nil), (*int64)(nil)).Return(nil, nil).Once()
			category := s.categoryRepo.EXPECT().GetByGUIDs(s.txCtx, []uuid.UUID{req.CategoryGUID}).Return([]entity.Category{{GUID: req.CategoryGUID}}, nil).Once()
			create := s.productRepo.EXPECT().Create(s.txCtx, mock.Anything).Return(repoErr).Once()
			mock.InOrder(tx, list, category, create)
		}},
		{name: "transaction error", args: args{request}, want: want{repoErr}, prepare: func(entity.RequestProductCreate) {
			s.productRepo.EXPECT().InsideTx(s.ctx, mock.Anything).Return(repoErr).Once()
		}},
		{name: "commit error", args: args{request}, want: want{repoErr}, prepare: func(req entity.RequestProductCreate) {
			tx := s.productRepo.EXPECT().InsideTx(s.ctx, mock.Anything).RunAndReturn(func(_ context.Context, fn func(context.Context) error) error {
				s.Require().NoError(fn(s.txCtx))
				return repoErr
			}).Once()
			list := s.productRepo.EXPECT().List(s.txCtx, &req.Name, (*uuid.UUID)(nil), (*int64)(nil), (*int64)(nil)).Return(nil, nil).Once()
			category := s.categoryRepo.EXPECT().GetByGUIDs(s.txCtx, []uuid.UUID{req.CategoryGUID}).Return([]entity.Category{{GUID: req.CategoryGUID}}, nil).Once()
			create := s.productRepo.EXPECT().Create(s.txCtx, mock.Anything).Return(nil).Once()
			mock.InOrder(tx, list, category, create)
		}},
	}
	for _, tc := range cases {
		s.Run(tc.name, func() {
			tc.prepare(tc.args.req)
			before := time.Now()
			got, err := s.srv.Create(s.ctx, tc.args.req)
			if tc.want.err != nil {
				s.Require().ErrorIs(err, tc.want.err)
				s.Equal(entity.Product{}, got, "failed creation must not return an unsaved product")
				return
			}
			s.Require().NoError(err)
			s.NotEqual(uuid.Nil, got.GUID)
			s.Equal(tc.args.req.Name, got.Name)
			s.Equal(tc.args.req.Price, got.Price)
			s.Equal(tc.args.req.Description, got.Description)
			s.Equal(tc.args.req.CategoryGUID, got.CategoryGUID)
			s.False(got.CreatedAt.Before(before))
			s.False(got.CreatedAt.After(time.Now()))
			s.Equal(got.CreatedAt, got.UpdatedAt)
		})
	}
}

type getByGUIDsProductSuite struct{ productSuite }

func TestGetByGUIDsProductSuite(t *testing.T) { suite.Run(t, new(getByGUIDsProductSuite)) }

func (s *getByGUIDsProductSuite) TestGetByGUIDs() {
	type args struct{ guids []uuid.UUID }
	type want struct {
		products []entity.Product
		err      error
	}
	item := productFixture()
	repoErr := errors.New("lookup failed")
	cases := []struct {
		name    string
		args    args
		want    want
		prepare func(args, want)
	}{
		{name: "single product found", args: args{[]uuid.UUID{item.GUID}}, want: want{products: []entity.Product{item}}},
		{name: "not found returns empty slice", args: args{[]uuid.UUID{item.GUID}}, want: want{products: []entity.Product{}}},
		{name: "repository error", args: args{[]uuid.UUID{item.GUID}}, want: want{err: repoErr}},
	}
	for _, tc := range cases {
		tc.prepare = func(a args, w want) {
			s.productRepo.EXPECT().GetByGUIDs(s.ctx, a.guids).Return(w.products, w.err).Once()
		}
		s.Run(tc.name, func() {
			tc.prepare(tc.args, tc.want)
			got, err := s.srv.GetByGUIDs(s.ctx, tc.args.guids)
			s.ErrorIs(err, tc.want.err)
			s.Equal(tc.want.products, got)
		})
	}
}

type deleteProductSuite struct{ productSuite }

func TestDeleteProductSuite(t *testing.T) { suite.Run(t, new(deleteProductSuite)) }

func (s *deleteProductSuite) TestDelete() {
	type args struct{ guid uuid.UUID }
	type want struct{ err error }
	item := productFixture()
	repoErr := errors.New("delete failed")
	cases := []struct {
		name    string
		args    args
		want    want
		prepare func(uuid.UUID)
	}{
		{name: "success", args: args{item.GUID}, prepare: func(guid uuid.UUID) {
			tx := s.expectTransaction()
			get := s.productRepo.EXPECT().GetByGUIDs(s.txCtx, []uuid.UUID{guid}).Return([]entity.Product{item}, nil).Once()
			del := s.productRepo.EXPECT().Delete(s.txCtx, guid).Return(nil).Once()
			mock.InOrder(tx, get, del)
		}},
		{name: "not found", args: args{item.GUID}, want: want{entity.ErrNotFound}, prepare: func(guid uuid.UUID) {
			tx := s.expectTransaction()
			get := s.productRepo.EXPECT().GetByGUIDs(s.txCtx, []uuid.UUID{guid}).Return(nil, nil).Once()
			mock.InOrder(tx, get)
		}},
		{name: "delete error", args: args{item.GUID}, want: want{repoErr}, prepare: func(guid uuid.UUID) {
			tx := s.expectTransaction()
			get := s.productRepo.EXPECT().GetByGUIDs(s.txCtx, []uuid.UUID{guid}).Return([]entity.Product{item}, nil).Once()
			del := s.productRepo.EXPECT().Delete(s.txCtx, guid).Return(repoErr).Once()
			mock.InOrder(tx, get, del)
		}},
		{name: "lookup error", args: args{item.GUID}, want: want{repoErr}, prepare: func(guid uuid.UUID) {
			tx := s.expectTransaction()
			get := s.productRepo.EXPECT().GetByGUIDs(s.txCtx, []uuid.UUID{guid}).Return(nil, repoErr).Once()
			mock.InOrder(tx, get)
		}},
		{name: "transaction error", args: args{item.GUID}, want: want{repoErr}, prepare: func(uuid.UUID) {
			s.productRepo.EXPECT().InsideTx(s.ctx, mock.Anything).Return(repoErr).Once()
		}},
	}
	for _, tc := range cases {
		s.Run(tc.name, func() {
			tc.prepare(tc.args.guid)
			s.ErrorIs(s.srv.Delete(s.ctx, tc.args.guid), tc.want.err)
		})
	}
}

type listProductSuite struct{ productSuite }

func TestListProductSuite(t *testing.T) { suite.Run(t, new(listProductSuite)) }

func (s *listProductSuite) TestList() {
	type args struct{ req entity.RequestProductList }
	type want struct {
		products []entity.Product
		err      error
	}
	item := productFixture()
	minPrice, maxPrice := int64(500), int64(1500)
	repoErr := errors.New("list failed")
	cases := []struct {
		name    string
		args    args
		want    want
		prepare func(args, want)
	}{
		{name: "success", args: args{entity.RequestProductList{CategoryGUID: &item.CategoryGUID, MinPrice: &minPrice, MaxPrice: &maxPrice}}, want: want{products: []entity.Product{item}}},
		{name: "without filters", want: want{products: []entity.Product{item}}},
		{name: "empty result", want: want{products: []entity.Product{}}},
		{name: "repository error", want: want{err: repoErr}},
	}
	for _, tc := range cases {
		tc.prepare = func(a args, w want) {
			s.productRepo.EXPECT().List(s.ctx, (*string)(nil), a.req.CategoryGUID, a.req.MinPrice, a.req.MaxPrice).Return(w.products, w.err).Once()
		}
		s.Run(tc.name, func() {
			tc.prepare(tc.args, tc.want)
			got, err := s.srv.List(s.ctx, tc.args.req)
			s.ErrorIs(err, tc.want.err)
			s.Equal(tc.want.products, got)
		})
	}
}

type updateProductSuite struct{ productSuite }

func TestUpdateProductSuite(t *testing.T) { suite.Run(t, new(updateProductSuite)) }

func (s *updateProductSuite) TestUpdate() {
	type args struct {
		guid uuid.UUID
		req  entity.RequestProductUpdate
	}
	type want struct {
		product entity.Product
		err     error
	}
	original := productFixture()
	changed := original
	changed.Name = "Updated product"
	changed.Price = 2000
	changed.Description = testutil.PtrString("Updated description")
	changed.CategoryGUID = uuid.FromStringOrNil("20000000-0000-4000-8000-000000000002")
	renamed := original
	renamed.Name = changed.Name
	repoErr := errors.New("update failed")
	full := args{original.GUID, entity.RequestProductUpdate{Name: changed.Name, Price: changed.Price, Description: changed.Description, CategoryGUID: changed.CategoryGUID}}
	nameOnly := args{original.GUID, entity.RequestProductUpdate{Name: changed.Name}}
	categoryOnly := args{original.GUID, entity.RequestProductUpdate{CategoryGUID: changed.CategoryGUID}}
	// Check the entity passed to persistence as well as the entity returned by the service.
	expectSave := func(expected entity.Product) *mock.Call {
		return s.productRepo.EXPECT().Update(s.txCtx, mock.MatchedBy(func(item entity.Product) bool {
			if !item.UpdatedAt.After(original.UpdatedAt) {
				return false
			}
			return item.GUID == expected.GUID && item.ID == expected.ID && item.Name == expected.Name &&
				item.Price == expected.Price && item.CategoryGUID == expected.CategoryGUID &&
				item.CreatedAt.Equal(expected.CreatedAt) &&
				((item.Description == nil && expected.Description == nil) ||
					(item.Description != nil && expected.Description != nil && *item.Description == *expected.Description))
		})).Return(nil).Once()
	}
	cases := []struct {
		name    string
		args    args
		want    want
		prepare func(args)
	}{
		{name: "full update", args: full, want: want{product: changed}, prepare: func(a args) {
			tx := s.expectTransaction()
			get := s.productRepo.EXPECT().GetByGUIDs(s.txCtx, []uuid.UUID{a.guid}).Return([]entity.Product{original}, nil).Once()
			list := s.productRepo.EXPECT().List(s.txCtx, &a.req.Name, (*uuid.UUID)(nil), (*int64)(nil), (*int64)(nil)).Return(nil, nil).Once()
			category := s.categoryRepo.EXPECT().GetByGUIDs(s.txCtx, []uuid.UUID{a.req.CategoryGUID}).Return([]entity.Category{{GUID: a.req.CategoryGUID}}, nil).Once()
			save := expectSave(changed)
			mock.InOrder(tx, get, list, category, save)
		}},
		{name: "partial update - name only", args: nameOnly, want: want{product: renamed}, prepare: func(a args) {
			tx := s.expectTransaction()
			get := s.productRepo.EXPECT().GetByGUIDs(s.txCtx, []uuid.UUID{a.guid}).Return([]entity.Product{original}, nil).Once()
			list := s.productRepo.EXPECT().List(s.txCtx, &a.req.Name, (*uuid.UUID)(nil), (*int64)(nil), (*int64)(nil)).Return(nil, nil).Once()
			save := expectSave(renamed)
			mock.InOrder(tx, get, list, save)
		}},
		{name: "omitted fields preserve values", args: args{guid: original.GUID}, want: want{product: original}, prepare: func(a args) {
			tx := s.expectTransaction()
			get := s.productRepo.EXPECT().GetByGUIDs(s.txCtx, []uuid.UUID{a.guid}).Return([]entity.Product{original}, nil).Once()
			save := expectSave(original)
			mock.InOrder(tx, get, save)
		}},
		{name: "same product name is allowed", args: args{original.GUID, entity.RequestProductUpdate{Name: original.Name}}, want: want{product: original}, prepare: func(a args) {
			tx := s.expectTransaction()
			get := s.productRepo.EXPECT().GetByGUIDs(s.txCtx, []uuid.UUID{a.guid}).Return([]entity.Product{original}, nil).Once()
			list := s.productRepo.EXPECT().List(s.txCtx, &a.req.Name, (*uuid.UUID)(nil), (*int64)(nil), (*int64)(nil)).Return([]entity.Product{original}, nil).Once()
			save := expectSave(original)
			mock.InOrder(tx, get, list, save)
		}},
		{name: "not found", args: full, want: want{err: entity.ErrNotFound}, prepare: func(a args) {
			tx := s.expectTransaction()
			get := s.productRepo.EXPECT().GetByGUIDs(s.txCtx, []uuid.UUID{a.guid}).Return(nil, nil).Once()
			mock.InOrder(tx, get)
		}},
		{name: "duplicate name", args: nameOnly, want: want{err: entity.ErrAlreadyExists}, prepare: func(a args) {
			tx := s.expectTransaction()
			get := s.productRepo.EXPECT().GetByGUIDs(s.txCtx, []uuid.UUID{a.guid}).Return([]entity.Product{original}, nil).Once()
			other := changed
			other.GUID = uuid.FromStringOrNil("10000000-0000-4000-8000-000000000002")
			list := s.productRepo.EXPECT().List(s.txCtx, &a.req.Name, (*uuid.UUID)(nil), (*int64)(nil), (*int64)(nil)).Return([]entity.Product{other}, nil).Once()
			mock.InOrder(tx, get, list)
		}},
		{name: "category not found", args: categoryOnly, want: want{err: entity.ErrNotFound}, prepare: func(a args) {
			tx := s.expectTransaction()
			get := s.productRepo.EXPECT().GetByGUIDs(s.txCtx, []uuid.UUID{a.guid}).Return([]entity.Product{original}, nil).Once()
			category := s.categoryRepo.EXPECT().GetByGUIDs(s.txCtx, []uuid.UUID{a.req.CategoryGUID}).Return(nil, nil).Once()
			mock.InOrder(tx, get, category)
		}},
		{name: "lookup error", args: full, want: want{err: repoErr}, prepare: func(a args) {
			tx := s.expectTransaction()
			get := s.productRepo.EXPECT().GetByGUIDs(s.txCtx, []uuid.UUID{a.guid}).Return(nil, repoErr).Once()
			mock.InOrder(tx, get)
		}},
		{name: "list error", args: nameOnly, want: want{err: repoErr}, prepare: func(a args) {
			tx := s.expectTransaction()
			get := s.productRepo.EXPECT().GetByGUIDs(s.txCtx, []uuid.UUID{a.guid}).Return([]entity.Product{original}, nil).Once()
			list := s.productRepo.EXPECT().List(s.txCtx, &a.req.Name, (*uuid.UUID)(nil), (*int64)(nil), (*int64)(nil)).Return(nil, repoErr).Once()
			mock.InOrder(tx, get, list)
		}},
		{name: "category lookup error", args: categoryOnly, want: want{err: repoErr}, prepare: func(a args) {
			tx := s.expectTransaction()
			get := s.productRepo.EXPECT().GetByGUIDs(s.txCtx, []uuid.UUID{a.guid}).Return([]entity.Product{original}, nil).Once()
			category := s.categoryRepo.EXPECT().GetByGUIDs(s.txCtx, []uuid.UUID{a.req.CategoryGUID}).Return(nil, repoErr).Once()
			mock.InOrder(tx, get, category)
		}},
		{name: "update error", args: args{guid: original.GUID}, want: want{err: repoErr}, prepare: func(a args) {
			tx := s.expectTransaction()
			get := s.productRepo.EXPECT().GetByGUIDs(s.txCtx, []uuid.UUID{a.guid}).Return([]entity.Product{original}, nil).Once()
			save := s.productRepo.EXPECT().Update(s.txCtx, mock.Anything).Return(repoErr).Once()
			mock.InOrder(tx, get, save)
		}},
		{name: "transaction error", args: full, want: want{err: repoErr}, prepare: func(args) {
			s.productRepo.EXPECT().InsideTx(s.ctx, mock.Anything).Return(repoErr).Once()
		}},
		{name: "commit error", args: args{guid: original.GUID}, want: want{err: repoErr}, prepare: func(a args) {
			tx := s.productRepo.EXPECT().InsideTx(s.ctx, mock.Anything).RunAndReturn(func(_ context.Context, fn func(context.Context) error) error {
				s.Require().NoError(fn(s.txCtx))
				return repoErr
			}).Once()
			get := s.productRepo.EXPECT().GetByGUIDs(s.txCtx, []uuid.UUID{a.guid}).Return([]entity.Product{original}, nil).Once()
			save := expectSave(original)
			mock.InOrder(tx, get, save)
		}},
	}
	for _, tc := range cases {
		s.Run(tc.name, func() {
			tc.prepare(tc.args)
			before := time.Now()
			got, err := s.srv.Update(s.ctx, tc.args.guid, tc.args.req)
			if tc.want.err != nil {
				s.Require().ErrorIs(err, tc.want.err)
				s.Equal(entity.Product{}, got, "failed update must not return uncommitted changes")
				return
			}
			s.Require().NoError(err)
			s.False(got.UpdatedAt.Before(before))
			s.False(got.UpdatedAt.After(time.Now()))
			expected := tc.want.product
			expected.UpdatedAt = got.UpdatedAt
			s.Equal(expected, got)
		})
	}
}
