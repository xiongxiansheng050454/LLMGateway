package postgres

import (
	"encoding/json"
	"errors"
	"testing"

	"LLMGateway/internal/domain"
	"LLMGateway/internal/store"
)

func TestPGRateLimitCRUDAndFilter(t *testing.T) {
	st := testStore(t)

	created, err := st.CreateRateLimit(domain.RateLimitInput{
		RuleName: strp("default user rpm"), TargetType: strp("user"), Metric: strp("rpm"), LimitValue: i64p(600), WindowSeconds: intp(60), Action: strp("reject"),
	})
	if err != nil {
		t.Fatalf("CreateRateLimit: %v", err)
	}
	if created["id"] != 1 || created["target_value"] != "*" || created["priority"] != 100 || created["enabled"] != true {
		t.Fatalf("unexpected rule: %+v", created)
	}

	if _, err := st.CreateRateLimit(domain.RateLimitInput{RuleName: strp("q"), TargetType: strp("model"), Metric: strp("tpd"), LimitValue: i64p(1000), WindowSeconds: intp(86400), Action: strp("queue"), Extras: json.RawMessage(`{"queue_timeout_seconds":30}`)}); err != nil {
		t.Fatalf("tpd queue rule rejected: %v", err)
	}
	if _, err := st.CreateRateLimit(domain.RateLimitInput{RuleName: strp("bad"), TargetType: strp("user"), Metric: strp("bogus"), LimitValue: i64p(1), WindowSeconds: intp(1), Action: strp("reject")}); !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("invalid metric err = %v, want ErrInvalid", err)
	}

	updated, err := st.UpdateRateLimit(1, domain.RateLimitInput{Enabled: boolp(false)})
	if err != nil {
		t.Fatalf("UpdateRateLimit: %v", err)
	}
	if updated["enabled"] != false || updated["rule_name"] != "default user rpm" {
		t.Fatalf("partial update lost fields: %+v", updated)
	}

	enabled := true
	enabledList, err := st.ListRateLimits(&enabled, 1, 20)
	if err != nil {
		t.Fatal(err)
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

func strp(value string) *string { return &value }
func intp(value int) *int       { return &value }
func i64p(value int64) *int64   { return &value }
func boolp(value bool) *bool    { return &value }
