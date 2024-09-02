package usecase

import (
	"fmt"
	"slices"

	"github.com/miu200521358/mlib_go/pkg/domain/delta"
	"github.com/miu200521358/mlib_go/pkg/domain/miter"
	"github.com/miu200521358/mlib_go/pkg/domain/mmath"
	"github.com/miu200521358/mlib_go/pkg/domain/pmx"
	"github.com/miu200521358/mlib_go/pkg/domain/vmd"
	"github.com/miu200521358/mlib_go/pkg/infrastructure/deform"
	"github.com/miu200521358/vmd_sizing_t3/pkg/model"
)

// FitBoneモーフ名
var sizing_morph_name = fmt.Sprintf("%s_%s", pmx.MLIB_PREFIX, "SizingBone")

func CreateSizingMorph(sizingSet *model.SizingSet) {
	sizingSet.SizingPmx.Morphs.RemoveByName(sizing_morph_name)

	createSizingMorph(sizingSet, sizing_morph_name)
	sizingSet.SizingPmx.Setup()
}

func createSizingMorph(sizingSet *model.SizingSet, morphName string) {
	offsets := make([]pmx.IMorphOffset, 0)

	originalModel := sizingSet.OriginalPmx
	sizingModel := sizingSet.SizingPmx

	if sizingSet.IsSizingArmStance {
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
	morph.SetName(morphName)
	morph.Offsets = offsets
	morph.MorphType = pmx.MORPH_TYPE_BONE
	morph.Panel = pmx.MORPH_PANEL_OTHER_LOWER_RIGHT
	morph.IsSystem = true
	sizingModel.Morphs.Append(morph)
}

func AddSizingMorph(motion *vmd.VmdMotion) *vmd.VmdMotion {
	if motion.MorphFrames != nil && motion.MorphFrames.Contains(sizing_morph_name) {
		return motion
	}

	// サイジングボーンモーフを適用
	mf := vmd.NewMorphFrame(float32(0))
	mf.Ratio = 1.0
	motion.AppendMorphFrame(sizing_morph_name, mf)
	return motion
}

func SizingLeg(sizingSet *model.SizingSet) {
	// 足補正
	originalModel := sizingSet.OriginalPmx
	originalMotion := sizingSet.OriginalVmd
	sizingModel := sizingSet.SizingPmx
	sizingMotion := sizingSet.OutputVmd

	frames := originalMotion.BoneFrames.RegisteredFrames(pmx.LEG_ALL_BONE_NAMES)

	originalAllDeltas := make([]*delta.VmdDeltas, len(frames))

	// 元モデルのデフォーム(IK ON)
	miter.IterParallelByList(frames, 100, func(data, index int) {
		frame := float32(data)
		vmdDeltas := delta.NewVmdDeltas(frame, originalModel.Bones, originalModel.Hash(), originalMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(originalModel, originalMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(originalModel, originalMotion, vmdDeltas, true, frame, pmx.LEG_FK_BONE_NAMES, false)
		originalAllDeltas[index] = vmdDeltas
	})

	// サイジング先にFKを焼き込み
	for _, vmdDeltas := range originalAllDeltas {
		for _, boneDelta := range vmdDeltas.Bones.Data {
			if boneDelta == nil || !sizingMotion.BoneFrames.Contains(boneDelta.Bone.Name()) ||
				!slices.Contains(pmx.LEG_FK_BONE_NAMES, boneDelta.Bone.Name()) {
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
		vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, false, frame, pmx.LEG_ALL_BONE_NAMES, false)
		sizingAllDeltas[index] = vmdDeltas
	})

	// サイジング先にIK結果を焼き込み
	for _, vmdDeltas := range sizingAllDeltas {
		for _, boneDelta := range vmdDeltas.Bones.Data {
			if boneDelta == nil || !sizingMotion.BoneFrames.Contains(boneDelta.Bone.Name()) ||
				!slices.Contains(pmx.LEG_IK_BONE_NAMES, boneDelta.Bone.Name()) {
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
