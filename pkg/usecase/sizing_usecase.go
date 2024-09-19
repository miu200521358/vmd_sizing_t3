package usecase

import (
	"slices"
	"sync"

	"github.com/miu200521358/mlib_go/pkg/domain/delta"
	"github.com/miu200521358/mlib_go/pkg/domain/miter"
	"github.com/miu200521358/mlib_go/pkg/domain/mmath"
	"github.com/miu200521358/mlib_go/pkg/domain/pmx"
	"github.com/miu200521358/mlib_go/pkg/domain/vmd"
	"github.com/miu200521358/mlib_go/pkg/infrastructure/deform"
	"github.com/miu200521358/mlib_go/pkg/mutils/mlog"
	"github.com/miu200521358/vmd_sizing_t3/pkg/model"
)

var leg_direction_bone_names = [][]string{
	{pmx.LEG.Left(), pmx.KNEE.Left(), pmx.HEEL.Left(), pmx.ANKLE.Left(), pmx.TOE.Left(), pmx.TOE_P.Left(),
		pmx.TOE_C.Left(), pmx.LEG_D.Left(), pmx.KNEE_D.Left(), pmx.HEEL_D.Left(), pmx.ANKLE_D.Left(),
		pmx.TOE_D.Left(), pmx.TOE_P_D.Left(), pmx.TOE_C_D.Left(), pmx.TOE_EX.Left(),
		pmx.LEG_IK_PARENT.Left(), pmx.LEG_IK.Left(), pmx.TOE_IK.Left()},
	{pmx.LEG.Right(), pmx.KNEE.Right(), pmx.HEEL.Right(), pmx.ANKLE.Right(), pmx.TOE.Right(), pmx.TOE_P.Right(),
		pmx.TOE_C.Right(), pmx.LEG_D.Right(), pmx.KNEE_D.Right(), pmx.HEEL_D.Right(), pmx.ANKLE_D.Right(),
		pmx.TOE_D.Right(), pmx.TOE_P_D.Right(), pmx.TOE_C_D.Right(), pmx.TOE_EX.Right(),
		pmx.LEG_IK_PARENT.Right(), pmx.LEG_IK.Right(), pmx.TOE_IK.Right()},
}
var leg_all_bone_names = append(leg_direction_bone_names[0], leg_direction_bone_names[1]...)
var move_bone_names = []string{pmx.ROOT.String(), pmx.CENTER.String(), pmx.GROOVE.String(),
	pmx.LEG_IK_PARENT.Left(), pmx.LEG_IK_PARENT.Right(), pmx.LEG_IK.Left(), pmx.LEG_IK.Right(),
	pmx.TOE_IK.Right(), pmx.TOE_IK.Left()}
var leg_ik_bone_names = []string{pmx.LEG_IK.Left(), pmx.LEG_IK.Right()}

var toe_ik_bone_names = []string{pmx.TOE_IK.Left(), pmx.TOE_IK.Right()}
var leg_fk_bone_names = []string{
	pmx.LEG.Left(), pmx.KNEE.Left(), pmx.HEEL.Left(), pmx.ANKLE.Left(), pmx.TOE.Left(), pmx.TOE_P.Left(),
	pmx.TOE_C.Left(), pmx.LEG_D.Left(), pmx.KNEE_D.Left(), pmx.HEEL_D.Left(), pmx.ANKLE_D.Left(),
	pmx.TOE_D.Left(), pmx.TOE_P_D.Left(), pmx.TOE_C_D.Left(), pmx.TOE_EX.Left(),
	pmx.LEG.Right(), pmx.KNEE.Right(), pmx.HEEL.Right(), pmx.ANKLE.Right(), pmx.TOE.Right(), pmx.TOE_P.Right(),
	pmx.TOE_C.Right(), pmx.LEG_D.Right(), pmx.KNEE_D.Right(), pmx.HEEL_D.Right(), pmx.ANKLE_D.Right(),
	pmx.TOE_D.Right(), pmx.TOE_P_D.Right(), pmx.TOE_C_D.Right(), pmx.TOE_EX.Right(),
}
var knee_bone_names = []string{pmx.KNEE.Left(), pmx.KNEE.Right()}

