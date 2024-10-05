package usecase

import (
	"github.com/miu200521358/mlib_go/pkg/domain/delta"
	"github.com/miu200521358/mlib_go/pkg/domain/miter"
	"github.com/miu200521358/mlib_go/pkg/domain/mmath"
	"github.com/miu200521358/mlib_go/pkg/domain/pmx"
	"github.com/miu200521358/mlib_go/pkg/domain/vmd"
	"github.com/miu200521358/mlib_go/pkg/infrastructure/deform"
	"github.com/miu200521358/mlib_go/pkg/infrastructure/repository"
	"github.com/miu200521358/mlib_go/pkg/mutils"
	"github.com/miu200521358/mlib_go/pkg/mutils/mi18n"
	"github.com/miu200521358/mlib_go/pkg/mutils/mlog"
	"github.com/miu200521358/vmd_sizing_t3/pkg/domain"
)

func getOriginalDeltas(sizingSet *domain.SizingSet) ([]int, []*delta.VmdDeltas) {

	originalModel := sizingSet.OriginalPmx
	originalMotion := sizingSet.OriginalVmd

	leftFrames := originalMotion.BoneFrames.RegisteredFrames(leg_direction_bone_names[0])
	rightFrames := originalMotion.BoneFrames.RegisteredFrames(leg_direction_bone_names[1])
	trunkFrames := originalMotion.BoneFrames.RegisteredFrames(trunk_lower_bone_names)

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

	return frames, originalAllDeltas
}

