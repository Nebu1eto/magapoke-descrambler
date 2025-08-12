package main

type XorShift32 struct {
	state uint32
}

func NewXorShift32(seed uint32) *XorShift32 {
	if seed == 0 {
		seed = 1
	}
	return &XorShift32{state: seed}
}

func (r *XorShift32) Next() uint32 {
	x := r.state
	x ^= x << 13
	x ^= x >> 17
	x ^= x << 5
	r.state = x
	return x
}
