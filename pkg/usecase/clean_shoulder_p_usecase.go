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

func CleanShoulderP(sizingSet *domain.SizingSet, setSize int) (bool, error) {
	if !sizingSet.IsCleanShoulderP || (sizingSet.IsCleanShoulderP && sizingSet.CompletedCleanShoulderP) {
		return false, nil
	}

	if !isValidCleanShoulderP(sizingSet) {
		return false, nil
	}

	originalModel := sizingSet.OriginalPmx
	originalMotion := sizingSet.OriginalVmd
	sizingMotion := sizingSet.OutputVmd

	mlog.I(mi18n.T("肩P最適化開始", map[string]interface{}{"No": sizingSet.Index + 1}))

	allFrames := make([][]int, 2)
	shoulderRotations := make([][]*mmath.MQuaternion, 2)
	armRotations := make([][]*mmath.MQuaternion, 2)
	allBlockSizes := make([]int, 2)

	for i, direction := range directions {
		frames := sizingMotion.BoneFrames.RegisteredFrames(shoulder_direction_bone_names[i])
		allBlockSizes[i], _ = miter.GetBlockSize(len(frames) * setSize)

		allFrames[i] = frames
		shoulderRotations[i] = make([]*mmath.MQuaternion, len(frames))
		armRotations[i] = make([]*mmath.MQuaternion, len(frames))

		shoulderRootBone := originalModel.Bones.GetByName(pmx.SHOULDER_ROOT.StringFromDirection(direction))
		shoulderBone := originalModel.Bones.GetByName(pmx.SHOULDER.StringFromDirection(direction))
		armBone := originalModel.Bones.GetByName(pmx.ARM.StringFromDirection(direction))

		// 元モデルのデフォーム(IK ON)
		if err := miter.IterParallelByList(frames, allBlockSizes[i], func(data, index int) {
			frame := float32(data)
			vmdDeltas := delta.NewVmdDeltas(frame, originalModel.Bones, originalModel.Hash(), sizingMotion.Hash())
			vmdDeltas.Morphs = deform.DeformMorph(originalModel, sizingMotion.MorphFrames, frame, nil)
			vmdDeltas = deform.DeformBoneByPhysicsFlag(originalModel, sizingMotion, vmdDeltas, true, frame, shoulder_direction_bone_names[i], false)

			shoulderRootDelta := vmdDeltas.Bones.Get(shoulderRootBone.Index())
			shoulderDelta := vmdDeltas.Bones.Get(shoulderBone.Index())
			armBoneDelta := vmdDeltas.Bones.Get(armBone.Index())

			shoulderRotations[i][index] = shoulderRootDelta.FilledGlobalMatrix().Inverted().Muled(shoulderDelta.FilledGlobalMatrix()).Quaternion()
			armRotations[i][index] = shoulderDelta.FilledGlobalMatrix().Inverted().Muled(armBoneDelta.FilledGlobalMatrix()).Quaternion()
		}, func(iterIndex, allCount int) {
			mlog.I(mi18n.T("肩P最適化01", map[string]interface{}{"No": sizingSet.Index + 1, "Direction": direction, "IterIndex": iterIndex, "AllCount": allCount}))
		}); err != nil {
			return false, err
		}
	}

	for i, direction := range directions {
		sizingMotion.BoneFrames.Delete(pmx.SHOULDER_P.StringFromDirection(direction))

		shoulderBoneName := pmx.SHOULDER.StringFromDirection(direction)
		armBoneName := pmx.ARM.StringFromDirection(direction)

		for j, iFrame := range allFrames[i] {
			frame := float32(iFrame)

			{
				bf := sizingMotion.BoneFrames.Get(shoulderBoneName).Get(frame)
				bf.Rotation = shoulderRotations[i][j]
				sizingMotion.InsertRegisteredBoneFrame(shoulderBoneName, bf)
			}
			{
				bf := sizingMotion.BoneFrames.Get(armBoneName).Get(frame)
				bf.Rotation = armRotations[i][j]
				sizingMotion.InsertRegisteredBoneFrame(armBoneName, bf)
			}
		}
	}

	// 中間キーフレのズレをチェック
	threshold := 0.02
	errorChan := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)

	for i, direction := range directions {
		go func(i int, direction string) {
			defer wg.Done()
			defer func() {
				// recoverによるpanicキャッチ
				if r := recover(); r != nil {
					errorChan <- miter.GetError()
				}
			}()

			frames := allFrames[i]

			shoulderRootBone := originalModel.Bones.GetByName(pmx.SHOULDER_ROOT.StringFromDirection(direction))
			shoulderBone := originalModel.Bones.GetByName(pmx.SHOULDER.StringFromDirection(direction))
			armBone := originalModel.Bones.GetByName(pmx.ARM.StringFromDirection(direction))

			shoulderBfs := sizingMotion.BoneFrames.Get(shoulderBone.Name())
			armBfs := sizingMotion.BoneFrames.Get(armBone.Name())

			logEndFrame := 0
			allCount := frames[len(frames)-1] - frames[0]

			for j, endFrame := range frames {
				if j == 0 {
					continue
				}
				startFrame := frames[j-1] + 1

				if endFrame-startFrame-1 <= 0 {
					continue
				}

				if endFrame%1000 == 0 && endFrame > logEndFrame {
					mlog.I(mi18n.T("肩P最適化02", map[string]interface{}{"No": sizingSet.Index + 1, "Direction": direction, "IterIndex": endFrame, "AllCount": allCount}))
					logEndFrame += 1000
				}

				for iFrame := startFrame + 1; iFrame < endFrame; iFrame++ {
					frame := float32(iFrame)

					originalVmdDeltas := delta.NewVmdDeltas(frame, originalModel.Bones, originalModel.Hash(), originalMotion.Hash())
					originalVmdDeltas.Morphs = deform.DeformMorph(originalModel, originalMotion.MorphFrames, frame, nil)
					originalVmdDeltas = deform.DeformBoneByPhysicsFlag(originalModel, originalMotion, originalVmdDeltas, true, frame, shoulder_direction_bone_names[i], false)

					cleanVmdDeltas := delta.NewVmdDeltas(frame, originalModel.Bones, originalModel.Hash(), sizingMotion.Hash())
					cleanVmdDeltas.Morphs = deform.DeformMorph(originalModel, sizingMotion.MorphFrames, frame, nil)
					cleanVmdDeltas = deform.DeformBoneByPhysicsFlag(originalModel, sizingMotion, cleanVmdDeltas, true, frame, shoulder_direction_bone_names[i], false)

					originalDelta := originalVmdDeltas.Bones.Get(armBone.Index())
					cleanDelta := cleanVmdDeltas.Bones.Get(armBone.Index())

					if originalDelta.FilledGlobalPosition().Distance(cleanDelta.FilledGlobalPosition()) > threshold {
						shoulderRootDelta := originalVmdDeltas.Bones.Get(shoulderRootBone.Index())
						shoulderDelta := originalVmdDeltas.Bones.Get(shoulderBone.Index())
						armBoneDelta := originalVmdDeltas.Bones.Get(armBone.Index())

						shoulderRotation := shoulderRootDelta.FilledGlobalMatrix().Inverted().Muled(shoulderDelta.FilledGlobalMatrix()).Quaternion()
						armRotation := shoulderDelta.FilledGlobalMatrix().Inverted().Muled(armBoneDelta.FilledGlobalMatrix()).Quaternion()

						shoulderBf := shoulderBfs.Get(frame)
						shoulderBf.Rotation = shoulderRotation
						shoulderBf.Registered = true
						shoulderBfs.Insert(shoulderBf)

						armBf := armBfs.Get(frame)
						armBf.Rotation = armRotation
						armBf.Registered = true
						armBfs.Insert(armBf)
					}
				}
			}
		}(i, direction)
	}

	// すべてのゴルーチンの完了を待つ
	wg.Wait()
	close(errorChan) // 全てのゴルーチンが終了したらチャネルを閉じる

	// チャネルからエラーを受け取る
	for err := range errorChan {
		if err != nil {
			return false, err
		}
	}

	sizingSet.CompletedCleanShoulderP = true
	return true, nil
}

func isValidCleanShoulderP(sizingSet *domain.SizingSet) bool {
	originalModel := sizingSet.OriginalPmx

	for _, direction := range directions {
		if !originalModel.Bones.ContainsByName(pmx.SHOULDER_P.StringFromDirection(direction)) {
			mlog.WT(mi18n.T("ボーン不足"), mi18n.T("肩P最適化ボーン不足", map[string]interface{}{
				"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.SHOULDER_P.StringFromDirection(direction)}))
			return false
		}

		if !originalModel.Bones.ContainsByName(pmx.SHOULDER.StringFromDirection(direction)) {
			mlog.WT(mi18n.T("ボーン不足"), mi18n.T("肩P最適化ボーン不足", map[string]interface{}{
				"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.SHOULDER.StringFromDirection(direction)}))
			return false
		}

		if !originalModel.Bones.ContainsByName(pmx.ARM.StringFromDirection(direction)) {
			mlog.WT(mi18n.T("ボーン不足"), mi18n.T("肩P最適化ボーン不足", map[string]interface{}{
				"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.ARM.StringFromDirection(direction)}))
			return false
		}
	}

	return true
}
