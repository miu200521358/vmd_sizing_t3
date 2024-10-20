package usecase

import (
	"fmt"
	"math"

	"github.com/miu200521358/mlib_go/pkg/domain/mmath"
	"github.com/miu200521358/mlib_go/pkg/domain/pmx"
)

func CreateOutputModel(model *pmx.PmxModel) (*pmx.PmxModel, error) {
	if sizingModel, _, err := AdjustPmxForSizing(model, false); err != nil {
		return nil, err
	} else {
		// 調整ボーン追加
		addAdjustBones(sizingModel)

		// 調整モーフ追加
		addAdjustMorphs(sizingModel)

		return sizingModel, nil
	}
}

func addAdjustMorphs(model *pmx.PmxModel) {
	displaySlot := model.DisplaySlots.GetByName("表情")

	for n := range 2 {
		var morphName string
		var morphEnglishName string
		var leftAngle float64
		var rightAngle float64

		switch n {
		case 0:
			morphName = "m_内股"
			morphEnglishName = "m_InnerThigh"
			leftAngle = -20
			rightAngle = 20
		case 1:
			morphName = "m_がに股"
			morphEnglishName = "m_OuterThigh"
			leftAngle = 20
			rightAngle = -20
		}

		{
			offsets := make([]pmx.IMorphOffset, 0)
			{
				legBone := model.Bones.GetByName(pmx.LEG.Left())
				rotVector := mmath.NewMQuaternionFromDegrees(1, 0, 0).MulVec3(
					legBone.Extend.NormalizedLocalAxisX.MuledScalar(leftAngle))
				{
					offset := pmx.NewBoneMorphOffset(legBone.Index())
					offset.Rotation = mmath.NewMQuaternionFromDegrees(rotVector.X, rotVector.Y, rotVector.Z)
					offsets = append(offsets, offset)

				}

				legIkBone := model.Bones.GetByName(pmx.LEG_IK.Left())
				{
					offset := pmx.NewBoneMorphOffset(legIkBone.Index())
					offset.Rotation = mmath.NewMQuaternionFromDegrees(rotVector.X, rotVector.Y, rotVector.Z)
					offsets = append(offsets, offset)
				}
			}
			{
				legBone := model.Bones.GetByName(pmx.LEG.Right())
				rotVector := mmath.NewMQuaternionFromDegrees(1, 0, 0).MulVec3(
					legBone.Extend.NormalizedLocalAxisX.MuledScalar(rightAngle))
				{
					offset := pmx.NewBoneMorphOffset(legBone.Index())
					offset.Rotation = mmath.NewMQuaternionFromDegrees(rotVector.X, rotVector.Y, rotVector.Z)
					offsets = append(offsets, offset)

				}

				legIkBone := model.Bones.GetByName(pmx.LEG_IK.Right())
				{
					offset := pmx.NewBoneMorphOffset(legIkBone.Index())
					offset.Rotation = mmath.NewMQuaternionFromDegrees(rotVector.X, rotVector.Y, rotVector.Z)
					offsets = append(offsets, offset)
				}
			}

			morph := pmx.NewMorph()
			morph.SetIndex(model.Morphs.Len())
			morph.SetName(morphName)
			morph.SetEnglishName(morphEnglishName)
			morph.Offsets = offsets
			morph.MorphType = pmx.MORPH_TYPE_BONE
			morph.Panel = pmx.MORPH_PANEL_OTHER_LOWER_RIGHT
			morph.IsSystem = true
			model.Morphs.Append(morph)

			displaySlot.References = append(displaySlot.References,
				&pmx.Reference{DisplayType: pmx.DISPLAY_TYPE_MORPH, DisplayIndex: morph.Index()})
		}
	}
}

func addAdjustBones(model *pmx.PmxModel) {
	addedBones := make(map[string]*pmx.Bone, 0)
	for _, boneIndex := range model.Bones.LayerSortedIndexes {
		bone := model.Bones.Get(boneIndex)
		if !(bone.IsStandard() && bone.Name() != pmx.ROOT.String() && bone.CanManipulate() && !bone.IsEffectorRotation() && !bone.IsEffectorTranslation() && len(bone.Extend.IkLinkBoneIndexes) == 0 && len(bone.Extend.IkTargetBoneIndexes) == 0) {
			continue
		}

		// 標準かつIKやエフェクタでないボーンに調整ボーンを追加
		adjustBone := pmx.NewBone()
		adjustBone.SetIndex(model.Bones.Len())
		adjustBone.SetName(fmt.Sprintf("%s_調整", bone.Name()))
		adjustBone.SetEnglishName(fmt.Sprintf("%s_Adjust", bone.EnglishName()))
		adjustBone.BoneFlag = pmx.BONE_FLAG_CAN_MANIPULATE | pmx.BONE_FLAG_IS_VISIBLE

		if bone.CanRotate() {
			adjustBone.BoneFlag |= pmx.BONE_FLAG_CAN_ROTATE
			bone.BoneFlag |= pmx.BONE_FLAG_IS_EXTERNAL_ROTATION
		}
		if bone.CanTranslate() {
			adjustBone.BoneFlag |= pmx.BONE_FLAG_CAN_TRANSLATE
			bone.BoneFlag |= pmx.BONE_FLAG_IS_EXTERNAL_TRANSLATION
		}
		if bone.HasFixedAxis() {
			adjustBone.BoneFlag |= pmx.BONE_FLAG_HAS_FIXED_AXIS
			adjustBone.FixedAxis = bone.FixedAxis.Copy()
		}
		if bone.HasLocalAxis() {
			adjustBone.BoneFlag |= pmx.BONE_FLAG_HAS_LOCAL_AXIS
			adjustBone.LocalAxisX = bone.LocalAxisX.Copy()
			adjustBone.LocalAxisZ = bone.LocalAxisZ.Copy()
		}

		afterIndex := -1
		adjustBone.ParentIndex = 0
		for _, parentIndex := range bone.Extend.ParentBoneIndexes {
			// 親ボーンが存在する場合、該当親ボーンの調整ボーンを親とする
			parentBone := model.Bones.Get(parentIndex)
			adjustParentBone := addedBones[fmt.Sprintf("%s_調整", parentBone.Name())]
			if adjustParentBone != nil {
				adjustBone.ParentIndex = adjustParentBone.Index()
				afterIndex = adjustParentBone.Index()
				break
			}
		}

		// 横方向に作成
		adjustBone.Position = bone.Position.Added(&mmath.MVec3{X: -10, Y: 0, Z: 0})

		model.Bones.Insert(adjustBone, afterIndex)
		addedBones[adjustBone.Name()] = adjustBone

		// 表示枠追加
		displaySlot := model.DisplaySlots.GetByName("調整用")
		if displaySlot == nil {
			displaySlot = pmx.NewDisplaySlot()
			displaySlot.SetIndex(model.DisplaySlots.Len())
			displaySlot.SetName("調整用")
			model.DisplaySlots.Append(displaySlot)
		}
		displaySlot.References = append(displaySlot.References,
			&pmx.Reference{DisplayType: pmx.DISPLAY_TYPE_BONE, DisplayIndex: adjustBone.Index()})

		bone.EffectIndex = adjustBone.Index()
		bone.EffectFactor = 1.0
	}

	// 全部追加が終わったら、変形階層を再設定
	minLayer := math.MaxInt
	for _, bone := range model.Bones.Data {
		if bone.Layer < minLayer {
			minLayer = bone.Layer
		}
	}

	for _, bone := range model.Bones.Data {
		bone.Layer -= minLayer
	}
}
