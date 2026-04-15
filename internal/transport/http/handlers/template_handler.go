package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	templatedomain "example.com/taskservice/internal/domain/template"
	templateusecase "example.com/taskservice/internal/usecase/template"
)

type TemplateHandler struct {
	usecase templateusecase.Usecase
}

func NewTemplateHandler(usecase templateusecase.Usecase) *TemplateHandler {
	return &TemplateHandler{usecase: usecase}
}

func (h *TemplateHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req templateMutationDTO
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	created, err := h.usecase.Create(r.Context(), req.toCreateInput())
	if err != nil {
		writeTemplateUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, newTemplateDTO(created))
}

func (h *TemplateHandler) List(w http.ResponseWriter, r *http.Request) {
	templates, err := h.usecase.List(r.Context())
	if err != nil {
		writeTemplateUsecaseError(w, err)
		return
	}

	response := make([]templateDTO, 0, len(templates))
	for i := range templates {
		response = append(response, newTemplateDTO(&templates[i]))
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *TemplateHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := getTemplateIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	tmpl, err := h.usecase.GetByID(r.Context(), id)
	if err != nil {
		writeTemplateUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newTemplateDTO(tmpl))
}

func (h *TemplateHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := getTemplateIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	var req templateMutationDTO
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	updated, err := h.usecase.Update(r.Context(), id, req.toUpdateInput())
	if err != nil {
		writeTemplateUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newTemplateDTO(updated))
}

func (h *TemplateHandler) Deactivate(w http.ResponseWriter, r *http.Request) {
	id, err := getTemplateIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if err := h.usecase.Deactivate(r.Context(), id); err != nil {
		writeTemplateUsecaseError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func getTemplateIDFromRequest(r *http.Request) (int64, error) {
	rawID := mux.Vars(r)["id"]
	if rawID == "" {
		return 0, errors.New("missing template id")
	}

	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		return 0, errors.New("invalid template id")
	}

	if id <= 0 {
		return 0, errors.New("invalid template id")
	}

	return id, nil
}

func writeTemplateUsecaseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, templatedomain.ErrNotFound):
		writeError(w, http.StatusNotFound, err)
	case errors.Is(err, templateusecase.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, err)
	default:
		writeError(w, http.StatusInternalServerError, err)
	}
}
