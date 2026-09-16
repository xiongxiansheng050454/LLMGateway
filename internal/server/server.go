package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type adminResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

type listResponse struct {
	List  []any `json:"list"`
	Total int   `json:"total"`
}

type app struct {
	store  *memoryStore
	client *http.Client
}

type memoryStore struct {
	mu            sync.Mutex
	nextChannelID int
	nextModelID   int
	nextPricingID int
	channels      map[int]*channel
	models        map[int]map[int]*channelModel
	pricing       map[string]*pricing
}

type channel struct {
	ID       int
	Name     string
	BaseURL  string
	APIKey   string
	AuthType string
	Status   int
	Weight   int
	Priority int
	Balance  *string
}

type channelModel struct {
	ID            int    `json:"id"`
	ModelName     string `json:"model_name"`
	UpstreamModel string `json:"upstream_model"`
	Enabled       bool   `json:"enabled"`
}

type pricing struct {
	ID                    int
	ChannelID             int
	ModelName             string
	InputPricePer1M       string
	OutputPricePer1M      string
	CachedInputPricePer1M string
	Currency              string
}

func NewHandler(dashboardDir string) http.Handler {
	a := &app{
		store:  newMemoryStore(),
		client: &http.Client{Timeout: 5 * time.Second},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthzHandler)
	mux.HandleFunc("/admin/", a.adminHandler)
	mux.HandleFunc("/admin", a.adminHandler)

	dashboard := http.FileServer(http.Dir(dashboardDir))
	mux.HandleFunc("/dashboard/index.html", dashboardIndexHandler(dashboardDir))
	mux.Handle("/dashboard/", http.StripPrefix("/dashboard", dashboard))
	mux.Handle("/", dashboard)
	return mux
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		nextChannelID: 1,
		nextModelID:   1,
		nextPricingID: 1,
		channels:      map[int]*channel{},
		models:        map[int]map[int]*channelModel{},
		pricing:       map[string]*pricing{},
	}
}

func dashboardIndexHandler(dashboardDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			writeMethodNotAllowed(w)
			return
		}

		file, err := os.Open(filepath.Join(dashboardDir, "index.html"))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer file.Close()

		info, err := file.Stat()
		if err != nil {
			http.NotFound(w, r)
			return
		}
		http.ServeContent(w, r, "index.html", info.ModTime(), file)
	}
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *app) adminHandler(w http.ResponseWriter, r *http.Request) {
	writeCORSHeaders(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	data, ok, errStatus, errMsg := a.adminData(r)
	if errStatus != 0 {
		writeAdminError(w, errStatus, errMsg)
		return
	}
	if !ok {
		writeAdminError(w, http.StatusNotFound, "not found")
		return
	}
	writeAdminOK(w, data)
}

func (a *app) adminData(r *http.Request) (any, bool, int, string) {
	if data, ok, status, msg := a.catalogData(r); ok || status != 0 {
		return data, ok, status, msg
	}
	if r.Method != http.MethodGet {
		return nil, false, http.StatusMethodNotAllowed, "method not allowed"
	}
	data, ok := dashboardStartupData(r)
	return data, ok, 0, ""
}

func dashboardStartupData(r *http.Request) (any, bool) {
	switch strings.TrimSuffix(r.URL.Path, "/") {
	case "/admin/stats/overview":
		return map[string]any{
			"request_count":     0,
			"success_count":     0,
			"error_count":       0,
			"total_tokens":      0,
			"total_cost":        "0.000000",
			"active_user_count": 0,
		}, true
	case "/admin/stats/daily", "/admin/channels", "/admin/usage-logs", "/admin/users", "/admin/rate-limits", "/admin/models":
		ParsePagination(r)
		return listResponse{List: []any{}, Total: 0}, true
	case "/admin/stats/channels":
		return map[string]any{"list": []any{}}, true
	default:
		return nil, false
	}
}

func (a *app) catalogData(r *http.Request) (any, bool, int, string) {
	path := strings.TrimSuffix(r.URL.Path, "/")
	parts := splitPath(path)
	if len(parts) < 2 || parts[0] != "admin" {
		return nil, false, 0, ""
	}

	if parts[1] == "channels" {
		return a.channelData(r, parts)
	}
	if parts[1] == "models" && len(parts) == 2 && r.Method == http.MethodGet {
		return a.listCatalogModels(r), true, 0, ""
	}
	if parts[1] == "pricing" && len(parts) == 2 {
		return a.pricingData(r)
	}
	return nil, false, 0, ""
}

func (a *app) channelData(r *http.Request, parts []string) (any, bool, int, string) {
	if len(parts) == 2 {
		switch r.Method {
		case http.MethodGet:
			return a.listChannels(r), true, 0, ""
		case http.MethodPost:
			return a.createChannel(r)
		}
	}

	if len(parts) < 3 {
		return nil, true, http.StatusMethodNotAllowed, "method not allowed"
	}
	channelID, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, true, http.StatusBadRequest, "invalid channel id"
	}
	if len(parts) == 3 {
		switch r.Method {
		case http.MethodPut:
			return a.updateChannel(r, channelID)
		case http.MethodDelete:
			return a.deleteChannel(channelID)
		}
	}
	if len(parts) == 4 && parts[3] == "status" && r.Method == http.MethodPut {
		return a.updateChannelStatus(r, channelID)
	}
	if len(parts) == 4 && parts[3] == "balance" && r.Method == http.MethodPut {
		return a.updateChannelBalance(r, channelID)
	}
	if len(parts) == 4 && parts[3] == "test" && r.Method == http.MethodPost {
		return a.testChannel(channelID)
	}
	if len(parts) == 4 && parts[3] == "remote-models" && r.Method == http.MethodPost {
		return a.remoteModels(channelID)
	}
	if len(parts) >= 4 && parts[3] == "models" {
		if len(parts) == 4 {
			switch r.Method {
			case http.MethodGet:
				return a.listChannelModels(channelID), true, 0, ""
			case http.MethodPost:
				return a.createChannelModel(r, channelID)
			}
		}
		if len(parts) == 5 {
			modelID, err := strconv.Atoi(parts[4])
			if err != nil {
				return nil, true, http.StatusBadRequest, "invalid model id"
			}
			switch r.Method {
			case http.MethodPut:
				return a.updateChannelModel(r, channelID, modelID)
			case http.MethodDelete:
				return a.deleteChannelModel(channelID, modelID)
			}
		}
	}
	return nil, true, http.StatusMethodNotAllowed, "method not allowed"
}

