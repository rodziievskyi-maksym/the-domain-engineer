package app

import (
	"context"
	"strings"

	"eats/backend/common"
	"eats/backend/common/shared"
)

type CustomerUUID struct {
	common.UUID
}

type Customer struct {
	CustomerUUID CustomerUUID
	Name         string
	Email        string
	Address      shared.Address
	PhoneNumber  string
}

type CustomerRepository interface {
	RegisterCustomer(ctx context.Context, customer Customer) error
}

func (s *Service) RegisterCustomer(ctx context.Context, customer Customer) error {
	var errDetails []common.ErrorDetails

	if customer.CustomerUUID.IsZero() {
		errDetails = append(errDetails, common.ErrorDetails{
			EntityType: "Customer",
			ErrorSlug:  "empty-customer-uuid",
		})
	}

	if strings.TrimSpace(customer.Name) == "" {
		errDetails = append(errDetails, common.ErrorDetails{
			EntityType: "Customer",
			EntityID:   customer.CustomerUUID.String(),
			ErrorSlug:  "empty-name",
		})
	}

	if strings.TrimSpace(customer.Email) == "" {
		errDetails = append(errDetails, common.ErrorDetails{
			EntityType: "Customer",
			EntityID:   customer.CustomerUUID.String(),
			ErrorSlug:  "empty-email",
		})
	}

	if customer.Address.IsZero() {
		errDetails = append(errDetails, common.ErrorDetails{
			EntityType: "Customer",
			EntityID:   customer.CustomerUUID.String(),
			ErrorSlug:  "customer-address-is-zero",
		})
	}

	if strings.TrimSpace(customer.PhoneNumber) == "" {
		errDetails = append(errDetails, common.ErrorDetails{
			EntityType: "Customer",
			EntityID:   customer.CustomerUUID.String(),
			ErrorSlug:  "empty-phone-number",
		})
	}

	if len(errDetails) > 0 {
		return common.NewInvalidInputError("invalid_customer_data", "Invalid customer data").WithDetails(errDetails)
	}

	return s.customerRepository.RegisterCustomer(ctx, customer)
}
