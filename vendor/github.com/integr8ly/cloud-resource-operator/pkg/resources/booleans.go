package resources

// Boolean to float64
func Btof64(b bool) float64 {
	if b {
		return float64(1)
	}
	return float64(0)
}
