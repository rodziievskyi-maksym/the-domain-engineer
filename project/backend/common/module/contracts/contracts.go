// Package contracts is the registry of module contracts. It's a separate package for historical reasons:
// Contracts could live one package up, in common/module. We keep it here for backward compatibility.
package contracts

import (
	"errors"

	deliveryModule "eats/backend/delivery/api/module/client"
	ordersModule "eats/backend/orders/api/module/client"
	reviewsModule "eats/backend/reviews/api/module/client"
)

type Contracts struct {
	ordersModule.Orders
	deliveryModule.Delivery
	reviewsModule.Reviews
}

func (c *Contracts) Verify() error {
	var err error

	if c.Orders == nil {
		err = errors.Join(err, errors.New("orders module contract is empty"))
	}
	if c.Delivery == nil {
		err = errors.Join(err, errors.New("delivery module contract is empty"))
	}
	if c.Reviews == nil {
		err = errors.Join(err, errors.New("reviews module contract is empty"))
	}

	return err
}
