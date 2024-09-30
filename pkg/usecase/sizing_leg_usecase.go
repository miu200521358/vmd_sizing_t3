package usecase

import (
	"github.com/miu200521358/mlib_go/pkg/domain/delta"
	"github.com/miu200521358/mlib_go/pkg/domain/miter"
	"github.com/miu200521358/mlib_go/pkg/domain/mmath"
	"github.com/miu200521358/mlib_go/pkg/domain/pmx"
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

	originalMotion := sizingSet.OriginalVmd
	sizingModel := sizingSet.SizingPmx
	sizingMotion := sizingSet.OutputVmd

	centerBone := sizingModel.Bones.GetByName(pmx.CENTER.String())
	grooveBone := sizingModel.Bones.GetByName(pmx.GROOVE.String())
	rightLegIkBone := sizingModel.Bones.GetByName(pmx.LEG_IK.Right())
	rightLegBone := sizingModel.Bones.GetByName(pmx.LEG.Right())
	rightKneeBone := sizingModel.Bones.GetByName(pmx.KNEE.Right())
	rightAnkleBone := sizingModel.Bones.Get(sizingModel.Bones.GetByName(pmx.LEG_IK.Right()).Ik.BoneIndex)
	// rightHeelBone := sizingModel.Bones.GetByName(pmx.HEEL.Right())
	rightToeBone := sizingModel.Bones.GetByName(pmx.TOE.Right())
	leftLegIkBone := sizingModel.Bones.GetByName(pmx.LEG_IK.Left())
	leftLegBone := sizingModel.Bones.GetByName(pmx.LEG.Left())
	leftKneeBone := sizingModel.Bones.GetByName(pmx.KNEE.Left())
	leftAnkleBone := sizingModel.Bones.Get(sizingModel.Bones.GetByName(pmx.LEG_IK.Left()).Ik.BoneIndex)
	// leftHeelBone := sizingModel.Bones.GetByName(pmx.HEEL.Left())
	leftToeBone := sizingModel.Bones.GetByName(pmx.TOE.Left())

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

	if mlog.IsVerbose() {
		outputPath := mutils.CreateOutputPath(sizingSet.OriginalVmdPath, "Leg_01_FK焼き込み")
		repository.NewVmdRepository().Save(outputPath, sizingMotion, true)
		mlog.V("Leg_01_FK焼き込み: %s", outputPath)
	}

	mlog.I(mi18n.T("足補正02", map[string]interface{}{"No": sizingSet.Index + 1}))

	sizingOffDeltas := make([]*delta.VmdDeltas, len(frames))
	centerPositions := make([]*mmath.MVec3, len(frames))
	groovePositions := make([]*mmath.MVec3, len(frames))
	rightLegIkPositions := make([]*mmath.MVec3, len(frames))
	rightHeelPositions := make([]*mmath.MVec3, len(frames))
	rightToePositions := make([]*mmath.MVec3, len(frames))

	// 先モデルのデフォーム(IK OFF)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, false, frame, leg_all_bone_names, false)
		sizingOffDeltas[index] = vmdDeltas

		// 右つま先から見た右足ボーンの相対位置を取得
		rightLegIkDelta := vmdDeltas.Bones.Get(rightLegIkBone.Index())
		rightLegDelta := vmdDeltas.Bones.Get(rightLegBone.Index())
		rightAnkleDelta := vmdDeltas.Bones.Get(rightAnkleBone.Index())
		rightToeDelta := vmdDeltas.Bones.Get(rightToeBone.Index())

		// 足首から見た足の位置を求める
		// 実際に接地するのはつま先もしくはかかとなので、一旦つま先を基準にする
		rightLegFkLocalPosition := rightToeDelta.FilledGlobalPosition().Subed(
			rightLegDelta.FilledGlobalPosition()).Subed(
			rightToeDelta.FilledGlobalPosition().Subed(
				rightAnkleDelta.FilledGlobalPosition()))
		rightLegIkLocalPosition := rightLegIkDelta.FilledGlobalPosition().Subed(
			rightLegDelta.FilledGlobalPosition())
		rightLegDiff := rightLegIkLocalPosition.Subed(rightLegFkLocalPosition)

		// 右足IKを動かさなかった場合のセンターと左足IKの位置を調整する用の値を保持
		// （この時点でキーフレに追加すると動きが変わる）
		centerBf := sizingMotion.BoneFrames.Get(centerBone.Name()).Get(frame)
		centerPositions[index] = centerBf.Position.Added(&mmath.MVec3{X: rightLegDiff.X, Y: 0, Z: rightLegDiff.Z})

		grooveBf := sizingMotion.BoneFrames.Get(grooveBone.Name()).Get(frame)
		groovePositions[index] = grooveBf.Position.Added(&mmath.MVec3{X: 0, Y: rightLegDiff.Y, Z: 0})

		rightLegIkBf := sizingMotion.BoneFrames.Get(rightLegIkBone.Name()).Get(frame)
		rightLegIkPositions[index] = rightLegIkBf.Position

		rightHeelPositions[index] = vmdDeltas.Bones.GetByName(pmx.HEEL.Right()).FilledGlobalPosition()
		rightToePositions[index] = vmdDeltas.Bones.GetByName(pmx.TOE.Right()).FilledGlobalPosition()
	})

	mlog.I(mi18n.T("足補正03", map[string]interface{}{"No": sizingSet.Index + 1}))

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

	if mlog.IsVerbose() {
		outputPath := mutils.CreateOutputPath(sizingSet.OriginalVmdPath, "Leg_02_右足基準補正")
		repository.NewVmdRepository().Save(outputPath, sizingMotion, true)
		mlog.V("Leg_02_右足基準補正: %s", outputPath)
	}

	mlog.I(mi18n.T("足補正04", map[string]interface{}{"No": sizingSet.Index + 1}))

	sizingCenterDeltas := make([]*delta.VmdDeltas, len(frames))
	leftLegIkPositions := make([]*mmath.MVec3, len(frames))

	// 先モデルのデフォーム(IK OFF+センター補正済み)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, false, frame, leg_all_bone_names, false)
		sizingCenterDeltas[index] = vmdDeltas

		// 左足首から見た左足ボーンの相対位置を取得
		leftLegIkDelta := vmdDeltas.Bones.Get(leftLegIkBone.Index())
		leftLegDelta := vmdDeltas.Bones.Get(leftLegBone.Index())
		leftAnkleDelta := vmdDeltas.Bones.Get(leftAnkleBone.Index())
		leftToeDelta := vmdDeltas.Bones.Get(leftToeBone.Index())

		leftLegIkLocalPosition := leftLegDelta.FilledGlobalPosition().Subed(
			leftLegIkDelta.FilledGlobalPosition())
		leftLegFkLocalPosition := leftLegDelta.FilledGlobalPosition().Subed(
			leftToeDelta.FilledGlobalPosition()).Subed(
			leftAnkleDelta.FilledGlobalPosition().Subed(
				leftToeDelta.FilledGlobalPosition()))
		leftLegDiff := leftLegFkLocalPosition.Subed(leftLegIkLocalPosition)

		// 左足IK
		leftLegIkBf := sizingMotion.BoneFrames.Get(leftLegIkBone.Name()).Get(frame)
		leftLegIkPositions[index] = leftLegIkBf.Position.Subed(leftLegDiff)
	})

	mlog.I(mi18n.T("足補正05", map[string]interface{}{"No": sizingSet.Index + 1}))

	// 左足の結果を登録
	for i, iFrame := range frames {
		frame := float32(iFrame)

		leftLegIkBf := sizingMotion.BoneFrames.Get(leftLegIkBone.Name()).Get(frame)
		leftLegIkBf.Position = leftLegIkPositions[i]
		sizingMotion.InsertRegisteredBoneFrame(leftLegIkBone.Name(), leftLegIkBf)
	}

	if mlog.IsVerbose() {
		outputPath := mutils.CreateOutputPath(sizingSet.OriginalVmdPath, "Leg_03_左足補正")
		repository.NewVmdRepository().Save(outputPath, sizingMotion, true)
		mlog.V("Leg_03_左足補正: %s", outputPath)
	}

	// センターのオフセットを保持
	for i, iFrame := range frames {
		frame := float32(iFrame)

		originalLeftLegIkBf := originalMotion.BoneFrames.Get(leftLegIkBone.Name()).Get(frame)
		originalRightLegIkBf := originalMotion.BoneFrames.Get(rightLegIkBone.Name()).Get(frame)

		// 地面に近い足底が同じ高さになるように調整
		originalLegLeftDelta := originalAllDeltas[i].Bones.GetByName(pmx.LEG.Left())
		// originalAnkleLeftDelta := originalAllDeltas[i].Bones.GetByName(pmx.ANKLE.Left())
		originalHeelLeftDelta := originalAllDeltas[i].Bones.GetByName(pmx.HEEL.Left())
		originalToeLeftDelta := originalAllDeltas[i].Bones.GetByName(pmx.TOE.Left())
		originalLegRightDelta := originalAllDeltas[i].Bones.GetByName(pmx.LEG.Right())
		// originalAnkleRightDelta := originalAllDeltas[i].Bones.GetByName(pmx.ANKLE.Right())
		originalHeelRightDelta := originalAllDeltas[i].Bones.GetByName(pmx.HEEL.Right())
		originalToeRightDelta := originalAllDeltas[i].Bones.GetByName(pmx.TOE.Right())

		// legLeftDelta := sizingCenterDeltas[i].Bones.GetByName(pmx.LEG.Left())
		// ankleLeftDelta := sizingCenterDeltas[i].Bones.GetByName(pmx.ANKLE.Left())
		heelLeftDelta := sizingCenterDeltas[i].Bones.GetByName(pmx.HEEL.Left())
		toeLeftDelta := sizingCenterDeltas[i].Bones.GetByName(pmx.TOE.Left())
		// legRightDelta := sizingCenterDeltas[i].Bones.GetByName(pmx.LEG.Right())
		// ankleRightDelta := sizingCenterDeltas[i].Bones.GetByName(pmx.ANKLE.Right())
		heelRightDelta := sizingCenterDeltas[i].Bones.GetByName(pmx.HEEL.Right())
		toeRightDelta := sizingCenterDeltas[i].Bones.GetByName(pmx.TOE.Right())

		minYIndex := mmath.ArgMin([]float64{
			originalHeelRightDelta.FilledGlobalPosition().Y,
			originalToeRightDelta.FilledGlobalPosition().Y,
			originalHeelLeftDelta.FilledGlobalPosition().Y,
			originalToeLeftDelta.FilledGlobalPosition().Y})

		originalMinY := []float64{
			originalHeelRightDelta.FilledGlobalPosition().Y,
			originalToeRightDelta.FilledGlobalPosition().Y,
			originalHeelLeftDelta.FilledGlobalPosition().Y,
			originalToeLeftDelta.FilledGlobalPosition().Y}[minYIndex]

		minY := []float64{
			heelRightDelta.FilledGlobalPosition().Y,
			toeRightDelta.FilledGlobalPosition().Y,
			heelLeftDelta.FilledGlobalPosition().Y,
			toeLeftDelta.FilledGlobalPosition().Y}[minYIndex]

		var scaledY float64
		if originalMinY > 1e-2 {
			scaledY = originalMinY * scale.Y
		} else {
			scaledY = originalMinY
		}
		y := scaledY - minY

		// センターの位置をスケールに合わせる
		originalCenterBf := originalMotion.BoneFrames.Get(centerBone.Name()).Get(frame)
		scaledCenter := originalCenterBf.Position.Muled(scale)

		x := scaledCenter.X - centerPositions[i].X
		z := scaledCenter.Z - centerPositions[i].Z

		centerPositions[i].Add(&mmath.MVec3{X: x, Y: 0, Z: z})
		groovePositions[i].Add(&mmath.MVec3{X: 0, Y: y, Z: 0})
		leftLegIkPositions[i].Add(&mmath.MVec3{X: x, Y: y, Z: z})
		rightLegIkPositions[i].Add(&mmath.MVec3{X: x, Y: y, Z: z})

		// 左足IK-Yの位置を調整
		if mmath.NearEquals(originalLeftLegIkBf.Position.Y, 0.0, 1e-2) {
			groovePositions[i].Y += -leftLegIkPositions[i].Y
			leftLegIkPositions[i].Y = 0.0
		} else {
			if originalToeLeftDelta.FilledGlobalPosition().Y <= originalHeelLeftDelta.FilledGlobalPosition().Y {
				// つま先の方がかかとより低い場合
				if toeLeftDelta.FilledGlobalPosition().Y+y < 1e-2 {
					leftLegIkPositions[i].Y += -(toeLeftDelta.FilledGlobalPosition().Y + y)
				} else {
					// 比率で求めた足首位置
					scaledLeftY := originalToeLeftDelta.FilledGlobalPosition().Y * scale.Y
					fixedLeftY := toeLeftDelta.FilledGlobalPosition().Y + y

					// 足の高さを1として、どの程度の高さに足首があるか
					t := originalToeLeftDelta.FilledGlobalPosition().Y /
						originalLegLeftDelta.FilledGlobalPosition().Y
					lerpY := mmath.LerpFloat(scaledLeftY, fixedLeftY, t)

					// 差分を足IKに加算する
					leftLegIkPositions[i].Y += (lerpY - fixedLeftY)
				}
			} else {
				// かかとの方がつま先より低い場合
				if heelLeftDelta.FilledGlobalPosition().Y+y < 1e-2 {
					leftLegIkPositions[i].Y += -(heelLeftDelta.FilledGlobalPosition().Y + y)
				} else {
					// 比率で求めた足首位置
					scaledLeftY := originalHeelLeftDelta.FilledGlobalPosition().Y * scale.Y
					fixedLeftY := heelLeftDelta.FilledGlobalPosition().Y + y

					// 元モデルの足の高さを1として、どの程度の高さに元モデルの足首があるか
					t := originalHeelLeftDelta.FilledGlobalPosition().Y /
						originalLegLeftDelta.FilledGlobalPosition().Y
					lerpY := mmath.LerpFloat(scaledLeftY, fixedLeftY, t)

					// 差分を足IKに加算する
					leftLegIkPositions[i].Y += (lerpY - fixedLeftY)
				}
			}
		}

		// 右足IK-Yの位置を調整
		if mmath.NearEquals(originalRightLegIkBf.Position.Y, 0.0, 1e-2) {
			groovePositions[i].Y += -rightLegIkPositions[i].Y
			rightLegIkPositions[i].Y = 0.0
		} else {
			if originalToeRightDelta.FilledGlobalPosition().Y <= originalHeelRightDelta.FilledGlobalPosition().Y {
				// つま先の方がかかとより低い場合
				if toeRightDelta.FilledGlobalPosition().Y+y < 1e-2 {
					rightLegIkPositions[i].Y += -(toeRightDelta.FilledGlobalPosition().Y + y)
				} else {
					// 比率で求めた足首位置
					scaledRightY := originalToeRightDelta.FilledGlobalPosition().Y * scale.Y
					fixedRightY := toeRightDelta.FilledGlobalPosition().Y + y

					// 元モデルの足の高さを1として、どの程度の高さに元モデルの足首があるか
					t := originalToeRightDelta.FilledGlobalPosition().Y /
						originalLegRightDelta.FilledGlobalPosition().Y
					lerpY := mmath.LerpFloat(scaledRightY, fixedRightY, t)

					// 差分を足IKに加算する
					rightLegIkPositions[i].Y += (lerpY - fixedRightY)
				}
			} else {
				// かかとの方がつま先より低い場合
				if heelRightDelta.FilledGlobalPosition().Y+y < 1e-2 {
					rightLegIkPositions[i].Y += -(heelRightDelta.FilledGlobalPosition().Y + y)
				} else {
					// 比率で求めた足首位置
					scaledRightY := originalHeelRightDelta.FilledGlobalPosition().Y * scale.Y
					fixedRightY := heelRightDelta.FilledGlobalPosition().Y + y

					// 元モデルの足の高さを1として、どの程度の高さに元モデルの足首があるか
					t := originalHeelRightDelta.FilledGlobalPosition().Y /
						originalLegRightDelta.FilledGlobalPosition().Y
					lerpY := mmath.LerpFloat(scaledRightY, fixedRightY, t)

					// 差分を足IKに加算する
					rightLegIkPositions[i].Y += (lerpY - fixedRightY)
				}
			}
		}

		if i > 0 {
			// 前と同じ位置なら同じ位置にする
			originalPrevRightLegIkBf := originalMotion.BoneFrames.Get(rightLegIkBone.Name()).Get(float32(frames[i-1]))
			if mmath.NearEquals(originalPrevRightLegIkBf.Position.X, originalRightLegIkBf.Position.X, 1e-2) {
				rightLegIkPositions[i].X = rightLegIkPositions[i-1].X
			}
			if mmath.NearEquals(originalPrevRightLegIkBf.Position.Z, originalRightLegIkBf.Position.Z, 1e-2) {
				rightLegIkPositions[i].Z = rightLegIkPositions[i-1].Z
			}

			originalPrevLeftLegIkBf := originalMotion.BoneFrames.Get(leftLegIkBone.Name()).Get(float32(frames[i-1]))
			if mmath.NearEquals(originalPrevLeftLegIkBf.Position.X, originalLeftLegIkBf.Position.X, 1e-2) {
				leftLegIkPositions[i].X = leftLegIkPositions[i-1].X
			}
			if mmath.NearEquals(originalPrevLeftLegIkBf.Position.Z, originalLeftLegIkBf.Position.Z, 1e-2) {
				leftLegIkPositions[i].Z = leftLegIkPositions[i-1].Z
			}
		}
	}

	mlog.I(mi18n.T("足補正06", map[string]interface{}{"No": sizingSet.Index + 1}))

	for i, iFrame := range frames {
		frame := float32(iFrame)

		centerBf := sizingMotion.BoneFrames.Get(centerBone.Name()).Get(frame)
		centerBf.Position = centerPositions[i]
		sizingMotion.InsertRegisteredBoneFrame(centerBone.Name(), centerBf)

		grooveBf := sizingMotion.BoneFrames.Get(grooveBone.Name()).Get(frame)
		grooveBf.Position = groovePositions[i]
		sizingMotion.InsertRegisteredBoneFrame(grooveBone.Name(), grooveBf)

		rightLegIkBf := sizingMotion.BoneFrames.Get(rightLegIkBone.Name()).Get(frame)
		rightLegIkBf.Position = rightLegIkPositions[i]
		sizingMotion.InsertRegisteredBoneFrame(rightLegIkBone.Name(), rightLegIkBf)

		leftLegIkBf := sizingMotion.BoneFrames.Get(leftLegIkBone.Name()).Get(frame)
		leftLegIkBf.Position = leftLegIkPositions[i]
		sizingMotion.InsertRegisteredBoneFrame(leftLegIkBone.Name(), leftLegIkBf)
	}

	if mlog.IsVerbose() {
		outputPath := mutils.CreateOutputPath(sizingSet.OriginalVmdPath, "Leg_04_移動オフセット")
		repository.NewVmdRepository().Save(outputPath, sizingMotion, true)
		mlog.V("Leg_04_移動オフセット: %s", outputPath)
	}

	mlog.I(mi18n.T("足補正07", map[string]interface{}{"No": sizingSet.Index + 1}))

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

	// サイジング先にFKを焼き込み
	for i, iFrame := range frames {
		frame := float32(iFrame)

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
		outputPath := mutils.CreateOutputPath(sizingSet.OriginalVmdPath, "Leg_05_FK再計算")
		repository.NewVmdRepository().Save(outputPath, sizingMotion, true)
		mlog.V("Leg_05_FK再計算: %s", outputPath)
	}

	mlog.I(mi18n.T("足補正08", map[string]interface{}{"No": sizingSet.Index + 1}))

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
