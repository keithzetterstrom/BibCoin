package blockchain

import (
	"hash/fnv"
	"math"
)

type rangeBounds struct {
	b1, b2 int
}

func mapRange(x, y rangeBounds, n int) int {
	return y.b1 + (n - x.b1) * (y.b2 - y.b1) / (x.b2 - x.b1)
}

func hash(s []byte) uint32 {
	h := fnv.New32a()
	h.Write(s)
	return h.Sum32()
}

func GetStakeholderIndexByHash(blockHash []byte, lastIndex int) int {
	blockHash = append(blockHash, []byte(stakeholderConst)...)
	hash := hash(blockHash)

	r1 := rangeBounds{0, math.MaxUint32}
	r2 := rangeBounds{0, lastIndex}

	n := int(hash)

	return mapRange(r1, r2, n)
}
