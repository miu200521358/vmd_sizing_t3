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
	"github.com/miu200521358/vmd_sizing_t3/pkg/domain"
)

func SizingLower(sizingSet *domain.SizingSet, frames []int, originalAllDeltas []*delta.VmdDeltas) {
	if !sizingSet.IsSizingLower || (sizingSet.IsSizingLower && sizingSet.CompletedSizingLower) {
		return
	}

	if !isValidSizingLower(sizingSet) {
		return
	}

	originalModel := sizingSet.OriginalPmx
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

	// 下半身から足中心のベクトル比率
	originalLowerVector := originalLegCenterBone.Position.Subed(originalLowerBone.Position)
	sizingLowerVector := legCenterBone.Position.Subed(lowerBone.Position)
	legCenterRatio := sizingLowerVector.Dived(originalLowerVector).Effective().One()

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

	sizingOffDeltas := make([]*delta.VmdDeltas, len(frames))
	sizingLegIkOnDeltas := make([]*delta.VmdDeltas, len(frames))
	centerPositions := make([]*mmath.MVec3, len(frames))
	groovePositions := make([]*mmath.MVec3, len(frames))
	lowerRotations := make([]*mmath.MQuaternion, len(frames))

	mlog.I(mi18n.T("下半身補正01", map[string]interface{}{"No": sizingSet.Index + 1}))

	// 先モデルのデフォーム(IK OFF)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, false, frame, leg_all_bone_names, false)
		sizingOffDeltas[index] = vmdDeltas

		// 足中心から見た下半身ボーンの相対位置を取得
		originalLegCenterDelta := originalAllDeltas[index].Bones.Get(originalLegCenterBone.Index())
		originalLowerDelta := originalAllDeltas[index].Bones.Get(originalLowerBone.Index())
		originalLegCenterLocalPosition := originalLowerDelta.FilledGlobalPosition().Subed(
			originalLegCenterDelta.FilledGlobalPosition())

		// サイジング先の足中心から、オリジナルの下半身位置を加算した時の結果
		legCenterDelta := vmdDeltas.Bones.Get(legCenterBone.Index())
		lowerDelta := vmdDeltas.Bones.Get(lowerBone.Index())
		sizingLowerGlobalPosition := lowerDelta.FilledGlobalPosition()
		sizingFixLowerGlobalPosition := legCenterDelta.FilledGlobalPosition().Added(
			originalLegCenterLocalPosition.Muled(legCenterRatio))

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
		lowerDiff := sizingFixLowerGlobalPosition.Subed(sizingLowerGlobalPosition)

		centerBf := sizingMotion.BoneFrames.Get(centerBone.Name()).Get(frame)
		centerPositions[index] = centerBf.Position.Added(
			&mmath.MVec3{X: lowerDiff.X, Y: 0, Z: lowerDiff.Z})

		grooveBf := sizingMotion.BoneFrames.Get(grooveBone.Name()).Get(frame)
		groovePositions[index] = grooveBf.Position.Added(
			&mmath.MVec3{X: 0, Y: lowerDiff.Y, Z: 0})

		{
			vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
			vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
			vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, true, frame, leg_all_bone_names, false)
			sizingLegIkOnDeltas[index] = vmdDeltas
		}
	})

	mlog.I(mi18n.T("下半身補正02", map[string]interface{}{"No": sizingSet.Index + 1}))

	for i, iFrame := range frames {
		frame := float32(iFrame)

		sizingCenterBf := sizingMotion.BoneFrames.Get(centerBone.Name()).Get(frame)
		sizingCenterBf.Position = centerPositions[i]
		sizingMotion.InsertRegisteredBoneFrame(centerBone.Name(), sizingCenterBf)

		sizingGrooveBf := sizingMotion.BoneFrames.Get(grooveBone.Name()).Get(frame)
		sizingGrooveBf.Position = groovePositions[i]
		sizingMotion.InsertRegisteredBoneFrame(grooveBone.Name(), sizingGrooveBf)

		sizingLowerBf := sizingMotion.BoneFrames.Get(lowerBone.Name()).Get(frame)
		sizingLowerBf.Rotation = lowerRotations[i]
		sizingMotion.InsertRegisteredBoneFrame(lowerBone.Name(), sizingLowerBf)
	}

	mlog.I(mi18n.T("下半身補正03", map[string]interface{}{"No": sizingSet.Index + 1}))

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

	mlog.I(mi18n.T("下半身補正04", map[string]interface{}{"No": sizingSet.Index + 1}))

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

	mlog.I(mi18n.T("下半身補正05", map[string]interface{}{"No": sizingSet.Index + 1}))

	leftKneeRotations := make([]*mmath.MQuaternion, len(frames))
	rightKneeRotations := make([]*mmath.MQuaternion, len(frames))

	// 先モデルの足角度追加補正(IK ON)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		// 足首を固定した場合のひざの回転
		{
			vmdDeltas := deform.DeformIk(sizingModel, sizingMotion, frame, leftAnkleIkBone,
				sizingLegIkOnDeltas[index].Bones.Get(leftAnkleBone.Index()).FilledGlobalPosition())
			leftKneeRotations[index] = vmdDeltas.Bones.Get(leftKneeBone.Index()).FilledFrameRotation()
		}
		{
			vmdDeltas := deform.DeformIk(sizingModel, sizingMotion, frame, rightAnkleIkBone,
				sizingLegIkOnDeltas[index].Bones.Get(rightAnkleBone.Index()).FilledGlobalPosition())
			rightKneeRotations[index] = vmdDeltas.Bones.Get(rightKneeBone.Index()).FilledFrameRotation()
		}
	})

	mlog.I(mi18n.T("下半身補正06", map[string]interface{}{"No": sizingSet.Index + 1}))

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

	mlog.I(mi18n.T("下半身補正07", map[string]interface{}{"No": sizingSet.Index + 1}))

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

	mlog.I(mi18n.T("下半身補正08", map[string]interface{}{"No": sizingSet.Index + 1}))

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
}
