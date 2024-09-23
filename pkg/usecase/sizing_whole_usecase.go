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

	centerBone := sizingModel.Bones.GetByName(pmx.CENTER.String())
	grooveBone := sizingModel.Bones.GetByName(pmx.GROOVE.String())
	rightLegIkBone := sizingModel.Bones.GetByName(pmx.LEG_IK.Right())
	rightLegBone := sizingModel.Bones.GetByName(pmx.LEG.Right())
	rightAnkleBone := sizingModel.Bones.Get(sizingModel.Bones.GetByName(pmx.LEG_IK.Right()).Ik.BoneIndex)
	leftLegIkBone := sizingModel.Bones.GetByName(pmx.LEG_IK.Left())
	leftLegBone := sizingModel.Bones.GetByName(pmx.LEG.Left())
	leftAnkleBone := sizingModel.Bones.Get(sizingModel.Bones.GetByName(pmx.LEG_IK.Left()).Ik.BoneIndex)

	leftFrames := originalMotion.BoneFrames.RegisteredFrames(leg_direction_bone_names[0])
	rightFrames := originalMotion.BoneFrames.RegisteredFrames(leg_direction_bone_names[1])
	trunkFrames := originalMotion.BoneFrames.RegisteredFrames(trunk_bone_names)
	m := make(map[int]struct{})
	frames := make([]int, 0, len(leftFrames)+len(rightFrames)+len(trunkFrames))
	leftZeroFrames := make([]float32, 0, len(leftFrames)+len(rightFrames)+len(trunkFrames))
	rightZeroFrames := make([]float32, 0, len(leftFrames)+len(rightFrames)+len(trunkFrames))
	for _, fs := range [][]int{leftFrames, rightFrames, trunkFrames} {
		for _, f := range fs {
			if _, ok := m[f]; ok {
				continue
			}

			frame := float32(f)
			leftLegIkBf := originalMotion.BoneFrames.Get(leftLegIkBone.Name()).Get(frame)
			if mmath.NearEquals(leftLegIkBf.Position.Y, 0, 1e-3) {
				leftZeroFrames = append(leftZeroFrames, frame)
			}
			rightLegIkBf := originalMotion.BoneFrames.Get(rightLegIkBone.Name()).Get(frame)
			if mmath.NearEquals(rightLegIkBf.Position.Y, 0, 1e-3) {
				rightZeroFrames = append(rightZeroFrames, frame)
			}

			m[f] = struct{}{}
			frames = append(frames, f)
		}
	}
	mmath.SortInts(frames)
	mmath.SortFloat32s(leftZeroFrames)
	mmath.SortFloat32s(rightZeroFrames)

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

	centerPositions := make([]*mmath.MVec3, len(frames))
	groovePositions := make([]*mmath.MVec3, len(frames))
	rightLegIkPositions := make([]*mmath.MVec3, len(frames))
	leftLegIkPositions := make([]*mmath.MVec3, len(frames))

	// サイジング先モデルのデフォーム(IK OFF)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, false, frame, leg_all_bone_names, false)
		sizingOffDeltas[index] = vmdDeltas

		// 右足首から見た右足ボーンの相対位置を取得
		rightLegIkBoneDelta := vmdDeltas.Bones.Get(rightLegIkBone.Index())
		rightLegBoneDelta := vmdDeltas.Bones.Get(rightLegBone.Index())
		rightAnkleBoneDelta := vmdDeltas.Bones.Get(rightAnkleBone.Index())

		rightLegFkLocalPosition := rightAnkleBoneDelta.FilledGlobalPosition().Subed(
			rightLegBoneDelta.FilledGlobalPosition())
		rightLegIkLocalPosition := rightLegIkBoneDelta.FilledGlobalPosition().Subed(
			rightLegBoneDelta.FilledGlobalPosition())
		rightLegDiff := rightLegIkLocalPosition.Subed(rightLegFkLocalPosition)

		// 右足IKを動かさなかった場合のセンターと左足IKの位置を調整する用の値を保持
		// （この時点でキーフレに追加すると動きが変わる）
		rightLegDiffs[index] = rightLegDiff

		centerBf := sizingMotion.BoneFrames.Get(centerBone.Name()).Get(frame)
		centerPositions[index] = centerBf.Position.Added(&mmath.MVec3{X: rightLegDiff.X, Y: 0, Z: rightLegDiff.Z})

		grooveBf := sizingMotion.BoneFrames.Get(grooveBone.Name()).Get(frame)
		groovePositions[index] = grooveBf.Position.Added(&mmath.MVec3{X: 0, Y: rightLegDiff.Y, Z: 0})

		rightLegIkBf := sizingMotion.BoneFrames.Get(rightLegIkBone.Name()).Get(frame)
		rightLegIkPositions[index] = rightLegIkBf.Position

		leftLegIkBf := sizingMotion.BoneFrames.Get(leftLegIkBone.Name()).Get(frame)
		leftLegIkPositions[index] = leftLegIkBf.Position.Added(rightLegDiff)
	})

	// 補正を登録(ここで参照すると結果がズレる)
	for i, vmdDeltas := range sizingOffDeltas {
		rightLegBoneDelta := vmdDeltas.Bones.Get(rightLegBone.Index())

		sizingCenterBf := sizingMotion.BoneFrames.Get(centerBone.Name()).Get(rightLegBoneDelta.Frame)
		sizingCenterBf.Position = centerPositions[i]
		sizingMotion.InsertRegisteredBoneFrame(centerBone.Name(), sizingCenterBf)

		sizingGrooveBf := sizingMotion.BoneFrames.Get(grooveBone.Name()).Get(rightLegBoneDelta.Frame)
		sizingGrooveBf.Position = groovePositions[i]
		sizingMotion.InsertRegisteredBoneFrame(grooveBone.Name(), sizingGrooveBf)

		sizingLeftLegIkBf := sizingMotion.BoneFrames.Get(leftLegIkBone.Name()).Get(rightLegBoneDelta.Frame)
		sizingLeftLegIkBf.Position = leftLegIkPositions[i]
		sizingMotion.InsertRegisteredBoneFrame(leftLegIkBone.Name(), sizingLeftLegIkBf)
	}

	sizingCenterDeltas := make([]*delta.VmdDeltas, len(frames))

	// サイジング先モデルのデフォーム(IK OFF+センター補正済み)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, false, frame, leg_all_bone_names, false)
		sizingCenterDeltas[index] = vmdDeltas

		// 左足首から見た左足ボーンの相対位置を取得
		leftLegIkBoneDelta := vmdDeltas.Bones.Get(leftLegIkBone.Index())
		leftLegBoneDelta := vmdDeltas.Bones.Get(leftLegBone.Index())
		leftAnkleBoneDelta := vmdDeltas.Bones.Get(leftAnkleBone.Index())

		leftLegFkLocalPosition := leftLegBoneDelta.FilledGlobalPosition().Subed(
			leftLegIkBoneDelta.FilledGlobalPosition())
		leftLegIkLocalPosition := leftLegBoneDelta.FilledGlobalPosition().Subed(
			leftAnkleBoneDelta.FilledGlobalPosition())
		leftLegDiff := leftLegIkLocalPosition.Subed(leftLegFkLocalPosition)

		// 左足IK
		leftLegIkBf := sizingMotion.BoneFrames.Get(leftLegIkBone.Name()).Get(frame)
		leftLegIkPositions[index] = leftLegIkBf.Position.Subed(leftLegDiff)
	})

	// 右足IKを動かさなかった場合の左足首の位置を調整
	for i, iFrame := range frames {
		frame := float32(iFrame)

		leftLegIkBf := sizingMotion.BoneFrames.Get(leftLegIkBone.Name()).Get(frame)
		leftLegIkBf.Position = leftLegIkPositions[i]
		sizingMotion.InsertRegisteredBoneFrame(leftLegIkBone.Name(), leftLegIkBf)
	}

	leftAdjustYs := make(map[float32]float64)
	for i, iFrame := range leftZeroFrames {
		if i == 0 {
			continue
		}
		frame := float32(iFrame)
		prevFrame := float32(leftZeroFrames[i-1])

		leftLegIkBf := sizingMotion.BoneFrames.Get(leftLegIkBone.Name()).Get(frame)
		leftLegIkY := 0 - leftLegIkBf.Position.Y

		prevLeftLegIkBf := sizingMotion.BoneFrames.Get(leftLegIkBone.Name()).Get(prevFrame)
		prevLeftLegIkY := 0 - prevLeftLegIkBf.Position.Y

		// Y=0の間を線形補間
		for f := prevFrame; f <= frame; f++ {
			y := mmath.LerpFloat(prevLeftLegIkY, leftLegIkY, float64(f-prevFrame)/float64(frame-prevFrame))
			leftAdjustYs[f] = y
		}
	}

	rightAdjustYs := make(map[float32]float64)
	for i, iFrame := range rightZeroFrames {
		if i == 0 {
			continue
		}
		frame := float32(iFrame)
		prevFrame := float32(rightZeroFrames[i-1])

		rightLegIkBf := sizingMotion.BoneFrames.Get(rightLegIkBone.Name()).Get(frame)
		rightLegIkY := 0 - rightLegIkBf.Position.Y

		prevRightLegIkBf := sizingMotion.BoneFrames.Get(rightLegIkBone.Name()).Get(prevFrame)
		prevRightLegIkY := 0 - prevRightLegIkBf.Position.Y

		// Y=0の間を線形補間
		for f := prevFrame; f <= frame; f++ {
			y := mmath.LerpFloat(prevRightLegIkY, rightLegIkY, float64(f-prevFrame)/float64(frame-prevFrame))
			rightAdjustYs[f] = y
		}
	}

	for i, iFrame := range frames {
		frame := float32(iFrame)

		ly := 0.0
		if y, ok := leftAdjustYs[frame]; ok {
			ly = y
		}

		ry := 0.0
		if y, ok := rightAdjustYs[frame]; ok {
			ry = y
		}

		grooveBf := sizingMotion.BoneFrames.Get(grooveBone.Name()).Get(frame)
		grooveBf.Position = groovePositions[i].Added(&mmath.MVec3{X: 0, Y: ly + ry, Z: 0})
		sizingMotion.InsertRegisteredBoneFrame(grooveBone.Name(), grooveBf)

		rightLegIkBf := sizingMotion.BoneFrames.Get(rightLegIkBone.Name()).Get(frame)
		rightLegIkBf.Position = rightLegIkPositions[i].Added(&mmath.MVec3{X: 0, Y: ly + ry, Z: 0})
		sizingMotion.InsertRegisteredBoneFrame(rightLegIkBone.Name(), rightLegIkBf)

		leftLegIkBf := sizingMotion.BoneFrames.Get(leftLegIkBone.Name()).Get(frame)
		leftLegIkBf.Position = leftLegIkPositions[i].Added(&mmath.MVec3{X: 0, Y: ly + ry, Z: 0})
		sizingMotion.InsertRegisteredBoneFrame(leftLegIkBone.Name(), leftLegIkBf)
	}
}
