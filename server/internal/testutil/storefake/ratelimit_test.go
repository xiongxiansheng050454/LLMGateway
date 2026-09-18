package storefake

import (
	"encoding/json"
	"errors"
	"testing"

	"LLMGateway/server/internal/domain"
	"LLMGateway/server/internal/store"
)

func strp(value string) *string { return &value }
func intp(value int) *int       { return &value }
func int64p(value int64) *int64 { return &value }
func boolp(value bool) *bool    { return &value }

func TestRateLimitCRUDAndFilter(t *testing.T) {
	st := New()

	created, err := st.CreateRateLimit(domain.RateLimitInput{
		RuleName:      strp("default user rpm"),
		TargetType:    strp("user"),
		Metric:        strp("rpm"),
		LimitValue:    int64p(600),
		WindowSeconds: intp(60),
		Action:        strp("reject"),
	})
	if err != nil {
		t.Fatalf("CreateRateLimit: %v", err)
	}
	if created.ID != 1 || created.TargetValue != "*" || created.Priority != 100 || created.Enabled != true {
		t.Fatalf("unexpected rule: %+v", created)
	}
	if created.Extras != nil && string(created.Extras) != "{}" {
		t.Fatalf("extras = %v, want {}", created.Extras)
	}

	if _, err := st.CreateRateLimit(domain.RateLimitInput{RuleName: strp("bad"), TargetType: strp("nope"), Metric: strp("rpm"), LimitValue: int64p(1), WindowSeconds: intp(1), Action: strp("reject")}); !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("invalid target_type err = %v, want ErrInvalid", err)
	}
	if _, err := st.CreateRateLimit(domain.RateLimitInput{RuleName: strp("bad"), TargetType: strp("user"), Metric: strp("bogus"), LimitValue: int64p(1), WindowSeconds: intp(1), Action: strp("reject")}); !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("invalid metric err = %v, want ErrInvalid", err)
	}
	if _, err := st.CreateRateLimit(domain.RateLimitInput{RuleName: strp("bad"), TargetType: strp("user"), Metric: strp("rpm"), LimitValue: int64p(0), WindowSeconds: intp(60), Action: strp("reject")}); !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("non-positive limit err = %v, want ErrInvalid", err)
	}

	// tpd must be accepted.
	if _, err := st.CreateRateLimit(domain.RateLimitInput{RuleName: strp("tpd"), TargetType: strp("model"), Metric: strp("tpd"), LimitValue: int64p(1000), WindowSeconds: intp(86400), Action: strp("queue"), Extras: json.RawMessage(`{"queue_timeout_seconds":30}`)}); err != nil {
		t.Fatalf("tpd rule rejected: %v", err)
	}

	// Partial update: only enabled.
	updated, err := st.UpdateRateLimit(1, domain.RateLimitInput{Enabled: boolp(false)})
	if err != nil {
		t.Fatalf("UpdateRateLimit: %v", err)
	}
	if updated.Enabled != false || updated.RuleName != "default user rpm" {
		t.Fatalf("partial update lost fields: %+v", updated)
	}

	enabled := true
	enabledList, err := st.ListRateLimits(&enabled, 1, 20)
	if err != nil {
		t.Fatalf("ListRateLimits: %v", err)
	}
	if enabledList.Total != 1 {
		t.Fatalf("enabled total = %d, want 1", enabledList.Total)
	}

	all, _ := st.ListRateLimits(nil, 1, 20)
	if all.Total != 2 {
		t.Fatalf("all total = %d, want 2", all.Total)
	}

	if err := st.DeleteRateLimit(1); err != nil {
		t.Fatalf("DeleteRateLimit: %v", err)
	}
	if err := st.DeleteRateLimit(1); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("second delete err = %v, want ErrNotFound", err)
	}
	if _, err := st.UpdateRateLimit(404, domain.RateLimitInput{Enabled: boolp(true)}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("update missing err = %v, want ErrNotFound", err)
	}
}
