package model

import "errors"

type Platform struct {
	// 0: Arr
	Next int
	Dest string
}

type Line []Platform

type Position []bool

func (l Line) ToPosition() Position {
	pos := make([]bool, len(l)*2-1)

	// A dead simple heuristic.
	// For every station:
	// - If the arrival time = 0, the train is at the platform
	// - If the arrival time = 1, the train is about to enter the station
	//   (so, between this station and the previous one)
	//   - Any other value, check the previous station:
	// - If previous station's train arrival >= this station's train arrival, put a train between them

	for i := range l {
		if l[i].Next == -1 {
			continue
		} else if l[i].Next == 0 {
			pos[i*2] = true
		} else if i == 0 {
			continue // start of the line
		} else if l[i].Next == 1 {
			pos[i*2-1] = true
		} else if l[i-1].Next >= l[i].Next {
			pos[i*2-1] = true
		}
	}
	return pos
}

func (p Position) ToString() string {
	s := make([]byte, len(p))

	for i := range p {
		if p[i] {
			s[i] = '*'
		} else {
			s[i] = '_'
		}
	}

	return string(s)
}

func (p Position) Reverse() Position {
	rev := make(Position, len(p))

	for i := range rev {
		rev[i] = p[len(p)-1-i]
	}

	return rev
}

func (p Position) Copy() Position {
	pp := make(Position, len(p))
	copy(pp, p)
	return pp
}

func NewPositionFromString(s string) (Position, error) {
	p := make(Position, len(s))

	for i, c := range s {
		switch c {
		case '*':
			p[i] = true
		case '_':
		default:
			return nil, errors.New("unrecognized character")
		}
	}

	return p, nil
}
