package usecase

import (
	"fmt"

	"github.com/miu200521358/mlib_go/pkg/mmath"
	"github.com/miu200521358/mlib_go/pkg/pmx"
	"github.com/miu200521358/mlib_go/pkg/vmd"
)

var root_bone_name string = fmt.Sprintf("%s_SIZING_ROOT", pmx.MLIB_PREFIX)

func SetupOriginalPmx(model *pmx.PmxModel) *pmx.PmxModel {
	// SIZING_ROOTボーンを先頭に追加
	rootBone := pmx.NewBone()
	rootBone.Index = model.Bones.Len()
	rootBone.Name = root_bone_name
	rootBone.Layer = -100
	model.Bones.Append(rootBone)

	for _, bone := range model.Bones.Data {
		if bone.ParentIndex == -1 {
			bone.ParentIndex = rootBone.Index
		}
	}

	rootBone.ParentIndex = -1

	return model
}

func SetupOriginalVmd(motion *vmd.VmdMotion) *vmd.VmdMotion {
	// SIZING_ROOTボーンを追加
	bf := vmd.NewBoneFrame(0)
	bf.Position = &mmath.MVec3{-7, 0, 10}
	motion.AppendBoneFrame(root_bone_name, bf)

	return motion
}
