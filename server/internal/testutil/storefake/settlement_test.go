package storefake

import (
	"errors"
	"testing"

	"LLMGateway/server/internal/store"
	domain "LLMGateway/server/internal/testutil/testtypes"
)

func TestSettleChatCompletionDebitsUserChannelAndWritesUsage(t *testing.T) {
	st := New()
	if _, err := st.CreateUser(domain.UserInput{Nickname: "Alice"}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.RechargeUser(1, domain.RechargeInput{Amount: "10.000000"}); err != nil {
		t.Fatal(err)
	}
	channelBalance := "5.000000"
	channel, err := st.CreateChannel(domain.ChannelInput{Name: "OpenAI", BaseURL: "https://api.test", APIKey: "sk", Status: 1, Balance: &channelBalance})
	if err != nil {
		t.Fatal(err)
	}

	usageID, err := st.SettleChatCompletion(domain.ChatSettlementInput{
		UserID:       1,
		ChannelID:    &channel.ID,
		Cost:         "1.250000",
		DebitChannel: true,
		Description:  "chat completion req-1",
		UsageLog:     successUsageInput("req-1", 1, channel.ID),
	})
	if err != nil {
		t.Fatalf("SettleChatCompletion: %v", err)
	}
	if usageID == 0 {
		t.Fatal("usage id not returned")
	}
	balance, _ := st.GetUserBalance(1)
	if balance.AvailableBalance != "8.750000" {
		t.Fatalf("user balance = %s, want 8.750000", balance.AvailableBalance)
	}
	secret, _ := st.GetChannelSecret(channel.ID)
	if secret.Balance == nil || *secret.Balance != "3.750000" {
		t.Fatalf("channel balance = %v, want 3.750000", secret.Balance)
	}
	logs, _ := st.ListUsageLogs(domain.UsageLogFilter{Page: 1, PageSize: 20})
	if logs.Total != 1 || logs.List[0].RequestID != "req-1" || logs.List[0].Status != "success" {
		t.Fatalf("unexpected usage logs: %+v", logs)
	}
	txs, _ := st.ListBalanceTransactions(1, 1, 20)
	if txs.Total != 2 || txs.List[0].TxType != "consume" {
		t.Fatalf("consume transaction missing: %+v", txs)
	}
}

func TestSettleChatCompletionRejectsInsufficientBalanceWithoutSuccessLog(t *testing.T) {
	st := New()
	if _, err := st.CreateUser(domain.UserInput{Nickname: "Alice"}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.RechargeUser(1, domain.RechargeInput{Amount: "1.000000"}); err != nil {
		t.Fatal(err)
	}
	channel, _ := st.CreateChannel(domain.ChannelInput{Name: "OpenAI", BaseURL: "https://api.test", APIKey: "sk", Status: 1})

	_, err := st.SettleChatCompletion(domain.ChatSettlementInput{UserID: 1, ChannelID: &channel.ID, Cost: "2.000000", Description: "chat", UsageLog: successUsageInput("req-2", 1, channel.ID)})
	if !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
	balance, _ := st.GetUserBalance(1)
	if balance.AvailableBalance != "1.000000" {
		t.Fatalf("balance changed: %s", balance.AvailableBalance)
	}
	logs, _ := st.ListUsageLogs(domain.UsageLogFilter{Page: 1, PageSize: 20})
	if logs.Total != 0 {
		t.Fatalf("success usage written despite failed settlement: %+v", logs)
	}
}

func TestSettleChatCompletionCostZeroWritesUsageWithoutDebit(t *testing.T) {
	st := New()
	if _, err := st.CreateUser(domain.UserInput{Nickname: "Alice"}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.RechargeUser(1, domain.RechargeInput{Amount: "1.000000"}); err != nil {
		t.Fatal(err)
	}
	channel, _ := st.CreateChannel(domain.ChannelInput{Name: "OpenAI", BaseURL: "https://api.test", APIKey: "sk", Status: 1})

	if _, err := st.SettleChatCompletion(domain.ChatSettlementInput{UserID: 1, ChannelID: &channel.ID, Cost: "0.000000", Description: "free", UsageLog: successUsageInput("req-free", 1, channel.ID)}); err != nil {
		t.Fatalf("SettleChatCompletion: %v", err)
	}
	balance, _ := st.GetUserBalance(1)
	if balance.AvailableBalance != "1.000000" {
		t.Fatalf("zero-cost settlement changed balance: %s", balance.AvailableBalance)
	}
	txs, _ := st.ListBalanceTransactions(1, 1, 20)
	if txs.Total != 1 {
		t.Fatalf("zero-cost settlement created consume transaction: %+v", txs)
	}
	logs, _ := st.ListUsageLogs(domain.UsageLogFilter{Page: 1, PageSize: 20})
	if logs.Total != 1 {
		t.Fatalf("zero-cost settlement did not write usage log: %+v", logs)
	}
}

func TestSettleChatCompletionDuplicateUsageRollsBack(t *testing.T) {
	st := New()
	if _, err := st.CreateUser(domain.UserInput{Nickname: "Alice"}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.RechargeUser(1, domain.RechargeInput{Amount: "10.000000"}); err != nil {
		t.Fatal(err)
	}
	channelBalance := "5.000000"
	channel, _ := st.CreateChannel(domain.ChannelInput{Name: "OpenAI", BaseURL: "https://api.test", APIKey: "sk", Status: 1, Balance: &channelBalance})
	if _, err := st.InsertUsageLog(successUsageInput("dup", 1, channel.ID)); err != nil {
		t.Fatal(err)
	}

	_, err := st.SettleChatCompletion(domain.ChatSettlementInput{UserID: 1, ChannelID: &channel.ID, Cost: "1.000000", DebitChannel: true, Description: "chat", UsageLog: successUsageInput("dup", 1, channel.ID)})
	if !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
	balance, _ := st.GetUserBalance(1)
	if balance.AvailableBalance != "10.000000" {
		t.Fatalf("user balance not rolled back: %s", balance.AvailableBalance)
	}
	secret, _ := st.GetChannelSecret(channel.ID)
	if secret.Balance == nil || *secret.Balance != "5.000000" {
		t.Fatalf("channel balance not rolled back: %v", secret.Balance)
	}
}

func successUsageInput(requestID string, userID, channelID int) domain.UsageLogInput {
	return domain.UsageLogInput{RequestID: requestID, UserID: &userID, ChannelID: &channelID, Model: "gpt", UpstreamModel: "up-gpt", InputTokens: 100, OutputTokens: 50, TotalTokens: 150, UnitPriceInputPer1M: "0.10000000", UnitPriceOutputPer1M: "0.20000000", TotalCost: "1.250000", Status: "success"}
}
