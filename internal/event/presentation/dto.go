package presentation

import "github.com/trvux/elc-go/internal/event/domain"

// createEventRequest is the public tracking-beacon payload.
type createEventRequest struct {
	Name       string  `json:"name"`
	EntityType *string `json:"entity_type"`
	EntityID   *string `json:"entity_id"`
	PagePath   *string `json:"page_path"`
	SessionID  *string `json:"session_id"`
}

type topViewedResponseItem struct {
	EntityID string `json:"entity_id"`
	Count    int    `json:"count"`
}

func toTopViewedResponse(rows []domain.EntityViewCount) []topViewedResponseItem {
	result := make([]topViewedResponseItem, len(rows))
	for i, row := range rows {
		result[i] = topViewedResponseItem{EntityID: row.EntityID, Count: row.Count}
	}
	return result
}
