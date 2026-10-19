package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/ecommerce/backend/internal/uploads/domain"
)

// Service validates and stores uploaded pictures.
type Service struct {
	validator domain.Validator
	store     domain.Store
}

// NewService creates an upload service.
func NewService(v domain.Validator, s domain.Store) *Service {
	return &Service{validator: v, store: s}
}

// Result is what the admin form needs after an upload.
type Result struct {
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	Width       int    `json:"width,omitempty"`
	Height      int    `json:"height,omitempty"`
	Size        int    `json:"size"`
}

// UploadImage checks the file and saves it under a random name. The name never comes from
// the customer's file name, so nothing can be overwritten or reach outside the folder.
func (s *Service) UploadImage(ctx context.Context, data []byte) (*Result, error) {
	info, err := s.validator.Check(data)
	if err != nil {
		return nil, err
	}
	name := uuid.New().String() + info.Ext
	if err := s.store.Save(ctx, name, data); err != nil {
		return nil, err
	}
	return &Result{URL: s.store.URL(name), ContentType: info.ContentType, Width: info.Width, Height: info.Height, Size: info.Size}, nil
}
