// internal/service/product_service.go - Improved error handling
package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/dukerupert/freyja/internal/api"
	"github.com/dukerupert/freyja/internal/repo"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/lib/pq"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// Define custom error types for better error handling
var (
	ErrProductNotFound = errors.New("product not found")
	ErrInvalidInput    = errors.New("invalid input")
	ErrDatabaseError   = errors.New("database error")
)

type ProductService struct {
	queries *repo.Queries
}

func NewProductService(queries *repo.Queries) *ProductService {
	return &ProductService{
		queries: queries,
	}
}

// Convert database Product to API Product
func (s *ProductService) dbProductToAPI(dbProduct repo.Product) api.Product {
	apiProduct := api.Product{
		Id:               openapi_types.UUID(dbProduct.ID.Bytes),
		Title:            dbProduct.Title,
		Handle:           dbProduct.Handle,
		Status:           api.ProductStatus(dbProduct.Status),
		IsGiftcard:       dbProduct.IsGiftcard,
		Discountable:     dbProduct.Discountable,
		SubscriptionEnabled: dbProduct.SubscriptionEnabled,
		CreatedAt:        time.Time(dbProduct.CreatedAt.Time),
		UpdatedAt:        time.Time(dbProduct.UpdatedAt.Time),
	}

	// Handle nullable fields
	if dbProduct.Subtitle.Valid {
		apiProduct.Subtitle = &dbProduct.Subtitle.String
	}
	if dbProduct.Description.Valid {
		apiProduct.Description = &dbProduct.Description.String
	}
	if dbProduct.Thumbnail.Valid {
		apiProduct.Thumbnail = &dbProduct.Thumbnail.String
	}
	if dbProduct.OriginCountry.Valid {
		apiProduct.OriginCountry = &dbProduct.OriginCountry.String
	}
	if dbProduct.Region.Valid {
		apiProduct.Region = &dbProduct.Region.String
	}
	if dbProduct.Farm.Valid {
		apiProduct.Farm = &dbProduct.Farm.String
	}
	if dbProduct.AltitudeMin.Valid {
		altMin := int(dbProduct.AltitudeMin.Int32)
		apiProduct.AltitudeMin = &altMin
	}
	if dbProduct.AltitudeMax.Valid {
		altMax := int(dbProduct.AltitudeMax.Int32)
		apiProduct.AltitudeMax = &altMax
	}
	if dbProduct.ProcessingMethod.Valid {
		processingMethod := api.ProcessingMethod(dbProduct.ProcessingMethod.String)
		apiProduct.ProcessingMethod = &processingMethod
	}
	if dbProduct.RoastLevel.Valid {
		roastLevel := api.RoastLevel(dbProduct.RoastLevel.String)
		apiProduct.RoastLevel = &roastLevel
	}
	if len(dbProduct.FlavorNotes) > 0 {
		flavorNotes := make([]string, len(dbProduct.FlavorNotes))
		for i, note := range dbProduct.FlavorNotes {
			flavorNotes[i] = note
		}
		apiProduct.FlavorNotes = &flavorNotes
	}
	if dbProduct.Varietal.Valid {
		apiProduct.Varietal = &dbProduct.Varietal.String
	}
	if dbProduct.HarvestDate.Valid {
		harvestDate := openapi_types.Date{Time: dbProduct.HarvestDate.Time}
		apiProduct.HarvestDate = &harvestDate
	}
	if dbProduct.WeightGrams.Valid {
		weight := int(dbProduct.WeightGrams.Int32)
		apiProduct.WeightGrams = &weight
	}
	if dbProduct.ProductTypeID.Valid {
		productTypeId := openapi_types.UUID(dbProduct.ProductTypeID.Bytes)
		apiProduct.ProductTypeId = &productTypeId
	}
	if dbProduct.CollectionID.Valid {
		collectionId := openapi_types.UUID(dbProduct.CollectionID.Bytes)
		apiProduct.CollectionId = &collectionId
	}
	
	// Handle metadata (JSONB field) - With json.RawMessage
	if len(dbProduct.Metadata) > 0 {
		var metadata map[string]interface{}
		if err := json.Unmarshal(dbProduct.Metadata, &metadata); err == nil {
			apiProduct.Metadata = &metadata
		}
	}
	
	// Handle subscription fields
	if len(dbProduct.SubscriptionIntervals) > 0 {
		intervals := make([]api.SubscriptionInterval, len(dbProduct.SubscriptionIntervals))
		for i, interval := range dbProduct.SubscriptionIntervals {
			intervals[i] = api.SubscriptionInterval(interval)
		}
		apiProduct.SubscriptionIntervals = &intervals
	}
	if dbProduct.MinSubscriptionQuantity.Valid {
		minQty := int(dbProduct.MinSubscriptionQuantity.Int32)
		apiProduct.MinSubscriptionQuantity = &minQty
	}
	if dbProduct.MaxSubscriptionQuantity.Valid {
		maxQty := int(dbProduct.MaxSubscriptionQuantity.Int32)
		apiProduct.MaxSubscriptionQuantity = &maxQty
	}
	if dbProduct.SubscriptionDiscountPercentage.Valid {
		// Convert pgtype.Numeric to float64
		discount, _ := dbProduct.SubscriptionDiscountPercentage.Float64Value()
		apiProduct.SubscriptionDiscountPercentage = &discount.Float64
	}
	if dbProduct.SubscriptionPriority.Valid {
		priority := int(dbProduct.SubscriptionPriority.Int32)
		apiProduct.SubscriptionPriority = &priority
	}

	return apiProduct
}

