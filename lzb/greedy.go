package lzb

import (
	"errors"
	"io"
)

// greedyFinder is an OpFinder that implements a simple greedy algorithm
// to finding operations.
type greedyFinder struct{}

// Greedy provides a greedy operation finder.
var Greedy OpFinder

// don't want to expose the initialization of Greedy
func init() {
	Greedy = greedyFinder{}
}

// errNoMatch indicates that no match could be found
var errNoMatch = errors.New("no match found")

// bestMatch provides the longest match reachable over the list of
// provided offsets.
func bestMatch(d *hashDict, offsets []int64) (m match, err error) {
	off := int64(-1)
	length := 1
	for i := len(offsets) - 1; i >= 0; i-- {
		n := d.buf.equalBytes(d.head, offsets[i], MaxLength)
		if n >= length {
			off, length = offsets[i], n
		}
	}
	if off < 0 || length == 1 {
		err = errNoMatch
		return
	}
	return match{distance: d.head - off, n: length}, nil
}

// errEmptyBuf indicates an empty buffer.
var errEmptyBuf = errors.New("empty buffer")

// potentialOffsets returns a list of offset positions where a match to
// at the current dictionary head can be identified.
func potentialOffsets(d *hashDict, p []byte) []int64 {
	start := d.start()
	offs := make([]int64, 0, 32)
	// add potential offsets with highest priority at the top
	for i := 1; i < 11; i++ {
		// distance 1 to 10
		off := d.head - int64(i)
		if start <= off {
			offs = append(offs, off)
		}
	}
	if len(p) == 4 {
		// distances from the hash table
		offs = append(offs, d.t4.Offsets(p)...)
	}
	return offs
}

// finds a single operation at the current head of the hash dictionary.
func findOp(d *hashDict) (op operation, err error) {
	p := make([]byte, 4)
	n, err := d.buf.ReadAt(p, d.head)
	if err != nil && err != io.EOF {
		return nil, err
	}
	if n <= 0 {
		if n < 0 {
			panic("ReadAt returned negative n")
		}
		return nil, errEmptyBuf
	}
	offs := potentialOffsets(d, p[:n])
	m, err := bestMatch(d, offs)
	if err == errNoMatch {
		return lit{b: p[0]}, nil
	}
	if err != nil {
		return nil, err
	}
	return m, nil
}

// findOps identifies a sequence of operations starting at the current
// head of the dictionary stored in s. If all is set the whole data
// buffer will be covered, if it is not set the last operation reaching
// the head will not be output. This functionality has been included to
// support the extension of the last operation if new data comes in.
func (g greedyFinder) findOps(s *State, all bool) (ops []operation, err error) {
	sd, ok := s.dict.(*hashDict)
	if !ok {
		panic("state doesn't contain hashDict")
	}
	d := *sd
	for d.head < d.buf.top {
		op, err := findOp(&d)
		if err != nil {
			return nil, err
		}
		if _, err = d.move(op.Len()); err != nil {
			return nil, err
		}
		ops = append(ops, op)
	}
	if !all && len(ops) > 0 {
		ops = ops[:len(ops)-1]
	}
	return ops, nil
}

// String implements the string function for the greedyFinder.
func (g greedyFinder) String() string { return "greedy finder" }
