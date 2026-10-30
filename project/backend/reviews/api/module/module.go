package module

import (
	"context"
	"eats/backend/reviews/api/module/client"
	"eats/backend/reviews/app"
)

type Reviews struct {
	service *app.Service
}

func New(service *app.Service) *Reviews {
	if service == nil {
		panic("service cannot be nil")
	}

	return &Reviews{service: service}
}

func (r Reviews) CalculateRestaurantRating(ctx context.Context, req client.CalculateRestaurantRatingRequest) (client.CalculateRestaurantRating, error) {
	rating, err := r.service.GetRestaurantRating(ctx, req.RestaurantUUID)
	if err != nil {
		return client.CalculateRestaurantRating{}, err
	}

	return client.CalculateRestaurantRating{
		Rating: rating,
	}, nil
}

//func (r Reviews) GetReviews(ctx context.Context, req client.GetReviewsRequest) (client.GetReviewsResponse, error) {
//	reviews, err := r.service.GetReviews(ctx, req.CustomerUUID, req.RestaurantUUID)
//	if err != nil {
//		return client.GetReviewsResponse{}, err
//	}
//
//	return client.GetReviewsResponse{
//		Reviews: clientReviewsFromAppReviews(reviews),
//	}, nil
//}
//
//func clientReviewsFromAppReviews(reviews []app.Review) []client.ReviewsResponse {
//	result := make([]client.ReviewsResponse, 0, len(reviews))
//	for _, reviewsResponse := range reviews {
//		result = append(result, client.ReviewsResponse{
//			ReviewUUID: reviewsResponse.ReviewUUID.UUID,
//			Comment:    reviewsResponse.Comment,
//			CreatedAt:  reviewsResponse.CreatedAt,
//		})
//	}
//
//	return result
//}
