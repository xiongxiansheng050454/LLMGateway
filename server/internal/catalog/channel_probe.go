package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"LLMGateway/server/internal/domain"
	"LLMGateway/server/internal/httpcommon"
)

const channelTestTimeout = 10 * time.Second

func (a *Server) testChannel(r *http.Request, channelID int) (any, bool, int, string) {
	var request struct {
		CheckAll *bool `json:"check_all"`
	}
	if err := httpcommon.ReadJSON(r, &request); err != nil {
		return nil, true, http.StatusBadRequest, "invalid json"
	}
	checkAll := request.CheckAll == nil || *request.CheckAll
	channel, err := a.store.GetChannelSecret(channelID)
	if err != nil {
		return httpcommon.Result(nil, err)
	}
	models, err := a.store.ListChannelModels(channelID)
	if err != nil {
		return httpcommon.Result(nil, err)
	}
	sort.Slice(models.List, func(i, j int) bool { return models.List[i].ID < models.List[j].ID })
	items := []domain.ChannelTestItemDTO{}
	for _, model := range models.List {
		if !model.Enabled {
			continue
		}
		items = append(items, a.testModel(r.Context(), channel, model))
		if !checkAll {
			break
		}
	}
	return domain.ChannelTestResultDTO{List: items}, true, 0, ""
}

func (a *Server) testModel(parent context.Context, channel *domain.Channel, model domain.ChannelModel) domain.ChannelTestItemDTO {
	item := domain.ChannelTestItemDTO{ModelAlias: model.ModelName, UpstreamModel: model.UpstreamModel}
	body, err := json.Marshal(map[string]any{
		"model":      model.UpstreamModel,
		"messages":   []map[string]string{{"role": "user", "content": "hi"}},
		"max_tokens": 1,
	})
	if err != nil {
		item.Error = "unable to create test request"
		return item
	}
	ctx, cancel := context.WithTimeout(parent, a.testTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(channel.BaseURL, "/")+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		item.Error = "unable to create test request"
		return item
	}
	req.Header.Set("Content-Type", "application/json")
	if channel.AuthType == "bearer" && channel.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+channel.APIKey)
	}
	started := time.Now()
	res, err := a.client.Do(req)
	item.LatencyMs = int(time.Since(started).Milliseconds())
	if err != nil {
		if ctx.Err() != nil {
			item.Error = "upstream request timed out"
		} else {
			item.Error = "upstream request failed"
		}
		return item
	}
	defer res.Body.Close()
	item.HTTPStatus = res.StatusCode
	item.OK = res.StatusCode >= http.StatusOK && res.StatusCode < http.StatusMultipleChoices
	if !item.OK {
		item.Error = fmt.Sprintf("upstream status %d", res.StatusCode)
	}
	return item
}
