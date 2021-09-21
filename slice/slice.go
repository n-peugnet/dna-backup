package slice

import "reflect"

type Slice []interface{}

type SliceDel int

type SliceIns struct {
	Idx   int
	Value []interface{}
}

type SlicePatch struct {
	Del []SliceDel
	Ins []SliceIns
}

func PatchSlice(source Slice, patch SlicePatch) (target Slice) {
	// apply Del part from patch to source into temp
	size := len(source) - len(patch.Del)
	temp := make(Slice, size)
	fill := 0
	prev := 0
	for _, del := range patch.Del {
		di := int(del)
		copy(temp[fill:], source[prev:di])
		fill += di - prev
		prev = di + 1
	}
	copy(temp[fill:], source[prev:])
	// apply Ins part from patch to temp into target
	for _, ins := range patch.Ins {
		size += len(ins.Value)
	}
	target = make(Slice, size)
	fill = 0
	prev = 0
	tpos := 0
	for _, ins := range patch.Ins {
		offset := ins.Idx - prev
		copy(target[fill:], temp[tpos:tpos+offset])
		fill += offset
		tpos += offset
		copy(target[fill:], ins.Value)
		fill += len(ins.Value)
		prev = ins.Idx + len(ins.Value)
	}
	return
}

func DiffSlice(source Slice, target Slice) (patch SlicePatch) {
	var si, ti int
	var found bool
	for ; si < len(source); si++ {
		for i := ti; i < len(target); i++ {
			found = reflect.DeepEqual(target[i], source[si])
			if found {
				if i != ti {
					patch.Ins = append(patch.Ins, SliceIns{ti, target[ti:i]})
				}
				ti = i + 1
				break
			}
		}
		if !found {
			patch.Del = append(patch.Del, SliceDel(si))
		}
	}
	if ti < len(target) {
		patch.Ins = append(patch.Ins, SliceIns{ti, target[ti:]})
	}
	return
}
