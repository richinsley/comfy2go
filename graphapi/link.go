package graphapi

import (
	"encoding/json"
	"errors"
)

type Link struct {
	ID         int
	OriginID   int
	OriginSlot int
	TargetID   int
	TargetSlot int
	Type       string
	// Internal flag to track serialization format
	// true = object format (subgraph links), false = tuple format (top-level links)
	isObjectFormat bool
}

func (l *Link) UnmarshalJSON(b []byte) error {
	// Try to unmarshal as array (tuple format) first
	var tmp []interface{}
	if err := json.Unmarshal(b, &tmp); err == nil {
		if len(tmp) != 6 {
			return errors.New("wrong number of fields in JSON array")
		}

		l.ID = int(tmp[0].(float64))
		l.OriginID = int(tmp[1].(float64))
		l.OriginSlot = int(tmp[2].(float64))
		l.TargetID = int(tmp[3].(float64))
		l.TargetSlot = int(tmp[4].(float64))
		l.Type, _ = tmp[5].(string)

		return nil
	}

	// Try to unmarshal as object (subgraph format)
	var obj struct {
		ID         int    `json:"id"`
		OriginID   int    `json:"origin_id"`
		OriginSlot int    `json:"origin_slot"`
		TargetID   int    `json:"target_id"`
		TargetSlot int    `json:"target_slot"`
		Type       string `json:"type"`
	}

	if err := json.Unmarshal(b, &obj); err != nil {
		return err
	}

	l.ID = obj.ID
	l.OriginID = obj.OriginID
	l.OriginSlot = obj.OriginSlot
	l.TargetID = obj.TargetID
	l.TargetSlot = obj.TargetSlot
	l.Type = obj.Type
	l.isObjectFormat = true

	return nil
}

func (l *Link) MarshalJSON() ([]byte, error) {
	// Use object format if it was deserialized from object format
	if l.isObjectFormat {
		obj := struct {
			ID         int    `json:"id"`
			OriginID   int    `json:"origin_id"`
			OriginSlot int    `json:"origin_slot"`
			TargetID   int    `json:"target_id"`
			TargetSlot int    `json:"target_slot"`
			Type       string `json:"type"`
		}{
			ID:         l.ID,
			OriginID:   l.OriginID,
			OriginSlot: l.OriginSlot,
			TargetID:   l.TargetID,
			TargetSlot: l.TargetSlot,
			Type:       l.Type,
		}
		return json.Marshal(obj)
	}

	// Default to tuple format
	tmp := []interface{}{
		l.ID,
		l.OriginID,
		l.OriginSlot,
		l.TargetID,
		l.TargetSlot,
		l.Type,
	}

	return json.Marshal(tmp)
}
