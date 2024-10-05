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

func CleanRoot(sizingSet *domain.SizingSet) {
	if !sizingSet.IsCleanRoot || (sizingSet.IsCleanRoot && sizingSet.CompletedCleanRoot) {
		return
	}

	if !isValidCleanRoot(sizingSet) {
		return
	}

	mlog.I(mi18n.T("全ての親クリーニング開始", map[string]interface{}{"No": sizingSet.Index + 1}))

	sizingModel := sizingSet.SizingPmx
	sizingMotion := sizingSet.OutputVmd

	rootRelativeBoneNames := []string{pmx.ROOT.String(), pmx.CENTER.String(), pmx.LEG_IK_PARENT.Left(), pmx.LEG_IK_PARENT.Right()}
	frames := sizingMotion.BoneFrames.RegisteredFrames(rootRelativeBoneNames)

	childLocalPositions := make([][]*mmath.MVec3, sizingModel.Bones.Len())
	childLocalRotations := make([][]*mmath.MQuaternion, sizingModel.Bones.Len())

	for _, boneName := range rootRelativeBoneNames {
		bone := sizingModel.Bones.GetByName(boneName)
		childLocalPositions[bone.Index()] = make([]*mmath.MVec3, len(frames))
		childLocalRotations[bone.Index()] = make([]*mmath.MQuaternion, len(frames))
	}

	// 元モデルのデフォーム(IK ON)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, true, frame, rootRelativeBoneNames, false)

		for _, boneName := range rootRelativeBoneNames {
			if boneName == pmx.ROOT.String() {
				continue
			}
			bone := sizingModel.Bones.GetByName(boneName)
			boneLocalPosition := vmdDeltas.Bones.Get(bone.Index()).FilledGlobalPosition().Subed(bone.Position)
			var boneLocalRotation *mmath.MQuaternion
			for _, boneIndex := range bone.Extend.ParentBoneIndexes {
				if boneLocalRotation == nil {
					boneLocalRotation = vmdDeltas.Bones.Get(boneIndex).FilledFrameRotation().Copy()
				} else {
					boneLocalRotation.Mul(vmdDeltas.Bones.Get(boneIndex).FilledFrameRotation())
				}
			}
			childLocalPositions[bone.Index()][index] = boneLocalPosition
			childLocalRotations[bone.Index()][index] = boneLocalRotation
		}
	})

	for _, boneName := range rootRelativeBoneNames {
		if boneName == pmx.ROOT.String() {
			continue
		}

		bone := sizingModel.Bones.GetByName(boneName)
		for j, frame := range frames {
			bf := sizingMotion.BoneFrames.Get(boneName).Get(float32(frame))
			bf.Position = childLocalPositions[bone.Index()][j]
			bf.Rotation = childLocalRotations[bone.Index()][j]
			sizingMotion.InsertRegisteredBoneFrame(boneName, bf)
		}
	}

	sizingMotion.BoneFrames.Delete(pmx.ROOT.String())
	sizingSet.CompletedCleanRoot = true
}

func isValidCleanRoot(sizingSet *domain.SizingSet) bool {
	sizingModel := sizingSet.SizingPmx

	if !sizingModel.Bones.ContainsByName(pmx.ROOT.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("全ての親クリーニングボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.ROOT.String()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.CENTER.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("全ての親クリーニングボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.CENTER.String()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.LEG_IK_PARENT.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("全ての親クリーニングボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.LEG_IK_PARENT.Left()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.LEG_IK_PARENT.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("全ての親クリーニングボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.LEG_IK_PARENT.Right()}))
		return false
	}

	return true
}
