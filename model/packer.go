package model

import (
	"fmt"

	"go.lepak.sg/mrtracker-backend/data/boards"
)

type PositionPacker func(map[string]Position) ([][]byte, error)

var _ PositionPacker = PackBoardV1

func PackBoardV1(ps map[string]Position) ([][]byte, error) {
	// TODO: refactor constants
	//out := make([][]byte, 3)
	//for i := range out {
	//	out[i] = make([]byte, 2*8) // 8 grids, 10 seg (2 bytes)
	//}

	var out [3][16]byte

	for name, specs := range boards.DevV1 {
		p, ok := ps[name]
		if !ok {
			return nil, fmt.Errorf("name not found in positions: %s", name)
		}

		if len(specs) != len(p) {
			return nil, fmt.Errorf("length mismatch: %s: len(specs)=%d len(p)=%d", name, len(specs), len(p))
		}

		for i, spec := range specs {
			if p[i] {
				// Turn on one bit in the output depending on where the spec says it should be
				byteOff := (spec.Grid-1)*2 + (spec.Seg-1)/8
				bitOff := (spec.Seg - 1) % 8
				out[spec.Chip][byteOff] |= 1 << bitOff
			}
		}
	}

	outSl := make([][]byte, 3)
	for i := range outSl {
		outSl[i] = out[i][:]
	}

	return outSl, nil
}
