package service

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"satellite-contact-window-deconfliction/backend/internal/dto"
	"satellite-contact-window-deconfliction/backend/internal/model"
	"satellite-contact-window-deconfliction/backend/internal/repository"
)

type SatelliteAssetService struct {
	repository *repository.SatelliteAssetRepository
	audit      *AuditService
}

func NewSatelliteAssetService(repository *repository.SatelliteAssetRepository, audit *AuditService) *SatelliteAssetService {
	return &SatelliteAssetService{repository: repository, audit: audit}
}

// WithContext returns a copy of the service whose repository and audit log are
// bound to the request context. When the client disconnects, in-flight queries
// are cancelled and open transactions roll back. Method signatures stay
// unchanged.
func (service *SatelliteAssetService) WithContext(ctx context.Context) *SatelliteAssetService {
	return &SatelliteAssetService{repository: service.repository.WithContext(ctx), audit: service.audit.WithContext(ctx)}
}

func (service *SatelliteAssetService) List(page, pageSize int, status, search string) ([]dto.SatelliteAssetResponse, dto.PageMeta, error) {
	assets, total, err := service.repository.List(page, pageSize, status, search)
	if err != nil {
		return nil, dto.PageMeta{}, Internal("could not list satellite assets", err)
	}
	responses := make([]dto.SatelliteAssetResponse, 0, len(assets))
	for _, asset := range assets {
		responses = append(responses, satelliteResponse(asset))
	}
	return responses, pageMeta(page, pageSize, total), nil
}

func (service *SatelliteAssetService) Get(id uint) (dto.SatelliteAssetResponse, error) {
	asset, err := service.repository.Get(id)
	if err != nil {
		return dto.SatelliteAssetResponse{}, MapRepositoryError("satellite asset", err)
	}
	return satelliteResponse(asset), nil
}

func (service *SatelliteAssetService) Create(request dto.CreateSatelliteAssetRequest, actor dto.Actor, requestID string) (dto.SatelliteAssetResponse, error) {
	asset := model.SatelliteAsset{
		SatelliteCode: strings.ToUpper(strings.TrimSpace(request.SatelliteCode)), Name: strings.TrimSpace(request.Name), OrbitClass: request.OrbitClass,
		SupportedBandsJSON: encodeStrings(uniqueStrings(request.SupportedBands)), PriorityWeight: request.PriorityWeight,
		MinimumContactSec: request.MinimumContactSec, AssetStatus: request.AssetStatus, Version: 1,
	}
	// Create the asset and record the audit event in one transaction so a
	// disconnect rolls both back instead of leaving an asset without its audit
	// trail.
	err := service.repository.DB().Transaction(func(tx *gorm.DB) error {
		txRepository := service.repository.WithDB(tx)
		if err := txRepository.Create(&asset); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "unique") {
				return Conflict("duplicate_satellite_code", "satellite_code already exists", err)
			}
			return Internal("could not create satellite asset", err)
		}
		return service.audit.RecordTx(tx, actor, requestID, "satellite.created", "satellite_asset", auditID(asset.ID), map[string]any{"satellite_code": asset.SatelliteCode}, nil, satelliteSummary(asset))
	})
	if err != nil {
		return dto.SatelliteAssetResponse{}, err
	}
	return satelliteResponse(asset), nil
}

func (service *SatelliteAssetService) Update(id uint, request dto.UpdateSatelliteAssetRequest, actor dto.Actor, requestID string) (dto.SatelliteAssetResponse, error) {
	before, err := service.repository.Get(id)
	if err != nil {
		return dto.SatelliteAssetResponse{}, MapRepositoryError("satellite asset", err)
	}
	values := map[string]any{
		"name": strings.TrimSpace(request.Name), "orbit_class": request.OrbitClass, "supported_bands_json": encodeStrings(uniqueStrings(request.SupportedBands)),
		"priority_weight": request.PriorityWeight, "minimum_contact_sec": request.MinimumContactSec, "asset_status": request.AssetStatus,
	}
	var after model.SatelliteAsset
	// Update the asset and record the audit event in one transaction so a
	// disconnect rolls the change back instead of committing it without a trace.
	err = service.repository.DB().Transaction(func(tx *gorm.DB) error {
		txRepository := service.repository.WithDB(tx)
		updated, err := txRepository.Update(id, request.ExpectedVersion, values)
		if err != nil {
			return Internal("could not update satellite asset", err)
		}
		if !updated {
			return Conflict("version_conflict", "satellite asset changed; reload before updating", nil)
		}
		loaded, err := txRepository.Get(id)
		if err != nil {
			return MapRepositoryError("satellite asset", err)
		}
		after = loaded
		return service.audit.RecordTx(tx, actor, requestID, "satellite.updated", "satellite_asset", auditID(id), map[string]any{"expected_version": request.ExpectedVersion}, satelliteSummary(before), satelliteSummary(after))
	})
	if err != nil {
		return dto.SatelliteAssetResponse{}, err
	}
	return satelliteResponse(after), nil
}

func satelliteResponse(asset model.SatelliteAsset) dto.SatelliteAssetResponse {
	return dto.SatelliteAssetResponse{
		ID: asset.ID, SatelliteCode: asset.SatelliteCode, Name: asset.Name, OrbitClass: asset.OrbitClass,
		SupportedBands: decodeStrings(asset.SupportedBandsJSON), PriorityWeight: asset.PriorityWeight, MinimumContactSec: asset.MinimumContactSec,
		AssetStatus: asset.AssetStatus, Version: asset.Version, UpdatedAt: asset.UpdatedAt,
	}
}

func satelliteSummary(asset model.SatelliteAsset) map[string]any {
	return map[string]any{"satellite_code": asset.SatelliteCode, "orbit_class": asset.OrbitClass, "bands": decodeStrings(asset.SupportedBandsJSON), "priority_weight": asset.PriorityWeight, "minimum_contact_sec": asset.MinimumContactSec, "status": asset.AssetStatus, "version": asset.Version}
}
