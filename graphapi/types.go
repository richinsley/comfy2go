package graphapi

import (
	"encoding/json"
)

type Pos struct {
	X float64
	Y float64
}

func (p *Pos) UnmarshalJSON(b []byte) error {
	var tmp []interface{}
	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}

	for i, v := range tmp {
		switch value := v.(type) {
		case float64:
			if i == 0 {
				p.X = value
			} else {
				p.Y = value
			}
		case int:
			if i == 0 {
				p.X = float64(value)
			} else {
				p.Y = float64(value)
			}
		}
	}

	return nil
}

func (p *Pos) MarshalJSON() ([]byte, error) {
	tmp := []float64{p.X, p.Y}
	return json.Marshal(tmp)
}

type Size struct {
	Width  float64
	Height float64
}

func (s *Size) UnmarshalJSON(b []byte) error {
	// First try to unmarshal as array
	var tmpArr []interface{}
	if err := json.Unmarshal(b, &tmpArr); err == nil && len(tmpArr) == 2 {
		for i, v := range tmpArr {
			switch value := v.(type) {
			case float64:
				if i == 0 {
					s.Width = value
				} else {
					s.Height = value
				}
			case int:
				if i == 0 {
					s.Width = float64(value)
				} else {
					s.Height = float64(value)
				}
			}
		}
		return nil
	}

	// If not array, try to unmarshal as map
	var tmpMap map[string]interface{}
	if err := json.Unmarshal(b, &tmpMap); err != nil {
		return err
	}

	for k, v := range tmpMap {
		switch value := v.(type) {
		case float64:
			if k == "0" {
				s.Width = value
			} else {
				s.Height = value
			}
		case int:
			if k == "0" {
				s.Width = float64(value)
			} else {
				s.Height = float64(value)
			}
		}
	}

	return nil
}

// func (s *Size) MarshalJSON() ([]byte, error) {
// 	tmp := map[string]float64{
// 		"0": s.Width,
// 		"1": s.Height,
// 	}
// 	return json.Marshal(tmp)
// }

// it seems the json code can have either an array of values, or a dictionary of values
// when marshaling, we'll always output as an array.
func (s *Size) MarshalJSON() ([]byte, error) {
	tmp := []float64{s.Width, s.Height}
	return json.Marshal(tmp)
}