// Convert API CreateProductRequest to database CreateProductParams
func (s *ProductService) apiCreateRequestToDBParams(req api.CreateProductRequest) repo.CreateProductParams {
	params := repo.CreateProductParams{
		Title:    req.Title,
		Handle:   req.Handle,
		Status:   "draft",
		IsGiftcard: false,
		Discountable: true,
		SubscriptionEnabled: false,
	}

	// Handle optional fields
	if req.Subtitle != nil {
		params.Subtitle = pgtype.Text{String: *req.Subtitle, Valid: true}
	}
	if req.Description != nil {
		params.Description = pgtype.Text{String: *req.Description, Valid: true}
	}
	if req.Thumbnail != nil {
		params.Thumbnail = pgtype.Text{String: *req.Thumbnail, Valid: true}
	}
	if req.Status != nil {
		params.Status = string(*req.Status)
	}
	if req.IsGiftcard != nil {
		params.IsGiftcard = *req.IsGiftcard
	}
	if req.Discountable != nil {
		params.Discountable = *req.Discountable
	}
	if req.OriginCountry != nil {
		params.OriginCountry = pgtype.Text{String: *req.OriginCountry, Valid: true}
	}
	if req.Region != nil {
		params.Region = pgtype.Text{String: *req.Region, Valid: true}
	}
	if req.Farm != nil {
		params.Farm = pgtype.Text{String: *req.Farm, Valid: true}
	}
	if req.AltitudeMin != nil {
		params.AltitudeMin = pgtype.Int4{Int32: int32(*req.AltitudeMin), Valid: true}
	}
	if req.AltitudeMax != nil {
		params.AltitudeMax = pgtype.Int4{Int32: int32(*req.AltitudeMax), Valid: true}
	}
	if req.ProcessingMethod != nil {
		params.ProcessingMethod = pgtype.Text{String: string(*req.ProcessingMethod), Valid: true}
	}
	if req.RoastLevel != nil {
		params.RoastLevel = pgtype.Text{String: string(*req.RoastLevel), Valid: true}
	}
	if req.FlavorNotes != nil && len(*req.FlavorNotes) > 0 {
		params.FlavorNotes = pq.StringArray(*req.FlavorNotes)
	}
	if req.Varietal != nil {
		params.Varietal = pgtype.Text{String: *req.Varietal, Valid: true}
	}
	if req.HarvestDate != nil {
		params.HarvestDate = pgtype.Date{Time: req.HarvestDate.Time, Valid: true}
	}
	if req.WeightGrams != nil {
		params.WeightGrams = pgtype.Int4{Int32: int32(*req.WeightGrams), Valid: true}
	}
	if req.ProductTypeId != nil {
		var uuidBytes [16]byte
		copy(uuidBytes[:], (*req.ProductTypeId)[:])
		params.ProductTypeID = pgtype.UUID{Bytes: uuidBytes, Valid: true}
	}
	if req.CollectionId != nil {
		var uuidBytes [16]byte
		copy(uuidBytes[:], (*req.CollectionId)[:])
		params.CollectionID = pgtype.UUID{Bytes: uuidBytes, Valid: true}
	}
	
	// Handle metadata (JSONB field) - With json.RawMessage
	if req.Metadata != nil {
		if metadataBytes, err := json.Marshal(*req.Metadata); err == nil {
			params.Metadata = json.RawMessage(metadataBytes)
		}
	}
	
	// Handle subscription fields
	if req.SubscriptionEnabled != nil {
		params.SubscriptionEnabled = *req.SubscriptionEnabled
	}
	if req.SubscriptionIntervals != nil && len(*req.SubscriptionIntervals) > 0 {
		intervals := make([]string, len(*req.SubscriptionIntervals))
		for i, interval := range *req.SubscriptionIntervals {
			intervals[i] = string(interval)
		}
		params.SubscriptionIntervals = pq.StringArray(intervals)
	}
	if req.MinSubscriptionQuantity != nil {
		params.MinSubscriptionQuantity = pgtype.Int4{Int32: int32(*req.MinSubscriptionQuantity), Valid: true}
	}
	if req.MaxSubscriptionQuantity != nil {
		params.MaxSubscriptionQuantity = pgtype.Int4{Int32: int32(*req.MaxSubscriptionQuantity), Valid: true}
	}
	if req.SubscriptionDiscountPercentage != nil {
		// Simple string conversion for numeric
		params.SubscriptionDiscountPercentage.Scan(fmt.Sprintf("%.2f", *req.SubscriptionDiscountPercentage))
	}
	if req.SubscriptionPriority != nil {
		params.SubscriptionPriority = pgtype.Int4{Int32: int32(*req.SubscriptionPriority), Valid: true}
	}

	return params
}

