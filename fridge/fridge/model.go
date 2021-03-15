package fridge

import "fmt"

type FridgeModel struct {
	Temp        FridgeTemperature `json:"T"`
	DesiredTemp FridgeTemperature `json:"D"`
	IsDoorOpen  bool              `json:"O"`
}

type FridgeTemperature float64

func (temp FridgeTemperature) String() string {
	return fmt.Sprintf("%.0f", float64(temp))
}

func (temp FridgeTemperature) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%v", temp)), nil
}
