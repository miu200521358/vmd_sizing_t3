package usecase

import (
	"fmt"
	"slices"
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

func CleanGrip(sizingSet *domain.SizingSet, setSize int) (bool, error) {
	if !sizingSet.IsCleanGrip || (sizingSet.IsCleanGrip && sizingSet.CompletedCleanGrip) {
		return false, nil
	}

	if !isValidCleanGrip(sizingSet) {
		return false, nil
	}

	originalModel := sizingSet.OriginalPmx
	originalMotion := sizingSet.OriginalVmd
	sizingMotion := sizingSet.OutputVmd

	// 握り拡散に相当するボーンがあるか取得
	gripBones := getGripBones(originalModel)

	if len(gripBones) == 0 {
		return false, nil
	}

	hasGripBoneFrame := false
	for _, gripBone := range gripBones {
		if sizingMotion.BoneFrames.ContainsActive(gripBone.Name()) {
			hasGripBoneFrame = true
			break
		}
	}

	if !hasGripBoneFrame {
		return false, nil
	}

	mlog.I(mi18n.T("握り最適化開始", map[string]interface{}{"No": sizingSet.Index + 1}))
	sizingMotion.Processing = true

	allFrames := make([][]int, 2)
	allVmdDeltas := make([][]*delta.VmdDeltas, 2)
	allBlockSizes := make([]int, 2)
	allBlockCounts := make([]int, 2)

	for i, direction := range directions {
		fingerBoneNames := make([]string, 0)
		fingerBoneNames = append(fingerBoneNames, finger_direction_bone_names[i]...)
		for _, gripBone := range gripBones {
			if gripBone.Direction() == direction {
				fingerBoneNames = append(fingerBoneNames, gripBone.Name())
			}
		}

		frames := sizingMotion.BoneFrames.RegisteredFrames(fingerBoneNames)
		allBlockSizes[i], allBlockCounts[i] = miter.GetBlockSize(len(frames) * setSize)

		allFrames[i] = frames
		allVmdDeltas[i] = make([]*delta.VmdDeltas, len(frames))

		// 元モデルのデフォーム(IK ON)
		if err := miter.IterParallelByList(frames, allBlockSizes[i], func(data, index int) {
			frame := float32(data)
			vmdDeltas := delta.NewVmdDeltas(frame, originalModel.Bones, originalModel.Hash(), sizingMotion.Hash())
			vmdDeltas.Morphs = deform.DeformMorph(originalModel, sizingMotion.MorphFrames, frame, nil)
			vmdDeltas = deform.DeformBoneByPhysicsFlag(originalModel, sizingMotion, vmdDeltas, true, frame, fingerBoneNames, false)

			allVmdDeltas[i][index] = vmdDeltas
		}, func(iterIndex, allCount int) {
			mlog.I(mi18n.T("握り最適化01", map[string]interface{}{"No": sizingSet.Index + 1, "Direction": direction, "IterIndex": fmt.Sprintf("%02d", iterIndex), "AllCount": fmt.Sprintf("%02d", allCount)}))
		}); err != nil {
			sizingMotion.Processing = false
			return false, err
		}
	}

	for _, gripBone := range gripBones {
		// 握り拡散ボーンフレームの削除
		sizingMotion.BoneFrames.Delete(gripBone.Name())
	}

	// 指本体の角度を保持
	for i, fingerBoneNames := range finger_direction_bone_names {
		directionVmdDeltas := allVmdDeltas[i]
		for _, vmdDeltas := range directionVmdDeltas {
			for _, fingerBoneName := range fingerBoneNames {
				fingerQuat := getFixRotationForGrip(originalModel, vmdDeltas, fingerBoneName)
				if fingerQuat != nil {
					boneDelta := vmdDeltas.Bones.GetByName(fingerBoneName)
					bf := sizingMotion.BoneFrames.Get(fingerBoneName).Get(boneDelta.Frame)
					bf.Rotation = fingerQuat
					sizingMotion.InsertRegisteredBoneFrame(fingerBoneName, bf)
				}
			}
		}
	}

	// 中間キーフレのズレをチェック
	threshold := 0.01
	var wg sync.WaitGroup

	for i, direction := range directions {
		frames := allFrames[i]
		fingerBoneNames := finger_direction_bone_names[i]

		logEndFrame := 0
		allCount := frames[len(frames)-1]
		for j, endFrame := range frames {
			if j == 0 {
				continue
			}
			startFrame := frames[j-1] + 1

			if endFrame-startFrame-1 <= 0 {
				continue
			}

			if endFrame > logEndFrame {
				mlog.I(mi18n.T("握り最適化02", map[string]interface{}{"No": sizingSet.Index + 1, "Direction": direction, "IterIndex": fmt.Sprintf("%04d", endFrame), "AllCount": fmt.Sprintf("%04d", allCount)}))
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
					originalVmdDeltas = deform.DeformBoneByPhysicsFlag(originalModel, originalMotion, originalVmdDeltas, true, frame, fingerBoneNames, false)
				}()

				go func() {
					defer wg.Done()
					cleanVmdDeltas = delta.NewVmdDeltas(frame, originalModel.Bones, originalModel.Hash(), sizingMotion.Hash())
					cleanVmdDeltas.Morphs = deform.DeformMorph(originalModel, sizingMotion.MorphFrames, frame, nil)
					cleanVmdDeltas = deform.DeformBoneByPhysicsFlag(originalModel, sizingMotion, cleanVmdDeltas, true, frame, fingerBoneNames, false)
				}()

				wg.Wait()

				for _, boneName := range fingerBoneNames {
					bone := originalModel.Bones.GetByName(boneName)
					originalDelta := originalVmdDeltas.Bones.Get(bone.Index())
					cleanDelta := cleanVmdDeltas.Bones.Get(bone.Index())

					if originalDelta.FilledGlobalPosition().Distance(cleanDelta.FilledGlobalPosition()) > threshold {
						// ボーンの位置がずれている場合、キーを追加
						for _, bn := range fingerBoneNames {
							fingerQuat := getFixRotationForGrip(originalModel, originalVmdDeltas, bn)
							if fingerQuat != nil {
								bf := sizingMotion.BoneFrames.Get(bn).Get(frame)
								bf.Rotation = fingerQuat
								sizingMotion.InsertRegisteredBoneFrame(bn, bf)
							}
						}

						break
					}
				}
			}
		}
	}

	sizingSet.CompletedCleanGrip = true
	sizingMotion.Processing = false
	return true, nil
}

