package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"satellite-contact-window-deconfliction/backend/internal/dto"
	"satellite-contact-window-deconfliction/backend/internal/service"
)

type GroundStationHandler struct{ service *service.GroundStationService }

func NewGroundStationHandler(service *service.GroundStationService) *GroundStationHandler {
	return &GroundStationHandler{service: service}
}

// serviceFor binds the service to the request context so that a client
// disconnect cancels in-flight queries and rolls back open transactions.
func (handler *GroundStationHandler) serviceFor(context *gin.Context) *service.GroundStationService {
	return handler.service.WithContext(context.Request.Context())
}

func (handler *GroundStationHandler) List(context *gin.Context) {
	page, pageSize := Pagination(context)
	stations, meta, err := handler.service.List(page, pageSize, context.Query("status"), context.Query("search"))
	if err != nil {
		WriteError(context, err)
		return
	}
	WritePage(context, stations, meta)
}

func (handler *GroundStationHandler) Get(context *gin.Context) {
	id, err := PathID(context)
	if err != nil {
		WriteError(context, err)
		return
	}
	station, err := handler.serviceFor(context).Get(id)
	if err != nil {
		WriteError(context, err)
		return
	}
	WriteData(context, http.StatusOK, station)
}

func (handler *GroundStationHandler) Create(context *gin.Context) {
	var request dto.CreateGroundStationRequest
	if err := BindAndValidate(context, &request); err != nil {
		WriteError(context, err)
		return
	}
	station, err := handler.serviceFor(context).Create(request, Actor(context), RequestID(context))
	if err != nil {
		WriteError(context, err)
		return
	}
	WriteData(context, http.StatusCreated, station)
}

func (handler *GroundStationHandler) Update(context *gin.Context) {
	id, err := PathID(context)
	if err != nil {
		WriteError(context, err)
		return
	}
	var request dto.UpdateGroundStationRequest
	if err := BindAndValidate(context, &request); err != nil {
		WriteError(context, err)
		return
	}
	station, err := handler.serviceFor(context).Update(id, request, Actor(context), RequestID(context))
	if err != nil {
		WriteError(context, err)
		return
	}
	WriteData(context, http.StatusOK, station)
}
