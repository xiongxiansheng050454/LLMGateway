package store

import (
	"context"

	"LLMGateway/server/internal/domain"
)

type QuotaStore interface {
	ListQuotaPolicies(domain.QuotaPolicyFilter) (domain.ListResponse[domain.QuotaPolicyDTO], error)
	CreateQuotaPolicy(domain.QuotaPolicyInput) (domain.QuotaPolicyDTO, error)
	UpdateQuotaPolicy(int, domain.QuotaPolicyInput) (domain.QuotaPolicyDTO, error)
	DeleteQuotaPolicy(int) error
	ListQuotaUsage(context.Context, domain.QuotaPolicyFilter) (domain.ListResponse[domain.QuotaUsageDTO], error)
	ReserveQuota(context.Context, domain.QuotaReserveInput) (domain.QuotaReservation, error)
	ReleaseQuota(context.Context, int64) error
	ReapExpiredQuotaReservations(context.Context, int) (int, error)
}
