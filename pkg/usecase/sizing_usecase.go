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
		for _, boneNames := range [][]string{{pmx.ARM.Left(), pmx.ELBOW.Left(), pmx.WRIST.Left()},
			{pmx.ARM.Right(), pmx.ELBOW.Right(), pmx.WRIST.Right()}} {
			armBoneName := boneNames[0]
			elbowBoneName := boneNames[1]
			wristBoneName := boneNames[2]

			// 腕
			armBone := sizingModel.Bones.GetByName(armBoneName)
			armOriginalBone := originalModel.Bones.GetByName(armBoneName)
			if armBone == nil || armOriginalBone == nil {
				continue
			}

			armBoneDirection := armBone.Extend.ChildRelativePosition.Normalized()
			armOriginalBoneDirection := armOriginalBone.Extend.ChildRelativePosition.Normalized()
			armOffsetQuat := mmath.NewMQuaternionRotate(armBoneDirection, armOriginalBoneDirection)

			armOffset := pmx.NewBoneMorphOffset(armBone.Index())
			armOffset.Rotation = armOffsetQuat
			offsets = append(offsets, armOffset)

			// ひじ
			elbowBone := sizingModel.Bones.GetByName(elbowBoneName)
			elbowOriginalBone := originalModel.Bones.GetByName(elbowBoneName)
			if elbowBone == nil || elbowOriginalBone == nil {
				continue
			}

			elbowBoneDirection := elbowBone.Extend.ChildRelativePosition.Normalized()
			elbowOriginalBoneDirection := elbowOriginalBone.Extend.ChildRelativePosition.Normalized()
			elbowOffsetQuat := mmath.NewMQuaternionRotate(elbowBoneDirection, elbowOriginalBoneDirection)

			elbowOffset := pmx.NewBoneMorphOffset(elbowBone.Index())
			elbowOffset.Rotation = elbowOffsetQuat.Muled(armOffsetQuat.Inverted())
			offsets = append(offsets, elbowOffset)

			// 手首
			wristBone := sizingModel.Bones.GetByName(wristBoneName)
			wristOriginalBone := originalModel.Bones.GetByName(wristBoneName)
			if wristBone == nil || wristOriginalBone == nil {
				continue
			}

			wristBoneDirection := wristBone.Extend.ChildRelativePosition.Normalized()
			wristOriginalBoneDirection := wristOriginalBone.Extend.ChildRelativePosition.Normalized()
			wristOffsetQuat := mmath.NewMQuaternionRotate(wristBoneDirection, wristOriginalBoneDirection)

			wristOffset := pmx.NewBoneMorphOffset(wristBone.Index())
			wristOffset.Rotation = wristOffsetQuat.Muled(elbowOffsetQuat.Inverted())
			offsets = append(offsets, wristOffset)
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
