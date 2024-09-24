package usecase

import (
	"github.com/miu200521358/mlib_go/pkg/domain/delta"
	"github.com/miu200521358/mlib_go/pkg/domain/miter"
	"github.com/miu200521358/mlib_go/pkg/domain/mmath"
	"github.com/miu200521358/mlib_go/pkg/domain/pmx"
	"github.com/miu200521358/mlib_go/pkg/infrastructure/deform"
	"github.com/miu200521358/mlib_go/pkg/mutils/mi18n"
	"github.com/miu200521358/mlib_go/pkg/mutils/mlog"
	"github.com/miu200521358/vmd_sizing_t3/pkg/model"
)

// 全身位置角度合わせ
func SizingWholeStance(sizingSet *model.SizingSet, scales *mmath.MVec3) {
	if !sizingSet.IsSizingWholeStance || (sizingSet.IsSizingWholeStance && sizingSet.CompletedSizingWholeStance) {
		return
	}

	if !isValidWholeStance(sizingSet) {
		return
	}

	// 下半身・足補正
	sizingWholeLowerLegStance(sizingSet, scales)
}

func isValidWholeStance(sizingSet *model.SizingSet) bool {
	originalModel := sizingSet.OriginalPmx
	sizingModel := sizingSet.SizingPmx

	// センター、グルーブ、下半身、右足IK、左足IKが存在するか

	if !originalModel.Bones.ContainsByName(pmx.CENTER.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("全身位置角度合わせボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.CENTER.String()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.GROOVE.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("全身位置角度合わせボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.GROOVE.String()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.LOWER.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("全身位置角度合わせボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.LOWER.String()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.LEG_IK.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("全身位置角度合わせボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.LEG_IK.Right()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.LEG_IK.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("全身位置角度合わせボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.LEG_IK.Left()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.CENTER.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("全身位置角度合わせボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("サイジング先モデル"), "BoneName": pmx.CENTER.String()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.GROOVE.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("全身位置角度合わせボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("サイジング先モデル"), "BoneName": pmx.GROOVE.String()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.LOWER.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("全身位置角度合わせボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("サイジング先モデル"), "BoneName": pmx.LOWER.String()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.LEG_IK.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("全身位置角度合わせボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("サイジング先モデル"), "BoneName": pmx.LEG_IK.Right()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.LEG_IK.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("全身位置角度合わせボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("サイジング先モデル"), "BoneName": pmx.LEG_IK.Left()}))
		return false
	}

	return true
}

func sizingWholeLowerLegStance(sizingSet *model.SizingSet, scales *mmath.MVec3) {
	// 足補正
	originalModel := sizingSet.OriginalPmx
	originalMotion := sizingSet.OriginalVmd
	sizingModel := sizingSet.SizingPmx
	sizingMotion := sizingSet.OutputVmd

	originalLegCenterBone := originalModel.Bones.GetByName(pmx.LEG_CENTER.String())
	originalLowerBone := originalModel.Bones.GetByName(pmx.LOWER.String())
	// originalLeftBone := originalModel.Bones.GetByName(pmx.LEG.Left())
	// originalRightBone := originalModel.Bones.GetByName(pmx.LEG.Right())

	centerBone := sizingModel.Bones.GetByName(pmx.CENTER.String())
	grooveBone := sizingModel.Bones.GetByName(pmx.GROOVE.String())
	lowerBone := sizingModel.Bones.GetByName(pmx.LOWER.String())
	legCenterBone := sizingModel.Bones.GetByName(pmx.LEG_CENTER.String())
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

	mlog.I(mi18n.T("全身位置角度合わせ01", map[string]interface{}{"No": sizingSet.Index + 1}))

	// 元モデルのデフォーム(IK ON)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		vmdDeltas := delta.NewVmdDeltas(frame, originalModel.Bones, originalModel.Hash(), originalMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(originalModel, originalMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(originalModel, originalMotion, vmdDeltas, true, frame, leg_all_bone_names, false)
		originalAllDeltas[index] = vmdDeltas
	})

	mlog.I(mi18n.T("全身位置角度合わせ02", map[string]interface{}{"No": sizingSet.Index + 1}))

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
	centerPositions := make([]*mmath.MVec3, len(frames))
	groovePositions := make([]*mmath.MVec3, len(frames))
	rightLegIkPositions := make([]*mmath.MVec3, len(frames))
	// lowerRotations := make([]*mmath.MQuaternion, len(frames))
	// rightLegRotations := make([]*mmath.MQuaternion, len(frames))
	// leftLegRotations := make([]*mmath.MQuaternion, len(frames))

	mlog.I(mi18n.T("全身位置角度合わせ03", map[string]interface{}{"No": sizingSet.Index + 1}))

	// サイジング先モデルのデフォーム(IK OFF)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, false, frame, leg_all_bone_names, false)
		sizingOffDeltas[index] = vmdDeltas

		// 右足首から見た右足ボーンの相対位置を取得
		rightLegIkDelta := vmdDeltas.Bones.Get(rightLegIkBone.Index())
		rightLegDelta := vmdDeltas.Bones.Get(rightLegBone.Index())
		rightAnkleDelta := vmdDeltas.Bones.Get(rightAnkleBone.Index())

		rightLegFkLocalPosition := rightAnkleDelta.FilledGlobalPosition().Subed(
			rightLegDelta.FilledGlobalPosition())
		rightLegIkLocalPosition := rightLegIkDelta.FilledGlobalPosition().Subed(
			rightLegDelta.FilledGlobalPosition())
		rightLegDiff := rightLegIkLocalPosition.Subed(rightLegFkLocalPosition)

		// 足中心から見た下半身ボーンの相対位置を取得
		originalLegCenterDelta := originalAllDeltas[index].Bones.Get(originalLegCenterBone.Index())
		originalLowerDelta := originalAllDeltas[index].Bones.Get(originalLowerBone.Index())
		originalLegCenterLocalPosition := originalLegCenterDelta.FilledGlobalMatrix().Inverted().MulVec3(
			originalLowerDelta.FilledGlobalPosition())

		// サイジング先の足中心から、オリジナルの下半身位置を加算した時の結果
		legCenterDelta := vmdDeltas.Bones.Get(legCenterBone.Index())
		lowerDelta := vmdDeltas.Bones.Get(lowerBone.Index())
		sizingLowerGlobalPosition := lowerDelta.FilledGlobalPosition()
		sizingFixLowerGlobalPosition := legCenterDelta.FilledGlobalMatrix().MulVec3(originalLegCenterLocalPosition)

		// originalLeftLegDelta := originalAllDeltas[index].Bones.Get(originalLeftBone.Index())
		// originalRightLegDelta := originalAllDeltas[index].Bones.Get(originalRightBone.Index())
		// originalLegUp := originalLeftLegDelta.FilledGlobalPosition().Subed(
		// 	originalRightLegDelta.FilledGlobalPosition())
		// originalLegDirection := originalLegCenterDelta.FilledGlobalPosition().Subed(
		// 	originalLowerDelta.FilledGlobalPosition())
		// originalLegSlope := mmath.NewMQuaternionFromDirection(
		// 	originalLegDirection.Normalized(), originalLegUp.Normalized())

		// sizingLeftLegDelta := vmdDeltas.Bones.Get(leftLegBone.Index())
		// sizingRightLegDelta := vmdDeltas.Bones.Get(rightLegBone.Index())
		// sizingLegUp := sizingLeftLegDelta.FilledGlobalPosition().Subed(
		// 	sizingRightLegDelta.FilledGlobalPosition())
		// sizingLegDirection := legCenterDelta.FilledGlobalPosition().Subed(
		// 	lowerDelta.FilledGlobalPosition())
		// sizingLegSlope := mmath.NewMQuaternionFromDirection(
		// 	sizingLegDirection.Normalized(), sizingLegUp.Normalized())

		// // 下半身の向きを元モデルと同じにする
		// lowerBf := sizingMotion.BoneFrames.Get(lowerBone.Name()).Get(frame)
		// lowerOffsetRotation := originalLegSlope.Muled(sizingLegSlope.Inverted())
		// lowerRotations[index] = lowerOffsetRotation.Muled(lowerBf.Rotation)
		lowerDiff := sizingLowerGlobalPosition.Subed(sizingFixLowerGlobalPosition)

		// rightLegBf := sizingMotion.BoneFrames.Get(rightLegBone.Name()).Get(frame)
		// rightLegRotations[index] = lowerOffsetRotation.Muled(rightLegBf.Rotation)

		// leftLegBf := sizingMotion.BoneFrames.Get(leftLegBone.Name()).Get(frame)
		// leftLegRotations[index] = lowerOffsetRotation.Muled(leftLegBf.Rotation)

		// 右足IKを動かさなかった場合のセンターと左足IKの位置を調整する用の値を保持
		// （この時点でキーフレに追加すると動きが変わる）
		centerBf := sizingMotion.BoneFrames.Get(centerBone.Name()).Get(frame)
		centerPositions[index] = centerBf.Position.Added(
			&mmath.MVec3{X: rightLegDiff.X + lowerDiff.X, Y: 0, Z: rightLegDiff.Z + lowerDiff.Z})

		grooveBf := sizingMotion.BoneFrames.Get(grooveBone.Name()).Get(frame)
		groovePositions[index] = grooveBf.Position.Added(
			&mmath.MVec3{X: 0, Y: rightLegDiff.Y + lowerDiff.Y, Z: 0})

		rightLegIkBf := sizingMotion.BoneFrames.Get(rightLegIkBone.Name()).Get(frame)
		rightLegIkPositions[index] = rightLegIkBf.Position
	})

	mlog.I(mi18n.T("全身位置角度合わせ04", map[string]interface{}{"No": sizingSet.Index + 1}))

	// 補正を登録
	for i, vmdDeltas := range sizingOffDeltas {
		rightLegBoneDelta := vmdDeltas.Bones.Get(rightLegBone.Index())

		sizingCenterBf := sizingMotion.BoneFrames.Get(centerBone.Name()).Get(rightLegBoneDelta.Frame)
		sizingCenterBf.Position = centerPositions[i]
		sizingMotion.InsertRegisteredBoneFrame(centerBone.Name(), sizingCenterBf)

		sizingGrooveBf := sizingMotion.BoneFrames.Get(grooveBone.Name()).Get(rightLegBoneDelta.Frame)
		sizingGrooveBf.Position = groovePositions[i]
		sizingMotion.InsertRegisteredBoneFrame(grooveBone.Name(), sizingGrooveBf)

		// sizingLowerBf := sizingMotion.BoneFrames.Get(lowerBone.Name()).Get(rightLegBoneDelta.Frame)
		// sizingLowerBf.Rotation = lowerRotations[i]
		// sizingMotion.InsertRegisteredBoneFrame(lowerBone.Name(), sizingLowerBf)

		// sizingRightLegBf := sizingMotion.BoneFrames.Get(rightLegBone.Name()).Get(rightLegBoneDelta.Frame)
		// sizingRightLegBf.Rotation = rightLegRotations[i]
		// sizingMotion.InsertRegisteredBoneFrame(rightLegBone.Name(), sizingRightLegBf)

		// sizingLeftLegBf := sizingMotion.BoneFrames.Get(leftLegBone.Name()).Get(rightLegBoneDelta.Frame)
		// sizingLeftLegBf.Rotation = leftLegRotations[i]
		// sizingMotion.InsertRegisteredBoneFrame(leftLegBone.Name(), sizingLeftLegBf)
	}

	sizingCenterDeltas := make([]*delta.VmdDeltas, len(frames))
	leftLegIkPositions := make([]*mmath.MVec3, len(frames))

	mlog.I(mi18n.T("全身位置角度合わせ05", map[string]interface{}{"No": sizingSet.Index + 1}))

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

	mlog.I(mi18n.T("全身位置角度合わせ06", map[string]interface{}{"No": sizingSet.Index + 1}))

	offsetXZs := make(map[float32]*mmath.MVec3)
	offsetYs := make(map[float32]*mmath.MVec3)

	// 左足の結果を登録＆センターのオフセットを保持
	for i, iFrame := range frames {
		frame := float32(iFrame)

		leftLegIkBf := sizingMotion.BoneFrames.Get(leftLegIkBone.Name()).Get(frame)
		leftLegIkBf.Position = leftLegIkPositions[i]
		sizingMotion.InsertRegisteredBoneFrame(leftLegIkBone.Name(), leftLegIkBf)

		originalLeftLegIkBf := originalMotion.BoneFrames.Get(leftLegIkBone.Name()).Get(frame)
		scaledLeftLegIk := originalLeftLegIkBf.Position.Muled(scales)

		// 右足首基準で求めたので、左足首基準で補正
		y := scaledLeftLegIk.Y - leftLegIkPositions[i].Y
		if rightLegIkPositions[i].Y+y < 0 {
			y += -(rightLegIkPositions[i].Y + y)
		}

		originalCenterBf := originalMotion.BoneFrames.Get(centerBone.Name()).Get(frame)
		scaledCenter := originalCenterBf.Position.Muled(scales)

		x := scaledCenter.X - centerPositions[i].X
		z := scaledCenter.Z - centerPositions[i].Z

		offsetXZs[frame] = &mmath.MVec3{X: x, Y: 0, Z: z}
		offsetYs[frame] = &mmath.MVec3{X: 0, Y: y, Z: 0}
	}

	mlog.I(mi18n.T("全身位置角度合わせ07", map[string]interface{}{"No": sizingSet.Index + 1}))

	for i, iFrame := range frames {
		frame := float32(iFrame)

		offsetPos := mmath.NewMVec3()

		if offsetXZ, ok := offsetXZs[frame]; ok {
			centerBf := sizingMotion.BoneFrames.Get(centerBone.Name()).Get(frame)
			centerBf.Position = centerPositions[i].Added(offsetXZ)
			sizingMotion.InsertRegisteredBoneFrame(centerBone.Name(), centerBf)

			offsetPos.Add(offsetXZ)
		}

		if offsetY, ok := offsetYs[frame]; ok {
			grooveBf := sizingMotion.BoneFrames.Get(grooveBone.Name()).Get(frame)
			grooveBf.Position = groovePositions[i].Added(offsetY)
			sizingMotion.InsertRegisteredBoneFrame(grooveBone.Name(), grooveBf)

			offsetPos.Add(offsetY)
		}

		rightLegIkBf := sizingMotion.BoneFrames.Get(rightLegIkBone.Name()).Get(frame)
		rightLegIkBf.Position = rightLegIkPositions[i].Added(offsetPos)
		sizingMotion.InsertRegisteredBoneFrame(rightLegIkBone.Name(), rightLegIkBf)

		leftLegIkBf := sizingMotion.BoneFrames.Get(leftLegIkBone.Name()).Get(frame)
		leftLegIkBf.Position = leftLegIkPositions[i].Added(offsetPos)
		sizingMotion.InsertRegisteredBoneFrame(leftLegIkBone.Name(), leftLegIkBf)
	}
}
