package usecase

import (
	"sync"

	"github.com/miu200521358/mlib_go/pkg/domain/delta"
	"github.com/miu200521358/mlib_go/pkg/domain/miter"
	"github.com/miu200521358/mlib_go/pkg/domain/mmath"
	"github.com/miu200521358/mlib_go/pkg/domain/pmx"
	"github.com/miu200521358/mlib_go/pkg/infrastructure/deform"
	"github.com/miu200521358/mlib_go/pkg/mutils/mi18n"
	"github.com/miu200521358/mlib_go/pkg/mutils/mlog"
	"github.com/miu200521358/vmd_sizing_t3/pkg/domain"
)

func CleanLegIkParent(sizingSet *domain.SizingSet) {
	if !sizingSet.IsCleanLegIkParent || (sizingSet.IsCleanLegIkParent && sizingSet.CompletedCleanLegIkParent) {
		return
	}

	if !isValidCleanLegIkParent(sizingSet) {
		return
	}

	mlog.I(mi18n.T("足IK親最適化開始", map[string]interface{}{"No": sizingSet.Index + 1}))

	originalModel := sizingSet.OriginalPmx
	originalMotion := sizingSet.OriginalVmd
	sizingMotion := sizingSet.OutputVmd

	originalLegIkParentLeftBone := originalModel.Bones.GetByName(pmx.LEG_IK_PARENT.Left())
	originalLegIkParentRightBone := originalModel.Bones.GetByName(pmx.LEG_IK_PARENT.Right())
	originalLegIkLeftBone := originalModel.Bones.GetByName(pmx.LEG_IK.Left())
	originalLegIkRightBone := originalModel.Bones.GetByName(pmx.LEG_IK.Right())

	if !sizingMotion.BoneFrames.ContainsActive(originalLegIkParentLeftBone.Name()) ||
		!sizingMotion.BoneFrames.ContainsActive(originalLegIkParentRightBone.Name()) {
		return
	}

	legIkRelativeBoneNames := []string{
		pmx.LEG_IK_PARENT.Left(), pmx.LEG_IK_PARENT.Right(), pmx.LEG_IK.Left(), pmx.LEG_IK.Right()}
	frames := sizingMotion.BoneFrames.RegisteredFrames(legIkRelativeBoneNames)

	legIkLeftPositions := make([]*mmath.MVec3, len(frames))
	legIkRightPositions := make([]*mmath.MVec3, len(frames))
	legIkLeftRotations := make([]*mmath.MQuaternion, len(frames))
	legIkRightRotations := make([]*mmath.MQuaternion, len(frames))

	mlog.I(mi18n.T("足IK親最適化01", map[string]interface{}{"No": sizingSet.Index + 1}))

	// 元モデルのデフォーム(IK OFF)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		vmdDeltas := delta.NewVmdDeltas(frame, originalModel.Bones, originalModel.Hash(), sizingMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(originalModel, sizingMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(originalModel, sizingMotion, vmdDeltas, false, frame, legIkRelativeBoneNames, false)

		for _, boneName := range []string{pmx.LEG_IK_PARENT.Left(), pmx.LEG_IK_PARENT.Right()} {
			bone := originalModel.Bones.GetByName(boneName)

			legIkLocalRotation := sizingMotion.BoneFrames.Get(bone.Name()).Get(frame).Rotation
			for _, boneIndex := range bone.Extend.ParentBoneIndexes {
				boneDelta := vmdDeltas.Bones.Get(boneIndex)
				if boneDelta == nil {
					continue
				}
				legIkLocalRotation = boneDelta.FilledFrameRotation().Muled(legIkLocalRotation)
			}
			legIkLocalPosition := vmdDeltas.Bones.Get(bone.Index()).FilledGlobalPosition().Subed(bone.Position)

			switch boneName {
			case pmx.LEG_IK_PARENT.Left():
				legIkLeftPositions[index] = legIkLocalPosition
				legIkLeftRotations[index] = legIkLocalRotation
			case pmx.LEG_IK_PARENT.Right():
				legIkRightPositions[index] = legIkLocalPosition
				legIkRightRotations[index] = legIkLocalRotation
			}
		}
	})

	for i, iFrame := range frames {
		frame := float32(iFrame)

		for _, bone := range []*pmx.Bone{originalLegIkLeftBone, originalLegIkRightBone} {
			bf := sizingMotion.BoneFrames.Get(bone.Name()).Get(frame)

			switch bone.Name() {
			case pmx.LEG_IK_PARENT.Left():
				bf.Position = legIkLeftPositions[i]
				bf.Rotation = legIkLeftRotations[i]
			case pmx.LEG_IK_PARENT.Right():
				bf.Position = legIkRightPositions[i]
				bf.Rotation = legIkRightRotations[i]
			}
			sizingMotion.InsertRegisteredBoneFrame(bone.Name(), bf)
		}
	}

	sizingMotion.BoneFrames.Delete(pmx.LEG_IK_PARENT.Left())
	sizingMotion.BoneFrames.Delete(pmx.LEG_IK_PARENT.Right())

	mlog.I(mi18n.T("足IK親最適化02", map[string]interface{}{"No": sizingSet.Index + 1}))

	// 中間キーフレのズレをチェック
	threshold := originalLegIkLeftBone.Position.Y * 0.05
	var wg sync.WaitGroup

	for i, endFrame := range frames {
		if i == 0 {
			continue
		}
		startFrame := frames[i-1] + 1
		legIkLeftBfs := sizingMotion.BoneFrames.Get(pmx.LEG_IK.Left())
		legIkRightBfs := sizingMotion.BoneFrames.Get(pmx.LEG_IK.Right())

		miter.IterParallelByCount(endFrame-startFrame-1, 500, func(index int) {
			frame := float32(startFrame + index + 1)

			wg.Add(2)
			var originalVmdDeltas, cleanVmdDeltas *delta.VmdDeltas

			go func() {
				defer wg.Done()
				originalVmdDeltas = delta.NewVmdDeltas(frame, originalModel.Bones, originalModel.Hash(), originalMotion.Hash())
				originalVmdDeltas.Morphs = deform.DeformMorph(originalModel, originalMotion.MorphFrames, frame, nil)
				originalVmdDeltas = deform.DeformBoneByPhysicsFlag(originalModel, originalMotion, originalVmdDeltas, false, frame, legIkRelativeBoneNames, false)
			}()

			go func() {
				defer wg.Done()
				cleanVmdDeltas = delta.NewVmdDeltas(frame, originalModel.Bones, originalModel.Hash(), sizingMotion.Hash())
				cleanVmdDeltas.Morphs = deform.DeformMorph(originalModel, sizingMotion.MorphFrames, frame, nil)
				cleanVmdDeltas = deform.DeformBoneByPhysicsFlag(originalModel, sizingMotion, cleanVmdDeltas, false, frame, legIkRelativeBoneNames, false)
			}()

			wg.Wait()

			wg.Add(2)

			go func() {
				defer wg.Done()

				originalLegIkLeftDelta := originalVmdDeltas.Bones.Get(originalLegIkLeftBone.Index())
				cleanLegIkLeftDelta := cleanVmdDeltas.Bones.Get(originalLegIkLeftBone.Index())

				if originalLegIkLeftDelta.FilledGlobalPosition().Distance(
					cleanLegIkLeftDelta.FilledGlobalPosition()) > threshold {
					// 足IKの位置がずれている場合、キーを追加

					legIkLeftPosition := originalLegIkLeftDelta.FilledGlobalPosition().Subed(
						originalLegIkLeftBone.Position)

					leftLegIkLocalRotation := sizingMotion.BoneFrames.Get(originalLegIkLeftBone.Name()).Get(frame).Rotation
					for _, boneIndex := range originalLegIkLeftBone.Extend.ParentBoneIndexes {
						boneDelta := originalVmdDeltas.Bones.Get(boneIndex)
						if boneDelta == nil {
							continue
						}
						leftLegIkLocalRotation = boneDelta.FilledFrameRotation().Muled(leftLegIkLocalRotation)
					}
					legIkLeftRotation := leftLegIkLocalRotation

					legIkLeftBf := sizingMotion.BoneFrames.Get(pmx.LEG_IK.Left()).Get(frame)
					legIkLeftBf.Position = legIkLeftPosition
					legIkLeftBf.Rotation = legIkLeftRotation
					legIkLeftBf.Registered = true
					legIkLeftBfs.Insert(legIkLeftBf)
				}
			}()

			go func() {
				defer wg.Done()

				originalLegIkRightDelta := originalVmdDeltas.Bones.Get(originalLegIkRightBone.Index())
				cleanLegIkRightDelta := cleanVmdDeltas.Bones.Get(originalLegIkRightBone.Index())

				if originalLegIkRightDelta.FilledGlobalPosition().Distance(
					cleanLegIkRightDelta.FilledGlobalPosition()) > threshold {
					// 足IKの位置がずれている場合、キーを追加

					legIkRightPosition := originalLegIkRightDelta.FilledGlobalPosition().Subed(
						originalLegIkRightBone.Position)

					leftLegIkLocalRotation := sizingMotion.BoneFrames.Get(originalLegIkRightBone.Name()).Get(frame).Rotation
					for _, boneIndex := range originalLegIkRightBone.Extend.ParentBoneIndexes {
						boneDelta := originalVmdDeltas.Bones.Get(boneIndex)
						if boneDelta == nil {
							continue
						}
						leftLegIkLocalRotation = boneDelta.FilledFrameRotation().Muled(leftLegIkLocalRotation)
					}
					legIkRightRotation := leftLegIkLocalRotation

					legIkRightBf := sizingMotion.BoneFrames.Get(pmx.LEG_IK.Right()).Get(frame)
					legIkRightBf.Position = legIkRightPosition
					legIkRightBf.Rotation = legIkRightRotation
					legIkRightBf.Registered = true
					legIkRightBfs.Insert(legIkRightBf)
				}
			}()

			wg.Wait()
		})
	}

	sizingSet.CompletedCleanLegIkParent = true
}

func isValidCleanLegIkParent(sizingSet *domain.SizingSet) bool {
	originalModel := sizingSet.OriginalPmx
	sizingModel := sizingSet.SizingPmx

	if !originalModel.Bones.ContainsByName(pmx.LEG_IK_PARENT.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足IK親最適化ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.LEG_IK_PARENT.Left()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.LEG_IK_PARENT.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足IK親最適化ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.LEG_IK_PARENT.Right()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.LEG_IK.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足IK親最適化ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.LEG_IK.Left()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.LEG_IK.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足IK親最適化ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.LEG_IK.Right()}))
		return false
	}

	// ----------

	if !sizingModel.Bones.ContainsByName(pmx.LEG_IK_PARENT.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足IK親最適化ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.LEG_IK_PARENT.Left()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.LEG_IK_PARENT.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足IK親最適化ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.LEG_IK_PARENT.Right()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.LEG_IK.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足IK親最適化ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.LEG_IK.Left()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.LEG_IK.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足IK親最適化ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.LEG_IK.Right()}))
		return false
	}

	return true
}
