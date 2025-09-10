package main

type KMisses struct {
	P []float64 // probability of exactly i bad events
}

func NewKMisses() (ret KMisses) {
	ret.P = make([]float64, 10)
	ret.P[0] = 1
	return ret
}

func (val *KMisses) Add(probGood float64) {
	if val.P[len(val.P)-1] > 1e-18 {
		val.P = append(val.P, 0)
		val.P = val.P[:cap(val.P)]
	}
	for i := len(val.P) - 1; i >= 1; i-- {
		val.P[i] = val.P[i]*probGood + val.P[i-1]*(1-probGood)
	}
	val.P[0] = val.P[0] * probGood
}

func (val KMisses) GetSum(n int) float64 {
	sum := 0.0
	for i := range min(n+1, len(val.P)) {
		sum += val.P[i]
	}
	if n+1 > len(val.P) {
		sum += val.P[len(val.P)-1] * float64(n+1-len(val.P))
	}
	return sum
}
