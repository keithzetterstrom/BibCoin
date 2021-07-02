package blockchain

type satoshies []int

type Satoshies interface {
	GetMaxIndex(s satoshies) int
	FindIndex(s satoshies, index int) bool
}

// GetMaxIndex returns max value of satoshies slice
func (satoshies) GetMaxIndex(s satoshies) int {
	max := 0
	for _, el := range s{
		if max < el {
			max = el
		}
	}
	return max
}

// FindIndex returns true if index exists in satoshies slice
func (satoshies) FindIndex(s satoshies, index int) bool {
	for _, el := range s{
		if index == el {
			return true
		}
	}
	return false
}
