package usecase

import (
	"fmt"
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

// FitBoneモーフ名
var arm_sizing_morph_name = fmt.Sprintf("%s_%s", pmx.MLIB_PREFIX, "ArmSizingBone")
var move_sizing_morph_name = fmt.Sprintf("%s_%s", pmx.MLIB_PREFIX, "MoveSizingBone")

func CreateSizingMorph(sizingSet *model.SizingSet) {
	sizingSet.SizingPmx.Morphs.RemoveByName(arm_sizing_morph_name)
	createArmStanceSizingMorph(sizingSet)

	sizingSet.SizingPmx.Morphs.RemoveByName(move_sizing_morph_name)
	createMoveSizingMorph(sizingSet)
}

func createMoveSizingMorph(sizingSet *model.SizingSet) {
	offsets := make([]pmx.IMorphOffset, 0)

	originalModel := sizingSet.OriginalPmx
	sizingModel := sizingSet.SizingPmx

	if originalModel != nil && sizingModel != nil && sizingSet.IsSizingMove {
		// 足の長さ比率
		legHeightRatio := sizingModel.Bones.GetByName(pmx.LEG_ROOT.Left()).Position.Length() /
			originalModel.Bones.GetByName(pmx.LEG_ROOT.Left()).Position.Length()
		// 腰の広さ比率
		hipWidthRatio := sizingModel.Bones.GetByName(pmx.LEG.Left()).Position.Distance(
			sizingModel.Bones.GetByName(pmx.LEG.Right()).Position) /
			originalModel.Bones.GetByName(pmx.LEG.Left()).Position.Distance(
				originalModel.Bones.GetByName(pmx.LEG.Right()).Position)

		// 移動補正
		{
			bone := sizingModel.Bones.GetByName(pmx.CENTER.String())
			offset := pmx.NewBoneMorphOffset(bone.Index())
			offset.CancelableScale = &mmath.MVec3{X: hipWidthRatio, Y: legHeightRatio, Z: hipWidthRatio}
			offsets = append(offsets, offset)
		}

		// // IK親補正
		// legIkParentHeightRatio := sizingModel.Bones.GetByName(pmx.LEG_IK_PARENT.Left()).Position.Length() /
		// 	originalModel.Bones.GetByName(pmx.LEG_IK_PARENT.Left()).Position.Length()
		// for _, boneName := range []string{pmx.LEG_IK_PARENT.Left(), pmx.LEG_IK_PARENT.Right()} {
		// 	bone := sizingModel.Bones.GetByName(boneName)
		// 	if bone == nil {
		// 		continue
		// 	}

		// 	offset := pmx.NewBoneMorphOffset(bone.Index())
		// 	offset.CancelableScale = &mmath.MVec3{X: legIkParentHeightRatio, Y: legIkParentHeightRatio, Z: legIkParentHeightRatio}
		// 	offsets = append(offsets, offset)
		// }

		// // IK補正
		// legIkHeightRatio := sizingModel.Bones.GetByName(pmx.LEG.Left()).Position.Distance(
		// 	sizingModel.Bones.GetByName(pmx.LEG_IK.Left()).Position) /
		// 	originalModel.Bones.GetByName(pmx.LEG.Left()).Position.Distance(
		// 		originalModel.Bones.GetByName(pmx.LEG_IK.Left()).Position)
		// legIkWidthRatio := (sizingModel.Bones.GetByName(pmx.LEG.Left()).Position.Distance(
		// 	sizingModel.Bones.GetByName(pmx.KNEE.Left()).Position) +
		// 	sizingModel.Bones.GetByName(pmx.KNEE.Left()).Position.Distance(
		// 		sizingModel.Bones.GetByName(pmx.ANKLE.Left()).Position)) /
		// 	(originalModel.Bones.GetByName(pmx.LEG.Left()).Position.Distance(
		// 		originalModel.Bones.GetByName(pmx.KNEE.Left()).Position) +
		// 		originalModel.Bones.GetByName(pmx.KNEE.Left()).Position.Distance(
		// 			originalModel.Bones.GetByName(pmx.ANKLE.Left()).Position))
		// for _, boneName := range []string{pmx.LEG_IK.Left(), pmx.LEG_IK.Right()} {
		// 	bone := sizingModel.Bones.GetByName(boneName)
		// 	if bone == nil {
		// 		continue
		// 	}

		// 	offset := pmx.NewBoneMorphOffset(bone.Index())
		// 	offset.CancelableScale = &mmath.MVec3{X: legIkWidthRatio, Y: legIkHeightRatio, Z: legIkWidthRatio}
		// 	offsets = append(offsets, offset)
		// }
	}

	morph := pmx.NewMorph()
	morph.SetIndex(sizingModel.Morphs.Len())
	morph.SetName(move_sizing_morph_name)
	morph.Offsets = offsets
	morph.MorphType = pmx.MORPH_TYPE_BONE
	morph.Panel = pmx.MORPH_PANEL_OTHER_LOWER_RIGHT
	morph.IsSystem = true
	sizingModel.Morphs.Append(morph)
}

func createArmStanceSizingMorph(sizingSet *model.SizingSet) {
	offsets := make([]pmx.IMorphOffset, 0)

	originalModel := sizingSet.OriginalPmx
	sizingModel := sizingSet.SizingPmx

	if originalModel != nil && sizingModel != nil && sizingSet.IsSizingArmStance {
		// 腕スタンス補正
		for _, boneNames := range [][]string{{pmx.ARM.Left(), pmx.ELBOW.Left(), pmx.WRIST.Left()},
			{pmx.ARM.Right(), pmx.ELBOW.Right(), pmx.WRIST.Right()}} {
			armBoneName := boneNames[0]
			elbowBoneName := boneNames[1]
			wristBoneName := boneNames[2]

			// 腕
			armBone := sizingModel.Bones.GetByName(armBoneName)
			armOriginalBone := originalModel.Bones.GetByName(armBoneName)
			if armBone == nil || armOriginalBone == nil {
				continue
			}

			armBoneDirection := armBone.Extend.ChildRelativePosition.Normalized()
			armOriginalBoneDirection := armOriginalBone.Extend.ChildRelativePosition.Normalized()
			armOffsetQuat := mmath.NewMQuaternionRotate(armBoneDirection, armOriginalBoneDirection)

			armOffset := pmx.NewBoneMorphOffset(armBone.Index())
			armOffset.Rotation = armOffsetQuat
			offsets = append(offsets, armOffset)

			// ひじ
			elbowBone := sizingModel.Bones.GetByName(elbowBoneName)
			elbowOriginalBone := originalModel.Bones.GetByName(elbowBoneName)
			if elbowBone == nil || elbowOriginalBone == nil {
				continue
			}

			elbowBoneDirection := elbowBone.Extend.ChildRelativePosition.Normalized()
			elbowOriginalBoneDirection := elbowOriginalBone.Extend.ChildRelativePosition.Normalized()
			elbowOffsetQuat := mmath.NewMQuaternionRotate(elbowBoneDirection, elbowOriginalBoneDirection)

			elbowOffset := pmx.NewBoneMorphOffset(elbowBone.Index())
			elbowOffset.Rotation = elbowOffsetQuat.Muled(armOffsetQuat.Inverted())
			offsets = append(offsets, elbowOffset)

			// 手首
			wristBone := sizingModel.Bones.GetByName(wristBoneName)
			wristOriginalBone := originalModel.Bones.GetByName(wristBoneName)
			if wristBone == nil || wristOriginalBone == nil {
				continue
			}

			wristBoneDirection := wristBone.Extend.ChildRelativePosition.Normalized()
			wristOriginalBoneDirection := wristOriginalBone.Extend.ChildRelativePosition.Normalized()
			wristOffsetQuat := mmath.NewMQuaternionRotate(wristBoneDirection, wristOriginalBoneDirection)

			wristOffset := pmx.NewBoneMorphOffset(wristBone.Index())
			wristOffset.Rotation = wristOffsetQuat.Muled(elbowOffsetQuat.Inverted())
			offsets = append(offsets, wristOffset)
		}
	}

	morph := pmx.NewMorph()
	morph.SetIndex(sizingModel.Morphs.Len())
	morph.SetName(arm_sizing_morph_name)
	morph.Offsets = offsets
	morph.MorphType = pmx.MORPH_TYPE_BONE
	morph.Panel = pmx.MORPH_PANEL_OTHER_LOWER_RIGHT
	morph.IsSystem = true
	sizingModel.Morphs.Append(morph)
}

func AddSizingMorph(motion *vmd.VmdMotion) *vmd.VmdMotion {
	if motion.MorphFrames != nil && motion.MorphFrames.Contains(arm_sizing_morph_name) &&
		motion.MorphFrames.Contains(move_sizing_morph_name) {
		return motion
	}

	// サイジングボーンモーフを適用
	{
		mf := vmd.NewMorphFrame(float32(0))
		mf.Ratio = 1.0
		motion.AppendMorphFrame(arm_sizing_morph_name, mf)
	}
	{
		mf := vmd.NewMorphFrame(float32(0))
		mf.Ratio = 1.0
		motion.AppendMorphFrame(move_sizing_morph_name, mf)
	}

	return motion
}

var leg_ik_bone_names = []string{pmx.LEG_IK.Left(), pmx.LEG_IK.Right(), pmx.TOE_IK.Right(), pmx.TOE_IK.Left()}
var leg_fk_bone_names = []string{
	pmx.LEG.Left(), pmx.KNEE.Left(), pmx.HEEL.Left(), pmx.ANKLE.Left(), pmx.TOE.Left(), pmx.TOE_P.Left(),
	pmx.TOE_C.Left(), pmx.LEG_D.Left(), pmx.KNEE_D.Left(), pmx.HEEL_D.Left(), pmx.ANKLE_D.Left(),
	pmx.TOE_D.Left(), pmx.TOE_P_D.Left(), pmx.TOE_C_D.Left(), pmx.TOE_EX.Left(),
	pmx.LEG.Right(), pmx.KNEE.Right(), pmx.HEEL.Right(), pmx.ANKLE.Right(), pmx.TOE.Right(), pmx.TOE_P.Right(),
	pmx.TOE_C.Right(), pmx.LEG_D.Right(), pmx.KNEE_D.Right(), pmx.HEEL_D.Right(), pmx.ANKLE_D.Right(),
	pmx.TOE_D.Right(), pmx.TOE_P_D.Right(), pmx.TOE_C_D.Right(), pmx.TOE_EX.Right(),
}
var leg_all_bone_names = append(leg_fk_bone_names, leg_ik_bone_names...)
var move_bone_names = []string{pmx.ROOT.String(), pmx.CENTER.String(), pmx.GROOVE.String(),
	pmx.LEG_IK_PARENT.Left(), pmx.LEG_IK_PARENT.Right(), pmx.LEG_IK.Left(),
	pmx.LEG_IK.Right(), pmx.TOE_IK.Right(), pmx.TOE_IK.Left()}

func SizingLeg(sizingSet *model.SizingSet) {
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

			// 最終的な足FKを焼き込み
			bf := vmd.NewBoneFrame(boneDelta.Frame)
			bf.Rotation = boneDelta.FilledTotalRotation()
			bf.Position = originalMotion.BoneFrames.Get(boneDelta.Bone.Name()).Get(boneDelta.Frame).Position
			bf.Curves = originalMotion.BoneFrames.Get(boneDelta.Bone.Name()).Get(boneDelta.Frame).Curves
			if bf.Curves == nil {
				bf.Curves = vmd.NewBoneCurves()
			}

			sizingMotion.InsertRegisteredBoneFrame(boneDelta.Bone.Name(), bf)
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
	for _, vmdDeltas := range sizingAllDeltas {
		for _, boneDelta := range vmdDeltas.Bones.Data {
			if boneDelta == nil || !sizingMotion.BoneFrames.Contains(boneDelta.Bone.Name()) ||
				!slices.Contains(leg_ik_bone_names, boneDelta.Bone.Name()) {
				continue
			}

			// 最終的な足IKを焼き込み
			bf := vmd.NewBoneFrame(boneDelta.Frame)
			bf.Position = boneDelta.Bone.Position.ToMat4().Inverted().MulVec3(
				vmdDeltas.Bones.Get(boneDelta.Bone.Ik.BoneIndex).FilledGlobalPosition())
			bf.Rotation = originalMotion.BoneFrames.Get(boneDelta.Bone.Name()).Get(boneDelta.Frame).Rotation
			bf.Curves = originalMotion.BoneFrames.Get(boneDelta.Bone.Name()).Get(boneDelta.Frame).Curves
			if bf.Curves == nil {
				bf.Curves = vmd.NewBoneCurves()
			}

			sizingMotion.InsertRegisteredBoneFrame(boneDelta.Bone.Name(), bf)
		}
	}
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
	if sizingSet.IsSizingMove {
		scales = &mmath.MVec3{X: hipWidthRatio, Y: legHeightRatio, Z: hipWidthRatio}
	} else {
		scales = mmath.MVec3One
	}

	stanceQuat := make(map[int]*mmath.MQuaternion)

	// 腕スタンス補正
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

	var wg sync.WaitGroup
	for _, boneName := range originalMotion.BoneFrames.Names() {
		wg.Add(1)
		go func(bfs *vmd.BoneNameFrames) {
			defer wg.Done()
			for _, frame := range bfs.Indexes.List() {
				originalBf := originalMotion.BoneFrames.Get(boneName).Get(frame)
				sizingBf := sizingMotion.BoneFrames.Get(boneName).Get(frame)
				if slices.Contains(move_bone_names, bfs.Name) {
					sizingBf.Position = originalBf.Position.Muled(scales)
					sizingMotion.BoneFrames.Get(boneName).Update(sizingBf)
				}

				bone := sizingModel.Bones.GetByName(boneName)
				if bone == nil {
					continue
				}
				if _, ok := stanceQuat[bone.Index()]; !ok {
					continue
				}

				sizingBf.Rotation = originalBf.Rotation.Muled(stanceQuat[bone.Index()])
				sizingMotion.BoneFrames.Get(boneName).Update(sizingBf)
			}
		}(originalMotion.BoneFrames.Get(boneName))
	}

	wg.Wait()
}