// 足スタンス補正
func SizingLegStance(sizingSet *model.SizingSet) {
	if !sizingSet.IsSizingLegStance || (sizingSet.IsSizingLegStance && sizingSet.CompletedSizingLegStance) {
		return
	}

	// 足補正
	originalModel := sizingSet.OriginalPmx
	originalMotion := sizingSet.OriginalVmd
	sizingModel := sizingSet.SizingPmx
	sizingMotion := sizingSet.OutputVmd

	leftFrames := originalMotion.BoneFrames.RegisteredFrames(leg_direction_bone_names[0])
	rightFrames := originalMotion.BoneFrames.RegisteredFrames(leg_direction_bone_names[1])
	frames := append(leftFrames, rightFrames...)
	mmath.SortInts(frames)
	frames = slices.Compact(frames)
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
			if boneDelta == nil || !boneDelta.Bone.IsLegFK() ||
				!((boneDelta.Bone.Direction() == "左" && mmath.Contains(leftFrames, int(boneDelta.Frame))) ||
					(boneDelta.Bone.Direction() == "右" && mmath.Contains(rightFrames, int(boneDelta.Frame)))) {
				continue
			}

			sizingBf := sizingMotion.BoneFrames.Get(boneDelta.Bone.Name()).Get(boneDelta.Frame)

			// 最終的な足FKを焼き込み
			sizingBf.Rotation = boneDelta.FilledFrameRotation()
			sizingMotion.InsertRegisteredBoneFrame(boneDelta.Bone.Name(), sizingBf)
		}
	}

	sizingAllDeltas := make([]*delta.VmdDeltas, len(frames))

	// サイジング先モデルのデフォーム(IK OFF)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, false, frame, leg_all_bone_names, false)
		sizingAllDeltas[index] = vmdDeltas
	})

	// サイジング先にIK結果を焼き込み
	for i, vmdDeltas := range sizingAllDeltas {
		for _, boneDelta := range vmdDeltas.Bones.Data {
			if boneDelta == nil || !slices.Contains(leg_ik_bone_names, boneDelta.Bone.Name()) ||
				!((boneDelta.Bone.Direction() == "左" && mmath.Contains(leftFrames, int(boneDelta.Frame))) ||
					(boneDelta.Bone.Direction() == "右" && mmath.Contains(rightFrames, int(boneDelta.Frame)))) {
				continue
			}
			direction := boneDelta.Bone.Direction()
			targetDelta := vmdDeltas.Bones.Get(boneDelta.Bone.Ik.BoneIndex)
			parentDelta := vmdDeltas.Bones.Get(boneDelta.Bone.ParentIndex)

			originalBf := originalMotion.BoneFrames.Get(boneDelta.Bone.Name()).Get(boneDelta.Frame)
			sizingBf := sizingMotion.BoneFrames.Get(boneDelta.Bone.Name()).Get(boneDelta.Frame)

			sizingBf.Position = targetDelta.FilledGlobalPosition().Subed(boneDelta.Bone.Position).Subed(
				parentDelta.FilledTotalPosition())

			if mmath.NearEquals(originalBf.Position.Y, 0, 1e-2) {
				// 足首のY座標が0の場合、0にする
				sizingBf.Position.Y = 0
			}

			sizingBf.Rotation = boneDelta.FilledFrameRotation()
			sizingMotion.InsertRegisteredBoneFrame(boneDelta.Bone.Name(), sizingBf)

			// 足首の回転(足底の傾き)
			originalHeelDelta := originalAllDeltas[i].Bones.GetByName(
				pmx.HEEL_D.StringFromDirection(direction))
			originalToeDelta := originalAllDeltas[i].Bones.GetByName(
				pmx.TOE_D.StringFromDirection(direction))
			originalSoleDirection := originalToeDelta.FilledGlobalPosition().Subed(
				originalHeelDelta.FilledGlobalPosition()).Normalized()

			// 足首の回転(足底横の傾き)
			originalToePDelta := originalAllDeltas[i].Bones.GetByName(
				pmx.TOE_P_D.StringFromDirection(direction))
			originalToeCDelta := originalAllDeltas[i].Bones.GetByName(
				pmx.TOE_C_D.StringFromDirection(direction))
			originalSoleSideDirection := originalToeCDelta.FilledGlobalPosition().Subed(
				originalToePDelta.FilledGlobalPosition()).Normalized()

			originalSoleQuat := mmath.NewMQuaternionFromDirection(originalSoleDirection, originalSoleSideDirection)

			sizingHeelDelta := sizingAllDeltas[i].Bones.GetByName(
				pmx.HEEL_D.StringFromDirection(direction))
			sizingToeDelta := sizingAllDeltas[i].Bones.GetByName(
				pmx.TOE_D.StringFromDirection(direction))
			sizingSoleDirection := sizingToeDelta.FilledGlobalPosition().Subed(
				sizingHeelDelta.FilledGlobalPosition()).Normalized()

			sizingToePDelta := sizingAllDeltas[i].Bones.GetByName(
				pmx.TOE_P_D.StringFromDirection(direction))
			sizingToeCDelta := sizingAllDeltas[i].Bones.GetByName(
				pmx.TOE_C_D.StringFromDirection(direction))
			sizingSoleSideDirection := sizingToeCDelta.FilledGlobalPosition().Subed(
				sizingToePDelta.FilledGlobalPosition()).Normalized()

			sizingSoleQuat := mmath.NewMQuaternionFromDirection(sizingSoleDirection, sizingSoleSideDirection)

			soleOffsetQuat := sizingSoleQuat.Muled(originalSoleQuat.Inverted())

			ankleBf := sizingMotion.BoneFrames.Get(targetDelta.Bone.Name()).Get(boneDelta.Frame)
			ankleBf.Rotation.Mul(soleOffsetQuat)

			sizingMotion.InsertRegisteredBoneFrame(targetDelta.Bone.Name(), ankleBf)
		}
	}

	for _, boneName := range leg_ik_bone_names {
		for m, frame := range originalMotion.BoneFrames.Get(boneName).Indexes.List() {
			if m == 0 {
				continue
			}
			prevFrame := originalMotion.BoneFrames.Get(boneName).Indexes.Prev(frame)

			originalPrevBf := originalMotion.BoneFrames.Get(boneName).Get(prevFrame)
			originalNowBf := originalMotion.BoneFrames.Get(boneName).Get(frame)

			if originalPrevBf == nil || originalNowBf == nil ||
				originalPrevBf.Position == nil || originalNowBf.Position == nil ||
				!(mmath.NearEquals(originalPrevBf.Position.X, originalNowBf.Position.X, 1e-2) ||
					mmath.NearEquals(originalPrevBf.Position.Y, originalNowBf.Position.Y, 1e-2) ||
					mmath.NearEquals(originalPrevBf.Position.Z, originalNowBf.Position.Z, 1e-2)) {
				continue
			}

			var prevBf, nowBf *vmd.BoneFrame
			for f := prevFrame; f <= frame; f++ {
				if f == prevFrame {
					prevBf = sizingMotion.BoneFrames.Get(boneName).Get(f)
					continue
				}
				if !sizingMotion.BoneFrames.Get(boneName).Contains(f) {
					continue
				}
				nowBf = sizingMotion.BoneFrames.Get(boneName).Get(f)

				// 元で前後のキーフレームが同じ座標の場合、座標を引き継ぐ
				if mmath.NearEquals(originalPrevBf.Position.X, originalNowBf.Position.X, 1e-2) {
					nowBf.Position.X = prevBf.Position.X
				}

				if mmath.NearEquals(originalPrevBf.Position.Y, originalNowBf.Position.Y, 1e-2) {
					nowBf.Position.Y = prevBf.Position.Y
				}

				if mmath.NearEquals(originalPrevBf.Position.Z, originalNowBf.Position.Z, 1e-2) {
					nowBf.Position.Z = prevBf.Position.Z
				}

				sizingMotion.BoneFrames.Get(boneName).Update(nowBf)
				prevBf = nowBf
			}
		}
	}

	for _, boneName := range toe_ik_bone_names {
		// つま先IKを削除
		sizingMotion.BoneFrames.Delete(boneName)
	}
}

