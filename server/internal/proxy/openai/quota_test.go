package openai

import "testing"

func TestEstimateRequestUsageUsesTokenizerAndMaxTokenPriority(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		defaultMax int
		wantOutput int
	}{
		{"max completion tokens", `{"model":"gpt","messages":[{"role":"user","content":"hello"}],"max_tokens":20,"max_completion_tokens":30}`, 40, 30},
		{"legacy max tokens", `{"model":"gpt","messages":[{"role":"user","content":"hello"}],"max_tokens":20}`, 40, 20},
		{"configured default", `{"model":"gpt","messages":[{"role":"user","content":"hello"}]}`, 40, 40},
		{"multiple choices", `{"model":"gpt","messages":[{"role":"user","content":"hello"}],"max_tokens":20,"n":3}`, 40, 60},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			estimate, err := estimateRequestUsage([]byte(tt.body), tt.defaultMax)
			if err != nil {
				t.Fatal(err)
			}
			if estimate.InputTokens <= 0 {
				t.Fatalf("input tokens = %d, want tokenizer estimate > 0", estimate.InputTokens)
			}
			if estimate.OutputTokens != tt.wantOutput || estimate.TotalTokens != estimate.InputTokens+tt.wantOutput {
				t.Fatalf("estimate = %+v, want output %d", estimate, tt.wantOutput)
			}
		})
	}
}
