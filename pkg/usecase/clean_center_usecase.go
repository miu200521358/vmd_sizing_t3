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

func CleanCenter(sizingSet *domain.SizingSet, setSize int) (bool, error) {
	if !sizingSet.IsCleanCenter || (sizingSet.IsCleanCenter && sizingSet.CompletedCleanCenter) {
		return false, nil
	}

	if !isValidCleanCenter(sizingSet) {
		return false, nil
	}

	originalModel := sizingSet.OriginalPmx
	originalMotion := sizingSet.OriginalVmd
	sizingMotion := sizingSet.OutputVmd

	isContainsActiveWaist := sizingMotion.BoneFrames.ContainsActive(pmx.WAIST.String())

	if !(sizingMotion.BoneFrames.ContainsActive(pmx.CENTER.String()) ||
		isContainsActiveWaist) {
		return false, nil
	}

	mlog.I(mi18n.T("センター最適化開始", map[string]interface{}{"No": sizingSet.Index + 1}))

	centerBone := originalModel.Bones.GetByName(pmx.CENTER.String())
	grooveBone := originalModel.Bones.GetByName(pmx.GROOVE.String())
	upperBone := originalModel.Bones.GetByName(pmx.UPPER.String())
	lowerBone := originalModel.Bones.GetByName(pmx.LOWER.String())
	// 腰がある場合、腰キャンセルが効いてるので、足も登録する
	waistCancelLeftBone := originalModel.Bones.GetByName(pmx.WAIST_CANCEL.Left())
	waistCancelRightBone := originalModel.Bones.GetByName(pmx.WAIST_CANCEL.Right())
	legLeftBone := originalModel.Bones.GetByName(pmx.LEG.Left())
	legRightBone := originalModel.Bones.GetByName(pmx.LEG.Right())

	centerRelativeBoneNames := []string{pmx.CENTER.String(), pmx.WAIST.String(), pmx.GROOVE.String(), pmx.UPPER.String(), pmx.UPPER2.String(), pmx.LOWER.String(), pmx.LEG.Left(), pmx.LEG.Right()}

	frames := sizingMotion.BoneFrames.RegisteredFrames(centerRelativeBoneNames)
	blockSize := miter.GetBlockSize(len(frames) * setSize)

	if len(frames) == 0 {
		return false, nil
	}

	centerPositions := make([]*mmath.MVec3, len(frames))
	groovePositions := make([]*mmath.MVec3, len(frames))
	upperRotations := make([]*mmath.MQuaternion, len(frames))
	lowerRotations := make([]*mmath.MQuaternion, len(frames))
	legLeftRotations := make([]*mmath.MQuaternion, len(frames))
	legRightRotations := make([]*mmath.MQuaternion, len(frames))

	mlog.I(mi18n.T("センター最適化01", map[string]interface{}{"No": sizingSet.Index + 1}))

	// 元モデルのデフォーム(IK ON)
	if err := miter.IterParallelByList(frames, blockSize, func(data, index int) {
		frame := float32(data)
		ikOnVmdDeltas := delta.NewVmdDeltas(frame, originalModel.Bones, originalModel.Hash(), sizingMotion.Hash())
		ikOnVmdDeltas.Morphs = deform.DeformMorph(originalModel, sizingMotion.MorphFrames, frame, nil)
		ikOnVmdDeltas = deform.DeformBoneByPhysicsFlag(originalModel, sizingMotion, ikOnVmdDeltas, true, frame, centerRelativeBoneNames, false)

		ikOffVmdDeltas := delta.NewVmdDeltas(frame, originalModel.Bones, originalModel.Hash(), sizingMotion.Hash())
		ikOffVmdDeltas.Morphs = deform.DeformMorph(originalModel, sizingMotion.MorphFrames, frame, nil)
		ikOffVmdDeltas = deform.DeformBoneByPhysicsFlag(originalModel, sizingMotion, ikOffVmdDeltas, false, frame, centerRelativeBoneNames, false)

		upperLocalPosition := ikOnVmdDeltas.Bones.Get(upperBone.Index()).FilledGlobalPosition().Subed(upperBone.Position)
		centerPositions[index] = &mmath.MVec3{X: upperLocalPosition.X, Y: 0, Z: upperLocalPosition.Z}
		groovePositions[index] = &mmath.MVec3{X: 0, Y: upperLocalPosition.Y, Z: 0}
		upperRotations[index] = ikOffVmdDeltas.Bones.Get(upperBone.Index()).FilledGlobalBoneRotation()
		lowerRotations[index] = ikOffVmdDeltas.Bones.Get(lowerBone.Index()).FilledGlobalBoneRotation()
		if isContainsActiveWaist {
			// 足は腰がある場合のみ
			legLeftRotations[index] = ikOffVmdDeltas.Bones.TotalBoneRotation(waistCancelLeftBone.Index()).Muled(ikOffVmdDeltas.Bones.Get(legLeftBone.Index()).FilledFrameRotation())
			legRightRotations[index] = ikOffVmdDeltas.Bones.TotalBoneRotation(waistCancelRightBone.Index()).Muled(ikOffVmdDeltas.Bones.Get(legRightBone.Index()).FilledFrameRotation())
		}
	}); err != nil {
		return false, err
	}

	for i, iFrame := range frames {
		frame := float32(iFrame)

		for _, bone := range []*pmx.Bone{centerBone, grooveBone, upperBone, lowerBone, legLeftBone, legRightBone} {
			bf := sizingMotion.BoneFrames.Get(bone.Name()).Get(frame)

			switch bone.Name() {
			case pmx.CENTER.String():
				bf.Position = centerPositions[i]
				bf.Rotation = mmath.NewMQuaternion()
			case pmx.GROOVE.String():
				bf.Position = groovePositions[i]
				bf.Rotation = mmath.NewMQuaternion()
			case pmx.UPPER.String():
				bf.Rotation = upperRotations[i]
			case pmx.LOWER.String():
				bf.Rotation = lowerRotations[i]
			case pmx.LEG.Left():
				if isContainsActiveWaist {
					bf.Rotation = legLeftRotations[i]
				}
			case pmx.LEG.Right():
				if isContainsActiveWaist {
					bf.Rotation = legRightRotations[i]
				}
			}

			if (bone.IsLegFK() && isContainsActiveWaist) || !bone.IsLegFK() {
				// 足は腰がある場合のみ
				sizingMotion.InsertRegisteredBoneFrame(bone.Name(), bf)
			}
		}
	}

	sizingMotion.BoneFrames.Delete(pmx.WAIST.String())

	mlog.I(mi18n.T("センター最適化02", map[string]interface{}{"No": sizingSet.Index + 1}))

	// 中間キーフレのズレをチェック
	threshold := 0.0005

	for i, endFrame := range frames {
		if i == 0 {
			continue
		}
		startFrame := frames[i-1] + 1

		if endFrame-startFrame-1 <= 0 {
			continue
		}

		if err := miter.IterParallelByCount(endFrame-startFrame-1, blockSize, func(index int) {
			frame := float32(startFrame + index + 1)

			var wg sync.WaitGroup
			wg.Add(2)
			var originalVmdDeltas, cleanVmdDeltas *delta.VmdDeltas

			go func() {
				defer wg.Done()
				originalVmdDeltas = delta.NewVmdDeltas(frame, originalModel.Bones, originalModel.Hash(), originalMotion.Hash())
				originalVmdDeltas.Morphs = deform.DeformMorph(originalModel, originalMotion.MorphFrames, frame, nil)
				originalVmdDeltas = deform.DeformBoneByPhysicsFlag(originalModel, originalMotion, originalVmdDeltas, false, frame, centerRelativeBoneNames, false)
			}()

			go func() {
				defer wg.Done()
				cleanVmdDeltas = delta.NewVmdDeltas(frame, originalModel.Bones, originalModel.Hash(), sizingMotion.Hash())
				cleanVmdDeltas.Morphs = deform.DeformMorph(originalModel, sizingMotion.MorphFrames, frame, nil)
				cleanVmdDeltas = deform.DeformBoneByPhysicsFlag(originalModel, sizingMotion, cleanVmdDeltas, false, frame, centerRelativeBoneNames, false)
			}()

			wg.Wait()

			{
				bone := originalModel.Bones.GetByName(pmx.UPPER2.String())
				originalDelta := originalVmdDeltas.Bones.Get(bone.Index())
				cleanDelta := cleanVmdDeltas.Bones.Get(bone.Index())

				if originalDelta.FilledGlobalPosition().Distance(cleanDelta.FilledGlobalPosition()) > threshold {
					// ボーンの位置がずれている場合、キーを追加
					localPosition := originalDelta.FilledGlobalPosition().Subed(cleanDelta.FilledGlobalPosition())

					{
						bf := sizingMotion.BoneFrames.Get(pmx.CENTER.String()).Get(frame)
						bf.Position.Add(&mmath.MVec3{X: localPosition.X, Y: 0, Z: localPosition.Z})
						sizingMotion.InsertRegisteredBoneFrame(pmx.CENTER.String(), bf)
					}
					{
						bf := sizingMotion.BoneFrames.Get(pmx.GROOVE.String()).Get(frame)
						bf.Position.Add(&mmath.MVec3{X: 0, Y: localPosition.Y, Z: 0})
						sizingMotion.InsertRegisteredBoneFrame(pmx.GROOVE.String(), bf)
					}
				}
			}
		}); err != nil {
			return false, err
		}
	}

	sizingSet.CompletedCleanCenter = true
	return true, nil
}

func isValidCleanCenter(sizingSet *domain.SizingSet) bool {
	originalModel := sizingSet.OriginalPmx

	if !originalModel.Bones.ContainsByName(pmx.CENTER.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("センター最適化ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.CENTER.String()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.GROOVE.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("センター最適化ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.GROOVE.String()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.UPPER.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("センター最適化ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.UPPER.String()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.LOWER.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("センター最適化ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.LOWER.String()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.LEG.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("センター最適化ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.LEG.Left()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.LEG.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("センター最適化ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.LEG.Right()}))
		return false
	}

	return true
}
