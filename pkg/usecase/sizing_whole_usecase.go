package usecase

import (
	"slices"

	"github.com/miu200521358/mlib_go/pkg/domain/delta"
	"github.com/miu200521358/mlib_go/pkg/domain/miter"
	"github.com/miu200521358/mlib_go/pkg/domain/mmath"
	"github.com/miu200521358/mlib_go/pkg/domain/pmx"
	"github.com/miu200521358/mlib_go/pkg/infrastructure/deform"
	"github.com/miu200521358/vmd_sizing_t3/pkg/model"
)

// 全身位置角度合わせ
func SizingWholeStance(sizingSet *model.SizingSet) {
	if !sizingSet.IsSizingWholeStance || (sizingSet.IsSizingWholeStance && sizingSet.CompletedSizingWholeStance) {
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

	sizingOffDeltas := make([]*delta.VmdDeltas, len(frames))

	// サイジング先モデルのデフォーム(IK OFF)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, false, frame, leg_all_bone_names, false)
		sizingOffDeltas[index] = vmdDeltas
	})

	// サイジング先にIK結果を焼き込み
	for i, vmdDeltas := range sizingOffDeltas {
		// 右足IKを動かさなかった場合のセンターと左足IKの位置を調整
		originalDeltas := originalAllDeltas[i]
		originalRightAnklePosition := originalDeltas.Bones.Get(originalModel.Bones.GetByName(pmx.LEG_IK.Right()).Ik.BoneIndex).FilledGlobalPosition()
		sizingRightAnklePosition := vmdDeltas.Bones.Get(sizingModel.Bones.GetByName(pmx.LEG_IK.Right()).Ik.BoneIndex).FilledGlobalPosition()
		rightAnkleDiff := sizingRightAnklePosition.Subed(originalRightAnklePosition)

		{
			boneName := pmx.LEG_IK.Right()
			boneDelta := vmdDeltas.Bones.GetByName(boneName)
			originalBf := originalMotion.BoneFrames.Get(boneName).Get(boneDelta.Frame)
			sizingBf := sizingMotion.BoneFrames.Get(boneName).Get(boneDelta.Frame)
			sizingBf.Position = originalBf.Position.Added(rightAnkleDiff)
			sizingMotion.InsertRegisteredBoneFrame(boneName, sizingBf)
		}
		{
			boneName := pmx.LEG_IK.Left()
			boneDelta := vmdDeltas.Bones.GetByName(boneName)
			originalBf := originalMotion.BoneFrames.Get(boneName).Get(boneDelta.Frame)
			sizingBf := sizingMotion.BoneFrames.Get(boneName).Get(boneDelta.Frame)
			sizingBf.Position = originalBf.Position.Added(rightAnkleDiff)
			sizingMotion.InsertRegisteredBoneFrame(boneName, sizingBf)
		}
		{
			boneName := pmx.CENTER.String()
			boneDelta := vmdDeltas.Bones.GetByName(boneName)
			originalBf := originalMotion.BoneFrames.Get(boneName).Get(boneDelta.Frame)
			sizingBf := sizingMotion.BoneFrames.Get(boneName).Get(boneDelta.Frame)
			sizingBf.Position = originalBf.Position.Added(rightAnkleDiff)
			sizingMotion.InsertRegisteredBoneFrame(boneName, sizingBf)
		}
	}
}
