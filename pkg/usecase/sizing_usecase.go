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
	"github.com/miu200521358/vmd_sizing_t3/pkg/model"
)

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
var leg_all_bone_names = append(leg_fk_bone_names,
	pmx.LEG_IK.Left(), pmx.LEG_IK.Right(), pmx.TOE_IK.Left(), pmx.TOE_IK.Right())
var move_bone_names = []string{pmx.ROOT.String(), pmx.CENTER.String(), pmx.GROOVE.String(),
	pmx.LEG_IK_PARENT.Left(), pmx.LEG_IK_PARENT.Right(), pmx.LEG_IK.Left(),
	pmx.LEG_IK.Right(), pmx.TOE_IK.Right(), pmx.TOE_IK.Left()}

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

	frames := originalMotion.BoneFrames.RegisteredFrames(leg_all_bone_names)

	originalAllDeltas := make([]*delta.VmdDeltas, len(frames))

	// 元モデルのデフォーム(IK ON)
	miter.IterParallelByList(frames, 100, func(data, index int) {
		frame := float32(data)
		vmdDeltas := delta.NewVmdDeltas(frame, originalModel.Bones, originalModel.Hash(), originalMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(originalModel, originalMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(originalModel, originalMotion, vmdDeltas, true, frame, leg_fk_bone_names, false)
		originalAllDeltas[index] = vmdDeltas
	})

	// サイジング先にFKを焼き込み
	for _, vmdDeltas := range originalAllDeltas {
		for _, boneDelta := range vmdDeltas.Bones.Data {
			if boneDelta == nil || !sizingMotion.BoneFrames.Contains(boneDelta.Bone.Name()) ||
				!slices.Contains(leg_fk_bone_names, boneDelta.Bone.Name()) {
				continue
			}

			originalBf := originalMotion.BoneFrames.Get(boneDelta.Bone.Name()).Get(boneDelta.Frame)

			// 最終的な足FKを焼き込み
			bf := vmd.NewBoneFrame(boneDelta.Frame)
			bf.Rotation = boneDelta.FilledFrameRotation()

			if originalBf.Position != nil {
				bf.Position = originalBf.Position.Copy()
			}

			if originalBf.Curves != nil {
				bf.Curves = originalBf.Curves.Copy()
			}

			if !slices.Contains(knee_bone_names, boneDelta.Bone.Name()) {
				// ひざ以外は登録対象
				bf.Registered = true
			}

			sizingMotion.InsertBoneFrame(boneDelta.Bone.Name(), bf)
		}
	}

	sizingAllDeltas := make([]*delta.VmdDeltas, len(frames))

	// サイジングモデルのデフォーム(IK OFF)
	miter.IterParallelByList(frames, 100, func(data, index int) {
		frame := float32(data)
		vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, false, frame, leg_all_bone_names, false)
		sizingAllDeltas[index] = vmdDeltas
	})

	// サイジング先にIK結果を焼き込み
	for i, vmdDeltas := range sizingAllDeltas {
		for _, boneDelta := range vmdDeltas.Bones.Data {
			if boneDelta == nil || !sizingMotion.BoneFrames.Contains(boneDelta.Bone.Name()) {
				continue
			}

			originalBf := originalMotion.BoneFrames.Get(boneDelta.Bone.Name()).Get(boneDelta.Frame)

			if slices.Contains(leg_ik_bone_names, boneDelta.Bone.Name()) {
				direction := string([]rune(boneDelta.Bone.Name())[0])
				targetDelta := vmdDeltas.Bones.Get(boneDelta.Bone.Ik.BoneIndex)

				// 最終的な足IKを焼き込み
				bf := vmd.NewBoneFrame(boneDelta.Frame)
				// mlog.I("[%s:%.0f](%s): %v <- %v", boneDelta.Bone.Name(), boneDelta.Frame,
				// 	targetDelta.Bone.Name(), targetDelta.GlobalPosition, boneDelta.GlobalPosition)

				bf.Position = targetDelta.FilledGlobalPosition().Subed(boneDelta.Bone.Position)
				if mmath.NearEquals(originalBf.Position.Y, 0, 1e-2) {
					// 足首のY座標が0の場合、0にする
					bf.Position.Y = 0
				}

				// 足首の回転(足底の傾き)
				originalHeelDelta := originalAllDeltas[i].Bones.GetByName(pmx.HEEL_D.StringFromDirection(direction))
				originalToeDelta := originalAllDeltas[i].Bones.GetByName(pmx.TOE_D.StringFromDirection(direction))
				originalSoleDirection := originalToeDelta.FilledGlobalPosition().Subed(
					originalHeelDelta.FilledGlobalPosition()).Normalized()

				sizingHeelDelta := sizingAllDeltas[i].Bones.GetByName(pmx.HEEL_D.StringFromDirection(direction))
				sizingToeDelta := sizingAllDeltas[i].Bones.GetByName(pmx.TOE_D.StringFromDirection(direction))
				sizingSoleDirection := sizingToeDelta.FilledGlobalPosition().Subed(
					sizingHeelDelta.FilledGlobalPosition()).Normalized()
				soleOffsetQuat := mmath.NewMQuaternionRotate(sizingSoleDirection, originalSoleDirection)

				bf.Rotation = boneDelta.FilledFrameRotation().Muled(soleOffsetQuat)

				if originalBf.Curves != nil {
					bf.Curves = originalBf.Curves.Copy()
				}

				sizingMotion.InsertRegisteredBoneFrame(boneDelta.Bone.Name(), bf)
			}
		}
	}

	for _, boneName := range leg_fk_bone_names {
		for m, frame := range sizingMotion.BoneFrames.Get(boneName).Indexes.List() {
			if m == 0 {
				continue
			}
			prevFrame := sizingMotion.BoneFrames.Get(boneName).Indexes.Prev(frame)

			originalPrevDelta := originalAllDeltas[m].Bones.GetByName(boneName)
			originalDelta := originalAllDeltas[m].Bones.GetByName(boneName)

			prevBf := sizingMotion.BoneFrames.Get(boneName).Get(prevFrame)
			bf := sizingMotion.BoneFrames.Get(boneName).Get(frame)

			if originalPrevDelta == nil || originalDelta == nil || prevBf == nil || bf == nil ||
				prevBf.Position == nil || bf.Position == nil {
				continue
			}

			// 元で前後のキーフレームが同じ座標の場合、座標を引き継ぐ
			if mmath.NearEquals(originalPrevDelta.FilledGlobalPosition().X,
				originalDelta.FilledGlobalPosition().X, 1e-2) {
				bf.Position.X = prevBf.Position.X
			}
			if mmath.NearEquals(originalPrevDelta.FilledGlobalPosition().Y,
				originalDelta.FilledGlobalPosition().Y, 1e-2) {
				bf.Position.Y = prevBf.Position.Y
			}
			if mmath.NearEquals(originalPrevDelta.FilledGlobalPosition().Z,
				originalDelta.FilledGlobalPosition().Z, 1e-2) {
				bf.Position.Z = prevBf.Position.Z
			}

			sizingMotion.BoneFrames.Get(boneName).Update(bf)
		}
	}

	for _, boneName := range toe_ik_bone_names {
		// つま先IKを削除
		sizingMotion.BoneFrames.Delete(boneName)
	}

	sizingSet.CompletedSizingLegStance = true
}

