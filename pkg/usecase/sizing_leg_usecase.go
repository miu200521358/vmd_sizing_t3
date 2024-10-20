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

func SizingLeg(sizingSet *domain.SizingSet, scale *mmath.MVec3, setSize int) (bool, error) {
	if !sizingSet.IsSizingLeg || (sizingSet.IsSizingLeg && sizingSet.CompletedSizingLeg) {
		return false, nil
	}

	if !isValidSizingLower(sizingSet) {
		return false, nil
	}

	mlog.I(mi18n.T("足補正開始", map[string]interface{}{"No": sizingSet.Index + 1}))

	originalModel := sizingSet.OriginalPmx
	originalMotion := sizingSet.OriginalVmd
	sizingModel := sizingSet.SizingPmx
	sizingMotion := sizingSet.OutputVmd

	// ------------

	originalLeftAnkleBone := originalModel.Bones.GetIkTarget(pmx.LEG_IK.Left())
	originalRightAnkleBone := originalModel.Bones.GetIkTarget(pmx.LEG_IK.Right())

	// ------------

	sizingCenterBone := sizingModel.Bones.GetByName(pmx.CENTER.String())
	sizingGrooveBone := sizingModel.Bones.GetByName(pmx.GROOVE.String())

	sizingLeftLegIkBone := sizingModel.Bones.GetByName(pmx.LEG_IK.Left())
	sizingLeftLegBone := sizingModel.Bones.GetByName(pmx.LEG.Left())
	sizingLeftKneeBone := sizingModel.Bones.GetByName(pmx.KNEE.Left())
	sizingLeftAnkleBone := sizingModel.Bones.GetIkTarget(pmx.LEG_IK.Left())
	sizingLeftToeIkBone := sizingModel.Bones.GetByName(pmx.TOE_IK.Left())
	sizingLeftToeBone := sizingModel.Bones.GetIkTarget(pmx.TOE_IK.Left())

	sizingRightLegIkBone := sizingModel.Bones.GetByName(pmx.LEG_IK.Right())
	sizingRightLegBone := sizingModel.Bones.GetByName(pmx.LEG.Right())
	sizingRightKneeBone := sizingModel.Bones.GetByName(pmx.KNEE.Right())
	sizingRightAnkleBone := sizingModel.Bones.GetIkTarget(pmx.LEG_IK.Right())
	sizingRightToeIkBone := sizingModel.Bones.GetByName(pmx.TOE_IK.Right())
	sizingRightToeBone := sizingModel.Bones.GetIkTarget(pmx.TOE_IK.Right())

	mlog.I(mi18n.T("足補正01", map[string]interface{}{"No": sizingSet.Index + 1}))

	frames := sizingMotion.BoneFrames.RegisteredFrames(all_lower_leg_bone_names)
	originalAllDeltas := make([]*delta.VmdDeltas, len(frames))
	blockSize := miter.GetBlockSize(len(frames) * setSize)

	if len(frames) == 0 {
		return false, nil
	}

	// 元モデルのデフォーム(IK ON)
	if err := miter.IterParallelByList(frames, blockSize, func(data, index int) {
		frame := float32(data)
		vmdDeltas := delta.NewVmdDeltas(frame, originalModel.Bones, originalModel.Hash(), originalMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(originalModel, originalMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(originalModel, originalMotion, vmdDeltas, true, frame, all_lower_leg_bone_names, false)
		originalAllDeltas[index] = vmdDeltas
	}); err != nil {
		return false, err
	}

	// サイジング先にFKを焼き込み
	for _, vmdDeltas := range originalAllDeltas {
		{
			// 足
			for _, boneName := range []string{pmx.LEG.Left(), pmx.LEG.Right()} {
				boneDelta := vmdDeltas.Bones.GetByName(boneName)
				if boneDelta == nil {
					continue
				}

				lowerDelta := vmdDeltas.Bones.GetByName(pmx.LOWER.String())

				sizingBf := sizingMotion.BoneFrames.Get(boneName).Get(boneDelta.Frame)
				sizingBf.Rotation = lowerDelta.FilledGlobalMatrix().Inverted().Muled(boneDelta.FilledGlobalMatrix()).Quaternion()
				sizingMotion.InsertRegisteredBoneFrame(boneName, sizingBf)
			}
		}
		{
			// ひざ・足首
			for _, boneName := range []string{pmx.KNEE.Left(), pmx.KNEE.Right(), pmx.ANKLE.Left(), pmx.ANKLE.Right()} {
				boneDelta := vmdDeltas.Bones.GetByName(boneName)
				if boneDelta == nil {
					continue
				}

				sizingBf := sizingMotion.BoneFrames.Get(boneName).Get(boneDelta.Frame)
				sizingBf.Rotation = boneDelta.FilledFrameRotation()
				sizingMotion.InsertRegisteredBoneFrame(boneName, sizingBf)
			}
		}
	}

	if mlog.IsVerbose() {
		kf := vmd.NewIkFrame(0)
		kf.Registered = true
		{
			kef := vmd.NewIkEnableFrame(0)
			kef.BoneName = sizingLeftLegIkBone.Name()
			kef.Enabled = false
			kf.IkList = append(kf.IkList, kef)
		}
		{
			kef := vmd.NewIkEnableFrame(0)
			kef.BoneName = sizingLeftToeIkBone.Name()
			kef.Enabled = false
			kf.IkList = append(kf.IkList, kef)
		}
		{
			kef := vmd.NewIkEnableFrame(0)
			kef.BoneName = sizingRightLegIkBone.Name()
			kef.Enabled = false
			kf.IkList = append(kf.IkList, kef)
		}
		{
			kef := vmd.NewIkEnableFrame(0)
			kef.BoneName = sizingRightToeIkBone.Name()
			kef.Enabled = false
			kf.IkList = append(kf.IkList, kef)
		}
		sizingMotion.InsertIkFrame(kf)

		outputPath := mutils.CreateOutputPath(sizingSet.OriginalVmdPath, "足補正01_FK焼き込み")
		repository.NewVmdRepository().Save(outputPath, sizingMotion, true)
		mlog.V("足補正01_FK焼き込み: %s", outputPath)
	}

	sizingMotion.BoneFrames.Delete(pmx.TOE_IK.Left())
	sizingMotion.BoneFrames.Delete(pmx.TOE_IK.Right())

	mlog.I(mi18n.T("足補正07", map[string]interface{}{"No": sizingSet.Index + 1}))

	centerPositions := make([]*mmath.MVec3, len(frames))
	groovePositions := make([]*mmath.MVec3, len(frames))

	centerTargetBones := []*pmx.Bone{
		sizingModel.Bones.GetByName(pmx.ANKLE.Right()), sizingModel.Bones.GetByName(pmx.ANKLE.Left()),
		sizingModel.Bones.GetByName(pmx.KNEE.Right()), sizingModel.Bones.GetByName(pmx.KNEE.Left()),
		sizingModel.Bones.GetByName(pmx.LEG.Right()), sizingModel.Bones.GetByName(pmx.LEG.Left()),
		sizingModel.Bones.GetByName(pmx.LOWER.String()),
	}

	// 先モデルのデフォーム
	if err := miter.IterParallelByList(frames, blockSize, func(data, index int) {
		frame := float32(data)

		vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, false, frame, all_lower_leg_bone_names, false)

		// 各関節の最も地面に近い位置からセンターを計算する
		centerTargetYs := []float64{
			mmath.Round(originalAllDeltas[index].Bones.GetByName(pmx.ANKLE.Right()).FilledGlobalPosition().Y, 0.1),
			mmath.Round(originalAllDeltas[index].Bones.GetByName(pmx.ANKLE.Left()).FilledGlobalPosition().Y, 0.1),
			mmath.Round(originalAllDeltas[index].Bones.GetByName(pmx.KNEE.Right()).FilledGlobalPosition().Y, 0.1),
			mmath.Round(originalAllDeltas[index].Bones.GetByName(pmx.KNEE.Left()).FilledGlobalPosition().Y, 0.1),
			mmath.Round(originalAllDeltas[index].Bones.GetByName(pmx.LEG.Right()).FilledGlobalPosition().Y, 0.1),
			mmath.Round(originalAllDeltas[index].Bones.GetByName(pmx.LEG.Left()).FilledGlobalPosition().Y, 0.1),
			mmath.Round(originalAllDeltas[index].Bones.GetByName(pmx.LOWER.String()).FilledGlobalPosition().Y, 0.1),
		}

		// 最もY位置が低い関節を処理対象とする(優先度が高いものが上)
		centerTargetBone := centerTargetBones[mmath.ArgMin(centerTargetYs)]
		originalCenterTargetDelta := originalAllDeltas[index].Bones.GetByName(centerTargetBone.Name())

		sizingCenterTargetDelta := vmdDeltas.Bones.GetByName(centerTargetBone.Name())

		originalCenterTargetY := originalCenterTargetDelta.FilledGlobalPosition().Y
		sizingCenterTargetY := sizingCenterTargetDelta.FilledGlobalPosition().Y

		if centerTargetBone.Name() == pmx.ANKLE.Left() || centerTargetBone.Name() == pmx.ANKLE.Right() {
			originalCenterTargetY -= originalModel.Bones.GetByName(centerTargetBone.Name()).Position.Y
			sizingCenterTargetY -= centerTargetBone.Position.Y
		}

		// 元モデルの対象ボーンのY位置*スケールから補正後のY位置を計算
		sizingFixCenterTargetY := originalCenterTargetY * scale.Y
		yDiff := sizingFixCenterTargetY - sizingCenterTargetY

		mlog.V("足補正07[%.0f][%s] originalY[%.4f], sizingY[%.4f], sizingFixY[%.4f], diff[%.4f]",
			frame, centerTargetBone.Name(), originalCenterTargetY, sizingCenterTargetY, sizingFixCenterTargetY, yDiff)

		// センターの位置をスケールに合わせる
		sizingCenterBf := sizingMotion.BoneFrames.Get(sizingCenterBone.Name()).Get(frame)
		centerPositions[index] = sizingCenterBf.Position.Muled(scale)

		sizingGrooveBf := sizingMotion.BoneFrames.Get(sizingGrooveBone.Name()).Get(frame)
		groovePositions[index] = sizingGrooveBf.Position.Added(&mmath.MVec3{X: 0, Y: yDiff, Z: 0})
	}); err != nil {
		return false, err
	}

	// 補正を登録
	for i, iFrame := range frames {
		frame := float32(iFrame)

		sizingCenterBf := sizingMotion.BoneFrames.Get(sizingCenterBone.Name()).Get(frame)
		sizingCenterBf.Position = centerPositions[i]
		sizingMotion.InsertRegisteredBoneFrame(sizingCenterBone.Name(), sizingCenterBf)

		sizingGrooveBf := sizingMotion.BoneFrames.Get(sizingGrooveBone.Name()).Get(frame)
		sizingGrooveBf.Position = groovePositions[i]
		sizingMotion.InsertRegisteredBoneFrame(sizingGrooveBone.Name(), sizingGrooveBf)
	}

	if mlog.IsVerbose() {
		title := "足補正07_センター補正"
		outputPath := mutils.CreateOutputPath(sizingSet.OriginalVmdPath, title)
		repository.NewVmdRepository().Save(outputPath, sizingMotion, true)
		mlog.V("%s: %s", title, outputPath)
	}

	mlog.I(mi18n.T("足補正08", map[string]interface{}{"No": sizingSet.Index + 1}))

	leftLegIkPositions := make([]*mmath.MVec3, len(frames))
	leftLegIkRotations := make([]*mmath.MQuaternion, len(frames))
	rightLegIkPositions := make([]*mmath.MVec3, len(frames))
	rightLegIkRotations := make([]*mmath.MQuaternion, len(frames))

	// 先モデルのデフォーム(IK OFF+センター補正済み)
	if err := miter.IterParallelByList(frames, blockSize, func(data, index int) {
		frame := float32(data)

		vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, false, frame, all_lower_leg_bone_names, false)

		originalLeftAnklePosition := originalAllDeltas[index].Bones.GetByName(pmx.ANKLE.Left()).FilledGlobalPosition()
		originalRightAnklePosition := originalAllDeltas[index].Bones.GetByName(pmx.ANKLE.Right()).FilledGlobalPosition()

		// 地面に近い足底が同じ高さになるように調整
		// originalLegLeftDelta := originalAllDeltas[index].Bones.GetByName(pmx.LEG.Left())
		originalLeftHeelDelta := originalAllDeltas[index].Bones.GetByName(pmx.HEEL.Left())
		originalLeftToeTailDelta := originalAllDeltas[index].Bones.GetByName(pmx.TOE_T.Left())
		// originalLegRightDelta := originalAllDeltas[index].Bones.GetByName(pmx.LEG.Right())
		// originalAnkleRightDelta := originalAllDeltas[index].Bones.GetByName(pmx.ANKLE.Right())
		originalRightHeelDelta := originalAllDeltas[index].Bones.GetByName(pmx.HEEL.Right())
		originalRightToeTailDelta := originalAllDeltas[index].Bones.GetByName(pmx.TOE_T.Right())

		sizingLeftAnkleDelta := vmdDeltas.Bones.GetByName(pmx.ANKLE.Left())
		sizingLeftHeelDelta := vmdDeltas.Bones.GetByName(pmx.HEEL.Left())
		sizingLeftToeDelta := vmdDeltas.Bones.GetByName(sizingLeftToeBone.Name())
		sizingLeftToeTailDelta := vmdDeltas.Bones.GetByName(pmx.TOE_T.Left())

		sizingRightAnkleDelta := vmdDeltas.Bones.GetByName(pmx.ANKLE.Right())
		sizingRightHeelDelta := vmdDeltas.Bones.GetByName(pmx.HEEL.Right())
		sizingRightToeDelta := vmdDeltas.Bones.GetByName(sizingRightToeBone.Name())
		sizingRightToeTailDelta := vmdDeltas.Bones.GetByName(pmx.TOE_T.Right())

		// 足IKから見た足首の位置
		leftLegIkPositions[index] = sizingLeftAnkleDelta.FilledGlobalPosition().Subed(sizingLeftLegIkBone.Position)
		rightLegIkPositions[index] = sizingRightAnkleDelta.FilledGlobalPosition().Subed(sizingRightLegIkBone.Position)

		calcLegIkPositionY(index, frame, "左", leftLegIkPositions, originalLeftAnkleBone, originalLeftAnklePosition,
			originalLeftToeTailDelta, originalLeftHeelDelta, sizingLeftToeTailDelta, sizingLeftHeelDelta, scale)

		calcLegIkPositionY(index, frame, "右", rightLegIkPositions, originalRightAnkleBone, originalRightAnklePosition,
			originalRightToeTailDelta, originalRightHeelDelta, sizingRightToeTailDelta, sizingRightHeelDelta, scale)

		// 足首から見たつま先IKの方向
		leftLegIkMat := sizingLeftToeIkBone.Position.Subed(sizingLeftLegIkBone.Position).Normalize().ToLocalMat()
		leftLegFkMat := sizingLeftToeDelta.FilledGlobalPosition().Subed(
			sizingLeftAnkleDelta.FilledGlobalPosition()).Normalize().ToLocalMat()
		leftLegIkRotations[index] = leftLegFkMat.Muled(leftLegIkMat.Inverted()).Quaternion()

		rightLegIkMat := sizingRightToeIkBone.Position.Subed(sizingRightLegIkBone.Position).Normalize().ToLocalMat()
		rightLegFkMat := sizingRightToeDelta.FilledGlobalPosition().Subed(
			sizingRightAnkleDelta.FilledGlobalPosition()).Normalize().ToLocalMat()
		rightLegIkRotations[index] = rightLegFkMat.Muled(rightLegIkMat.Inverted()).Quaternion()
	}); err != nil {
		return false, err
	}

	for i, iFrame := range frames {
		frame := float32(iFrame)

		originalLeftAnklePosition := originalAllDeltas[i].Bones.GetByName(pmx.ANKLE.Left()).FilledGlobalPosition()
		originalRightAnklePosition := originalAllDeltas[i].Bones.GetByName(pmx.ANKLE.Right()).FilledGlobalPosition()

		if i < len(frames)-1 {
			originalLeftAnkleNextPosition := originalAllDeltas[i+1].Bones.GetByName(pmx.ANKLE.Left()).FilledGlobalPosition()
			originalRightAnkleNextPosition := originalAllDeltas[i+1].Bones.GetByName(pmx.ANKLE.Right()).FilledGlobalPosition()

			// 前と同じ位置なら同じ位置にする
			if mmath.NearEquals(originalRightAnkleNextPosition.X, originalRightAnklePosition.X, 1e-2) {
				rightLegIkPositions[i].X = rightLegIkPositions[i+1].X
			}
			if mmath.NearEquals(originalRightAnkleNextPosition.Y, originalRightAnklePosition.Y, 1e-2) {
				rightLegIkPositions[i].Y = rightLegIkPositions[i+1].Y
			}
			if mmath.NearEquals(originalRightAnkleNextPosition.Z, originalRightAnklePosition.Z, 1e-2) {
				rightLegIkPositions[i].Z = rightLegIkPositions[i+1].Z
			}

			if mmath.NearEquals(originalLeftAnkleNextPosition.X, originalLeftAnklePosition.X, 1e-2) {
				leftLegIkPositions[i].X = leftLegIkPositions[i+1].X
			}
			if mmath.NearEquals(originalLeftAnkleNextPosition.Y, originalLeftAnklePosition.Y, 1e-2) {
				leftLegIkPositions[i].Y = leftLegIkPositions[i+1].Y
			}
			if mmath.NearEquals(originalLeftAnkleNextPosition.Z, originalLeftAnklePosition.Z, 1e-2) {
				leftLegIkPositions[i].Z = leftLegIkPositions[i+1].Z
			}
		}

		rightLegIkBf := sizingMotion.BoneFrames.Get(sizingRightLegIkBone.Name()).Get(frame)
		rightLegIkBf.Position = rightLegIkPositions[i]
		rightLegIkBf.Rotation = rightLegIkRotations[i]
		sizingMotion.InsertRegisteredBoneFrame(sizingRightLegIkBone.Name(), rightLegIkBf)

		leftLegIkBf := sizingMotion.BoneFrames.Get(sizingLeftLegIkBone.Name()).Get(frame)
		leftLegIkBf.Position = leftLegIkPositions[i]
		leftLegIkBf.Rotation = leftLegIkRotations[i]
		sizingMotion.InsertRegisteredBoneFrame(sizingLeftLegIkBone.Name(), leftLegIkBf)
	}

	if mlog.IsVerbose() {
		sizingMotion.IkFrames.Delete(0)

		title := "足補正08_足IK補正"
		outputPath := mutils.CreateOutputPath(sizingSet.OriginalVmdPath, title)
		repository.NewVmdRepository().Save(outputPath, sizingMotion, true)
		mlog.V("%s: %s", title, outputPath)
	}

	mlog.I(mi18n.T("足補正09", map[string]interface{}{"No": sizingSet.Index + 1}))

	leftLegRotations := make([]*mmath.MQuaternion, len(frames))
	leftKneeRotations := make([]*mmath.MQuaternion, len(frames))
	leftAnkleRotations := make([]*mmath.MQuaternion, len(frames))
	rightLegRotations := make([]*mmath.MQuaternion, len(frames))
	rightKneeRotations := make([]*mmath.MQuaternion, len(frames))
	rightAnkleRotations := make([]*mmath.MQuaternion, len(frames))

	// 足IK再計算
	// 元モデルのデフォーム(IK ON)
	if err := miter.IterParallelByList(frames, blockSize, func(data, index int) {
		frame := float32(data)

		vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, true, frame, all_lower_leg_bone_names, false)

		leftLegRotations[index] = vmdDeltas.Bones.Get(sizingLeftLegBone.Index()).FilledFrameRotation()
		leftKneeRotations[index] = vmdDeltas.Bones.Get(sizingLeftKneeBone.Index()).FilledFrameRotation()
		leftAnkleRotations[index] = vmdDeltas.Bones.Get(sizingLeftAnkleBone.Index()).FilledFrameRotation()

		rightLegRotations[index] = vmdDeltas.Bones.Get(sizingRightLegBone.Index()).FilledFrameRotation()
		rightKneeRotations[index] = vmdDeltas.Bones.Get(sizingRightKneeBone.Index()).FilledFrameRotation()
		rightAnkleRotations[index] = vmdDeltas.Bones.Get(sizingRightAnkleBone.Index()).FilledFrameRotation()
	}); err != nil {
		return false, err
	}

	registerLegFk(frames, sizingMotion, sizingLeftLegBone, sizingLeftKneeBone, sizingLeftAnkleBone, sizingRightLegBone,
		sizingRightKneeBone, sizingRightAnkleBone, leftLegRotations, leftKneeRotations, leftAnkleRotations,
		rightLegRotations, rightKneeRotations, rightAnkleRotations)

	if mlog.IsVerbose() {
		sizingMotion.IkFrames.Delete(0)

		title := "足補正09_FK再計算"
		outputPath := mutils.CreateOutputPath(sizingSet.OriginalVmdPath, title)
		repository.NewVmdRepository().Save(outputPath, sizingMotion, true)
		mlog.V("%s: %s", title, outputPath)
	}

	sizingSet.CompletedSizingLeg = true
	return true, nil
}

