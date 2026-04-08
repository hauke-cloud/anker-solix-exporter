package anker

import (
	"fmt"
)

// parseFloat safely converts a string to float64
func parseFloat(s string) (float64, error) {
	if s == "" {
		return 0, nil
	}
	var val float64
	_, err := fmt.Sscanf(s, "%f", &val)
	return val, err
}