func SizingLeg(sizingSet *domain.SizingSet, scale *mmath.MVec3) ([]int, []*delta.VmdDeltas) {
	if !sizingSet.IsSizingLeg || (sizingSet.IsSizingLeg && sizingSet.CompletedSizingLeg) {
		return nil, nil
	}

	if !isValidSizingLower(sizingSet) {
		return nil, nil
	}

	originalModel := sizingSet.OriginalPmx
	originalMotion := sizingSet.OriginalVmd
	sizingModel := sizingSet.SizingPmx
	sizingMotion := sizingSet.OutputVmd

	originalLeftAnkleBone := originalModel.Bones.GetByName(pmx.ANKLE.Left())
	// originalLeftToeTailBone := originalModel.Bones.GetByName(pmx.TOE_T.Left())
	// originalLeftHeelBone := originalModel.Bones.GetByName(pmx.HEEL.Left())
	originalRightAnkleBone := originalModel.Bones.GetByName(pmx.ANKLE.Right())
	// originalRightToeTailBone := originalModel.Bones.GetByName(pmx.TOE_T.Right())
	// originalRightHeelBone := originalModel.Bones.GetByName(pmx.HEEL.Right())

	centerBone := sizingModel.Bones.GetByName(pmx.CENTER.String())
	grooveBone := sizingModel.Bones.GetByName(pmx.GROOVE.String())

	leftLegIkBone := sizingModel.Bones.GetByName(pmx.LEG_IK.Left())
	leftLegBone := sizingModel.Bones.GetByName(pmx.LEG.Left())
	leftKneeBone := sizingModel.Bones.GetByName(pmx.KNEE.Left())
	leftAnkleBone := sizingModel.Bones.GetIkTarget(pmx.LEG_IK.Left())
	// leftHeelBone := sizingModel.Bones.GetByName(pmx.HEEL.Left())
	leftToeIkBone := sizingModel.Bones.GetByName(pmx.TOE_IK.Left())
	leftToeBone := sizingModel.Bones.GetIkTarget(pmx.TOE_IK.Left())
	// leftToeTailBone := sizingModel.Bones.GetByName(pmx.TOE_T.Left())

	rightLegIkBone := sizingModel.Bones.GetByName(pmx.LEG_IK.Right())
	rightLegBone := sizingModel.Bones.GetByName(pmx.LEG.Right())
	rightKneeBone := sizingModel.Bones.GetByName(pmx.KNEE.Right())
	rightAnkleBone := sizingModel.Bones.GetIkTarget(pmx.LEG_IK.Right())
	// rightHeelBone := sizingModel.Bones.GetByName(pmx.HEEL.Right())
	rightToeIkBone := sizingModel.Bones.GetByName(pmx.TOE_IK.Right())
	rightToeBone := sizingModel.Bones.GetIkTarget(pmx.TOE_IK.Right())
	// rightToeTailBone := sizingModel.Bones.GetByName(pmx.TOE_T.Right())

	mlog.I(mi18n.T("足補正開始", map[string]interface{}{"No": sizingSet.Index + 1}))

	frames, originalAllDeltas := getOriginalDeltas(sizingSet)

	mlog.I(mi18n.T("足補正01", map[string]interface{}{"No": sizingSet.Index + 1}))

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

	// 足IK・つま先IKを削除
	sizingMotion.BoneFrames.Delete(pmx.LEG_IK.Left())
	sizingMotion.BoneFrames.Delete(pmx.LEG_IK.Right())
	sizingMotion.BoneFrames.Delete(pmx.TOE_IK.Left())
	sizingMotion.BoneFrames.Delete(pmx.TOE_IK.Right())

	if mlog.IsVerbose() {
		outputPath := mutils.CreateOutputPath(sizingSet.OriginalVmdPath, "足補正01_FK焼き込み")
		repository.NewVmdRepository().Save(outputPath, sizingMotion, true)
		mlog.V("足補正01_FK焼き込み: %s", outputPath)
	}

	mlog.I(mi18n.T("足補正02", map[string]interface{}{"No": sizingSet.Index + 1}))

	sizingOffDeltas := make([]*delta.VmdDeltas, len(frames))
	centerPositions := make([]*mmath.MVec3, len(frames))
	groovePositions := make([]*mmath.MVec3, len(frames))

	// 先モデルのデフォーム(IK OFF)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, false, frame, leg_all_bone_names, false)
		sizingOffDeltas[index] = vmdDeltas

		originalAnkleLeftDelta := originalAllDeltas[index].Bones.GetByName(pmx.ANKLE.Left())
		originalAnkleRightDelta := originalAllDeltas[index].Bones.GetByName(pmx.ANKLE.Right())

		// 元モデルの足首位置
		originalLeftLegGlobalPosition := originalAnkleLeftDelta.FilledGlobalPosition()
		originalLeftAnkleY := originalLeftLegGlobalPosition.Y - originalLeftAnkleBone.Position.Y

		originalRightLegGlobalPosition := originalAnkleRightDelta.FilledGlobalPosition()
		originalRightAnkleY := originalRightLegGlobalPosition.Y - originalRightAnkleBone.Position.Y

		// 足首のY座標を元モデルの足首のY座標*スケールに合わせる
		sizingLeftAnkleY := originalLeftAnkleY * scale.Y
		sizingRightAnkleY := originalRightAnkleY * scale.Y

		leftAnkleDelta := vmdDeltas.Bones.Get(leftAnkleBone.Index())
		rightAnkleDelta := vmdDeltas.Bones.Get(rightAnkleBone.Index())

		// 現時点の足首のY座標（足首の高さを除く）
		actualLeftAnkleY := leftAnkleDelta.FilledGlobalPosition().Y - leftAnkleBone.Position.Y
		actualRightAnkleY := rightAnkleDelta.FilledGlobalPosition().Y - rightAnkleBone.Position.Y

		leftAnkleDiff := sizingLeftAnkleY - actualLeftAnkleY
		rightAnkleDiff := sizingRightAnkleY - actualRightAnkleY
		ankleDiffY := max(leftAnkleDiff, rightAnkleDiff)

		mlog.V("足補正02[%.0f] originalLeftY[%.4f], sizingLeftY[%.4f], fkLeftY[%.4f], LDiff[%.4f], originalRightY[%.4f], sizingRightY[%.4f], fkRightY[%.4f], RDiff[%.4f], ankleDiffY[%.4f]",
			frame, originalLeftAnkleY, sizingLeftAnkleY, actualLeftAnkleY, leftAnkleDiff, originalRightAnkleY, sizingRightAnkleY, actualRightAnkleY, rightAnkleDiff, ankleDiffY)

		// センターの位置をスケールに合わせる
		originalCenterBf := originalMotion.BoneFrames.Get(centerBone.Name()).Get(frame)
		originalGrooveBf := originalMotion.BoneFrames.Get(grooveBone.Name()).Get(frame)

		// 右足IKを動かさなかった場合のセンターと左足IKの位置を調整する用の値を保持
		// （この時点でキーフレに追加すると動きが変わる）
		centerPositions[index] = originalCenterBf.Position.Muled(scale)
		groovePositions[index] = originalGrooveBf.Position.Added(&mmath.MVec3{X: 0, Y: ankleDiffY, Z: 0})
	})

	mlog.I(mi18n.T("足補正03", map[string]interface{}{"No": sizingSet.Index + 1}))

	// 補正を登録
	for i, iFrame := range frames {
		frame := float32(iFrame)

		sizingCenterBf := sizingMotion.BoneFrames.Get(centerBone.Name()).Get(frame)
		sizingCenterBf.Position = centerPositions[i]
		sizingMotion.InsertRegisteredBoneFrame(centerBone.Name(), sizingCenterBf)

		sizingGrooveBf := sizingMotion.BoneFrames.Get(grooveBone.Name()).Get(frame)
		sizingGrooveBf.Position = groovePositions[i]
		sizingMotion.InsertRegisteredBoneFrame(grooveBone.Name(), sizingGrooveBf)
	}

	if mlog.IsVerbose() {
		outputPath := mutils.CreateOutputPath(sizingSet.OriginalVmdPath, "足補正03_センター補正")
		repository.NewVmdRepository().Save(outputPath, sizingMotion, true)
		mlog.V("足補正03_センター補正: %s", outputPath)
	}

	mlog.I(mi18n.T("足補正04", map[string]interface{}{"No": sizingSet.Index + 1}))

	leftLegIkPositions := make([]*mmath.MVec3, len(frames))
	rightLegIkPositions := make([]*mmath.MVec3, len(frames))
	leftLegIkRotations := make([]*mmath.MQuaternion, len(frames))
	rightLegIkRotations := make([]*mmath.MQuaternion, len(frames))

	// 先モデルのデフォーム(IK OFF+センター補正済み)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, false, frame, leg_all_bone_names, false)

		originalLeftLegIkBf := originalMotion.BoneFrames.Get(leftLegIkBone.Name()).Get(frame)
		originalRightLegIkBf := originalMotion.BoneFrames.Get(rightLegIkBone.Name()).Get(frame)

		// 地面に近い足底が同じ高さになるように調整
		// originalLegLeftDelta := originalAllDeltas[index].Bones.GetByName(pmx.LEG.Left())
		originalHeelLeftDelta := originalAllDeltas[index].Bones.GetByName(pmx.HEEL.Left())
		originalToeTailLeftDelta := originalAllDeltas[index].Bones.GetByName(pmx.TOE_T.Left())
		// originalLegRightDelta := originalAllDeltas[index].Bones.GetByName(pmx.LEG.Right())
		// originalAnkleRightDelta := originalAllDeltas[index].Bones.GetByName(pmx.ANKLE.Right())
		originalHeelRightDelta := originalAllDeltas[index].Bones.GetByName(pmx.HEEL.Right())
		originalToeTailRightDelta := originalAllDeltas[index].Bones.GetByName(pmx.TOE_T.Right())

		leftLegIkDelta := vmdDeltas.Bones.GetByName(pmx.LEG_IK.Left())
		leftAnkleDelta := vmdDeltas.Bones.GetByName(pmx.ANKLE.Left())
		leftHeelDelta := vmdDeltas.Bones.GetByName(pmx.HEEL.Left())
		leftToeTailDelta := vmdDeltas.Bones.GetByName(pmx.TOE_T.Left())

		rightLegIkDelta := vmdDeltas.Bones.GetByName(pmx.LEG_IK.Right())
		rightAnkleDelta := vmdDeltas.Bones.GetByName(pmx.ANKLE.Right())
		rightHeelDelta := vmdDeltas.Bones.GetByName(pmx.HEEL.Right())
		rightToeTailDelta := vmdDeltas.Bones.GetByName(pmx.TOE_T.Right())

		// 足IKから見た足首の位置
		leftLegIkPositions[index] = leftAnkleDelta.FilledGlobalPosition().Subed(leftLegIkDelta.FilledGlobalPosition())
		rightLegIkPositions[index] = rightAnkleDelta.FilledGlobalPosition().Subed(rightLegIkDelta.FilledGlobalPosition())

		// 左足IK-Yの位置を調整
		if mmath.NearEquals(originalLeftLegIkBf.Position.Y, 0, 1e-2) {
			leftLegIkPositions[index].Y = 0
		} else {
			if originalToeTailLeftDelta.FilledGlobalPosition().Y <= originalHeelLeftDelta.FilledGlobalPosition().Y {
				// つま先の方がかかとより低い場合
				originalLeftToeTailY := originalToeTailLeftDelta.FilledGlobalPosition().Y

				// つま先のY座標を元モデルのつま先のY座標*スケールに合わせる
				sizingLeftToeTailY := originalLeftToeTailY * scale.Y

				// 現時点のつま先のY座標
				actualLeftToeTailY := leftToeTailDelta.FilledGlobalPosition().Y

				leftToeDiff := sizingLeftToeTailY - actualLeftToeTailY
				lerpLeftToeDiff := mmath.LerpFloat(leftToeDiff, 0,
					originalToeTailLeftDelta.FilledGlobalPosition().Y/originalLeftAnkleBone.Position.Y)
				// 足首Y位置に近付くにつれて補正を弱める
				leftLegIkPositions[index].Y += lerpLeftToeDiff
				mlog.V("足補正04[%.0f][左つま先] originalLeftY[%.4f], sizingLeftY[%.4f], actualLeftY[%.4f], diff[%.4f], lerp[%.4f]",
					frame, originalLeftToeTailY, sizingLeftToeTailY, actualLeftToeTailY, leftToeDiff, lerpLeftToeDiff)
			} else {
				// かかとの方がつま先より低い場合
				originalLeftHeelY := originalHeelLeftDelta.FilledGlobalPosition().Y

				// かかとのY座標を元モデルのかかとのY座標*スケールに合わせる
				sizingLeftHeelY := originalLeftHeelY * scale.Y

				// 現時点のかかとのY座標
				actualLeftHeelY := leftHeelDelta.FilledGlobalPosition().Y

				leftHeelDiff := sizingLeftHeelY - actualLeftHeelY
				lerpLeftHeelDiff := mmath.LerpFloat(leftHeelDiff, 0,
					originalHeelLeftDelta.FilledGlobalPosition().Y/originalLeftAnkleBone.Position.Y)
				// 足首Y位置に近付くにつれて補正を弱める
				leftLegIkPositions[index].Y += lerpLeftHeelDiff

				mlog.V("足補正04[%.0f][左かかと] originalLeftY[%.4f], sizingLeftY[%.4f], actualLeftY[%.4f], diff[%.4f], lerp[%.4f]",
					frame, originalLeftHeelY, sizingLeftHeelY, actualLeftHeelY, leftHeelDiff, lerpLeftHeelDiff)
			}
		}

		// 右足IK-Yの位置を調整
		if mmath.NearEquals(originalRightLegIkBf.Position.Y, 0, 1e-2) {
			rightLegIkPositions[index].Y = 0
		} else {
			if originalToeTailRightDelta.FilledGlobalPosition().Y <= originalHeelRightDelta.FilledGlobalPosition().Y {
				// つま先の方がかかとより低い場合
				originalRightToeY := originalToeTailRightDelta.FilledGlobalPosition().Y

				// つま先のY座標を元モデルのつま先のY座標*スケールに合わせる
				sizingRightToeY := originalRightToeY * scale.Y

				// 現時点のつま先のY座標
				actualRightToeY := rightToeTailDelta.FilledGlobalPosition().Y

				rightToeDiff := sizingRightToeY - actualRightToeY
				lerpRightToeDiff := mmath.LerpFloat(rightToeDiff, 0,
					originalToeTailRightDelta.FilledGlobalPosition().Y/originalRightAnkleBone.Position.Y)
				// 足首Y位置に近付くにつれて補正を弱める
				rightLegIkPositions[index].Y += lerpRightToeDiff

				mlog.V("足補正04[%.0f][右つま先] originalRightY[%.4f], sizingRightY[%.4f], actualRightY[%.4f], diff[%.4f], lerp[%.4f]",
					frame, originalRightToeY, sizingRightToeY, actualRightToeY, rightToeDiff, lerpRightToeDiff)
			} else {
				// かかとの方がつま先より低い場合
				originalRightHeelY := originalHeelRightDelta.FilledGlobalPosition().Y

				// かかとのY座標を元モデルのかかとのY座標*スケールに合わせる
				sizingRightHeelY := originalRightHeelY * scale.Y

				// 現時点のかかとのY座標
				actualRightHeelY := rightHeelDelta.FilledGlobalPosition().Y

				rightHeelDiff := sizingRightHeelY - actualRightHeelY
				lerpRightHeelDiff := mmath.LerpFloat(rightHeelDiff, 0,
					originalHeelRightDelta.FilledGlobalPosition().Y/originalRightAnkleBone.Position.Y)
				// 足首Y位置に近付くにつれて補正を弱める
				rightLegIkPositions[index].Y += lerpRightHeelDiff

				mlog.V("足補正04[%.0f][右かかと] originalRightY[%.4f], sizingRightY[%.4f], actualRightY[%.4f], diff[%.4f], lerp[%.4f]",
					frame, originalRightHeelY, sizingRightHeelY, actualRightHeelY, rightHeelDiff, lerpRightHeelDiff)
			}
		}

		// 足首から見たつま先IKの方向
		leftLegIkMat := vmdDeltas.Bones.Get(leftToeIkBone.Index()).FilledGlobalPosition().Subed(
			vmdDeltas.Bones.Get(leftLegIkBone.Index()).FilledGlobalPosition()).Normalize().ToLocalMat()
		leftLegFkMat := vmdDeltas.Bones.Get(leftToeBone.Index()).FilledGlobalPosition().Subed(
			vmdDeltas.Bones.Get(leftAnkleBone.Index()).FilledGlobalPosition()).Normalize().ToLocalMat()
		leftLegIkRotations[index] = leftLegFkMat.Muled(leftLegIkMat.Inverted()).Quaternion()

		rightLegIkMat := vmdDeltas.Bones.Get(rightToeIkBone.Index()).FilledGlobalPosition().Subed(
			vmdDeltas.Bones.Get(rightLegIkBone.Index()).FilledGlobalPosition()).Normalize().ToLocalMat()
		rightLegFkMat := vmdDeltas.Bones.Get(rightToeBone.Index()).FilledGlobalPosition().Subed(
			vmdDeltas.Bones.Get(rightAnkleBone.Index()).FilledGlobalPosition()).Normalize().ToLocalMat()
		rightLegIkRotations[index] = rightLegFkMat.Muled(rightLegIkMat.Inverted()).Quaternion()
	})

	mlog.I(mi18n.T("足補正05", map[string]interface{}{"No": sizingSet.Index + 1}))

	// 足IKを一旦戻す
	for _, frame := range originalMotion.BoneFrames.Get(leftLegIkBone.Name()).Indexes.List() {
		bf := originalMotion.BoneFrames.Get(leftLegIkBone.Name()).Get(frame).Copy().(*vmd.BoneFrame)
		sizingMotion.AppendRegisteredBoneFrame(leftLegIkBone.Name(), bf)
	}

	for _, frame := range originalMotion.BoneFrames.Get(rightLegIkBone.Name()).Indexes.List() {
		bf := originalMotion.BoneFrames.Get(rightLegIkBone.Name()).Get(frame).Copy().(*vmd.BoneFrame)
		sizingMotion.AppendRegisteredBoneFrame(rightLegIkBone.Name(), bf)
	}

	for i, iFrame := range frames {
		frame := float32(iFrame)

		originalLeftLegIkBf := originalMotion.BoneFrames.Get(leftLegIkBone.Name()).Get(frame)
		originalRightLegIkBf := originalMotion.BoneFrames.Get(rightLegIkBone.Name()).Get(frame)

		if i > 0 {
			// 前と同じ位置なら同じ位置にする
			originalPrevRightLegIkBf := originalMotion.BoneFrames.Get(rightLegIkBone.Name()).Get(float32(frames[i-1]))
			if mmath.NearEquals(originalPrevRightLegIkBf.Position.X, originalRightLegIkBf.Position.X, 1e-2) {
				rightLegIkPositions[i].X = rightLegIkPositions[i-1].X
			}
			if mmath.NearEquals(originalPrevRightLegIkBf.Position.Y, originalRightLegIkBf.Position.Y, 1e-2) {
				rightLegIkPositions[i].Y = rightLegIkPositions[i-1].Y
			}
			if mmath.NearEquals(originalPrevRightLegIkBf.Position.Z, originalRightLegIkBf.Position.Z, 1e-2) {
				rightLegIkPositions[i].Z = rightLegIkPositions[i-1].Z
			}

			originalPrevLeftLegIkBf := originalMotion.BoneFrames.Get(leftLegIkBone.Name()).Get(float32(frames[i-1]))
			if mmath.NearEquals(originalPrevLeftLegIkBf.Position.X, originalLeftLegIkBf.Position.X, 1e-2) {
				leftLegIkPositions[i].X = leftLegIkPositions[i-1].X
			}
			if mmath.NearEquals(originalPrevLeftLegIkBf.Position.Y, originalLeftLegIkBf.Position.Y, 1e-2) {
				leftLegIkPositions[i].Y = leftLegIkPositions[i-1].Y
			}
			if mmath.NearEquals(originalPrevLeftLegIkBf.Position.Z, originalLeftLegIkBf.Position.Z, 1e-2) {
				leftLegIkPositions[i].Z = leftLegIkPositions[i-1].Z
			}
		}

		rightLegIkBf := sizingMotion.BoneFrames.Get(rightLegIkBone.Name()).Get(frame)
		rightLegIkBf.Position = rightLegIkPositions[i]
		rightLegIkBf.Rotation = rightLegIkRotations[i]
		sizingMotion.InsertRegisteredBoneFrame(rightLegIkBone.Name(), rightLegIkBf)

		leftLegIkBf := sizingMotion.BoneFrames.Get(leftLegIkBone.Name()).Get(frame)
		leftLegIkBf.Position = leftLegIkPositions[i]
		leftLegIkBf.Rotation = leftLegIkRotations[i]
		sizingMotion.InsertRegisteredBoneFrame(leftLegIkBone.Name(), leftLegIkBf)
	}

	if mlog.IsVerbose() {
		outputPath := mutils.CreateOutputPath(sizingSet.OriginalVmdPath, "足補正05_足IK補正")
		repository.NewVmdRepository().Save(outputPath, sizingMotion, true)
		mlog.V("足補正05_足IK補正: %s", outputPath)
	}

	mlog.I(mi18n.T("足補正06", map[string]interface{}{"No": sizingSet.Index + 1}))

	leftLegRotations := make([]*mmath.MQuaternion, len(frames))
	leftKneeRotations := make([]*mmath.MQuaternion, len(frames))
	leftAnkleRotations := make([]*mmath.MQuaternion, len(frames))
	rightLegRotations := make([]*mmath.MQuaternion, len(frames))
	rightKneeRotations := make([]*mmath.MQuaternion, len(frames))
	rightAnkleRotations := make([]*mmath.MQuaternion, len(frames))

	// 先モデルのデフォーム(IK ON)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, false, frame, leg_all_bone_names, false)

		{
			leftLegIkGlobalPosition := vmdDeltas.Bones.Get(leftLegIkBone.Index()).FilledGlobalPosition()
			vmdDeltas := deform.DeformIk(sizingModel, sizingMotion, frame, leftLegIkBone, leftLegIkGlobalPosition)
			leftLegRotations[index] = vmdDeltas.Bones.Get(leftLegBone.Index()).FilledFrameRotation()
			leftKneeRotations[index] = vmdDeltas.Bones.Get(leftKneeBone.Index()).FilledFrameRotation()
			leftAnkleRotations[index] = vmdDeltas.Bones.Get(leftAnkleBone.Index()).FilledFrameRotation()
		}
		{
			rightLegIkGlobalPosition := vmdDeltas.Bones.Get(rightLegIkBone.Index()).FilledGlobalPosition()
			vmdDeltas := deform.DeformIk(sizingModel, sizingMotion, frame, rightLegIkBone, rightLegIkGlobalPosition)
			rightLegRotations[index] = vmdDeltas.Bones.Get(rightLegBone.Index()).FilledFrameRotation()
			rightKneeRotations[index] = vmdDeltas.Bones.Get(rightKneeBone.Index()).FilledFrameRotation()
			rightAnkleRotations[index] = vmdDeltas.Bones.Get(rightAnkleBone.Index()).FilledFrameRotation()
		}
	})

	mlog.I(mi18n.T("足補正07", map[string]interface{}{"No": sizingSet.Index + 1}))

	// つま先IKを削除
	sizingMotion.BoneFrames.Delete(pmx.TOE_IK.Left())
	sizingMotion.BoneFrames.Delete(pmx.TOE_IK.Right())

	// サイジング先にFKを焼き込み
	for i, iFrame := range frames {
		frame := float32(iFrame)

		{
			bf := sizingMotion.BoneFrames.Get(leftLegIkBone.Name()).Get(frame)
			bf.Position = leftLegIkPositions[i]
			bf.Rotation = leftLegIkRotations[i]
			sizingMotion.InsertRegisteredBoneFrame(leftLegIkBone.Name(), bf)
		}
		{
			bf := sizingMotion.BoneFrames.Get(leftLegBone.Name()).Get(frame)
			bf.Rotation = leftLegRotations[i]
			sizingMotion.InsertRegisteredBoneFrame(leftLegBone.Name(), bf)
		}
		{
			bf := sizingMotion.BoneFrames.Get(leftKneeBone.Name()).Get(frame)
			bf.Rotation = leftKneeRotations[i]
			sizingMotion.InsertRegisteredBoneFrame(leftKneeBone.Name(), bf)
		}
		{
			bf := sizingMotion.BoneFrames.Get(leftAnkleBone.Name()).Get(frame)
			bf.Rotation = leftAnkleRotations[i]
			sizingMotion.InsertRegisteredBoneFrame(leftAnkleBone.Name(), bf)
		}
		{
			bf := sizingMotion.BoneFrames.Get(rightLegIkBone.Name()).Get(frame)
			bf.Position = rightLegIkPositions[i]
			bf.Rotation = rightLegIkRotations[i]
			sizingMotion.InsertRegisteredBoneFrame(rightLegIkBone.Name(), bf)
		}
		{
			bf := sizingMotion.BoneFrames.Get(rightLegBone.Name()).Get(frame)
			bf.Rotation = rightLegRotations[i]
			sizingMotion.InsertRegisteredBoneFrame(rightLegBone.Name(), bf)
		}
		{
			bf := sizingMotion.BoneFrames.Get(rightKneeBone.Name()).Get(frame)
			bf.Rotation = rightKneeRotations[i]
			sizingMotion.InsertRegisteredBoneFrame(rightKneeBone.Name(), bf)
		}
		{
			bf := sizingMotion.BoneFrames.Get(rightAnkleBone.Name()).Get(frame)
			bf.Rotation = rightAnkleRotations[i]
			sizingMotion.InsertRegisteredBoneFrame(rightAnkleBone.Name(), bf)
		}
	}

	if mlog.IsVerbose() {
		outputPath := mutils.CreateOutputPath(sizingSet.OriginalVmdPath, "足補正07_FK再計算")
		repository.NewVmdRepository().Save(outputPath, sizingMotion, true)
		mlog.V("足補正07_FK再計算: %s", outputPath)
	}

	sizingSet.CompletedSizingLeg = true

	return frames, originalAllDeltas
}

