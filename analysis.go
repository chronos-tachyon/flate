package flate

func isText(buf []byte) bool {
	for _, ch := range buf {
		if ch < 0x09 {
			return false
		}
		if ch > 0x0d && ch < 0x20 {
			return false
		}
		if ch == 0x7f {
			return false
		}
	}
	return true
}

func studyFrequenciesLLD(tokens []token) (freqLL []uint32, freqD []uint32) {
	freqLL = make([]uint32, physicalNumLLCodes)
	freqD = make([]uint32, physicalNumDCodes)
	for _, t := range tokens {
		if symLL, _, _ := t.symbolLL(); symLL >= 0 {
			freqLL[symLL]++
		}

		if symD, _, _ := t.symbolD(); symD >= 0 {
			freqD[symD]++
		}
	}
	return
}

func studyFrequenciesX(tokens []token) (freqX []uint32) {
	freqX = make([]uint32, physicalNumXCodes)
	for _, t := range tokens {
		if symX, _, _ := t.symbolX(); symX >= 0 {
			freqX[symX]++
		}
	}
	return
}

func sizesAllZeroes(sizes []byte) bool {
	for i, length := uint(0), uint(len(sizes)); i < length; i++ {
		if sizes[i] != 0 {
			return false
		}
	}
	return true
}

func sizeRunLength(sizes []byte, size byte) uint {
	var count uint
	for i, length := uint(0), uint(len(sizes)); i < length; i++ {
		if sizes[i] != size {
			break
		}
		count++
	}
	return count
}

func encodeTreeTokens(xtokens []token, sizes []byte, min uint) ([]token, uint) {
	i := uint(0)
	sizesLen := uint(len(sizes))
	for i < sizesLen {
		if i >= min && sizesAllZeroes(sizes[i:]) {
			break
		}

		if size := sizes[i]; size != 0 {
			xtokens = append(xtokens, makeTreeLenToken(size))
			i++

			run := sizeRunLength(sizes[i:], size)
			for run >= 9 {
				xtokens = append(xtokens, makeTreeDupToken(6))
				i += 6
				run -= 6
			}
			if run > 6 {
				xtokens = append(xtokens, makeTreeDupToken(4))
				i += 4
				run -= 4
			}
			if run > 2 {
				xtokens = append(xtokens, makeTreeDupToken(run))
				i += run
				run = 0
			}
			for run != 0 {
				xtokens = append(xtokens, makeTreeLenToken(size))
				i++
				run--
			}
			continue
		}

		run := sizeRunLength(sizes[i:], 0)
		if (i+run) > min && sizesAllZeroes(sizes[i:]) {
			run = min
		}
		for run >= 141 {
			xtokens = append(xtokens, makeTreeZeroRunToken(138))
			i += 138
			run -= 138
		}
		if run > 138 {
			xtokens = append(xtokens, makeTreeZeroRunToken(136))
			i += 136
			run -= 136
		}
		if run > 3 {
			xtokens = append(xtokens, makeTreeZeroRunToken(run))
			i += run
			run = 0
		}
		for run != 0 {
			xtokens = append(xtokens, makeTreeLenToken(0))
			i++
			run--
		}
	}
	sizesNum := i
	return xtokens, sizesNum
}