// // 足スタンス補正
// func SizingLegStance2(sizingSet *model.SizingSet) {
// 	if !sizingSet.IsSizingLegStance || (sizingSet.IsSizingLegStance && sizingSet.CompletedSizingLegStance) {
// 		return
// 	}

// 	// 足補正
// 	originalModel := sizingSet.OriginalPmx
// 	originalMotion := sizingSet.OriginalVmd
// 	sizingModel := sizingSet.SizingPmx
// 	sizingMotion := sizingSet.OutputVmd

// 	frames := originalMotion.BoneFrames.RegisteredFrames(leg_all_bone_names)

// 	originalAllDeltas := make([]*delta.VmdDeltas, len(frames))

// 	// 元モデルのデフォーム(IK ON)
// 	miter.IterParallelByList(frames, 100, func(data, index int) {
// 		frame := float32(data)
// 		vmdDeltas := delta.NewVmdDeltas(frame, originalModel.Bones, originalModel.Hash(), originalMotion.Hash())
// 		vmdDeltas.Morphs = deform.DeformMorph(originalModel, originalMotion.MorphFrames, frame, nil)
// 		vmdDeltas = deform.DeformBoneByPhysicsFlag(originalModel, originalMotion, vmdDeltas, true, frame, leg_fk_bone_names, false)
// 		originalAllDeltas[index] = vmdDeltas
// 	})