func calcLegIkPositionY(
	index int,
	frame float32,
	direction string,
	legIkPositions []*mmath.MVec3,
	originalAnkleBone *pmx.Bone,
	originalAnklePosition *mmath.MVec3,
	originalToeTailDelta, originalHeelDelta, sizingToeTailDelta, sizingHeelDelta *delta.BoneDelta,
	scale *mmath.MVec3,
) {

	// 左足IK-Yの位置を調整
	if mmath.NearEquals(originalAnklePosition.Y, 0, 1e-2) {
		legIkPositions[index].Y = 0
		return
	}

	if originalToeTailDelta.FilledGlobalPosition().Y <= originalHeelDelta.FilledGlobalPosition().Y {
		// つま先の方がかかとより低い場合
		originalLeftToeTailY := originalToeTailDelta.FilledGlobalPosition().Y

		// つま先のY座標を元モデルのつま先のY座標*スケールに合わせる
		sizingLeftToeTailY := originalLeftToeTailY * scale.Y

		// 現時点のつま先のY座標
		actualLeftToeTailY := sizingToeTailDelta.FilledGlobalPosition().Y

		leftToeDiff := sizingLeftToeTailY - actualLeftToeTailY
		lerpLeftToeDiff := mmath.LerpFloat(leftToeDiff, 0,
			originalToeTailDelta.FilledGlobalPosition().Y/originalAnkleBone.Position.Y)
		// 足首Y位置に近付くにつれて補正を弱める
		legIkPositions[index].Y += lerpLeftToeDiff
		mlog.V("足補正08[%.0f][%sつま先] originalLeftY[%.4f], sizingLeftY[%.4f], actualLeftY[%.4f], diff[%.4f], lerp[%.4f]",
			frame, direction, originalLeftToeTailY, sizingLeftToeTailY, actualLeftToeTailY, leftToeDiff, lerpLeftToeDiff)

		return
	}

	// かかとの方がつま先より低い場合
	originalLeftHeelY := originalHeelDelta.FilledGlobalPosition().Y

	// かかとのY座標を元モデルのかかとのY座標*スケールに合わせる
	sizingLeftHeelY := originalLeftHeelY * scale.Y

	// 現時点のかかとのY座標
	actualLeftHeelY := sizingHeelDelta.FilledGlobalPosition().Y

	leftHeelDiff := sizingLeftHeelY - actualLeftHeelY
	lerpLeftHeelDiff := mmath.LerpFloat(leftHeelDiff, 0,
		originalHeelDelta.FilledGlobalPosition().Y/originalAnkleBone.Position.Y)
	// 足首Y位置に近付くにつれて補正を弱める
	legIkPositions[index].Y += lerpLeftHeelDiff

	mlog.V("足補正08[%.0f][%sかかと] originalLeftY[%.4f], sizingLeftY[%.4f], actualLeftY[%.4f], diff[%.4f], lerp[%.4f]",
		frame, direction, originalLeftHeelY, sizingLeftHeelY, actualLeftHeelY, leftHeelDiff, lerpLeftHeelDiff)
}

