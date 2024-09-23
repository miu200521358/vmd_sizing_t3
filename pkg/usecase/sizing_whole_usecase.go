package usecase

import (
	"github.com/miu200521358/mlib_go/pkg/domain/delta"
	"github.com/miu200521358/mlib_go/pkg/domain/miter"
	"github.com/miu200521358/mlib_go/pkg/domain/mmath"
	"github.com/miu200521358/mlib_go/pkg/domain/pmx"
	"github.com/miu200521358/mlib_go/pkg/infrastructure/deform"
	"github.com/miu200521358/vmd_sizing_t3/pkg/model"
)

// 全身位置角度合わせ
func SizingWholeStance(sizingSet *model.SizingSet) {
	if !sizingSet.IsSizingWholeStance || (sizingSet.IsSizingWholeStance && sizingSet.CompletedSizingWholeStance) {
		return
	}
	// TODO　必須ボーンチェック

	// 足補正
	originalModel := sizingSet.OriginalPmx
	originalMotion := sizingSet.OriginalVmd
	sizingModel := sizingSet.SizingPmx
	sizingMotion := sizingSet.OutputVmd

	leftFrames := originalMotion.BoneFrames.RegisteredFrames(leg_direction_bone_names[0])
	rightFrames := originalMotion.BoneFrames.RegisteredFrames(leg_direction_bone_names[1])
	trunkFrames := originalMotion.BoneFrames.RegisteredFrames(trunk_bone_names)
	m := make(map[int]struct{})
	frames := make([]int, 0, len(leftFrames)+len(rightFrames)+len(trunkFrames))
	for _, fs := range [][]int{leftFrames, rightFrames, trunkFrames} {
		for _, f := range fs {
			if _, ok := m[f]; ok {
				continue
			}
			m[f] = struct{}{}
			frames = append(frames, f)
		}
	}
	mmath.SortInts(frames)
	originalAllDeltas := make([]*delta.VmdDeltas, len(frames))

	// 元モデルのデフォーム(IK ON)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		vmdDeltas := delta.NewVmdDeltas(frame, originalModel.Bones, originalModel.Hash(), originalMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(originalModel, originalMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(originalModel, originalMotion, vmdDeltas, true, frame, leg_all_bone_names, false)
		originalAllDeltas[index] = vmdDeltas
	})

	// サイジング先にFKを焼き込み
	for _, vmdDeltas := range originalAllDeltas {
		for _, boneDelta := range vmdDeltas.Bones.Data {
			if boneDelta == nil || !boneDelta.Bone.IsLegFK() {
				continue
			}

			sizingBf := sizingMotion.BoneFrames.Get(boneDelta.Bone.Name()).Get(boneDelta.Frame)

			// 最終的な足FKを焼き込み
			sizingBf.Rotation = boneDelta.FilledFrameRotation()
			sizingMotion.InsertRegisteredBoneFrame(boneDelta.Bone.Name(), sizingBf)
		}
	}

	sizingOffDeltas := make([]*delta.VmdDeltas, len(frames))
	rightLegDiffs := make([]*mmath.MVec3, len(frames))

	// サイジング先モデルのデフォーム(IK OFF)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, false, frame, leg_all_bone_names, false)
		sizingOffDeltas[index] = vmdDeltas

		// 右足首から見た右足ボーンの相対位置を取得
		rightLegIkBone := sizingModel.Bones.GetByName(pmx.LEG_IK.Right())
		rightLegIkBoneDelta := vmdDeltas.Bones.Get(rightLegIkBone.Index())

		rightLegBone := sizingModel.Bones.GetByName(pmx.LEG.Right())
		rightLegBoneDelta := vmdDeltas.Bones.Get(rightLegBone.Index())

		rightAnkleBone := sizingModel.Bones.Get(sizingModel.Bones.GetByName(pmx.LEG_IK.Right()).Ik.BoneIndex)
		rightAnkleBoneDelta := vmdDeltas.Bones.Get(rightAnkleBone.Index())

		rightLegFkLocalPosition := rightAnkleBoneDelta.FilledGlobalPosition().Subed(
			rightLegBoneDelta.FilledGlobalPosition())
		rightLegIkLocalPosition := rightLegIkBoneDelta.FilledGlobalPosition().Subed(
			rightLegBoneDelta.FilledGlobalPosition())
		rightLegDiff := rightLegIkLocalPosition.Subed(rightLegFkLocalPosition)

		// 右足IKを動かさなかった場合のセンターと左足IKの位置を調整する用の値を保持
		// （この時点でキーフレに追加すると動きが変わる）
		rightLegDiffs[index] = rightLegDiff
	})

	// 補正を登録
	for i, vmdDeltas := range sizingOffDeltas {
		rightLegBoneDelta := vmdDeltas.Bones.Get(sizingModel.Bones.GetByName(pmx.LEG.Right()).Index())

		rightLegDiff := rightLegDiffs[i]
		grooveBone := sizingModel.Bones.GetByName(pmx.GROOVE.String())
		sizingGrooveBf := sizingMotion.BoneFrames.Get(grooveBone.Name()).Get(rightLegBoneDelta.Frame)
		sizingGrooveBf.Position.Add(rightLegDiff)
		sizingMotion.InsertRegisteredBoneFrame(grooveBone.Name(), sizingGrooveBf)

		leftLegIkBone := sizingModel.Bones.GetByName(pmx.LEG_IK.Left())
		sizingLeftBf := sizingMotion.BoneFrames.Get(leftLegIkBone.Name()).Get(rightLegBoneDelta.Frame)
		sizingLeftBf.Position.Add(rightLegDiff)
		sizingMotion.InsertRegisteredBoneFrame(leftLegIkBone.Name(), sizingLeftBf)
	}

	sizingCenterDeltas := make([]*delta.VmdDeltas, len(frames))
	leftLegDiffs := make([]*mmath.MVec3, len(frames))

	// サイジング先モデルのデフォーム(IK OFF+センター補正済み)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, false, frame, leg_all_bone_names, false)
		sizingCenterDeltas[index] = vmdDeltas

		// 左足首から見た左足ボーンの相対位置を取得
		leftLegIkBone := sizingModel.Bones.GetByName(pmx.LEG_IK.Left())
		leftLegIkBoneDelta := vmdDeltas.Bones.Get(leftLegIkBone.Index())

		leftLegBone := sizingModel.Bones.GetByName(pmx.LEG.Left())
		leftLegBoneDelta := vmdDeltas.Bones.Get(leftLegBone.Index())

		leftAnkleBone := sizingModel.Bones.Get(sizingModel.Bones.GetByName(pmx.LEG_IK.Left()).Ik.BoneIndex)
		leftAnkleBoneDelta := vmdDeltas.Bones.Get(leftAnkleBone.Index())

		leftLegFkLocalPosition := leftLegBoneDelta.FilledGlobalPosition().Subed(
			leftLegIkBoneDelta.FilledGlobalPosition())
		leftLegIkLocalPosition := leftLegBoneDelta.FilledGlobalPosition().Subed(
			leftAnkleBoneDelta.FilledGlobalPosition())
		leftLegDiff := leftLegIkLocalPosition.Subed(leftLegFkLocalPosition)

		// 左足IKを動かさなかった場合のセンターと右足IKの位置を調整する用の値を保持
		leftLegDiffs[index] = leftLegDiff
	})

	// 右足IKを動かさなかった場合の左足首の位置を調整
	for i, vmdDeltas := range sizingOffDeltas {
		leftLegIkBone := sizingModel.Bones.GetByName(pmx.LEG_IK.Left())
		leftLegIkBoneDelta := vmdDeltas.Bones.Get(leftLegIkBone.Index())
		leftLegDiff := leftLegDiffs[i]

		// 左足IK補正
		sizingBf := sizingMotion.BoneFrames.Get(leftLegIkBone.Name()).Get(leftLegIkBoneDelta.Frame)
		sizingBf.Position.Sub(leftLegDiff)
		sizingMotion.InsertRegisteredBoneFrame(leftLegIkBone.Name(), sizingBf)
	}
}
