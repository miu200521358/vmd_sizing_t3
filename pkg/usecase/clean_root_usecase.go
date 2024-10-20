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

func CleanRoot(sizingSet *domain.SizingSet) bool {
	if !sizingSet.IsCleanRoot || (sizingSet.IsCleanRoot && sizingSet.CompletedCleanRoot) {
		return false
	}

	if !isValidCleanRoot(sizingSet) {
		return false
	}

	originalModel := sizingSet.OriginalPmx
	originalMotion := sizingSet.OriginalVmd
	sizingMotion := sizingSet.OutputVmd

	if !sizingMotion.BoneFrames.ContainsActive(pmx.ROOT.String()) {
		return false
	}

	mlog.I(mi18n.T("全ての親最適化開始", map[string]interface{}{"No": sizingSet.Index + 1}))

	rootRelativeBoneNames := []string{pmx.ROOT.String(), pmx.CENTER.String(), pmx.LEG_IK_PARENT.Left(), pmx.LEG_IK_PARENT.Right()}
	frames := sizingMotion.BoneFrames.RegisteredFrames(rootRelativeBoneNames)

	if len(frames) == 0 {
		return false
	}

	childLocalPositions := make([][]*mmath.MVec3, originalModel.Bones.Len())
	childLocalRotations := make([][]*mmath.MQuaternion, originalModel.Bones.Len())

	for _, boneName := range rootRelativeBoneNames {
		bone := originalModel.Bones.GetByName(boneName)
		childLocalPositions[bone.Index()] = make([]*mmath.MVec3, len(frames))
		childLocalRotations[bone.Index()] = make([]*mmath.MQuaternion, len(frames))
	}

	mlog.I(mi18n.T("全ての親最適化01", map[string]interface{}{"No": sizingSet.Index + 1}))

	// 元モデルのデフォーム(IK ON)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		vmdDeltas := delta.NewVmdDeltas(frame, originalModel.Bones, originalModel.Hash(), sizingMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(originalModel, sizingMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(originalModel, sizingMotion, vmdDeltas, true, frame, rootRelativeBoneNames, false)

		for _, boneName := range rootRelativeBoneNames {
			if boneName == pmx.ROOT.String() {
				continue
			}
			bone := originalModel.Bones.GetByName(boneName)
			childLocalPositions[bone.Index()][index] =
				vmdDeltas.Bones.Get(bone.Index()).FilledGlobalPosition().Subed(bone.Position)
			childLocalRotations[bone.Index()][index] =
				vmdDeltas.Bones.Get(bone.Index()).FilledGlobalBoneRotation()
		}
	})

	for _, boneName := range rootRelativeBoneNames {
		if boneName == pmx.ROOT.String() {
			continue
		}

		bone := originalModel.Bones.GetByName(boneName)
		for j, frame := range frames {
			bf := sizingMotion.BoneFrames.Get(boneName).Get(float32(frame))
			bf.Position = childLocalPositions[bone.Index()][j]
			bf.Rotation = childLocalRotations[bone.Index()][j]
			sizingMotion.InsertRegisteredBoneFrame(boneName, bf)
		}
	}

	sizingMotion.BoneFrames.Delete(pmx.ROOT.String())

	mlog.I(mi18n.T("全ての親最適化02", map[string]interface{}{"No": sizingSet.Index + 1}))

	// 中間キーフレのズレをチェック
	threshold := 0.01
	var wg sync.WaitGroup

	for i, endFrame := range frames {
		if i == 0 {
			continue
		}
		startFrame := frames[i-1] + 1

		if endFrame-startFrame-1 <= 0 {
			continue
		}

		miter.IterParallelByCount(endFrame-startFrame-1, 500, func(index int) {
			frame := float32(startFrame + index + 1)

			wg.Add(2)
			var originalVmdDeltas, cleanVmdDeltas *delta.VmdDeltas

			go func() {
				defer wg.Done()
				originalVmdDeltas = delta.NewVmdDeltas(frame, originalModel.Bones, originalModel.Hash(), originalMotion.Hash())
				originalVmdDeltas.Morphs = deform.DeformMorph(originalModel, originalMotion.MorphFrames, frame, nil)
				originalVmdDeltas = deform.DeformBoneByPhysicsFlag(originalModel, originalMotion, originalVmdDeltas, false, frame, rootRelativeBoneNames, false)
			}()

			go func() {
				defer wg.Done()
				cleanVmdDeltas = delta.NewVmdDeltas(frame, originalModel.Bones, originalModel.Hash(), sizingMotion.Hash())
				cleanVmdDeltas.Morphs = deform.DeformMorph(originalModel, sizingMotion.MorphFrames, frame, nil)
				cleanVmdDeltas = deform.DeformBoneByPhysicsFlag(originalModel, sizingMotion, cleanVmdDeltas, false, frame, rootRelativeBoneNames, false)
			}()

			wg.Wait()

			wg.Add(3)

			for _, boneName := range rootRelativeBoneNames {
				if boneName == pmx.ROOT.String() {
					continue
				}

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

			wg.Wait()
		})
	}

	sizingSet.CompletedCleanRoot = true
	return true
}

func isValidCleanRoot(sizingSet *domain.SizingSet) bool {
	originalModel := sizingSet.SizingPmx

	if !originalModel.Bones.ContainsByName(pmx.ROOT.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("全ての親最適化ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.ROOT.String()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.CENTER.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("全ての親最適化ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.CENTER.String()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.LEG_IK_PARENT.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("全ての親最適化ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.LEG_IK_PARENT.Left()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.LEG_IK_PARENT.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("全ての親最適化ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.LEG_IK_PARENT.Right()}))
		return false
	}

	return true
}