// 	// サイジング先にFKを焼き込み
// 	for _, vmdDeltas := range originalAllDeltas {
// 		for _, boneDelta := range vmdDeltas.Bones.Data {
// 			if boneDelta == nil || !sizingMotion.BoneFrames.Contains(boneDelta.Bone.Name()) ||
// 				!slices.Contains(leg_fk_bone_names, boneDelta.Bone.Name()) {
// 				continue
// 			}

// 			originalBf := originalMotion.BoneFrames.Get(boneDelta.Bone.Name()).Get(boneDelta.Frame)

// 			// 最終的な足FKを焼き込み
// 			bf := vmd.NewBoneFrame(boneDelta.Frame)
// 			bf.Rotation = boneDelta.FilledFrameRotation()

// 			if originalBf.Position != nil {
// 				bf.Position = originalBf.Position.Copy()
// 			}

// 			if originalBf.Curves != nil {
// 				bf.Curves = originalBf.Curves.Copy()
// 			}

// 			sizingMotion.InsertRegisteredBoneFrame(boneDelta.Bone.Name(), bf)
// 		}
// 	}

// 	sizingAllDeltas := make([]*delta.VmdDeltas, len(frames))

// 	// サイジングモデルのデフォーム(IK OFF)
// 	miter.IterParallelByList(frames, 100, func(data, index int) {
// 		frame := float32(data)
// 		vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
// 		vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
// 		vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, false, frame, leg_all_bone_names, false)
// 		sizingAllDeltas[index] = vmdDeltas
// 	})

// 	// サイジング先にIK結果を焼き込み
// 	for i, vmdDeltas := range sizingAllDeltas {
// 		for _, boneDelta := range vmdDeltas.Bones.Data {
// 			if boneDelta == nil || !sizingMotion.BoneFrames.Contains(boneDelta.Bone.Name()) {
// 				continue
// 			}

// 			originalBf := originalMotion.BoneFrames.Get(boneDelta.Bone.Name()).Get(boneDelta.Frame)

// 			if slices.Contains(leg_ik_bone_names, boneDelta.Bone.Name()) {
// 				direction := string([]rune(boneDelta.Bone.Name())[0])
// 				targetDelta := vmdDeltas.Bones.Get(boneDelta.Bone.Ik.BoneIndex)
// 				parentDelta := vmdDeltas.Bones.Get(boneDelta.Bone.ParentIndex)

