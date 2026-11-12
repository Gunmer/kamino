package conflicts

import "sort"

// Opcode is one diff opcode over two sequences.
type Opcode struct {
	Tag    string
	I1, I2 int
	J1, J2 int
}

type match struct {
	A, B, Size int
}

type sequenceMatcher struct {
	a        []string
	b        []string
	b2j      map[string][]int
	autojunk bool
}

func newSequenceMatcher(a, b []string, autojunk bool) *sequenceMatcher {
	m := &sequenceMatcher{a: a, b: b, autojunk: autojunk}
	m.chainB()
	return m
}

func (m *sequenceMatcher) chainB() {
	counts := make(map[string]int, len(m.b))
	for _, item := range m.b {
		counts[item]++
	}
	m.b2j = make(map[string][]int, len(counts))
	for index, item := range m.b {
		m.b2j[item] = append(m.b2j[item], index)
	}
	if m.autojunk && len(m.b) >= 200 {
		popular := len(m.b)/100 + 1
		for item, indices := range m.b2j {
			if len(indices) > popular {
				delete(m.b2j, item)
			}
		}
	}
}

func (m *sequenceMatcher) findLongestMatch(alo, ahi, blo, bhi int) match {
	bestI, bestJ, bestSize := alo, blo, 0
	j2len := map[int]int{}
	for i := alo; i < ahi; i++ {
		newJ2Len := map[int]int{}
		for _, j := range m.b2j[m.a[i]] {
			if j < blo {
				continue
			}
			if j >= bhi {
				break
			}
			k := j2len[j-1] + 1
			newJ2Len[j] = k
			if k > bestSize {
				bestI, bestJ, bestSize = i-k+1, j-k+1, k
			}
		}
		j2len = newJ2Len
	}
	return match{A: bestI, B: bestJ, Size: bestSize}
}

// GetMatchingBlocks returns the matching blocks, including the final sentinel.
func (m *sequenceMatcher) GetMatchingBlocks() []match {
	type region struct{ alo, ahi, blo, bhi int }
	queue := []region{{0, len(m.a), 0, len(m.b)}}
	blocks := make([]match, 0)
	for len(queue) > 0 {
		current := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		found := m.findLongestMatch(current.alo, current.ahi, current.blo, current.bhi)
		if found.Size > 0 {
			blocks = append(blocks, found)
			if current.alo < found.A && current.blo < found.B {
				queue = append(queue, region{current.alo, found.A, current.blo, found.B})
			}
			if found.A+found.Size < current.ahi && found.B+found.Size < current.bhi {
				queue = append(queue, region{found.A + found.Size, current.ahi, found.B + found.Size, current.bhi})
			}
		}
	}
	sort.Slice(blocks, func(i, j int) bool {
		if blocks[i].A != blocks[j].A {
			return blocks[i].A < blocks[j].A
		}
		return blocks[i].B < blocks[j].B
	})
	combined := make([]match, 0)
	i1, j1, k1 := 0, 0, 0
	for _, block := range blocks {
		if i1+k1 == block.A && j1+k1 == block.B {
			k1 += block.Size
			continue
		}
		if k1 > 0 {
			combined = append(combined, match{A: i1, B: j1, Size: k1})
		}
		i1, j1, k1 = block.A, block.B, block.Size
	}
	if k1 > 0 {
		combined = append(combined, match{A: i1, B: j1, Size: k1})
	}
	combined = append(combined, match{A: len(m.a), B: len(m.b), Size: 0})
	return combined
}

// GetOpcodes returns the opcodes for the two sequences.
func (m *sequenceMatcher) GetOpcodes() []Opcode {
	i, j := 0, 0
	opcodes := make([]Opcode, 0)
	for _, block := range m.GetMatchingBlocks() {
		tag := ""
		if i < block.A && j < block.B {
			tag = "replace"
		} else if i < block.A {
			tag = "delete"
		} else if j < block.B {
			tag = "insert"
		}
		if tag != "" {
			opcodes = append(opcodes, Opcode{Tag: tag, I1: i, I2: block.A, J1: j, J2: block.B})
		}
		i, j = block.A+block.Size, block.B+block.Size
		if block.Size > 0 {
			opcodes = append(opcodes, Opcode{Tag: "equal", I1: block.A, I2: i, J1: block.B, J2: j})
		}
	}
	return opcodes
}

// GetGroupedOpcodes groups opcodes with no more than n context lines.
func (m *sequenceMatcher) GetGroupedOpcodes(n int) [][]Opcode {
	codes := m.GetOpcodes()
	if len(codes) == 0 {
		codes = []Opcode{{Tag: "equal", I1: 0, I2: 1, J1: 0, J2: 1}}
	}
	codes = append([]Opcode(nil), codes...)
	if codes[0].Tag == "equal" {
		c := &codes[0]
		if c.I1 < c.I2-n {
			c.I1 = c.I2 - n
		}
		if c.J1 < c.J2-n {
			c.J1 = c.J2 - n
		}
	}
	if codes[len(codes)-1].Tag == "equal" {
		c := &codes[len(codes)-1]
		if c.I2 > c.I1+n {
			c.I2 = c.I1 + n
		}
		if c.J2 > c.J1+n {
			c.J2 = c.J1 + n
		}
	}
	groups := make([][]Opcode, 0)
	group := make([]Opcode, 0)
	double := n + n
	for _, code := range codes {
		if code.Tag == "equal" && code.I2-code.I1 > double {
			group = append(group, Opcode{Tag: code.Tag, I1: code.I1, I2: code.I1 + n, J1: code.J1, J2: code.J1 + n})
			groups = append(groups, group)
			group = []Opcode{{Tag: code.Tag, I1: code.I2 - n, I2: code.I2, J1: code.J2 - n, J2: code.J2}}
			continue
		}
		group = append(group, code)
	}
	if len(group) > 0 && !(len(group) == 1 && group[0].Tag == "equal") {
		groups = append(groups, group)
	}
	return groups
}
