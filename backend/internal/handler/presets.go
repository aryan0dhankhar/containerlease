package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/aryan0dhankhar/containerlease/pkg/config"
)

// PresetsHandler returns available provisioning presets
type PresetsHandler struct {
	config *config.Config
	log    *slog.Logger
}

// NewPresetsHandler creates a new presets handler
func NewPresetsHandler(cfg *config.Config, log *slog.Logger) *PresetsHandler {
	return &PresetsHandler{config: cfg, log: log}
}

// ServeHTTP implements the HTTP handler for presets
func (h *PresetsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	type PresetResponse struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		CPUMilli    int    `json:"cpuMilli"`
		MemoryMB    int    `json:"memoryMB"`
		DurationMin int    `json:"durationMin"`
	}

	presets := make([]PresetResponse, 0)
	for id, preset := range h.config.Presets {
		presets = append(presets, PresetResponse{
			ID:          id,
			Name:        preset.Name,
			CPUMilli:    preset.CPUMilli,
			MemoryMB:    preset.MemoryMB,
			DurationMin: preset.DurationMin,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"presets": presets,
	})
}
