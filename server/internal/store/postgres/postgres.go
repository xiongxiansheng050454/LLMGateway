package postgres

import (
	"errors"
	"fmt"
	"time"

	"LLMGateway/server/internal/crypto"
	"LLMGateway/server/internal/db/sqlc"
	"LLMGateway/server/internal/store"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store is the PostgreSQL-backed implementation of store.Store.
//
// SQL access uses sqlc-generated queries from db/queries. The cipher is
// required to encrypt upstream channel api keys into api_key_ciphertext and to
// decrypt them for GetChannelSecret; there is no plaintext fallback.
type Store struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
	cipher  *crypto.Cipher
	breaker store.ChannelBreakerConfig
	now     func() time.Time
}

var (
	_ store.Store              = (*Store)(nil)
	_ store.ChannelStore       = (*Store)(nil)
	_ store.ChannelHealthStore = (*Store)(nil)
)

func New(pool *pgxpool.Pool, cipher *crypto.Cipher) *Store {
	return &Store{
		pool:    pool,
		queries: sqlc.New(pool),
		cipher:  cipher,
		breaker: store.DefaultChannelBreakerConfig(),
		now:     time.Now,
	}
}

// mapError converts database errors into the store error vocabulary:
// missing rows become ErrNotFound and constraint/format violations become
// ErrInvalid.
func mapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return store.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505", // unique_violation
			"23503", // foreign_key_violation
			"23502", // not_null_violation
			"22P02", // invalid_text_representation
			"22003", // numeric_value_out_of_range
			"22007", // invalid_datetime_format
			"22008": // datetime_field_overflow
			return fmt.Errorf("%w: %s", store.ErrInvalid, pgErr.Message)
		}
	}
	return err
}

// textValue renders a nullable text value (sqlc may infer interface{} after a
// COALESCE) as a string. A nil/empty value means "unset".
func textValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case []byte:
		return string(typed)
	default:
		return fmt.Sprint(typed)
	}
}

func textOrEmpty(value pgtype.Text) string {
	if value.Valid {
		return value.String
	}
	return ""
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func optionalTimestamp(value pgtype.Timestamptz) *string {
	if !value.Valid {
		return nil
	}
	formatted := value.Time.UTC().Format(time.RFC3339)
	return &formatted
}

func timestampValue(value string) pgtype.Timestamptz {
	if value == "" {
		return pgtype.Timestamptz{}
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: parsed, Valid: true}
}

func optionalInt(value pgtype.Int8) *int {
	if !value.Valid {
		return nil
	}
	converted := int(value.Int64)
	return &converted
}

func int8Value(value *int) pgtype.Int8 {
	if value == nil {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: int64(*value), Valid: true}
}

func channelDTO(id int64, name, baseURL, authType string, status, weight, priority int32, balance string, modelCount int32) map[string]any {
	return map[string]any{
		"id":          int(id),
		"name":        name,
		"base_url":    baseURL,
		"auth_type":   authType,
		"status":      int(status),
		"weight":      int(weight),
		"priority":    int(priority),
		"balance":     optionalString(balance),
		"model_count": int(modelCount),
	}
}

func pricingDTO(id, channelID int64, channelName, modelName, upstreamModel, inputPrice, outputPrice, cachedPrice, currency string) map[string]any {
	return map[string]any{
		"id":                        int(id),
		"channel_id":                int(channelID),
		"channel_name":              channelName,
		"model_name":                modelName,
		"upstream_model":            upstreamModel,
		"input_price_per_1m":        inputPrice,
		"output_price_per_1m":       outputPrice,
		"cached_input_price_per_1m": cachedPrice,
		"currency":                  currency,
	}
}
