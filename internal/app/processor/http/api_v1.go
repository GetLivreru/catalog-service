package rprocessor

import (
	"net/http"

	"github.com/gorilla/mux"

	rhandler "github.com/GetLivreru/catalog-service/internal/app/handler/http"
)

func v1RegCategoryHandler(r *mux.Router, h rhandler.Category) {
	reg(r, http.MethodPost, "/category/create", http.HandlerFunc(h.Create))
	reg(r, http.MethodPatch, "/category/{guid}", http.HandlerFunc(h.Update))
	reg(r, http.MethodDelete, "/category/{guid}", http.HandlerFunc(h.Delete))
	reg(r, http.MethodPost, "/category/list", http.HandlerFunc(h.List))
}

func v1RegProductHandler(r *mux.Router, h rhandler.Product) {
	reg(r, http.MethodPost, "/product/create", http.HandlerFunc(h.Create))
	reg(r, http.MethodPatch, "/product/{guid}", http.HandlerFunc(h.Update))
	reg(r, http.MethodDelete, "/product/{guid}", http.HandlerFunc(h.Delete))
	reg(r, http.MethodPost, "/product/list", http.HandlerFunc(h.List))
}