func getFixRotationForGrip(
	originalModel *pmx.PmxModel,
	vmdDeltas *delta.VmdDeltas,
	fingerBoneName string,
) *mmath.MQuaternion {
	fingerBone := originalModel.Bones.GetByName(fingerBoneName)
	if fingerBone.IsTail() {
		return nil
	}

	boneDelta := vmdDeltas.Bones.Get(fingerBone.Index())
	if boneDelta == nil {
		return nil
	}

	var fingerConfigParentBone *pmx.Bone
	for _, parentName := range fingerBone.Config().ParentBoneNames {
		if originalModel.Bones.ContainsByName(parentName.StringFromDirection(fingerBone.Direction())) {
			fingerConfigParentBone = originalModel.Bones.GetByName(parentName.StringFromDirection(fingerBone.Direction()))
			break
		}
	}
	if fingerConfigParentBone == nil {
		return nil
	}

	parentDelta := vmdDeltas.Bones.Get(fingerConfigParentBone.Index())
	return parentDelta.FilledGlobalMatrix().Inverted().Muled(boneDelta.FilledGlobalMatrix()).Quaternion()
}

func getGripBones(originalModel *pmx.PmxModel) []*pmx.Bone {
	gripBones := make([]*pmx.Bone, 0)
	gripBoneIndexes := make([]int, 0)

	for _, direction := range directions {
		wristBone := originalModel.Bones.GetByName(pmx.WRIST.StringFromDirection(direction))
		for _, boneName := range []string{pmx.THUMB_TAIL.StringFromDirection(direction),
			pmx.INDEX_TAIL.StringFromDirection(direction), pmx.MIDDLE_TAIL.StringFromDirection(direction),
			pmx.RING_TAIL.StringFromDirection(direction), pmx.PINKY_TAIL.StringFromDirection(direction)} {
			fingerTailBone := originalModel.Bones.GetByName(boneName)
			if fingerTailBone == nil {
				continue
			}
			for _, parentIndex := range fingerTailBone.Extend.ParentBoneIndexes {
				parentBone := originalModel.Bones.Get(parentIndex)
				if parentBone.Index() == wristBone.Index() {
					break
				}

				if parentBone.IsEffectorRotation() || parentBone.IsEffectorTranslation() {
					// 手首までのボーンで付与親である場合、握り拡散とみなす
					gripBones = append(gripBones, parentBone)
					gripBoneIndexes = append(gripBoneIndexes, parentBone.Index())

					if !slices.Contains(gripBoneIndexes, parentBone.EffectIndex) {
						// まだ付与親ボーンが追加されていない場合、付与親ボーンも追加
						gripBones = append(gripBones, originalModel.Bones.Get(parentBone.EffectIndex))
						gripBoneIndexes = append(gripBoneIndexes, parentBone.EffectIndex)
					}
				}
			}
		}
		for _, boneIndex := range wristBone.Extend.ChildBoneIndexes {
			bone := originalModel.Bones.Get(boneIndex)
			if len(bone.Extend.EffectiveBoneIndexes) > 0 {
				for _, effectiveBoneIndex := range bone.Extend.EffectiveBoneIndexes {
					effectiveBone := originalModel.Bones.Get(effectiveBoneIndex)
					if !slices.Contains(gripBoneIndexes, effectiveBone.Index()) {
						// 手首からのボーンで付与親が付いてるである場合、握り拡散とみなす
						gripBones = append(gripBones, effectiveBone)
						gripBoneIndexes = append(gripBoneIndexes, effectiveBone.Index())
					}
				}
				if !slices.Contains(gripBoneIndexes, bone.Index()) {
					// 手首からのボーンで付与親が付いてるである場合、握り拡散とみなす
					gripBones = append(gripBones, bone)
					gripBoneIndexes = append(gripBoneIndexes, bone.Index())
				}
			}
		}
	}

	return gripBones
}

func isValidCleanGrip(sizingSet *domain.SizingSet) bool {
	originalModel := sizingSet.OriginalPmx

	for _, direction := range directions {
		for _, boneName := range []string{pmx.THUMB1.StringFromDirection(direction), pmx.THUMB2.StringFromDirection(direction), pmx.INDEX1.StringFromDirection(direction), pmx.INDEX2.StringFromDirection(direction), pmx.INDEX3.StringFromDirection(direction),
			pmx.MIDDLE1.StringFromDirection(direction), pmx.MIDDLE2.StringFromDirection(direction), pmx.MIDDLE3.StringFromDirection(direction),
			pmx.RING1.StringFromDirection(direction), pmx.RING2.StringFromDirection(direction), pmx.RING3.StringFromDirection(direction),
			pmx.PINKY1.StringFromDirection(direction), pmx.PINKY2.StringFromDirection(direction), pmx.PINKY3.StringFromDirection(direction)} {

			if !originalModel.Bones.ContainsByName(boneName) {
				mlog.WT(mi18n.T("ボーン不足"), mi18n.T("握り拡散最適化ボーン不足", map[string]interface{}{
					"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": boneName}))
				return false
			}
		}
	}

	return true
}