func (a *app) listChannels(r *http.Request) any {
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	list := []any{}
	for _, ch := range a.store.channels {
		list = append(list, a.channelDTO(ch))
	}
	return listResponse{List: list, Total: len(list)}
}

func (a *app) createChannel(r *http.Request) (any, bool, int, string) {
	var req struct {
		Name     string  `json:"name"`
		BaseURL  string  `json:"base_url"`
		APIKey   string  `json:"api_key"`
		AuthType string  `json:"auth_type"`
		Status   int     `json:"status"`
		Weight   int     `json:"weight"`
		Priority int     `json:"priority"`
		Balance  *string `json:"balance"`
	}
	if err := readJSON(r, &req); err != nil {
		return nil, true, http.StatusBadRequest, "invalid json"
	}
	if strings.TrimSpace(req.APIKey) == "" {
		return nil, true, http.StatusBadRequest, "api_key is required"
	}
	if req.AuthType == "" {
		req.AuthType = "bearer"
	}
	if req.Weight == 0 {
		req.Weight = 100
	}
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	ch := &channel{ID: a.store.nextChannelID, Name: req.Name, BaseURL: req.BaseURL, APIKey: req.APIKey, AuthType: req.AuthType, Status: req.Status, Weight: req.Weight, Priority: req.Priority, Balance: cleanBalance(req.Balance)}
	a.store.nextChannelID++
	a.store.channels[ch.ID] = ch
	return a.channelDTO(ch), true, 0, ""
}

func (a *app) updateChannel(r *http.Request, id int) (any, bool, int, string) {
	var req struct {
		Name     string  `json:"name"`
		BaseURL  string  `json:"base_url"`
		APIKey   string  `json:"api_key"`
		AuthType string  `json:"auth_type"`
		Status   int     `json:"status"`
		Weight   int     `json:"weight"`
		Priority int     `json:"priority"`
		Balance  *string `json:"balance"`
	}
	if err := readJSON(r, &req); err != nil {
		return nil, true, http.StatusBadRequest, "invalid json"
	}
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	ch, ok := a.store.channels[id]
	if !ok {
		return nil, true, http.StatusNotFound, "channel not found"
	}
	ch.Name, ch.BaseURL, ch.AuthType, ch.Status, ch.Weight, ch.Priority = req.Name, req.BaseURL, req.AuthType, req.Status, req.Weight, req.Priority
	if strings.TrimSpace(req.APIKey) != "" {
		ch.APIKey = req.APIKey
	}
	ch.Balance = cleanBalance(req.Balance)
	return a.channelDTO(ch), true, 0, ""
}

