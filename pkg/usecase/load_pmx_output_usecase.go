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

		return sizingModel, nil
	}
}

func addAdjustBones(model *pmx.PmxModel) {
	addedBones := make(map[string]*pmx.Bone, 0)
	for _, boneIndex := range model.Bones.LayerSortedIndexes {
		bone := model.Bones.Get(boneIndex)
		if !(bone.IsStandard() && bone.CanManipulate() && !bone.IsEffectorRotation() && !bone.IsEffectorTranslation() && len(bone.Extend.IkLinkBoneIndexes) == 0 && len(bone.Extend.IkTargetBoneIndexes) == 0) {
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
