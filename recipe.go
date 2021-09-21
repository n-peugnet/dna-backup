package main

import "reflect"

type Recipe []Chunk

type RecipeDel int

type RecipeIns struct {
	Idx   int
	Value []Chunk
}

type RecipePatch struct {
	Del []RecipeDel
	Ins []RecipeIns
}

func patchRecipe(source Recipe, patch RecipePatch) (target Recipe) {
	// apply Del part from patch to source into temp
	size := len(source) - len(patch.Del)
	temp := make(Recipe, size)
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
	target = make(Recipe, size)
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

func diffRecipe(source Recipe, target Recipe) (patch RecipePatch) {
	var si, ti int
	var found bool
	for ; si < len(source); si++ {
		for i := ti; i < len(target); i++ {
			found = reflect.DeepEqual(target[i], source[si])
			if found {
				if i != ti {
					patch.Ins = append(patch.Ins, RecipeIns{ti, target[ti:i]})
				}
				ti = i + 1
				break
			}
		}
		if !found {
			patch.Del = append(patch.Del, RecipeDel(si))
		}
	}
	if ti < len(target) {
		patch.Ins = append(patch.Ins, RecipeIns{ti, target[ti:]})
	}
	return
}
