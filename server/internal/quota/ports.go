package quota

import (
	"context"
)

type Port interface {
	ListQuotaPolicies(QuotaPolicyFilter) (ListResponse[QuotaPolicyDTO], error)
	CreateQuotaPolicy(QuotaPolicyInput) (QuotaPolicyDTO, error)
	UpdateQuotaPolicy(int, QuotaPolicyInput) (QuotaPolicyDTO, error)
	DeleteQuotaPolicy(int) error
	ListQuotaUsage(context.Context, QuotaPolicyFilter) (ListResponse[QuotaUsageDTO], error)
	ReserveQuota(context.Context, QuotaReserveInput) (QuotaReservation, error)
	ReleaseQuota(context.Context, int64) error
	ReapExpiredQuotaReservations(context.Context, int) (int, error)
}