func isValidSizingLower(sizingSet *domain.SizingSet) bool {
	originalModel := sizingSet.OriginalPmx
	sizingModel := sizingSet.SizingPmx

	// センター、グルーブ、下半身、右足IK、左足IKが存在するか

	if !originalModel.Bones.ContainsByName(pmx.CENTER.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.CENTER.String()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.GROOVE.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.GROOVE.String()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.LOWER.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.LOWER.String()}))
		return false
	}

	// ------------------------------

	if !originalModel.Bones.ContainsByName(pmx.LEG_IK.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.LEG_IK.Left()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.LEG.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.LEG.Left()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.KNEE.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.KNEE.Left()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.ANKLE.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.ANKLE.Left()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.TOE_IK.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.TOE_IK.Left()}))
		return false
	}

	// ------------------------------

	if !originalModel.Bones.ContainsByName(pmx.LEG_IK.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.LEG_IK.Right()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.LEG.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.LEG.Right()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.KNEE.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.KNEE.Right()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.ANKLE.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.ANKLE.Right()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.TOE_IK.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.TOE_IK.Left()}))
		return false
	}

	// ------------------------------

	if !sizingModel.Bones.ContainsByName(pmx.CENTER.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.CENTER.String()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.GROOVE.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.GROOVE.String()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.LOWER.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.LOWER.String()}))
		return false
	}

	// ------------------------------

	if !sizingModel.Bones.ContainsByName(pmx.LEG_IK.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.LEG_IK.Left()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.LEG.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.LEG.Left()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.KNEE.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.KNEE.Left()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.ANKLE.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.ANKLE.Left()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.TOE_IK.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.TOE_IK.Left()}))
		return false
	}

	// ------------------------------

	if !sizingModel.Bones.ContainsByName(pmx.LEG_IK.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.LEG_IK.Right()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.LEG.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.LEG.Right()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.KNEE.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.KNEE.Right()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.ANKLE.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.ANKLE.Right()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.TOE_IK.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.TOE_IK.Left()}))
		return false
	}

	return true
}