// func deformLegIk(
// 	index int,
// 	frame float32,
// 	sizingModel *pmx.PmxModel,
// 	sizingMotion *vmd.VmdMotion,
// 	originalAllDeltas []*delta.VmdDeltas,
// 	sizingDeltas *delta.VmdDeltas,
// 	originalSrcBone *pmx.Bone,
// 	originalDstBone *pmx.Bone,
// 	sizingSrcBone *pmx.Bone,
// 	sizingDstBone *pmx.Bone,
// 	sizingIkBone *pmx.Bone,
// 	sizingSlopeMat *mmath.MMat4,
// 	scale float64,
// ) (dstIkDeltas *delta.VmdDeltas, diffSrcRotation *mmath.MQuaternion, sizingFixDstGlobalPosition *mmath.MVec3) {
// 	// 元から見た先の相対位置を取得
// 	originalSrcDelta := originalAllDeltas[index].Bones.Get(originalSrcBone.Index())
// 	originalDstDelta := originalAllDeltas[index].Bones.Get(originalDstBone.Index())

// 	// 元から見た先の相対位置をスケールに合わせる
// 	originalSrcLocalPosition := originalDstDelta.FilledGlobalPosition().Subed(originalSrcDelta.FilledGlobalPosition())
// 	sizingDstLocalPosition := originalSrcLocalPosition.MuledScalar(scale)
// 	sizingDstSlopeLocalPosition := sizingSlopeMat.MulVec3(sizingDstLocalPosition)

