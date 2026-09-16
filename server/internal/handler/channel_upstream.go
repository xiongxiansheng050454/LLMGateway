package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func (a *Server) remoteModels(channelID int) (any, bool, int, string) {
	ch, err := a.store.GetChannelSecret(channelID)
	if err != nil {
		return errorResponse(err)
	}
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(ch.BaseURL, "/")+"/v1/models", nil)
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}, true, 0, ""
	}
	if ch.AuthType == "bearer" && ch.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+ch.APIKey)
	}
	res, err := a.client.Do(req)
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}, true, 0, ""
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return map[string]any{"ok": false, "error": fmt.Sprintf("upstream status %d", res.StatusCode)}, true, 0, ""
	}
	var body struct {
		Data   []map[string]any `json:"data"`
		Models []map[string]any `json:"models"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return map[string]any{"ok": false, "error": err.Error()}, true, 0, ""
	}
	models := body.Models
	if models == nil {
		models = body.Data
	}
	return map[string]any{"ok": true, "models": models}, true, 0, ""
}
