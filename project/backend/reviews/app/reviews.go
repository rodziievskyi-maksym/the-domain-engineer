package app

import (
	"context"
	"eats/backend/common"
	"time"
)

type ReviewUUID struct {
	common.UUID
}

type Review struct {
	ReviewUUID     ReviewUUID
	RestaurantUUID common.UUID
	CustomerUUID   common.UUID
	Comment        string
	Rating         float64
	CreatedAt      time.Time
}

func (s *Service) GetRestaurantRating(ctx context.Context, restaurantUUID common.UUID) (float64, error) {
	// Implementation for getting restaurant rating
	return 9.5, nil
}

func (s *Service) GetReviews(ctx context.Context, customerUUID, restaurantUUID common.UUID) ([]Review, error) {
	// Implementation for getting reviews
	return []Review{}, nil
}
