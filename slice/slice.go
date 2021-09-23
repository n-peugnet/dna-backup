package slice

import (
	"fmt"
	"reflect"
)

type Slice []interface{}

type Del int

type Ins struct {
	Idx   int
	Value []interface{}
}

type insData struct {
	idx   int
	count int
}

type Delta struct {
	Del []Del
	Ins []Ins
}

func (d Delta) String() string {
	data := make([]insData, len(d.Ins))
	for i, ins := range d.Ins {
		data[i] = insData{ins.Idx, len(ins.Value)}
	}
	return fmt.Sprintf("{Del: %d Ins: %+v}", d.Del, data)
}

func Patch(source Slice, delta Delta) (target Slice) {
	// apply Del part from patch to source into temp
	size := len(source) - len(delta.Del)
	temp := make(Slice, size)
	fill := 0
	prev := 0
	for _, del := range delta.Del {
		di := int(del)
		copy(temp[fill:], source[prev:di])
		fill += di - prev
		prev = di + 1
	}
	copy(temp[fill:], source[prev:])
	// apply Ins part from patch to temp into target
	for _, ins := range delta.Ins {
		size += len(ins.Value)
	}
	target = make(Slice, size)
	fill = 0
	prev = 0
	tpos := 0
	for _, ins := range delta.Ins {
		offset := ins.Idx - prev
		copy(target[fill:], temp[tpos:tpos+offset])
		fill += offset
		tpos += offset
		copy(target[fill:], ins.Value)
		fill += len(ins.Value)
		prev = ins.Idx + len(ins.Value)
	}
	copy(target[fill:], temp[tpos:])
	return
}

func Diff(source Slice, target Slice) (delta Delta) {
	var si, ti int
	var found bool
	for ; si < len(source); si++ {
		for i := ti; i < len(target); i++ {
			found = reflect.DeepEqual(target[i], source[si])
			if found {
				if i != ti {
					delta.Ins = append(delta.Ins, Ins{ti, target[ti:i]})
				}
				ti = i + 1
				break
			}
		}
		if !found {
			delta.Del = append(delta.Del, Del(si))
		}
	}
	if ti < len(target) {
		delta.Ins = append(delta.Ins, Ins{ti, target[ti:]})
	}
	return
}
