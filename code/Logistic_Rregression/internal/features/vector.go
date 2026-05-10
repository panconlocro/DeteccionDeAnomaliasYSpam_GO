package features

type Feature struct {
	Index int
	Value float64
}

type SparseVector []Feature

func AppendNonZero(v SparseVector, index int, value float64) SparseVector {
	if value == 0 {
		return v
	}
	return append(v, Feature{Index: index, Value: value})
}
