package hcategory

import (
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/gorilla/mux"

	"github.com/GetLivreru/catalog-service/internal/app/entity"
	rhandler "github.com/GetLivreru/catalog-service/internal/app/handler/http"
	"github.com/GetLivreru/catalog-service/internal/app/service"
	"github.com/GetLivreru/catalog-service/internal/pkg/http/binding"
	"github.com/GetLivreru/catalog-service/internal/pkg/http/httph"
)

type handler struct{ srv service.Category }

func NewHandler(srv service.Category) rhandler.Category { return &handler{srv: srv} }
func (h *handler) Create(w http.ResponseWriter, r *http.Request) {
	var req entity.RequestCategoryCreate
	if err := binding.ScanAndValidateJSON(r, &req); err != nil {
		httph.HandleError(w, err)
		return
	}
	item, err := h.srv.Create(r.Context(), req)
	if err != nil {
		httph.HandleError(w, err)
		return
	}
	resp := entity.ResponseCategoryCreate{GUID: item.GUID, Name: item.Name, CreatedAt: item.CreatedAt}
	httph.SendJSON(w, http.StatusCreated, resp)
}

func (h *handler) Update(w http.ResponseWriter, r *http.Request) {
	guid, err := uuid.FromString(mux.Vars(r)["guid"])
	if err != nil || guid == uuid.Nil {
		httph.HandleError(w, entity.ErrIncorrectParameters)
		return
	}
	var req entity.RequestCategoryUpdate
	if err := binding.ScanAndValidateJSON(r, &req); err != nil {
		httph.HandleError(w, err)
		return
	}
	item, err := h.srv.Update(r.Context(), guid, req)
	if err != nil {
		httph.HandleError(w, err)
		return
	}
	resp := entity.ResponseCategoryUpdate{GUID: item.GUID, Name: item.Name, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
	httph.SendJSON(w, http.StatusOK, resp)
}

func (h *handler) Delete(w http.ResponseWriter, r *http.Request) {
	guid, err := uuid.FromString(mux.Vars(r)["guid"])
	if err != nil || guid == uuid.Nil {
		httph.HandleError(w, entity.ErrIncorrectParameters)
		return
	}
	if err := h.srv.Delete(r.Context(), guid); err != nil {
		httph.HandleError(w, err)
		return
	}
	httph.SendEmpty(w, http.StatusOK)
}

func (h *handler) List(w http.ResponseWriter, r *http.Request) {
	item, err := h.srv.List(r.Context())
	if err != nil {
		httph.HandleError(w, err)
		return
	}
	resp := entity.ResponseCategoryList{Data: make([]entity.ResponseCategoryListItem, 0, len(item))}
	for _, item := range item {
		resp.Data = append(resp.Data, entity.ResponseCategoryListItem{GUID: item.GUID, Name: item.Name, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt})
	}
	httph.SendJSON(w, http.StatusOK, resp)
}
