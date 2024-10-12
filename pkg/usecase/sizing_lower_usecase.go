package usecase

import (
	"fmt"

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

func SizingLower(sizingSet *domain.SizingSet) {
	if !sizingSet.IsSizingLower || (sizingSet.IsSizingLower && sizingSet.CompletedSizingLower) {
		return
	}

	if !isValidSizingLower(sizingSet) {
		return
	}

	originalModel := sizingSet.OriginalPmx
	originalMotion := sizingSet.OriginalVmd
	sizingModel := sizingSet.SizingPmx
	sizingMotion := sizingSet.OutputVmd

	originalLegCenterBone := originalModel.Bones.GetByName(pmx.LEG_CENTER.String())
	originalLowerRootBone := originalModel.Bones.GetByName(pmx.LOWER_ROOT.String())
	originalLowerBone := originalModel.Bones.GetByName(pmx.LOWER.String())
	originalLeftLegBone := originalModel.Bones.GetByName(pmx.LEG.Left())
	originalRightLegBone := originalModel.Bones.GetByName(pmx.LEG.Right())

	centerBone := sizingModel.Bones.GetByName(pmx.CENTER.String())
	grooveBone := sizingModel.Bones.GetByName(pmx.GROOVE.String())
	lowerRootBone := sizingModel.Bones.GetByName(pmx.LOWER_ROOT.String())
	lowerBone := sizingModel.Bones.GetByName(pmx.LOWER.String())
	legCenterBone := sizingModel.Bones.GetByName(pmx.LEG_CENTER.String())
	rightLegIkBone := sizingModel.Bones.GetByName(pmx.LEG_IK.Right())
	rightToeBone := sizingModel.Bones.Get(sizingModel.Bones.GetByName(pmx.TOE_IK.Right()).Ik.BoneIndex)
	rightToeIkBone := sizingModel.Bones.GetByName(pmx.TOE_IK.Right())
	rightLegBone := sizingModel.Bones.GetByName(pmx.LEG.Right())
	rightKneeBone := sizingModel.Bones.GetByName(pmx.KNEE.Right())
	rightAnkleBone := sizingModel.Bones.Get(sizingModel.Bones.GetByName(pmx.LEG_IK.Right()).Ik.BoneIndex)
	leftLegIkBone := sizingModel.Bones.GetByName(pmx.LEG_IK.Left())
	leftToeBone := sizingModel.Bones.Get(sizingModel.Bones.GetByName(pmx.TOE_IK.Left()).Ik.BoneIndex)
	leftToeIkBone := sizingModel.Bones.GetByName(pmx.TOE_IK.Left())
	leftLegBone := sizingModel.Bones.GetByName(pmx.LEG.Left())
	leftKneeBone := sizingModel.Bones.GetByName(pmx.KNEE.Left())
	leftAnkleBone := sizingModel.Bones.Get(sizingModel.Bones.GetByName(pmx.LEG_IK.Left()).Ik.BoneIndex)

	// 左ひざIK
	leftKneeIkBone := pmx.NewBoneByName(fmt.Sprintf("%s%sIk", pmx.MLIB_PREFIX, leftKneeBone.Name()))
	leftKneeIkBone.Position = leftKneeBone.Position
	leftKneeIkBone.Ik = pmx.NewIk()
	leftKneeIkBone.Ik.BoneIndex = leftKneeBone.Index()
	leftKneeIkBone.Ik.LoopCount = leftLegIkBone.Ik.LoopCount
	leftKneeIkBone.Ik.UnitRotation = leftLegIkBone.Ik.UnitRotation
	leftKneeIkBone.Ik.Links = make([]*pmx.IkLink, 1)
	leftKneeIkBone.Ik.Links[0] = pmx.NewIkLink()
	leftKneeIkBone.Ik.Links[0].BoneIndex = leftLegBone.Index()

	// 右ひざIK
	rightKneeIkBone := pmx.NewBoneByName(fmt.Sprintf("%s%sIk", pmx.MLIB_PREFIX, rightKneeBone.Name()))
	rightKneeIkBone.Position = rightKneeBone.Position
	rightKneeIkBone.Ik = pmx.NewIk()
	rightKneeIkBone.Ik.BoneIndex = rightKneeBone.Index()
	rightKneeIkBone.Ik.LoopCount = rightLegIkBone.Ik.LoopCount
	rightKneeIkBone.Ik.UnitRotation = rightLegIkBone.Ik.UnitRotation
	rightKneeIkBone.Ik.Links = make([]*pmx.IkLink, 1)
	rightKneeIkBone.Ik.Links[0] = pmx.NewIkLink()
	rightKneeIkBone.Ik.Links[0].BoneIndex = rightLegBone.Index()

	// 左足首IK
	leftAnkleIkBone := pmx.NewBoneByName(fmt.Sprintf("%s%sIk", pmx.MLIB_PREFIX, leftAnkleBone.Name()))
	leftAnkleIkBone.Position = leftAnkleBone.Position
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
	rightAnkleIkBone.Position = rightAnkleBone.Position
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

	mlog.I(mi18n.T("下半身補正開始", map[string]interface{}{"No": sizingSet.Index + 1}))

	frames := sizingMotion.BoneFrames.RegisteredFrames([]string{pmx.LOWER.String(), pmx.CENTER.String(), pmx.GROOVE.String(), pmx.LEG.Left(), pmx.LEG.Right(), pmx.KNEE.Left(), pmx.KNEE.Right(), pmx.ANKLE.Left(), pmx.ANKLE.Right()})

	sizingLowerDeltas := make([]*delta.VmdDeltas, len(frames))
	// sizingLegIkOnDeltas := make([]*delta.VmdDeltas, len(frames))
	lowerRotations := make([]*mmath.MQuaternion, len(frames))

	mlog.I(mi18n.T("下半身補正01", map[string]interface{}{"No": sizingSet.Index + 1}))

	// 先モデルのデフォーム(IK ON)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)

		sizingVmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
		sizingVmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
		sizingVmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, sizingVmdDeltas, true, frame, leg_all_bone_names, false)
		sizingLowerDeltas[index] = sizingVmdDeltas
	})

	// 元モデル初期姿勢ベクトル
	originalInitialLowerDirection := originalLegCenterBone.Position.Subed(originalLowerBone.Position)
	// originalInitialLowerUp := originalLeftLegBone.Position.Subed(originalRightLegBone.Position)
	// originalInitialLowerSlope := mmath.NewMQuaternionFromDirection(
	// 	originalInitialLowerDirection.Normalized(), originalInitialLowerUp.Normalized())

	// 先モデル初期姿勢ベクトル
	sizingInitialLowerDirection := legCenterBone.Position.Subed(lowerBone.Position)
	sizingInitialLowerUp := leftLegBone.Position.Subed(rightLegBone.Position)
	sizingInitialLowerSlope := mmath.NewMQuaternionFromDirection(
		sizingInitialLowerDirection.Normalized(), sizingInitialLowerUp.Normalized())

	sizingInitialLowerRatio := sizingInitialLowerDirection.Length() / originalInitialLowerDirection.Length()

	originalLowerLocalMat := originalLowerBone.Position.Subed(originalLowerRootBone.Position).ToMat4()
	originalLegCenterVector := originalLegCenterBone.Position.Subed(originalLowerBone.Position).Normalized()
	originalLegCenterLocalPosition := originalLegCenterBone.Position.Subed(originalLowerBone.Position)

	sizingLowerLocalMat := lowerBone.Position.Subed(lowerRootBone.Position).ToMat4()
	sizingLegCenterVector := legCenterBone.Position.Subed(lowerBone.Position).Normalized()
	sizingLegCenterLocalPosition := legCenterBone.Position.Subed(lowerBone.Position)

	// 元モデルと先モデルの足中心の差分（主にZ方向）
	sizingLegCenterDiffVector := originalLegCenterLocalPosition.MuledScalar(sizingInitialLowerRatio).Subed(sizingLegCenterLocalPosition)

	// 元モデルのデフォーム(IK ON)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)

		vmdDeltas := delta.NewVmdDeltas(frame, originalModel.Bones, originalModel.Hash(), originalMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(originalModel, originalMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(originalModel, originalMotion, vmdDeltas, true, frame, leg_all_bone_names, false)

		originalLegCenterDelta := vmdDeltas.Bones.Get(originalLegCenterBone.Index())
		originalLowerRootDelta := vmdDeltas.Bones.Get(originalLowerRootBone.Index())
		originalLowerDelta := vmdDeltas.Bones.Get(originalLowerBone.Index())
		originalLeftLegDelta := vmdDeltas.Bones.Get(originalLeftLegBone.Index())
		originalRightLegDelta := vmdDeltas.Bones.Get(originalRightLegBone.Index())

		originalTwistQuat, _ := originalLowerDelta.FilledFrameRotation().SeparateTwistByAxis(originalLegCenterVector)

		// 元モデルで下半身軸回転だけを加味した場合の足中心グローバル行列
		originalTwistLegCenterGlobalPosition := originalLowerRootDelta.FilledGlobalMatrix().Muled(originalLowerLocalMat).Muled(originalTwistQuat.ToMat4()).MulVec3(originalLegCenterLocalPosition)

		// 下半身軸回転だけした時の足中心グローバル行列から見た全回転加味した足中心の差分
		originalLegCenterLocalDiff := originalLegCenterDelta.FilledGlobalPosition().Subed(originalTwistLegCenterGlobalPosition)

		// 左足から右足へのベクトル
		originalLowerUp := originalLeftLegDelta.FilledGlobalPosition().Subed(
			originalRightLegDelta.FilledGlobalPosition()).Normalized()

		sizingLowerDelta := sizingLowerDeltas[index].Bones.Get(lowerBone.Index())
		sizingLowerRootDelta := sizingLowerDeltas[index].Bones.Get(lowerRootBone.Index())
		sizingLegCenterDelta := sizingLowerDeltas[index].Bones.Get(legCenterBone.Index())

		sizingTwistQuat, _ := sizingLowerDelta.FilledFrameRotation().SeparateTwistByAxis(sizingLegCenterVector)

		// 先モデルの下半身根元から見た足中心のベクトル
		sizingLegCenterFixGlobalPosition := sizingLowerRootDelta.FilledGlobalMatrix().Muled(sizingLowerLocalMat).Muled(sizingTwistQuat.ToMat4()).MulVec3(sizingLegCenterLocalPosition.Added(originalLegCenterLocalDiff.MuledScalar(sizingInitialLowerRatio).Added(sizingLegCenterDiffVector)))
		sizingLowerDirection := sizingLegCenterFixGlobalPosition.Subed(sizingLowerDelta.FilledGlobalPosition())
		sizingLowerSlope := mmath.NewMQuaternionFromDirection(sizingLowerDirection.Normalized(), originalLowerUp)

		mlog.V("下半身補正01[%.0f] originalTwistLegCenterGlobalPosition[%v], originalLegCenterPosition[%v], originalLegCenterLocalDiff[%v], sizingLegCenterFixGlobalPosition[%v], sizingLegCenterGlobalPosition[%v], sizingLowerDirection[%v]", frame, originalTwistLegCenterGlobalPosition, originalLegCenterDelta.FilledGlobalPosition(), originalLegCenterLocalDiff, sizingLegCenterFixGlobalPosition, sizingLegCenterDelta.FilledGlobalPosition(), sizingLowerDirection)

		lowerRotations[index] = sizingLowerSlope.Muled(sizingInitialLowerSlope.Inverted())
	})

	// 下半身の回転を補正登録
	for i, iFrame := range frames {
		frame := float32(iFrame)

		sizingLowerBf := sizingMotion.BoneFrames.Get(lowerBone.Name()).Get(frame)
		sizingLowerBf.Rotation = lowerRotations[i]
		sizingMotion.InsertRegisteredBoneFrame(lowerBone.Name(), sizingLowerBf)
	}

	if mlog.IsVerbose() {
		outputPath := mutils.CreateOutputPath(sizingSet.OriginalVmdPath, "下半身補正01_下半身回転")
		repository.NewVmdRepository().Save(outputPath, sizingMotion, true)
		mlog.V("下半身補正01_下半身回転: %s", outputPath)
	}

	sizingLowerFixDeltas := make([]*delta.VmdDeltas, len(frames))
	centerPositions := make([]*mmath.MVec3, len(frames))
	groovePositions := make([]*mmath.MVec3, len(frames))

	mlog.I(mi18n.T("下半身補正02", map[string]interface{}{"No": sizingSet.Index + 1}))

	frames = sizingMotion.BoneFrames.RegisteredFrames([]string{pmx.LOWER.String(), pmx.CENTER.String(), pmx.GROOVE.String()})

	// 先モデルのデフォーム(下半身補正済み)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, true, frame, leg_all_bone_names, false)
		sizingLowerFixDeltas[index] = vmdDeltas

		legCenterDelta := sizingLowerDeltas[index].Bones.Get(legCenterBone.Index())
		legCenterFixDelta := sizingLowerFixDeltas[index].Bones.Get(legCenterBone.Index())

		lowerDiff := legCenterDelta.FilledGlobalPosition().Subed(legCenterFixDelta.FilledGlobalPosition())

		centerBf := sizingMotion.BoneFrames.Get(centerBone.Name()).Get(frame)
		centerPositions[index] = centerBf.Position.Added(&mmath.MVec3{X: lowerDiff.X, Y: 0, Z: lowerDiff.Z})

		grooveBf := sizingMotion.BoneFrames.Get(grooveBone.Name()).Get(frame)
		groovePositions[index] = grooveBf.Position.Added(&mmath.MVec3{X: 0, Y: lowerDiff.Y, Z: 0})
	})

	// センターの移動を補正登録
	for i, iFrame := range frames {
		frame := float32(iFrame)

		centerBf := sizingMotion.BoneFrames.Get(centerBone.Name()).Get(frame)
		centerBf.Position = centerPositions[i]
		sizingMotion.InsertRegisteredBoneFrame(centerBone.Name(), centerBf)

		grooveBf := sizingMotion.BoneFrames.Get(grooveBone.Name()).Get(frame)
		grooveBf.Position = groovePositions[i]
		sizingMotion.InsertRegisteredBoneFrame(grooveBone.Name(), grooveBf)
	}

	if mlog.IsVerbose() {
		outputPath := mutils.CreateOutputPath(sizingSet.OriginalVmdPath, "下半身補正02_センター")
		repository.NewVmdRepository().Save(outputPath, sizingMotion, true)
		mlog.V("下半身補正02_センター: %s", outputPath)
	}

	// // ひざと足首を一旦除去
	// sizingMotion.BoneFrames.Delete(leftKneeBone.Name())
	// sizingMotion.BoneFrames.Delete(leftAnkleBone.Name())
	// sizingMotion.BoneFrames.Delete(rightKneeBone.Name())
	// sizingMotion.BoneFrames.Delete(rightAnkleBone.Name())

	// sizingResultDeltas := make([]*delta.VmdDeltas, len(frames))

	// // 先モデルのデフォーム(IK ON)
	// miter.IterParallelByList(frames, 500, func(data, index int) {
	// 	frame := float32(data)
	// 	vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
	// 	vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
	// 	vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, true, frame, leg_all_bone_names, false)
	// 	sizingResultDeltas[index] = vmdDeltas
	// })

	// // サイジング先にFKを焼き込み
	// for _, vmdDeltas := range sizingResultDeltas {
	// 	for _, boneDelta := range vmdDeltas.Bones.Data {
	// 		if boneDelta == nil || !boneDelta.Bone.IsLegFK() {
	// 			continue
	// 		}

	// 		sizingBf := sizingMotion.BoneFrames.Get(boneDelta.Bone.Name()).Get(boneDelta.Frame)

	// 		// 最終的な足FKを焼き込み
	// 		sizingBf.Rotation = boneDelta.FilledFrameRotation()
	// 		sizingMotion.InsertRegisteredBoneFrame(boneDelta.Bone.Name(), sizingBf)
	// 	}
	// }

	mlog.I(mi18n.T("下半身補正03", map[string]interface{}{"No": sizingSet.Index + 1}))

	frames = sizingMotion.BoneFrames.RegisteredFrames([]string{pmx.LOWER.String(), pmx.LEG.Left(), pmx.LEG.Right()})

	leftLegRotations := make([]*mmath.MQuaternion, len(frames))
	rightLegRotations := make([]*mmath.MQuaternion, len(frames))

	// 先モデルの足角度追加補正(IK ON)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)

		// ひざを固定した場合の足の回転
		{
			vmdDeltas := deform.DeformIk(sizingModel, sizingMotion, frame, leftKneeIkBone,
				sizingLowerDeltas[index].Bones.Get(leftKneeBone.Index()).FilledGlobalPosition())
			leftLegRotations[index] = vmdDeltas.Bones.Get(leftLegBone.Index()).FilledFrameRotation()
		}
		{
			vmdDeltas := deform.DeformIk(sizingModel, sizingMotion, frame, rightKneeIkBone,
				sizingLowerDeltas[index].Bones.Get(rightKneeBone.Index()).FilledGlobalPosition())
			rightLegRotations[index] = vmdDeltas.Bones.Get(rightLegBone.Index()).FilledFrameRotation()
		}
	})

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

	if mlog.IsVerbose() {
		outputPath := mutils.CreateOutputPath(sizingSet.OriginalVmdPath, "下半身補正03_足")
		repository.NewVmdRepository().Save(outputPath, sizingMotion, true)
		mlog.V("下半身補正03_足: %s", outputPath)
	}

	mlog.I(mi18n.T("下半身補正04", map[string]interface{}{"No": sizingSet.Index + 1}))

	frames = sizingMotion.BoneFrames.RegisteredFrames([]string{pmx.LOWER.String(), pmx.KNEE.Left(), pmx.KNEE.Right()})

	leftKneeRotations := make([]*mmath.MQuaternion, len(frames))
	rightKneeRotations := make([]*mmath.MQuaternion, len(frames))

	// 先モデルの足角度追加補正(IK ON)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		// 足首を固定した場合のひざの回転
		{
			vmdDeltas := deform.DeformIk(sizingModel, sizingMotion, frame, leftAnkleIkBone,
				sizingLowerDeltas[index].Bones.Get(leftLegIkBone.Index()).FilledGlobalPosition())
			leftKneeRotations[index] = vmdDeltas.Bones.Get(leftKneeBone.Index()).FilledFrameRotation()
		}
		{
			vmdDeltas := deform.DeformIk(sizingModel, sizingMotion, frame, rightAnkleIkBone,
				sizingLowerDeltas[index].Bones.Get(rightLegIkBone.Index()).FilledGlobalPosition())
			rightKneeRotations[index] = vmdDeltas.Bones.Get(rightKneeBone.Index()).FilledFrameRotation()
		}
	})

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

	if mlog.IsVerbose() {
		outputPath := mutils.CreateOutputPath(sizingSet.OriginalVmdPath, "下半身補正04_ひざ")
		repository.NewVmdRepository().Save(outputPath, sizingMotion, true)
		mlog.V("下半身補正04_ひざ: %s", outputPath)
	}

	mlog.I(mi18n.T("下半身補正05", map[string]interface{}{"No": sizingSet.Index + 1}))

	frames = sizingMotion.BoneFrames.RegisteredFrames([]string{pmx.LOWER.String(), pmx.ANKLE.Left(), pmx.ANKLE.Right()})

	leftAnkleRotations := make([]*mmath.MQuaternion, len(frames))
	rightAnkleRotations := make([]*mmath.MQuaternion, len(frames))

	// 先モデルの足角度追加補正(IK ON)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		// つま先を固定した場合の足首の回転
		{
			vmdDeltas := deform.DeformIk(sizingModel, sizingMotion, frame, leftToeIkBone,
				sizingLowerDeltas[index].Bones.Get(leftToeBone.Index()).FilledGlobalPosition())
			leftAnkleRotations[index] = vmdDeltas.Bones.Get(leftAnkleBone.Index()).FilledFrameRotation()
		}
		{
			vmdDeltas := deform.DeformIk(sizingModel, sizingMotion, frame, rightToeIkBone,
				sizingLowerDeltas[index].Bones.Get(rightToeBone.Index()).FilledGlobalPosition())
			rightAnkleRotations[index] = vmdDeltas.Bones.Get(rightAnkleBone.Index()).FilledFrameRotation()
		}
	})

	// 補正を登録
	for i, iFrame := range frames {
		frame := float32(iFrame)

		sizingLeftAnkleBf := sizingMotion.BoneFrames.Get(leftAnkleBone.Name()).Get(frame)
		sizingLeftAnkleBf.Rotation = leftAnkleRotations[i]
		sizingMotion.InsertRegisteredBoneFrame(leftAnkleBone.Name(), sizingLeftAnkleBf)

		sizingRightAnkleBf := sizingMotion.BoneFrames.Get(rightAnkleBone.Name()).Get(frame)
		sizingRightAnkleBf.Rotation = rightAnkleRotations[i]
		sizingMotion.InsertRegisteredBoneFrame(rightAnkleBone.Name(), sizingRightAnkleBf)
	}

	if mlog.IsVerbose() {
		outputPath := mutils.CreateOutputPath(sizingSet.OriginalVmdPath, "下半身補正05_足首")
		repository.NewVmdRepository().Save(outputPath, sizingMotion, true)
		mlog.V("下半身補正05_足首: %s", outputPath)
	}

	// mlog.I(mi18n.T("下半身補正09", map[string]interface{}{"No": sizingSet.Index + 1}))

	// sizingResultDeltas := make([]*delta.VmdDeltas, len(frames))

	// // 元モデルのデフォーム(IK ON)
	// miter.IterParallelByList(frames, 500, func(data, index int) {
	// 	frame := float32(data)
	// 	vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
	// 	vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
	// 	vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, true, frame, leg_all_bone_names, false)
	// 	sizingResultDeltas[index] = vmdDeltas
	// })

	// mlog.I(mi18n.T("下半身補正10", map[string]interface{}{"No": sizingSet.Index + 1}))

	// // サイジング先にFKを焼き込み
	// for _, vmdDeltas := range sizingResultDeltas {
	// 	for _, boneDelta := range vmdDeltas.Bones.Data {
	// 		if boneDelta == nil || !boneDelta.Bone.IsLegFK() {
	// 			continue
	// 		}

	// 		sizingBf := sizingMotion.BoneFrames.Get(boneDelta.Bone.Name()).Get(boneDelta.Frame)

	// 		// 最終的な足FKを焼き込み
	// 		sizingBf.Rotation = boneDelta.FilledFrameRotation()
	// 		sizingMotion.InsertRegisteredBoneFrame(boneDelta.Bone.Name(), sizingBf)
	// 	}
	// }

	sizingSet.CompletedSizingLower = true
}
