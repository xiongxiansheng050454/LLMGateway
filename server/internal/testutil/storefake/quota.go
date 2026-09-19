package storefake

import (
	"context"
	"fmt"
	"sort"
	"time"

	"LLMGateway/server/internal/money"
	domain "LLMGateway/server/internal/quota"
	"LLMGateway/server/internal/store"
	usage "LLMGateway/server/internal/usage"
)

type fakeQuotaBucket struct {
	policyID                   int
	start, end                 time.Time
	usedTokens, reservedTokens int64
	usedCost, reservedCost     money.Amount
}

type fakeQuotaReservation struct {
	id              int64
	requestID       string
	userID, keyID   int
	estimatedTokens int64
	estimatedCost   money.Amount
	status          string
	expiresAt       time.Time
	items           []string
}

func (s *Store) ListQuotaPolicies(filter domain.QuotaPolicyFilter) (domain.ListResponse[domain.QuotaPolicyDTO], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := []domain.QuotaPolicyDTO{}
	for _, policy := range s.quotaPolicies {
		if filter.ScopeType != "" && string(policy.ScopeType) != filter.ScopeType || filter.ScopeID > 0 && policy.ScopeID != filter.ScopeID || filter.Enabled != nil && policy.Enabled != *filter.Enabled {
			continue
		}
		list = append(list, domain.QuotaPolicyToDTO(*policy))
	}
	sort.Slice(list, func(i, j int) bool { return list[i].ID < list[j].ID })
	return domain.ListResponse[domain.QuotaPolicyDTO]{List: list, Total: len(list)}, nil
}

func (s *Store) CreateQuotaPolicy(in domain.QuotaPolicyInput) (domain.QuotaPolicyDTO, error) {
	policy, err := domain.NormalizeQuotaPolicy(in, nil)
	if err != nil {
		return domain.QuotaPolicyDTO{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, current := range s.quotaPolicies {
		if current.ScopeType == policy.ScopeType && current.ScopeID == policy.ScopeID && current.PeriodType == policy.PeriodType {
			return domain.QuotaPolicyDTO{}, fmt.Errorf("%w: duplicate quota policy", store.ErrInvalid)
		}
	}
	policy.ID = s.nextQuotaPolicyID
	s.nextQuotaPolicyID++
	s.quotaPolicies[policy.ID] = &policy
	return domain.QuotaPolicyToDTO(policy), nil
}

func (s *Store) UpdateQuotaPolicy(id int, in domain.QuotaPolicyInput) (domain.QuotaPolicyDTO, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing := s.quotaPolicies[id]
	if existing == nil {
		return domain.QuotaPolicyDTO{}, store.ErrNotFound
	}
	policy, err := domain.NormalizeQuotaPolicy(in, existing)
	if err != nil {
		return domain.QuotaPolicyDTO{}, err
	}
	policy.ID = id
	s.quotaPolicies[id] = &policy
	return domain.QuotaPolicyToDTO(policy), nil
}

func (s *Store) DeleteQuotaPolicy(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.quotaPolicies[id] == nil {
		return store.ErrNotFound
	}
	delete(s.quotaPolicies, id)
	return nil
}

func (s *Store) ListQuotaUsage(_ context.Context, filter domain.QuotaPolicyFilter) (domain.ListResponse[domain.QuotaUsageDTO], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := []domain.QuotaUsageDTO{}
	for _, bucket := range s.quotaBuckets {
		policy := s.quotaPolicies[bucket.policyID]
		if policy == nil || filter.ScopeType != "" && string(policy.ScopeType) != filter.ScopeType || filter.ScopeID > 0 && policy.ScopeID != filter.ScopeID {
			continue
		}
		list = append(list, domain.QuotaUsageDTO{PolicyID: policy.ID, PolicyName: policy.PolicyName, ScopeType: policy.ScopeType, ScopeID: policy.ScopeID, PeriodType: policy.PeriodType, PeriodStart: bucket.start.Format(time.RFC3339), PeriodEnd: bucket.end.Format(time.RFC3339), TokenLimit: policy.TokenLimit, UsedTokens: bucket.usedTokens, ReservedTokens: bucket.reservedTokens, CostLimit: policy.CostLimit, UsedCost: money.Format6(bucket.usedCost), ReservedCost: money.Format6(bucket.reservedCost)})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].PolicyID < list[j].PolicyID })
	return domain.ListResponse[domain.QuotaUsageDTO]{List: list, Total: len(list)}, nil
}