// 				// 最終的な足IKを焼き込み
// 				bf := vmd.NewBoneFrame(boneDelta.Frame)
// 				// mlog.I("[%s:%.0f](%s): %v <- %v", boneDelta.Bone.Name(), boneDelta.Frame,
// 				// 	targetDelta.Bone.Name(), targetDelta.GlobalPosition, boneDelta.GlobalPosition)

// 				bf.Position = targetDelta.FilledGlobalPosition().Subed(boneDelta.Bone.Position).Subed(
// 					parentDelta.FilledTotalPosition())
// 				if mmath.NearEquals(originalBf.Position.Y, 0, 1e-2) {
// 					// 足首のY座標が0の場合、0にする
// 					bf.Position.Y = 0
// 				}

// 				// 足首の回転(足底の傾き)
// 				originalHeelDelta := originalAllDeltas[i].Bones.GetByName(pmx.HEEL_D.StringFromDirection(direction))
// 				originalToeDelta := originalAllDeltas[i].Bones.GetByName(pmx.TOE_D.StringFromDirection(direction))
// 				originalSoleDirection := originalToeDelta.FilledGlobalPosition().Subed(
// 					originalHeelDelta.FilledGlobalPosition()).Normalized()

// 				sizingHeelDelta := sizingAllDeltas[i].Bones.GetByName(pmx.HEEL_D.StringFromDirection(direction))
// 				sizingToeDelta := sizingAllDeltas[i].Bones.GetByName(pmx.TOE_D.StringFromDirection(direction))
// 				sizingSoleDirection := sizingToeDelta.FilledGlobalPosition().Subed(
// 					sizingHeelDelta.FilledGlobalPosition()).Normalized()
// 				soleOffsetQuat := mmath.NewMQuaternionRotate(sizingSoleDirection, originalSoleDirection)

// 				bf.Rotation = soleOffsetQuat.Muled(boneDelta.FilledFrameRotation())

// 				if originalBf.Curves != nil {
// 					bf.Curves = originalBf.Curves.Copy()
// 				}

// 				sizingMotion.InsertRegisteredBoneFrame(boneDelta.Bone.Name(), bf)
// 			}
// 		}
// 	}

// 	sizingSet.CompletedSizingLegStance = true
// }

