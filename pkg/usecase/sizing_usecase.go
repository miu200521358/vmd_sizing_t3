package usecase

import (
	"fmt"

	"github.com/miu200521358/mlib_go/pkg/domain/mmath"
	"github.com/miu200521358/mlib_go/pkg/domain/pmx"
	"github.com/miu200521358/mlib_go/pkg/domain/vmd"
	"github.com/miu200521358/vmd_sizing_t3/pkg/model"
)

// FitBoneモーフ名
var sizing_morph_name = fmt.Sprintf("%s_%s", pmx.MLIB_PREFIX, "SizingBone")

func CreateSizingMorph(originalModel, sizingModel *pmx.PmxModel, sizingSet *model.SizingSet) *pmx.PmxModel {
	sizingModel.Morphs.RemoveByName(sizing_morph_name)

	createSizingMorph(originalModel, sizingModel, sizingSet, sizing_morph_name)
	sizingModel.Setup()

	return sizingModel
}

func createSizingMorph(originalModel, sizingModel *pmx.PmxModel, sizingSet *model.SizingSet, morphName string) {
	offsets := make([]pmx.IMorphOffset, 0)

	if sizingSet.IsSizingArmStance {
		// スタンス補正
		for _, boneName := range []string{pmx.ARM.Left(), pmx.ARM.Right()} {
			bone := sizingModel.Bones.GetByName(boneName)
			originalBone := originalModel.Bones.GetByName(boneName)
			if bone == nil || originalBone == nil {
				continue
			}

			boneDirection := bone.Extend.ChildRelativePosition.Normalized()
			originalBoneDirection := originalBone.Extend.ChildRelativePosition.Normalized()
			offsetQuat := mmath.NewMQuaternionRotate(boneDirection, originalBoneDirection)

			offset := pmx.NewBoneMorphOffset(bone.Index())
			offset.Rotation = offsetQuat
			offsets = append(offsets, offset)
		}
	}

	morph := pmx.NewMorph()
	morph.SetIndex(sizingModel.Morphs.Len())
	morph.SetName(morphName)
	morph.Offsets = offsets
	morph.MorphType = pmx.MORPH_TYPE_BONE
	morph.Panel = pmx.MORPH_PANEL_OTHER_LOWER_RIGHT
	morph.IsSystem = true
	sizingModel.Morphs.Append(morph)
}

func AddSizingMorph(motion *vmd.VmdMotion) *vmd.VmdMotion {
	if motion.MorphFrames != nil && motion.MorphFrames.Contains(sizing_morph_name) {
		return motion
	}

	// サイジングボーンモーフを適用
	mf := vmd.NewMorphFrame(float32(0))
	mf.Ratio = 1.0
	motion.AppendMorphFrame(sizing_morph_name, mf)
	return motion
}