func (a *app) updateChannelStatus(r *http.Request, id int) (any, bool, int, string) {
	var req struct {
		Status int `json:"status"`
	}
	if err := readJSON(r, &req); err != nil {
		return nil, true, http.StatusBadRequest, "invalid json"
	}
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	ch, ok := a.store.channels[id]
	if !ok {
		return nil, true, http.StatusNotFound, "channel not found"
	}
	ch.Status = req.Status
	return a.channelDTO(ch), true, 0, ""
}

func (a *app) updateChannelBalance(r *http.Request, id int) (any, bool, int, string) {
	var req struct {
		Balance string `json:"balance"`
		Delta   string `json:"delta"`
	}
	if err := readJSON(r, &req); err != nil {
		return nil, true, http.StatusBadRequest, "invalid json"
	}
	if req.Balance == "" && req.Delta == "" {
		return nil, true, http.StatusBadRequest, "balance or delta is required"
	}
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	ch, ok := a.store.channels[id]
	if !ok {
		return nil, true, http.StatusNotFound, "channel not found"
	}
	base := 0.0
	if req.Balance != "" {
		base, _ = strconv.ParseFloat(req.Balance, 64)
	} else if ch.Balance != nil {
		base, _ = strconv.ParseFloat(*ch.Balance, 64)
	}
	if req.Delta != "" {
		d, _ := strconv.ParseFloat(req.Delta, 64)
		base += d
	}
	b := fmt.Sprintf("%.6f", base)
	ch.Balance = &b
	return a.channelDTO(ch), true, 0, ""
}

func (a *app) deleteChannel(id int) (any, bool, int, string) {
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	delete(a.store.channels, id)
	delete(a.store.models, id)
	for key, p := range a.store.pricing {
		if p.ChannelID == id {
			delete(a.store.pricing, key)
		}
	}
	return map[string]any{"deleted": true}, true, 0, ""
}

func (a *app) testChannel(id int) (any, bool, int, string) {
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	items := []any{}
	for _, m := range a.store.models[id] {
		items = append(items, map[string]any{"model_alias": m.ModelName, "upstream_model": m.UpstreamModel, "http_status": 0, "latency_ms": 0, "ok": false, "error": "not tested in MVP"})
	}
	return map[string]any{"list": items}, true, 0, ""
}

func (a *app) listChannelModels(channelID int) any {
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	list := []any{}
	for _, m := range a.store.models[channelID] {
		list = append(list, *m)
	}
	return listResponse{List: list, Total: len(list)}
}

func (a *app) createChannelModel(r *http.Request, channelID int) (any, bool, int, string) {
	var req channelModel
	if err := readJSON(r, &req); err != nil {
		return nil, true, http.StatusBadRequest, "invalid json"
	}
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	if _, ok := a.store.channels[channelID]; !ok {
		return nil, true, http.StatusNotFound, "channel not found"
	}
	req.ID = a.store.nextModelID
	a.store.nextModelID++
	if a.store.models[channelID] == nil {
		a.store.models[channelID] = map[int]*channelModel{}
	}
	m := req
	a.store.models[channelID][m.ID] = &m
	return m, true, 0, ""
}

func (a *app) updateChannelModel(r *http.Request, channelID, modelID int) (any, bool, int, string) {
	var req struct {
		UpstreamModel string `json:"upstream_model"`
		Enabled       bool   `json:"enabled"`
	}
	if err := readJSON(r, &req); err != nil {
		return nil, true, http.StatusBadRequest, "invalid json"
	}
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	m, ok := a.store.models[channelID][modelID]
	if !ok {
		return nil, true, http.StatusNotFound, "model mapping not found"
	}
	m.UpstreamModel, m.Enabled = req.UpstreamModel, req.Enabled
	return *m, true, 0, ""
}

func (a *app) deleteChannelModel(channelID, modelID int) (any, bool, int, string) {
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	delete(a.store.models[channelID], modelID)
	return map[string]any{"deleted": true}, true, 0, ""
}

func (a *app) listCatalogModels(r *http.Request) any {
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	byName := map[string]map[string]any{}
	for channelID, models := range a.store.models {
		ch := a.store.channels[channelID]
		if ch == nil {
			continue
		}
		for _, m := range models {
			if r.URL.Query().Get("status") == "1" && !m.Enabled {
				continue
			}
			entry := byName[m.ModelName]
			if entry == nil {
				entry = map[string]any{"model_name": m.ModelName, "status": 1, "channels": []any{}}
				byName[m.ModelName] = entry
			}
			entry["channels"] = append(entry["channels"].([]any), map[string]any{"channel_id": channelID, "channel_name": ch.Name, "upstream_model": m.UpstreamModel, "enabled": m.Enabled})
		}
	}
	list := []any{}
	for _, item := range byName {
		list = append(list, item)
	}
	return listResponse{List: list, Total: len(list)}
}