func Sizing(sizingSet *model.SizingSet) {
	originalModel := sizingSet.OriginalPmx
	originalMotion := sizingSet.OriginalVmd
	sizingModel := sizingSet.SizingPmx
	sizingMotion := sizingSet.OutputVmd

	if originalModel == nil || sizingModel == nil || originalMotion == nil || sizingMotion == nil {
		return
	}

	var scales *mmath.MVec3
	if sizingSet.IsSizingMove {
		// 足の長さ比率(XZ)
		legLengthRatio := (sizingModel.Bones.GetByName(pmx.LEG.Left()).Position.Distance(
			sizingModel.Bones.GetByName(pmx.KNEE.Left()).Position) +
			sizingModel.Bones.GetByName(pmx.KNEE.Left()).Position.Distance(
				sizingModel.Bones.GetByName(pmx.ANKLE.Left()).Position)) /
			(originalModel.Bones.GetByName(pmx.LEG.Left()).Position.Distance(
				originalModel.Bones.GetByName(pmx.KNEE.Left()).Position) +
				originalModel.Bones.GetByName(pmx.KNEE.Left()).Position.Distance(
					originalModel.Bones.GetByName(pmx.ANKLE.Left()).Position))
		// 足の長さ比率(Y)
		legHeightRatio := sizingModel.Bones.GetByName(pmx.LEG.Left()).Position.Distance(
			sizingModel.Bones.GetByName(pmx.LEG_IK.Left()).Position) /
			originalModel.Bones.GetByName(pmx.LEG.Left()).Position.Distance(
				originalModel.Bones.GetByName(pmx.LEG_IK.Left()).Position)

		scales = &mmath.MVec3{X: legLengthRatio, Y: legHeightRatio, Z: legLengthRatio}

		mlog.I("legHeightRatio: %.5f", legLengthRatio)
	}

	stanceQuat := make(map[int]*mmath.MQuaternion)

	// 腕スタンス補正
	if sizingSet.IsSizingArmStance {
		for _, boneNames := range [][]string{{pmx.ARM.Left(), pmx.ELBOW.Left(), pmx.WRIST.Left()},
			{pmx.ARM.Right(), pmx.ELBOW.Right(), pmx.WRIST.Right()}} {
			armBoneName := boneNames[0]
			elbowBoneName := boneNames[1]
			wristBoneName := boneNames[2]

			// 腕
			armBone := sizingModel.Bones.GetByName(armBoneName)
			armOriginalBone := originalModel.Bones.GetByName(armBoneName)
			if armBone != nil && armOriginalBone != nil {
				armBoneDirection := armBone.Extend.ChildRelativePosition.Normalized()
				armOriginalBoneDirection := armOriginalBone.Extend.ChildRelativePosition.Normalized()
				stanceQuat[armBone.Index()] = mmath.NewMQuaternionRotate(armBoneDirection, armOriginalBoneDirection)
			}

			// ひじ
			elbowBone := sizingModel.Bones.GetByName(elbowBoneName)
			elbowOriginalBone := originalModel.Bones.GetByName(elbowBoneName)
			if elbowBone != nil && elbowOriginalBone != nil {
				elbowBoneDirection := elbowBone.Extend.ChildRelativePosition.Normalized()
				elbowOriginalBoneDirection := elbowOriginalBone.Extend.ChildRelativePosition.Normalized()
				elbowOffsetQuat := mmath.NewMQuaternionRotate(elbowBoneDirection, elbowOriginalBoneDirection)
				stanceQuat[elbowBone.Index()] = elbowOffsetQuat.Muled(stanceQuat[armBone.Index()].Inverted())
			}

			// 手首
			wristBone := sizingModel.Bones.GetByName(wristBoneName)
			wristOriginalBone := originalModel.Bones.GetByName(wristBoneName)
			if wristBone != nil && wristOriginalBone != nil {
				wristBoneDirection := wristBone.Extend.ChildRelativePosition.Normalized()
				wristOriginalBoneDirection := wristOriginalBone.Extend.ChildRelativePosition.Normalized()
				wristOffsetQuat := mmath.NewMQuaternionRotate(wristBoneDirection, wristOriginalBoneDirection)
				stanceQuat[wristBone.Index()] = wristOffsetQuat.Muled(stanceQuat[elbowBone.Index()].Inverted())
			}
		}
	}

	var wg sync.WaitGroup
	for _, boneName := range originalMotion.BoneFrames.Names() {
		wg.Add(1)

		go func(originalBfs, sizingBfs *vmd.BoneNameFrames) {
			defer wg.Done()
			for _, frame := range originalBfs.Indexes.List() {
				originalBf := originalMotion.BoneFrames.Get(boneName).Get(frame)
				sizingBf := sizingBfs.Get(frame)
				if originalBf == nil || sizingBf == nil {
					continue
				}

				// 移動補正
				if scales != nil && slices.Contains(move_bone_names, originalBfs.Name) {
					sizingBf.Position = originalBf.Position.Muled(scales)
					sizingBfs.Update(sizingBf)
				}

				// 回転補正
				bone := sizingModel.Bones.GetByName(boneName)
				if bone != nil {
					if _, ok := stanceQuat[bone.Index()]; ok {
						sizingBf.Rotation = originalBf.Rotation.Muled(stanceQuat[bone.Index()])
						sizingBfs.Update(sizingBf)
					}
				}
			}
		}(originalMotion.BoneFrames.Get(boneName), sizingMotion.BoneFrames.Get(boneName))
	}

	wg.Wait()

	sizingSet.CompletedSizingMove = true
	sizingSet.CompletedSizingArmStance = true
}
