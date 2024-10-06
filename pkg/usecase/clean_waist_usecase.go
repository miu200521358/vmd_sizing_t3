package usecase

import (
	"github.com/miu200521358/mlib_go/pkg/domain/delta"
	"github.com/miu200521358/mlib_go/pkg/domain/miter"
	"github.com/miu200521358/mlib_go/pkg/domain/mmath"
	"github.com/miu200521358/mlib_go/pkg/domain/pmx"
	"github.com/miu200521358/mlib_go/pkg/infrastructure/deform"
	"github.com/miu200521358/mlib_go/pkg/mutils/mi18n"
	"github.com/miu200521358/mlib_go/pkg/mutils/mlog"
	"github.com/miu200521358/vmd_sizing_t3/pkg/domain"
)

func CleanWaist(sizingSet *domain.SizingSet) {
	if !sizingSet.IsCleanWaist || (sizingSet.IsCleanWaist && sizingSet.CompletedCleanWaist) {
		return
	}

	if !isValidCleanWaist(sizingSet) {
		return
	}

	originalModel := sizingSet.OriginalPmx
	sizingMotion := sizingSet.OutputVmd

	if !sizingMotion.BoneFrames.ContainsActive(pmx.WAIST.String()) {
		return
	}

	mlog.I(mi18n.T("腰最適化開始", map[string]interface{}{"No": sizingSet.Index + 1}))

	waistBone := originalModel.Bones.GetByName(pmx.WAIST.String())

	waistRelativeBoneNames := []string{pmx.WAIST.String(), pmx.UPPER.String(), pmx.LOWER.String(), pmx.LEG.Left(), pmx.LEG.Right()}
	frames := sizingMotion.BoneFrames.RegisteredFrames(waistRelativeBoneNames)

	upperRotations := make([]*mmath.MQuaternion, len(frames))
	lowerRotations := make([]*mmath.MQuaternion, len(frames))
	legLeftRotations := make([]*mmath.MQuaternion, len(frames))
	legRightRotations := make([]*mmath.MQuaternion, len(frames))

	// 元モデルのデフォーム(IK ON)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		vmdDeltas := delta.NewVmdDeltas(frame, originalModel.Bones, originalModel.Hash(), sizingMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(originalModel, sizingMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(originalModel, sizingMotion, vmdDeltas, true, frame, waistRelativeBoneNames, false)

		for _, boneName := range waistRelativeBoneNames {
			if boneName == pmx.WAIST.String() {
				continue
			}

			bone := originalModel.Bones.GetByName(boneName)
			localRotation := sizingMotion.BoneFrames.Get(bone.Name()).Get(frame).Rotation
			for _, boneIndex := range bone.Extend.ParentBoneIndexes {
				boneDelta := vmdDeltas.Bones.Get(boneIndex)
				if boneDelta == nil {
					continue
				}
				localRotation = boneDelta.FilledFrameRotation().Muled(localRotation)
				// 親を上に行きすぎないように途中で終了
				if boneIndex == waistBone.Index() {
					break
				} else if (boneName == pmx.LEG.Left() || boneName == pmx.LEG.Right()) &&
					boneDelta.Bone.Name() == pmx.WAIST_CANCEL.StringFromDirection(bone.Direction()) {
					break
				}
			}

			switch boneName {
			case pmx.UPPER.String():
				upperRotations[index] = localRotation
			case pmx.LOWER.String():
				lowerRotations[index] = localRotation
			case pmx.LEG.Left():
				legLeftRotations[index] = localRotation
			case pmx.LEG.Right():
				legRightRotations[index] = localRotation
			}
		}
	})

	for i, iFrame := range frames {
		frame := float32(iFrame)

		for _, boneName := range waistRelativeBoneNames {
			if boneName == pmx.WAIST.String() {
				continue
			}
			bf := sizingMotion.BoneFrames.Get(boneName).Get(frame)

			switch boneName {
			case pmx.UPPER.String():
				bf.Rotation = upperRotations[i]
			case pmx.LOWER.String():
				bf.Rotation = lowerRotations[i]
			case pmx.LEG.Left():
				bf.Rotation = legLeftRotations[i]
			case pmx.LEG.Right():
				bf.Rotation = legRightRotations[i]
			}
			sizingMotion.InsertRegisteredBoneFrame(boneName, bf)
		}
	}

	sizingMotion.BoneFrames.Delete(pmx.WAIST.String())

	sizingSet.CompletedCleanWaist = true
}

func isValidCleanWaist(sizingSet *domain.SizingSet) bool {
	originalModel := sizingSet.SizingPmx

	if !originalModel.Bones.ContainsByName(pmx.WAIST.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("腰最適化ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.WAIST.String()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.UPPER.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("腰最適化ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.UPPER.String()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.LOWER.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("腰最適化ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.LOWER.String()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.LEG.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("腰最適化ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.LEG.Left()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.LEG.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("腰最適化ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.LEG.Right()}))
		return false
	}

	return true
}