func Sizing(sizingSet *model.SizingSet) {
	originalModel := sizingSet.OriginalPmx
	originalMotion := sizingSet.OriginalVmd
	sizingModel := sizingSet.SizingPmx
	sizingMotion := sizingSet.OutputVmd

	if originalModel == nil || sizingModel == nil || originalMotion == nil || sizingMotion == nil {
		return
	}

	// 足の長さ比率
	legHeightRatio := sizingModel.Bones.GetByName(pmx.LEG_ROOT.Left()).Position.Length() /
		originalModel.Bones.GetByName(pmx.LEG_ROOT.Left()).Position.Length()
	// 腰の広さ比率
	hipWidthRatio := sizingModel.Bones.GetByName(pmx.LEG.Left()).Position.Distance(
		sizingModel.Bones.GetByName(pmx.LEG.Right()).Position) /
		originalModel.Bones.GetByName(pmx.LEG.Left()).Position.Distance(
			originalModel.Bones.GetByName(pmx.LEG.Right()).Position)

	var scales *mmath.MVec3
	if !sizingSet.IsSizingMove || (sizingSet.IsSizingMove && !sizingSet.CompletedSizingMove) {
		if sizingSet.IsSizingMove {
			scales = &mmath.MVec3{X: hipWidthRatio, Y: legHeightRatio, Z: hipWidthRatio}
		} else {
			scales = mmath.MVec3One
		}
	}

	stanceQuat := make(map[int]*mmath.MQuaternion)

	// 腕スタンス補正
	if !sizingSet.IsSizingArmStance || (sizingSet.IsSizingArmStance && !sizingSet.CompletedSizingArmStance) {
		for _, boneNames := range [][]string{{pmx.ARM.Left(), pmx.ELBOW.Left(), pmx.WRIST.Left()},
			{pmx.ARM.Right(), pmx.ELBOW.Right(), pmx.WRIST.Right()}} {
			armBoneName := boneNames[0]
			elbowBoneName := boneNames[1]
			wristBoneName := boneNames[2]

			// 腕
			armBone := sizingModel.Bones.GetByName(armBoneName)
			armOriginalBone := originalModel.Bones.GetByName(armBoneName)
			if sizingSet.IsSizingArmStance && armBone != nil && armOriginalBone != nil {
				armBoneDirection := armBone.Extend.ChildRelativePosition.Normalized()
				armOriginalBoneDirection := armOriginalBone.Extend.ChildRelativePosition.Normalized()
				stanceQuat[armBone.Index()] = mmath.NewMQuaternionRotate(armBoneDirection, armOriginalBoneDirection)
			} else {
				stanceQuat[armBone.Index()] = mmath.MQuaternionIdent
			}

			// ひじ
			elbowBone := sizingModel.Bones.GetByName(elbowBoneName)
			elbowOriginalBone := originalModel.Bones.GetByName(elbowBoneName)
			if sizingSet.IsSizingArmStance && elbowBone != nil && elbowOriginalBone != nil {
				elbowBoneDirection := elbowBone.Extend.ChildRelativePosition.Normalized()
				elbowOriginalBoneDirection := elbowOriginalBone.Extend.ChildRelativePosition.Normalized()
				elbowOffsetQuat := mmath.NewMQuaternionRotate(elbowBoneDirection, elbowOriginalBoneDirection)
				stanceQuat[elbowBone.Index()] = elbowOffsetQuat.Muled(stanceQuat[armBone.Index()].Inverted())
			} else {
				stanceQuat[elbowBone.Index()] = mmath.MQuaternionIdent
			}

			// 手首
			wristBone := sizingModel.Bones.GetByName(wristBoneName)
			wristOriginalBone := originalModel.Bones.GetByName(wristBoneName)
			if sizingSet.IsSizingArmStance && wristBone != nil || wristOriginalBone != nil {
				wristBoneDirection := wristBone.Extend.ChildRelativePosition.Normalized()
				wristOriginalBoneDirection := wristOriginalBone.Extend.ChildRelativePosition.Normalized()
				wristOffsetQuat := mmath.NewMQuaternionRotate(wristBoneDirection, wristOriginalBoneDirection)
				stanceQuat[wristBone.Index()] = wristOffsetQuat.Muled(stanceQuat[elbowBone.Index()].Inverted())
			} else {
				stanceQuat[wristBone.Index()] = mmath.MQuaternionIdent
			}
		}
	}

	var wg sync.WaitGroup
	for _, boneName := range originalMotion.BoneFrames.Names() {
		wg.Add(1)

		if !sizingSet.IsSizingLegStance && slices.Contains(leg_fk_bone_names, boneName) {
			// 足スタンス補正なしの場合、キーフレ置き直し
			sizingMotion.BoneFrames.Append(vmd.NewBoneNameFrames(boneName))
		}

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

				// 足スタンス補正なし
				if !sizingSet.IsSizingLegStance && slices.Contains(leg_fk_bone_names, boneName) {
					sizingBf.Position = originalBf.Position.Copy()
					sizingBf.Rotation = originalBf.Rotation.Copy()
					if originalBf.Curves != nil {
						sizingBf.Curves = originalBf.Curves.Copy()
					}
					sizingBfs.Append(sizingBf)
				}
			}
		}(originalMotion.BoneFrames.Get(boneName), sizingMotion.BoneFrames.Get(boneName))
	}

	wg.Wait()

	sizingSet.CompletedSizingMove = true
	sizingSet.CompletedSizingArmStance = true
}