func (a *app) pricingData(r *http.Request) (any, bool, int, string) {
	switch r.Method {
	case http.MethodGet:
		return a.listPricing(), true, 0, ""
	case http.MethodPost:
		return a.upsertPricing(r)
	case http.MethodDelete:
		return a.deletePricing(r)
	default:
		return nil, true, http.StatusMethodNotAllowed, "method not allowed"
	}
}

func (a *app) listPricing() any {
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	list := []any{}
	for _, p := range a.store.pricing {
		list = append(list, a.pricingDTO(p))
	}
	return listResponse{List: list, Total: len(list)}
}

func (a *app) upsertPricing(r *http.Request) (any, bool, int, string) {
	var req struct {
		ChannelID             int    `json:"channel_id"`
		ModelName             string `json:"model_name"`
		InputPricePer1M       string `json:"input_price_per_1m"`
		OutputPricePer1M      string `json:"output_price_per_1m"`
		CachedInputPricePer1M string `json:"cached_input_price_per_1m"`
		Currency              string `json:"currency"`
	}
	if err := readJSON(r, &req); err != nil {
		return nil, true, http.StatusBadRequest, "invalid json"
	}
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	key := pricingKey(req.ChannelID, req.ModelName)
	p := a.store.pricing[key]
	if p == nil {
		p = &pricing{ID: a.store.nextPricingID, ChannelID: req.ChannelID, ModelName: req.ModelName}
		a.store.nextPricingID++
		a.store.pricing[key] = p
	}
	p.InputPricePer1M, p.OutputPricePer1M, p.CachedInputPricePer1M, p.Currency = req.InputPricePer1M, req.OutputPricePer1M, req.CachedInputPricePer1M, req.Currency
	if p.Currency == "" {
		p.Currency = "USD"
	}
	return a.pricingDTO(p), true, 0, ""
}

func (a *app) deletePricing(r *http.Request) (any, bool, int, string) {
	var req struct {
		ChannelID int    `json:"channel_id"`
		ModelName string `json:"model_name"`
	}
	if err := readJSON(r, &req); err != nil && err != io.EOF {
		return nil, true, http.StatusBadRequest, "invalid json"
	}
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	delete(a.store.pricing, pricingKey(req.ChannelID, req.ModelName))
	return map[string]any{"deleted": true}, true, 0, ""
}

func (a *app) remoteModels(channelID int) (any, bool, int, string) {
	a.store.mu.Lock()
	ch := a.store.channels[channelID]
	a.store.mu.Unlock()
	if ch == nil {
		return nil, true, http.StatusNotFound, "channel not found"
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

func (a *app) channelDTO(ch *channel) map[string]any {
	modelCount := len(a.store.models[ch.ID])
	return map[string]any{"id": ch.ID, "name": ch.Name, "base_url": ch.BaseURL, "auth_type": ch.AuthType, "status": ch.Status, "weight": ch.Weight, "priority": ch.Priority, "balance": ch.Balance, "model_count": modelCount}
}

func (a *app) pricingDTO(p *pricing) map[string]any {
	channelName, upstream := "", ""
	if ch := a.store.channels[p.ChannelID]; ch != nil {
		channelName = ch.Name
	}
	for _, m := range a.store.models[p.ChannelID] {
		if m.ModelName == p.ModelName {
			upstream = m.UpstreamModel
			break
		}
	}
	return map[string]any{"id": p.ID, "channel_id": p.ChannelID, "channel_name": channelName, "model_name": p.ModelName, "upstream_model": upstream, "input_price_per_1m": p.InputPricePer1M, "output_price_per_1m": p.OutputPricePer1M, "cached_input_price_per_1m": p.CachedInputPricePer1M, "currency": p.Currency}
}

func splitPath(path string) []string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 1 && parts[0] == "" {
		return nil
	}
	return parts
}
func readJSON(r *http.Request, v any) error { return json.NewDecoder(r.Body).Decode(v) }
func cleanBalance(balance *string) *string {
	if balance == nil || *balance == "" {
		return nil
	}
	return balance
}
func pricingKey(channelID int, model string) string { return fmt.Sprintf("%d:%s", channelID, model) }

func ParsePagination(r *http.Request) (int, int) {
	q := r.URL.Query()
	page := parsePositiveInt(q.Get("page"), 1)
	pageSize := parsePositiveInt(q.Get("page_size"), 20)
	return page, pageSize
}

func parsePositiveInt(value string, fallback int) int {
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func writeAdminOK(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, adminResponse{Code: 0, Message: "ok", Data: data})
}

func writeAdminError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, adminResponse{Code: status, Message: message, Data: map[string]any{}})
}

func writeMethodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
}

func writeCORSHeaders(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Vary", "Origin")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type")
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
