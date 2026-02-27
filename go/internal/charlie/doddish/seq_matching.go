package doddish

func (seq Seq) MatchAll(tokens ...TokenMatcher) bool {
	if len(tokens) != seq.Len() {
		return false
	}

	for i, m := range tokens {
		if !m.Match(seq.At(i)) {
			return false
		}
	}

	return true
}

func (seq Seq) MatchStart(tokens ...TokenMatcher) bool {
	if len(tokens) > seq.Len() {
		return false
	}

	for i, m := range tokens {
		if !m.Match(seq.At(i)) {
			return false
		}
	}

	return true
}

func (seq Seq) MatchEnd(tokens ...TokenMatcher) (ok bool, left, right Seq) {
	if len(tokens) > seq.Len() {
		return ok, left, right
	}

	for i := seq.Len() - 1; i >= 0; i-- {
		partition := seq.At(i)
		j := len(tokens) - (seq.Len() - i)

		if j < 0 {
			break
		}

		m := tokens[j]

		if !m.Match(partition) {
			return ok, left, right
		}

		left = seq[:i]
		right = seq[i:]
	}

	ok = true

	return ok, left, right
}

func (seq Seq) PartitionFavoringRight(
	m TokenMatcher,
) (ok bool, left, right Seq, partition Token) {
	for i := seq.Len() - 1; i >= 0; i-- {
		partition = seq.At(i)

		if m.Match(partition) {
			ok = true
			left = seq[:i]
			right = seq[i+1:]
			return ok, left, right, partition
		}
	}

	return ok, left, right, partition
}

func (seq Seq) PartitionFavoringLeft(
	m TokenMatcher,
) (ok bool, left, right Seq, partition Token) {
	for i := range seq {
		partition = seq.At(i)

		if m.Match(partition) {
			ok = true
			left = seq[:i]
			right = seq[i+1:]
			return ok, left, right, partition
		}
	}

	return ok, left, right, partition
}
