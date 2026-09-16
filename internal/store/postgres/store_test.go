package postgres

import (
	"errors"
	"testing"

	"LLMGateway/internal/domain"
	"LLMGateway/internal/store"
)

func TestNewImplementsStore(t *testing.T) {
	var st store.Store = New(nil)
	if st == nil {
		t.Fatal("New returned nil")
	}
}

func TestUnimplementedMethodsReturnErrNotImplemented(t *testing.T) {
	st := New(nil)

	if _, err := st.ListChannels(); !errors.Is(err, store.ErrNotImplemented) {
		t.Fatalf("ListChannels err = %v, want ErrNotImplemented", err)
	}
	if _, err := st.CreateChannel(domain.ChannelInput{}); !errors.Is(err, store.ErrNotImplemented) {
		t.Fatalf("CreateChannel err = %v, want ErrNotImplemented", err)
	}
	if _, err := st.UpdateChannel(1, domain.ChannelInput{}); !errors.Is(err, store.ErrNotImplemented) {
		t.Fatalf("UpdateChannel err = %v, want ErrNotImplemented", err)
	}
	if _, err := st.UpdateChannelStatus(1, 1); !errors.Is(err, store.ErrNotImplemented) {
		t.Fatalf("UpdateChannelStatus err = %v, want ErrNotImplemented", err)
	}
	if _, err := st.UpdateChannelBalance(1, "1.000000", ""); !errors.Is(err, store.ErrNotImplemented) {
		t.Fatalf("UpdateChannelBalance err = %v, want ErrNotImplemented", err)
	}
	if err := st.DeleteChannel(1); !errors.Is(err, store.ErrNotImplemented) {
		t.Fatalf("DeleteChannel err = %v, want ErrNotImplemented", err)
	}
	if _, err := st.GetChannelSecret(1); !errors.Is(err, store.ErrNotImplemented) {
		t.Fatalf("GetChannelSecret err = %v, want ErrNotImplemented", err)
	}
	if _, err := st.ListChannelModels(1); !errors.Is(err, store.ErrNotImplemented) {
		t.Fatalf("ListChannelModels err = %v, want ErrNotImplemented", err)
	}
	if _, err := st.CreateChannelModel(1, domain.ChannelModel{}); !errors.Is(err, store.ErrNotImplemented) {
		t.Fatalf("CreateChannelModel err = %v, want ErrNotImplemented", err)
	}
	if _, err := st.UpdateChannelModel(1, 1, "m", true); !errors.Is(err, store.ErrNotImplemented) {
		t.Fatalf("UpdateChannelModel err = %v, want ErrNotImplemented", err)
	}
	if err := st.DeleteChannelModel(1, 1); !errors.Is(err, store.ErrNotImplemented) {
		t.Fatalf("DeleteChannelModel err = %v, want ErrNotImplemented", err)
	}
	if _, err := st.ListCatalogModels(true); !errors.Is(err, store.ErrNotImplemented) {
		t.Fatalf("ListCatalogModels err = %v, want ErrNotImplemented", err)
	}
	if _, err := st.ListPricing(); !errors.Is(err, store.ErrNotImplemented) {
		t.Fatalf("ListPricing err = %v, want ErrNotImplemented", err)
	}
	if _, err := st.UpsertPricing(domain.PricingInput{}); !errors.Is(err, store.ErrNotImplemented) {
		t.Fatalf("UpsertPricing err = %v, want ErrNotImplemented", err)
	}
	if err := st.DeletePricing(domain.DeletePricingInput{}); !errors.Is(err, store.ErrNotImplemented) {
		t.Fatalf("DeletePricing err = %v, want ErrNotImplemented", err)
	}
	if _, err := st.TestChannel(1); !errors.Is(err, store.ErrNotImplemented) {
		t.Fatalf("TestChannel err = %v, want ErrNotImplemented", err)
	}
}
