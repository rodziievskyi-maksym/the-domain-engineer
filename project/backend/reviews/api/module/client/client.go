package client

import (
	"context"
	"eats/backend/common"
)

type Reviews interface {
	CalculateRestaurantRating(ctx context.Context, req CalculateRestaurantRatingRequest) (CalculateRestaurantRating, error)
}

type CalculateRestaurantRatingRequest struct {
	RestaurantUUID common.UUID
}

type CalculateRestaurantRating struct {
	Rating float64
}