func (s *ProductService) CreateProduct(ctx context.Context, req api.CreateProductRequest) (*api.Product, error) {
	params := s.apiCreateRequestToDBParams(req)
	
	dbProduct, err := s.queries.CreateProduct(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create product: %w", err)
	}

	apiProduct := s.dbProductToAPI(dbProduct)
	return &apiProduct, nil
}

func (s *ProductService) GetProduct(ctx context.Context, id openapi_types.UUID) (*api.Product, error) {
	var uuidBytes [16]byte
	copy(uuidBytes[:], id[:])
	pgUUID := pgtype.UUID{Bytes: uuidBytes, Valid: true}

	dbProduct, err := s.queries.GetProduct(ctx, pgUUID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrProductNotFound
		}
		return nil, fmt.Errorf("failed to get product: %w", err)
	}

	apiProduct := s.dbProductToAPI(dbProduct)
	return &apiProduct, nil
}

func (s *ProductService) ListProducts(ctx context.Context, limit, offset int32) ([]api.Product, error) {
	params := repo.ListProductsParams{
		Limit:  limit,
		Offset: offset,
	}

	dbProducts, err := s.queries.ListProducts(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list products: %w", err)
	}

	apiProducts := make([]api.Product, len(dbProducts))
	for i, dbProduct := range dbProducts {
		apiProducts[i] = s.dbProductToAPI(dbProduct)
	}

	return apiProducts, nil
}

func (s *ProductService) SearchProducts(ctx context.Context, query string, limit, offset int32) ([]api.Product, error) {
	params := repo.SearchProductsParams{
		Column1: pgtype.Text{String: query, Valid: true},
		Limit:   limit,
		Offset:  offset,
	}

	dbProducts, err := s.queries.SearchProducts(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to search products: %w", err)
	}

	apiProducts := make([]api.Product, len(dbProducts))
	for i, dbProduct := range dbProducts {
		apiProducts[i] = s.dbProductToAPI(dbProduct)
	}

	return apiProducts, nil
}

func (s *ProductService) DeleteProduct(ctx context.Context, id openapi_types.UUID) error {
	var uuidBytes [16]byte
	copy(uuidBytes[:], id[:])
	pgUUID := pgtype.UUID{Bytes: uuidBytes, Valid: true}

	err := s.queries.SoftDeleteProduct(ctx, pgUUID)
	if err != nil {
		return fmt.Errorf("failed to delete product: %w", err)
	}

	return nil
}

func (s *ProductService) CountProducts(ctx context.Context) (int64, error) {
	return s.queries.CountProducts(ctx)
}