package store

import (
	"errors"

	"LLMGateway/internal/domain"
)

var (
	ErrNotFound       = errors.New("not found")
	ErrInvalid        = errors.New("invalid")
	ErrNotImplemented = errors.New("not implemented")
)

// Store is the persistence port used by the HTTP handler layer.
//
// Implementations must not expose sensitive values (upstream channel api_key,
// gateway key plaintext) through response DTOs.
type Store interface {
	ListChannels() (domain.ListResponse, error)
	CreateChannel(domain.ChannelInput) (map[string]any, error)
	UpdateChannel(int, domain.ChannelInput) (map[string]any, error)
	UpdateChannelStatus(int, int) (map[string]any, error)
	UpdateChannelBalance(int, string, string) (map[string]any, error)
	DeleteChannel(int) error
	GetChannelSecret(int) (*domain.Channel, error)
	ListChannelModels(int) (domain.ListResponse, error)
	CreateChannelModel(int, domain.ChannelModel) (domain.ChannelModel, error)
	UpdateChannelModel(int, int, string, bool) (domain.ChannelModel, error)
	DeleteChannelModel(int, int) error
	ListCatalogModels(bool) (domain.ListResponse, error)
	ListPricing() (domain.ListResponse, error)
	UpsertPricing(domain.PricingInput) (map[string]any, error)
	DeletePricing(domain.DeletePricingInput) error
	TestChannel(int) (map[string]any, error)
}
