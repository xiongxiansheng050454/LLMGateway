package httpcommon

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	apperrors "LLMGateway/server/internal/errors"
)

func TestAdminResultConstructors(t *testing.T) {
	data := map[string]string{"value": "ok"}
	got := Handled(data)
	if !got.Handled || got.Status != 0 || got.Message != "" || got.Data == nil {
		t.Fatalf("Handled result = %+v", got)
	}
	if got.Data.(map[string]string)["value"] != "ok" {
		t.Fatalf("Handled data = %#v", got.Data)
	}

	if got := Unhandled(); got.Handled || got.Status != 0 || got.Data != nil || got.Message != "" {
		t.Fatalf("Unhandled result = %+v", got)
	}

	got = HTTPError(http.StatusBadRequest, "invalid input")
	if !got.Handled || got.Status != http.StatusBadRequest || got.Message != "invalid input" || got.Data != nil {
		t.Fatalf("HTTPError result = %+v", got)
	}
}

func TestResultMapsErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantMsg    string
	}{
		{name: "success"},
		{name: "not found", err: apperrors.ErrNotFound, wantStatus: http.StatusNotFound, wantMsg: "not found"},
		{name: "invalid", err: fmt.Errorf("%w: bad value", apperrors.ErrInvalid), wantStatus: http.StatusBadRequest, wantMsg: "bad value"},
		{name: "unexpected", err: errors.New("database unavailable"), wantStatus: http.StatusInternalServerError, wantMsg: "database unavailable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Result("payload", tt.err)
			if !got.Handled || got.Status != tt.wantStatus || got.Message != tt.wantMsg {
				t.Fatalf("Result = %+v", got)
			}
			if tt.err == nil && got.Data != "payload" {
				t.Fatalf("Result data = %#v", got.Data)
			}
			if tt.err != nil && got.Data != nil {
				t.Fatalf("error result data = %#v", got.Data)
			}
		})
	}
}

func TestNoBody(t *testing.T) {
	got := NoBody(nil)
	if !got.Handled || got.Status != 0 || got.Message != "" || got.Data.(map[string]any)["deleted"] != true {
		t.Fatalf("NoBody success = %+v", got)
	}

	got = NoBody(apperrors.ErrNotFound)
	if !got.Handled || got.Status != http.StatusNotFound || got.Message != "not found" || got.Data != nil {
		t.Fatalf("NoBody error = %+v", got)
	}
}