// 	// Fixさせた新しい先のグローバル位置を取得
// 	sizingSrcDelta := sizingDeltas.Bones.Get(sizingSrcBone.Index())
// 	sizingFixDstGlobalPosition = sizingSrcDelta.FilledGlobalPosition().Added(sizingDstSlopeLocalPosition)

// 	// IK結果を返す
// 	dstIkDeltas = deform.DeformIk(sizingModel, sizingMotion, sizingDeltas, frame, sizingIkBone,
// 		sizingFixDstGlobalPosition, []string{sizingSrcBone.Name(), sizingDstBone.Name()})

// 	originalSrcRotation := originalAllDeltas[index].Bones.Get(originalSrcBone.Index()).FilledFrameRotation()
// 	sizingSrcRotation := dstIkDeltas.Bones.Get(sizingSrcBone.Index()).FilledFrameRotation()

// 	// IK結果の回転差分
// 	diffSrcRotation = sizingSrcRotation.Muled(originalSrcRotation.Inverted()).Inverted()

// 	return dstIkDeltas, diffSrcRotation, sizingFixDstGlobalPosition
// }

func registerLegFk(
	frames []int,
	sizingMotion *vmd.VmdMotion,
	sizingLeftLegBone, sizingLeftKneeBone, sizingLeftAnkleBone,
	sizingRightLegBone, sizingRightKneeBone, sizingRightAnkleBone *pmx.Bone,
	leftLegRotations, leftKneeRotations, leftAnkleRotations,
	rightLegRotations, rightKneeRotations, rightAnkleRotations []*mmath.MQuaternion,
) (sizingLeftFkAllDeltas, sizingRightFkAllDeltas []*delta.VmdDeltas) {

	// サイジング先にFKを焼き込み
	for i, iFrame := range frames {
		frame := float32(iFrame)

		{
			bf := sizingMotion.BoneFrames.Get(sizingLeftLegBone.Name()).Get(frame)
			bf.Rotation = leftLegRotations[i]
			sizingMotion.InsertRegisteredBoneFrame(sizingLeftLegBone.Name(), bf)
		}
		{
			bf := sizingMotion.BoneFrames.Get(sizingLeftKneeBone.Name()).Get(frame)
			bf.Rotation = leftKneeRotations[i]
			sizingMotion.InsertRegisteredBoneFrame(sizingLeftKneeBone.Name(), bf)
		}
		{
			bf := sizingMotion.BoneFrames.Get(sizingLeftAnkleBone.Name()).Get(frame)
			bf.Rotation = leftAnkleRotations[i]
			sizingMotion.InsertRegisteredBoneFrame(sizingLeftAnkleBone.Name(), bf)
		}
		{
			bf := sizingMotion.BoneFrames.Get(sizingRightLegBone.Name()).Get(frame)
			bf.Rotation = rightLegRotations[i]
			sizingMotion.InsertRegisteredBoneFrame(sizingRightLegBone.Name(), bf)
		}
		{
			bf := sizingMotion.BoneFrames.Get(sizingRightKneeBone.Name()).Get(frame)
			bf.Rotation = rightKneeRotations[i]
			sizingMotion.InsertRegisteredBoneFrame(sizingRightKneeBone.Name(), bf)
		}
		{
			bf := sizingMotion.BoneFrames.Get(sizingRightAnkleBone.Name()).Get(frame)
			bf.Rotation = rightAnkleRotations[i]
			sizingMotion.InsertRegisteredBoneFrame(sizingRightAnkleBone.Name(), bf)
		}
	}

	return sizingLeftFkAllDeltas, sizingRightFkAllDeltas
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
