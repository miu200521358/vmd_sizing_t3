package usecase

import (
	"sync"

	"github.com/miu200521358/mlib_go/pkg/domain/delta"
	"github.com/miu200521358/mlib_go/pkg/domain/miter"
	"github.com/miu200521358/mlib_go/pkg/domain/mmath"
	"github.com/miu200521358/mlib_go/pkg/domain/pmx"
	"github.com/miu200521358/mlib_go/pkg/domain/vmd"
	"github.com/miu200521358/mlib_go/pkg/infrastructure/deform"
	"github.com/miu200521358/mlib_go/pkg/mutils/mi18n"
	"github.com/miu200521358/mlib_go/pkg/mutils/mlog"
	"github.com/miu200521358/vmd_sizing_t3/pkg/domain"
)

func CleanLegIkParent(sizingSet *domain.SizingSet, setSize int) (bool, error) {
	if !sizingSet.IsCleanLegIkParent || (sizingSet.IsCleanLegIkParent && sizingSet.CompletedCleanLegIkParent) {
		return false, nil
	}

	if !isValidCleanLegIkParent(sizingSet) {
		return false, nil
	}

	originalModel := sizingSet.OriginalPmx
	originalMotion := sizingSet.OriginalVmd
	sizingMotion := sizingSet.OutputVmd

	if !(sizingMotion.BoneFrames.ContainsActive(pmx.LEG_IK_PARENT.Left()) ||
		sizingMotion.BoneFrames.ContainsActive(pmx.LEG_IK_PARENT.Right())) {
		return false, nil
	}

	mlog.I(mi18n.T("足IK親最適化開始", map[string]interface{}{"No": sizingSet.Index + 1}))
	sizingMotion.Processing = true

	legIkRelativeBoneNames := []string{
		pmx.LEG_IK_PARENT.Left(), pmx.LEG_IK_PARENT.Right(), pmx.LEG_IK.Left(), pmx.LEG_IK.Right()}
	legIkBoneNames := []string{pmx.LEG_IK.Left(), pmx.LEG_IK.Right()}
	frames := sizingMotion.BoneFrames.RegisteredFrames(legIkRelativeBoneNames)
	blockSize, _ := miter.GetBlockSize(len(frames) * setSize)

	if len(frames) == 0 {
		sizingMotion.Processing = false
		return false, nil
	}

	legIkLeftPositions := make([]*mmath.MVec3, len(frames))
	legIkRightPositions := make([]*mmath.MVec3, len(frames))
	legIkLeftRotations := make([]*mmath.MQuaternion, len(frames))
	legIkRightRotations := make([]*mmath.MQuaternion, len(frames))

	// 元モデルのデフォーム(IK OFF)
	if err := miter.IterParallelByList(frames, blockSize, func(data, index int) {
		frame := float32(data)
		vmdDeltas := delta.NewVmdDeltas(frame, originalModel.Bones, originalModel.Hash(), sizingMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(originalModel, sizingMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(originalModel, sizingMotion, vmdDeltas, false, frame, legIkRelativeBoneNames, false)

		for _, boneName := range legIkBoneNames {
			bone := originalModel.Bones.GetByName(boneName)

			legIkLocalPosition := vmdDeltas.Bones.Get(bone.Index()).FilledGlobalPosition().Subed(bone.Position)
			legIkLocalRotation := vmdDeltas.Bones.Get(bone.Index()).FilledGlobalBoneRotation()

			switch boneName {
			case pmx.LEG_IK.Left():
				legIkLeftPositions[index] = legIkLocalPosition
				legIkLeftRotations[index] = legIkLocalRotation
			case pmx.LEG_IK.Right():
				legIkRightPositions[index] = legIkLocalPosition
				legIkRightRotations[index] = legIkLocalRotation
			}
		}
	}, func(iterIndex, allCount int) {
		mlog.I(mi18n.T("足IK親最適化01", map[string]interface{}{"No": sizingSet.Index + 1, "IterIndex": iterIndex, "AllCount": allCount}))
	}); err != nil {
		sizingMotion.Processing = false
		return false, err
	}

	for i, iFrame := range frames {
		frame := float32(iFrame)

		for _, boneName := range legIkBoneNames {
			bone := originalModel.Bones.GetByName(boneName)
			bf := sizingMotion.BoneFrames.Get(bone.Name()).Get(frame)

			switch bone.Name() {
			case pmx.LEG_IK.Left():
				bf.Position = legIkLeftPositions[i]
				bf.Rotation = legIkLeftRotations[i]
			case pmx.LEG_IK.Right():
				bf.Position = legIkRightPositions[i]
				bf.Rotation = legIkRightRotations[i]
			}
			sizingMotion.InsertRegisteredBoneFrame(bone.Name(), bf)
		}
	}

	sizingMotion.BoneFrames.Delete(pmx.LEG_IK_PARENT.Left())
	sizingMotion.BoneFrames.Delete(pmx.LEG_IK_PARENT.Right())

	// 中間キーフレのズレをチェック
	threshold := 0.01
	var wg sync.WaitGroup

	logEndFrame := 0
	allCount := frames[len(frames)-1] - frames[0]
	for i, endFrame := range frames {
		if i == 0 {
			continue
		}
		startFrame := frames[i-1] + 1

		if endFrame-startFrame-1 <= 0 {
			continue
		}

		if endFrame%1000 == 0 && endFrame > logEndFrame {
			mlog.I(mi18n.T("足IK親最適化02", map[string]interface{}{"No": sizingSet.Index + 1, "IterIndex": endFrame, "AllCount": allCount}))
			logEndFrame += 1000
		}

		for iFrame := startFrame + 1; iFrame < endFrame; iFrame++ {
			frame := float32(iFrame)

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

			for _, boneName := range legIkBoneNames {
				go func(boneName string, bfs *vmd.BoneNameFrames) {
					defer wg.Done()

					bone := originalModel.Bones.GetByName(boneName)
					originalDelta := originalVmdDeltas.Bones.Get(bone.Index())
					cleanDelta := cleanVmdDeltas.Bones.Get(bone.Index())

					if originalDelta.FilledGlobalPosition().Distance(cleanDelta.FilledGlobalPosition()) > threshold {
						// ボーンの位置がずれている場合、キーを追加
						bf := bfs.Get(frame)
						bf.Position = originalDelta.FilledGlobalPosition().Subed(bone.Position)
						bf.Rotation = originalDelta.FilledGlobalBoneRotation()
						bf.Registered = true
						bfs.Insert(bf)
					}
				}(boneName, sizingMotion.BoneFrames.Get(boneName))
			}
		}
	}

	sizingSet.CompletedCleanLegIkParent = true
	sizingMotion.Processing = false
	return true, nil
}

func isValidCleanLegIkParent(sizingSet *domain.SizingSet) bool {
	originalModel := sizingSet.OriginalPmx

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

	return true
}
