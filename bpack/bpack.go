package bpack

type IndexEnt struct {
	Key    string
	Offset uint64
}

type Index []IndexEnt

func (idx Index) Len() int           { return len(idx) }
func (idx Index) Swap(i, j int)      { idx[i], idx[j] = idx[j], idx[i] }
func (idx Index) Less(i, j int) bool { return KeyCmp(idx[i].Key, idx[j].Key) < 0 }

func (idx Index) Search(key string) (int, bool) {
	lo := 0
	hi := len(idx) - 1
	for lo <= hi {
		mid := (hi + lo) / 2
		switch KeyCmp(idx[mid].Key, key) {
		case 1:
			hi = mid - 1
		case -1:
			lo = mid + 1
		case 0:
			return mid, true
		}
	}
	return -1, false

}

func KeyCmp(l, r string) int {
	if len(l) != len(r) {
		if len(l) < len(r) {
			return -1
		} else {
			return 1
		}
	}
	for i := range l {
		if l[i] != r[i] {
			if l[i] < r[i] {
				return -1
			} else {
				return 1
			}
		}
	}
	return 0
}
