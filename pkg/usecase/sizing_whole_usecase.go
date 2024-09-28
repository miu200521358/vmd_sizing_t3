package usecase

import (
	"fmt"

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

// 下半身＋足補正
func sizingWholeLowerLegStance(sizingSet *model.SizingSet, scales *mmath.MVec3) {
	originalModel := sizingSet.OriginalPmx
	originalMotion := sizingSet.OriginalVmd
	sizingModel := sizingSet.SizingPmx
	sizingMotion := sizingSet.OutputVmd

	originalLegCenterBone := originalModel.Bones.GetByName(pmx.LEG_CENTER.String())
	originalLowerBone := originalModel.Bones.GetByName(pmx.LOWER.String())
	originalLeftBone := originalModel.Bones.GetByName(pmx.LEG.Left())
	originalRightBone := originalModel.Bones.GetByName(pmx.LEG.Right())

	centerBone := sizingModel.Bones.GetByName(pmx.CENTER.String())
	grooveBone := sizingModel.Bones.GetByName(pmx.GROOVE.String())
	lowerBone := sizingModel.Bones.GetByName(pmx.LOWER.String())
	legCenterBone := sizingModel.Bones.GetByName(pmx.LEG_CENTER.String())
	rightLegIkBone := sizingModel.Bones.GetByName(pmx.LEG_IK.Right())
	rightToeBone := sizingModel.Bones.Get(sizingModel.Bones.GetByName(pmx.TOE_IK.Right()).Ik.BoneIndex)
	rightToeIkBone := sizingModel.Bones.GetByName(pmx.TOE_IK.Right())
	rightKneeBone := sizingModel.Bones.GetByName(pmx.KNEE.Right())
	rightLegBone := sizingModel.Bones.GetByName(pmx.LEG.Right())
	rightAnkleBone := sizingModel.Bones.Get(sizingModel.Bones.GetByName(pmx.LEG_IK.Right()).Ik.BoneIndex)
	leftLegIkBone := sizingModel.Bones.GetByName(pmx.LEG_IK.Left())
	leftToeBone := sizingModel.Bones.Get(sizingModel.Bones.GetByName(pmx.TOE_IK.Left()).Ik.BoneIndex)
	leftToeIkBone := sizingModel.Bones.GetByName(pmx.TOE_IK.Left())
	leftKneeBone := sizingModel.Bones.GetByName(pmx.KNEE.Left())
	leftLegBone := sizingModel.Bones.GetByName(pmx.LEG.Left())
	leftAnkleBone := sizingModel.Bones.Get(sizingModel.Bones.GetByName(pmx.LEG_IK.Left()).Ik.BoneIndex)

	// 左ひざIK
	leftKneeIkBone := pmx.NewBoneByName(fmt.Sprintf("%s%sIk", pmx.MLIB_PREFIX, leftKneeBone.Name()))
	leftKneeIkBone.Ik = pmx.NewIk()
	leftKneeIkBone.Ik.BoneIndex = leftKneeBone.Index()
	leftKneeIkBone.Ik.LoopCount = leftLegIkBone.Ik.LoopCount
	leftKneeIkBone.Ik.UnitRotation = leftLegIkBone.Ik.UnitRotation
	leftKneeIkBone.Ik.Links = make([]*pmx.IkLink, 1)
	leftKneeIkBone.Ik.Links[0] = pmx.NewIkLink()
	leftKneeIkBone.Ik.Links[0].BoneIndex = leftLegBone.Index()

	// 右ひざIK
	rightKneeIkBone := pmx.NewBoneByName(fmt.Sprintf("%s%sIk", pmx.MLIB_PREFIX, rightKneeBone.Name()))
	rightKneeIkBone.Ik = pmx.NewIk()
	rightKneeIkBone.Ik.BoneIndex = rightKneeBone.Index()
	rightKneeIkBone.Ik.LoopCount = rightLegIkBone.Ik.LoopCount
	rightKneeIkBone.Ik.UnitRotation = rightLegIkBone.Ik.UnitRotation
	rightKneeIkBone.Ik.Links = make([]*pmx.IkLink, 1)
	rightKneeIkBone.Ik.Links[0] = pmx.NewIkLink()
	rightKneeIkBone.Ik.Links[0].BoneIndex = rightLegBone.Index()

	// 左足首IK
	leftAnkleIkBone := pmx.NewBoneByName(fmt.Sprintf("%s%sIk", pmx.MLIB_PREFIX, leftAnkleBone.Name()))
	leftAnkleIkBone.Ik = pmx.NewIk()
	leftAnkleIkBone.Ik.BoneIndex = leftAnkleBone.Index()
	leftAnkleIkBone.Ik.LoopCount = leftLegIkBone.Ik.LoopCount
	leftAnkleIkBone.Ik.UnitRotation = leftLegIkBone.Ik.UnitRotation
	leftAnkleIkBone.Ik.Links = make([]*pmx.IkLink, 1)
	leftAnkleIkBone.Ik.Links[0] = pmx.NewIkLink()
	leftAnkleIkBone.Ik.Links[0].BoneIndex = leftKneeBone.Index()
	leftAnkleIkBone.Ik.Links[0].AngleLimit = true
	leftAnkleIkBone.Ik.Links[0].MinAngleLimit = leftLegIkBone.Ik.Links[0].MinAngleLimit.Copy()
	leftAnkleIkBone.Ik.Links[0].MaxAngleLimit = leftLegIkBone.Ik.Links[0].MaxAngleLimit.Copy()

	// 右足首IK
	rightAnkleIkBone := pmx.NewBoneByName(fmt.Sprintf("%s%sIk", pmx.MLIB_PREFIX, rightAnkleBone.Name()))
	rightAnkleIkBone.Ik = pmx.NewIk()
	rightAnkleIkBone.Ik.BoneIndex = rightAnkleBone.Index()
	rightAnkleIkBone.Ik.LoopCount = rightLegIkBone.Ik.LoopCount
	rightAnkleIkBone.Ik.UnitRotation = rightLegIkBone.Ik.UnitRotation
	rightAnkleIkBone.Ik.Links = make([]*pmx.IkLink, 1)
	rightAnkleIkBone.Ik.Links[0] = pmx.NewIkLink()
	rightAnkleIkBone.Ik.Links[0].BoneIndex = rightKneeBone.Index()
	rightAnkleIkBone.Ik.Links[0].AngleLimit = true
	rightAnkleIkBone.Ik.Links[0].MinAngleLimit = rightLegIkBone.Ik.Links[0].MinAngleLimit
	rightAnkleIkBone.Ik.Links[0].MaxAngleLimit = rightLegIkBone.Ik.Links[0].MaxAngleLimit

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
	lowerRotations := make([]*mmath.MQuaternion, len(frames))

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

		originalLeftLegDelta := originalAllDeltas[index].Bones.Get(originalLeftBone.Index())
		originalRightLegDelta := originalAllDeltas[index].Bones.Get(originalRightBone.Index())
		originalLegUp := originalLeftLegDelta.FilledGlobalPosition().Subed(
			originalRightLegDelta.FilledGlobalPosition())
		originalLegDirection := originalLegCenterDelta.FilledGlobalPosition().Subed(
			originalLowerDelta.FilledGlobalPosition())
		originalLegSlope := mmath.NewMQuaternionFromDirection(
			originalLegDirection.Normalized(), originalLegUp.Normalized())

		sizingLeftLegDelta := vmdDeltas.Bones.Get(leftLegBone.Index())
		sizingRightLegDelta := vmdDeltas.Bones.Get(rightLegBone.Index())
		sizingLegUp := sizingLeftLegDelta.FilledGlobalPosition().Subed(
			sizingRightLegDelta.FilledGlobalPosition())
		sizingLegDirection := legCenterDelta.FilledGlobalPosition().Subed(
			lowerDelta.FilledGlobalPosition())
		sizingLegSlope := mmath.NewMQuaternionFromDirection(
			sizingLegDirection.Normalized(), sizingLegUp.Normalized())

		// 下半身の向きを元モデルと同じにする
		lowerBf := sizingMotion.BoneFrames.Get(lowerBone.Name()).Get(frame)
		lowerOffsetRotation := originalLegSlope.Muled(sizingLegSlope.Inverted())
		lowerRotations[index] = lowerOffsetRotation.Muled(lowerBf.Rotation)
		lowerDiff := sizingLowerGlobalPosition.Subed(sizingFixLowerGlobalPosition)

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
	}

	mlog.I(mi18n.T("全身位置角度合わせ05", map[string]interface{}{"No": sizingSet.Index + 1}))

	sizingCenterDeltas := make([]*delta.VmdDeltas, len(frames))
	leftLegIkPositions := make([]*mmath.MVec3, len(frames))

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

		originalRightLegIkBf := originalMotion.BoneFrames.Get(rightLegIkBone.Name()).Get(frame)
		originalLeftLegIkBf := originalMotion.BoneFrames.Get(leftLegIkBone.Name()).Get(frame)

		if mmath.NearEquals(originalRightLegIkBf.Position.Y, 0.0, 1e-2) {
			rightLegIkBf.Position.Y = 0
			sizingMotion.InsertRegisteredBoneFrame(rightLegIkBone.Name(), rightLegIkBf)
		}
		if mmath.NearEquals(originalLeftLegIkBf.Position.Y, 0.0, 1e-2) {
			leftLegIkBf.Position.Y = 0
			sizingMotion.InsertRegisteredBoneFrame(leftLegIkBone.Name(), leftLegIkBf)
		}

		if i > 0 {
			originalPrevRightLegIkBf := originalMotion.BoneFrames.Get(rightLegIkBone.Name()).Get(float32(frames[i-1]))
			if originalPrevRightLegIkBf.Position.NearEquals(originalRightLegIkBf.Position, 1e-2) {
				rightPrevLegIkBf := sizingMotion.BoneFrames.Get(rightLegIkBone.Name()).Get(float32(frames[i-1]))
				rightLegIkBf.Position = rightPrevLegIkBf.Position.Copy()
				sizingMotion.InsertRegisteredBoneFrame(rightLegIkBone.Name(), rightLegIkBf)
			}

			originalPrevLeftLegIkBf := originalMotion.BoneFrames.Get(leftLegIkBone.Name()).Get(float32(frames[i-1]))
			if originalPrevLeftLegIkBf.Position.NearEquals(originalLeftLegIkBf.Position, 1e-2) {
				leftPrevLegIkBf := sizingMotion.BoneFrames.Get(leftLegIkBone.Name()).Get(float32(frames[i-1]))
				leftLegIkBf.Position = leftPrevLegIkBf.Position.Copy()
				sizingMotion.InsertRegisteredBoneFrame(leftLegIkBone.Name(), leftLegIkBf)
			}
		}
	}

	mlog.I(mi18n.T("全身位置角度合わせ08", map[string]interface{}{"No": sizingSet.Index + 1}))
	sizingLegIkOnDeltas := make([]*delta.VmdDeltas, len(frames))

	// サイジング先モデルのデフォーム(IK ON+センター・足周り補正済み)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, true, frame, leg_all_bone_names, false)
		sizingLegIkOnDeltas[index] = vmdDeltas
	})

	mlog.I(mi18n.T("全身位置角度合わせ09", map[string]interface{}{"No": sizingSet.Index + 1}))

	for i, iFrame := range frames {
		frame := float32(iFrame)

		sizingLowerBf := sizingMotion.BoneFrames.Get(lowerBone.Name()).Get(frame)
		sizingLowerBf.Rotation = lowerRotations[i]
		sizingMotion.InsertRegisteredBoneFrame(lowerBone.Name(), sizingLowerBf)
	}

	mlog.I(mi18n.T("全身位置角度合わせ10", map[string]interface{}{"No": sizingSet.Index + 1}))

	leftLegRotations := make([]*mmath.MQuaternion, len(frames))
	rightLegRotations := make([]*mmath.MQuaternion, len(frames))

	// 先モデルの足角度追加補正(IK ON)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		// ひざを固定した場合の足の回転
		{
			vmdDeltas := deform.DeformIk(sizingModel, sizingMotion, frame, leftKneeIkBone,
				sizingLegIkOnDeltas[index].Bones.Get(leftKneeBone.Index()).FilledGlobalPosition())
			leftLegRotations[index] = vmdDeltas.Bones.Get(leftLegBone.Index()).FilledFrameRotation()
		}
		{
			vmdDeltas := deform.DeformIk(sizingModel, sizingMotion, frame, rightKneeIkBone,
				sizingLegIkOnDeltas[index].Bones.Get(rightKneeBone.Index()).FilledGlobalPosition())
			rightLegRotations[index] = vmdDeltas.Bones.Get(rightLegBone.Index()).FilledFrameRotation()
		}
	})

	mlog.I(mi18n.T("全身位置角度合わせ11", map[string]interface{}{"No": sizingSet.Index + 1}))

	// 補正を登録
	for i, iFrame := range frames {
		frame := float32(iFrame)

		sizingRightLegBf := sizingMotion.BoneFrames.Get(rightLegBone.Name()).Get(frame)
		sizingRightLegBf.Rotation = rightLegRotations[i]
		sizingMotion.InsertRegisteredBoneFrame(rightLegBone.Name(), sizingRightLegBf)

		sizingLeftLegBf := sizingMotion.BoneFrames.Get(leftLegBone.Name()).Get(frame)
		sizingLeftLegBf.Rotation = leftLegRotations[i]
		sizingMotion.InsertRegisteredBoneFrame(leftLegBone.Name(), sizingLeftLegBf)
	}

	mlog.I(mi18n.T("全身位置角度合わせ12", map[string]interface{}{"No": sizingSet.Index + 1}))

	leftKneeRotations := make([]*mmath.MQuaternion, len(frames))
	rightKneeRotations := make([]*mmath.MQuaternion, len(frames))
	// leftLegIkPositions = make([]*mmath.MVec3, len(frames))
	// rightLegIkPositions = make([]*mmath.MVec3, len(frames))

	// 先モデルの足角度追加補正(IK ON)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		// 足首を固定した場合のひざの回転
		{
			vmdDeltas := deform.DeformIk(sizingModel, sizingMotion, frame, leftAnkleIkBone,
				sizingLegIkOnDeltas[index].Bones.Get(leftAnkleBone.Index()).FilledGlobalPosition())
			leftKneeRotations[index] = vmdDeltas.Bones.Get(leftKneeBone.Index()).FilledFrameRotation()

			// // 足IKから見た左足首の位置
			// leftLegIkPositions[index] = vmdDeltas.Bones.Get(leftAnkleBone.Index()).FilledGlobalPosition().Subed(leftLegIkBone.Position)
		}
		{
			vmdDeltas := deform.DeformIk(sizingModel, sizingMotion, frame, rightAnkleIkBone,
				sizingLegIkOnDeltas[index].Bones.Get(rightAnkleBone.Index()).FilledGlobalPosition())
			rightKneeRotations[index] = vmdDeltas.Bones.Get(rightKneeBone.Index()).FilledFrameRotation()

			// // 足IKから見た右足首の位置
			// rightLegIkPositions[index] = vmdDeltas.Bones.Get(rightAnkleBone.Index()).FilledGlobalPosition().Subed(rightLegIkBone.Position)
		}
	})

	mlog.I(mi18n.T("全身位置角度合わせ13", map[string]interface{}{"No": sizingSet.Index + 1}))

	// 補正を登録
	for i, iFrame := range frames {
		frame := float32(iFrame)

		sizingRightKneeBf := sizingMotion.BoneFrames.Get(rightKneeBone.Name()).Get(frame)
		sizingRightKneeBf.Rotation = rightKneeRotations[i]
		sizingMotion.InsertRegisteredBoneFrame(rightKneeBone.Name(), sizingRightKneeBf)

		sizingLeftKneeBf := sizingMotion.BoneFrames.Get(leftKneeBone.Name()).Get(frame)
		sizingLeftKneeBf.Rotation = leftKneeRotations[i]
		sizingMotion.InsertRegisteredBoneFrame(leftKneeBone.Name(), sizingLeftKneeBf)
	}

	mlog.I(mi18n.T("全身位置角度合わせ14", map[string]interface{}{"No": sizingSet.Index + 1}))

	leftAnkleRotations := make([]*mmath.MQuaternion, len(frames))
	rightAnkleRotations := make([]*mmath.MQuaternion, len(frames))

	// 先モデルの足角度追加補正(IK ON)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		// つま先を固定した場合の足首の回転
		{
			vmdDeltas := deform.DeformIk(sizingModel, sizingMotion, frame, leftToeIkBone,
				sizingLegIkOnDeltas[index].Bones.Get(leftToeBone.Index()).FilledGlobalPosition())
			leftAnkleRotations[index] = vmdDeltas.Bones.Get(leftAnkleBone.Index()).FilledFrameRotation()
		}
		{
			vmdDeltas := deform.DeformIk(sizingModel, sizingMotion, frame, rightToeIkBone,
				sizingLegIkOnDeltas[index].Bones.Get(rightToeBone.Index()).FilledGlobalPosition())
			rightAnkleRotations[index] = vmdDeltas.Bones.Get(rightAnkleBone.Index()).FilledFrameRotation()
		}
	})

	mlog.I(mi18n.T("全身位置角度合わせ15", map[string]interface{}{"No": sizingSet.Index + 1}))

	// 補正を登録
	for i, iFrame := range frames {
		frame := float32(iFrame)

		sizingLeftAnkleBf := sizingMotion.BoneFrames.Get(leftAnkleBone.Name()).Get(frame)
		sizingLeftAnkleBf.Rotation = leftAnkleRotations[i]
		sizingMotion.InsertRegisteredBoneFrame(leftAnkleBone.Name(), sizingLeftAnkleBf)

		sizingRightAnkleBf := sizingMotion.BoneFrames.Get(rightAnkleBone.Name()).Get(frame)
		sizingRightAnkleBf.Rotation = rightAnkleRotations[i]
		sizingMotion.InsertRegisteredBoneFrame(rightAnkleBone.Name(), sizingRightAnkleBf)

		// sizingLeftIkBf := sizingMotion.BoneFrames.Get(leftLegIkBone.Name()).Get(frame)
		// sizingLeftIkBf.Position = leftLegIkPositions[i]
		// sizingMotion.InsertRegisteredBoneFrame(leftLegIkBone.Name(), sizingLeftIkBf)

		// sizingRightIkBf := sizingMotion.BoneFrames.Get(rightLegIkBone.Name()).Get(frame)
		// sizingRightIkBf.Position = rightLegIkPositions[i]
		// sizingMotion.InsertRegisteredBoneFrame(rightLegIkBone.Name(), sizingRightIkBf)
	}
}
