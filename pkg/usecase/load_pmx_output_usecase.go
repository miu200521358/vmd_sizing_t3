package usecase

import "github.com/miu200521358/mlib_go/pkg/domain/pmx"

func CreateOutputModel(model *pmx.PmxModel) (*pmx.PmxModel, error) {
	if sizingModel, _, err := AdjustPmxForSizing(model, false); err != nil {
		return nil, err
	} else {
		return sizingModel, nil
	}

	// for _, boneIndex := range model.Bones.LayerSortedIndexes {
	// 	bone := model.Bones.Get(boneIndex)
	// 	if bone.IsStandard() {

	// 	}
	// }
}