func (s *Store) ReserveQuota(_ context.Context, in domain.QuotaReserveInput) (domain.QuotaReservation, error) {
	cost, err := money.Parse6(in.EstimatedCost)
	if err != nil {
		return domain.QuotaReservation{}, store.ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reapQuotaLocked(16, in.UserID, in.APIKeyID)
	policies := []*domain.QuotaPolicy{}
	for _, policy := range s.quotaPolicies {
		if policy.Enabled && (policy.ScopeType == domain.QuotaScopeUser && policy.ScopeID == in.UserID || policy.ScopeType == domain.QuotaScopeAPIKey && policy.ScopeID == in.APIKeyID) {
			policies = append(policies, policy)
		}
	}
	if len(policies) == 0 {
		return domain.QuotaReservation{}, nil
	}
	sort.Slice(policies, func(i, j int) bool { return policies[i].ID < policies[j].ID })
	keys := []string{}
	for _, policy := range policies {
		start, end, _ := domain.QuotaPeriodBounds(s.now(), policy.PeriodType)
		key := fakeQuotaBucketKey(policy.ID, start)
		bucket := s.quotaBuckets[key]
		if bucket == nil {
			bucket = &fakeQuotaBucket{policyID: policy.ID, start: start, end: end}
			s.quotaBuckets[key] = bucket
		}
		if policy.TokenLimit != nil && bucket.usedTokens+bucket.reservedTokens+in.EstimatedTokens > *policy.TokenLimit {
			return domain.QuotaReservation{}, store.ErrQuotaExceeded
		}
		if policy.CostLimit != nil {
			limit, _ := money.Parse6(*policy.CostLimit)
			if bucket.usedCost.Add(bucket.reservedCost).Add(cost).Cmp(limit) > 0 {
				return domain.QuotaReservation{}, store.ErrQuotaExceeded
			}
		}
		keys = append(keys, key)
	}
	id := s.nextQuotaReservationID
	s.nextQuotaReservationID++
	for _, key := range keys {
		b := s.quotaBuckets[key]
		b.reservedTokens += in.EstimatedTokens
		b.reservedCost = b.reservedCost.Add(cost)
	}
	s.quotaReservations[id] = &fakeQuotaReservation{id: id, requestID: in.RequestID, userID: in.UserID, keyID: in.APIKeyID, estimatedTokens: in.EstimatedTokens, estimatedCost: cost, status: "pending", expiresAt: in.ExpiresAt, items: keys}
	return domain.QuotaReservation{ID: id, RequestID: in.RequestID, EstimatedTokens: in.EstimatedTokens, EstimatedCost: money.Format6(cost)}, nil
}

func (s *Store) ReleaseQuota(_ context.Context, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.releaseQuotaLocked(id, "released")
}
func (s *Store) ReapExpiredQuotaReservations(_ context.Context, limit int) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.reapQuotaLocked(limit, 0, 0), nil
}

func (s *Store) releaseQuotaLocked(id int64, status string) error {
	r := s.quotaReservations[id]
	if r == nil || r.status != "pending" {
		return nil
	}
	for _, key := range r.items {
		b := s.quotaBuckets[key]
		b.reservedTokens -= r.estimatedTokens
		b.reservedCost = b.reservedCost.Sub(r.estimatedCost)
	}
	r.status = status
	return nil
}
func (s *Store) reapQuotaLocked(limit, userID, keyID int) int {
	if limit <= 0 {
		limit = 100
	}
	count := 0
	for id, r := range s.quotaReservations {
		if count >= limit {
			break
		}
		if r.status == "pending" && !r.expiresAt.After(s.now()) && (userID == 0 || r.userID == userID || r.keyID == keyID) {
			_ = s.releaseQuotaLocked(id, "expired")
			count++
		}
	}
	return count
}
func fakeQuotaBucketKey(policyID int, start time.Time) string {
	return fmt.Sprintf("%d:%s", policyID, start.UTC().Format(time.RFC3339))
}

func (s *Store) settleQuotaLocked(in usage.ChatSettlementInput, actualTokens int64, actualCost money.Amount) error {
	id := in.ReservationID
	if id == 0 {
		return nil
	}
	r := s.quotaReservations[id]
	if r == nil {
		return store.ErrNotFound
	}
	if r.status == "settled" {
		return nil
	}
	if r.status != "pending" {
		return store.ErrInvalid
	}
	if r.requestID != in.UsageLog.RequestID || r.userID != in.UserID || r.keyID != in.APIKeyID {
		return store.ErrInvalid
	}
	if actualTokens > r.estimatedTokens || actualCost.Cmp(r.estimatedCost) > 0 {
		return store.ErrInvalid
	}
	for _, key := range r.items {
		b := s.quotaBuckets[key]
		b.reservedTokens -= r.estimatedTokens
		b.reservedCost = b.reservedCost.Sub(r.estimatedCost)
		b.usedTokens += actualTokens
		b.usedCost = b.usedCost.Add(actualCost)
	}
	r.status = "settled"
	return nil
}

func (s *Store) cleanupQuotaLocked(userID, keyID int) {
	for id, reservation := range s.quotaReservations {
		if reservation.userID == userID && (keyID == 0 || reservation.keyID == keyID) {
			_ = s.releaseQuotaLocked(id, "released")
			delete(s.quotaReservations, id)
		}
	}
}
