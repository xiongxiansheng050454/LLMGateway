package httpapi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestChatCompletionSuccessUsesStoreSettlementPort(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "proxy", "openai_proxy.go"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(content)
	if !strings.Contains(source, "SettleChatCompletion") {
		t.Fatal("chat completion success path must call the store settlement port")
	}
	if strings.Contains(source, "DebitUserBalance") || strings.Contains(source, "UpdateChannelBalance") {
		t.Fatal("chat completion success path must not sequence debit/channel balance store calls directly")
	}
}
