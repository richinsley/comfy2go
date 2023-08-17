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
}

func (l *Link) UnmarshalJSON(b []byte) error {
	var tmp []interface{}
	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}

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

func (l *Link) MarshalJSON() ([]byte, error) {
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
