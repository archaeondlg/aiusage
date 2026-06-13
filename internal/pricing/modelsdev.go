package pricing

import (
	"encoding/json"
)

// models.dev pricing integration.

type modelsDevJSON struct {
	ID    *string                  `json:"id"`
	Name  *string                  `json:"name"`
	Models map[string]modelsDevModel `json:"models"`
	// Flat format: model entries directly at root level.
	Cost  *modelsDevCost  `json:"cost"`
	Limit *modelsDevLimit `json:"limit"`
}

type modelsDevModel struct {
	ID    *string         `json:"id"`
	Name  *string         `json:"name"`
	Cost  *modelsDevCost  `json:"cost"`
	Limit *modelsDevLimit `json:"limit"`
}

type modelsDevCost struct {
	Input      *float64 `json:"input"`
	Output     *float64 `json:"output"`
	CacheRead  *float64 `json:"cache_read"`
	CacheWrite *float64 `json:"cache_write"`
}

type modelsDevLimit struct {
	Context *uint64 `json:"context"`
}

// loadModelsDevJSONMissing loads models.dev pricing for models not already in the map.
func (m *PricingMap) loadModelsDevJSONMissing(data []byte) int {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return 0
	}

	// Check format: provider-based (entries have "models" field) or flat (entries have "cost").
	firstHasModels := false
	for _, v := range raw {
		var entry modelsDevJSON
		if err := json.Unmarshal(v, &entry); err == nil && entry.Models != nil {
			firstHasModels = true
		}
		break
	}

	if firstHasModels {
		loaded := 0
		for _, v := range raw {
			var entry modelsDevJSON
			if err := json.Unmarshal(v, &entry); err != nil {
				continue
			}
			if entry.Models != nil {
				loaded += m.loadModelsDevModels(entry.Models)
			}
		}
		return loaded
	}

	// Flat format.
	return m.loadModelsDevModelsFlat(raw)
}

func (m *PricingMap) loadModelsDevModels(models map[string]modelsDevModel) int {
	loaded := 0
	for key, model := range models {
		modelID := key
		if model.ID != nil {
			modelID = *model.ID
		}
		if _, exists := m.entries[modelID]; exists {
			continue
		}
		if model.Cost == nil || model.Cost.Input == nil || model.Cost.Output == nil {
			continue
		}
		input := *model.Cost.Input / 1_000_000.0
		output := *model.Cost.Output / 1_000_000.0
		cacheReadExplicit := model.Cost.CacheRead != nil
		cacheCreate := input * 1.25
		if model.Cost.CacheWrite != nil {
			cacheCreate = *model.Cost.CacheWrite / 1_000_000.0
		}
		cacheRead := input * 0.1
		if model.Cost.CacheRead != nil {
			cacheRead = *model.Cost.CacheRead / 1_000_000.0
		}
		m.entries[modelID] = Pricing{
			Input:              input,
			Output:             output,
			CacheCreate:        cacheCreate,
			CacheRead:          cacheRead,
			CacheReadExplicit: cacheReadExplicit,
			FastMultiplier:    1.0,
		}
		if model.Limit != nil && model.Limit.Context != nil {
			m.contextLimits[modelID] = *model.Limit.Context
		}
		loaded++
	}
	return loaded
}

func (m *PricingMap) loadModelsDevModelsFlat(raw map[string]json.RawMessage) int {
	loaded := 0
	for key, v := range raw {
		var model modelsDevModel
		if err := json.Unmarshal(v, &model); err != nil {
			continue
		}
		modelID := key
		if model.ID != nil {
			modelID = *model.ID
		}
		if _, exists := m.entries[modelID]; exists {
			continue
		}
		if model.Cost == nil || model.Cost.Input == nil || model.Cost.Output == nil {
			continue
		}
		input := *model.Cost.Input / 1_000_000.0
		output := *model.Cost.Output / 1_000_000.0
		cacheReadExplicit := model.Cost.CacheRead != nil
		cacheCreate := input * 1.25
		if model.Cost.CacheWrite != nil {
			cacheCreate = *model.Cost.CacheWrite / 1_000_000.0
		}
		cacheRead := input * 0.1
		if model.Cost.CacheRead != nil {
			cacheRead = *model.Cost.CacheRead / 1_000_000.0
		}
		m.entries[modelID] = Pricing{
			Input:              input,
			Output:             output,
			CacheCreate:        cacheCreate,
			CacheRead:          cacheRead,
			CacheReadExplicit: cacheReadExplicit,
			FastMultiplier:    1.0,
		}
		if model.Limit != nil && model.Limit.Context != nil {
			m.contextLimits[modelID] = *model.Limit.Context
		}
		loaded++
	}
	return loaded
}

