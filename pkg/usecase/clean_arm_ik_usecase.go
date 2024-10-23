package usecase

import (
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

func CleanArmIk(sizingSet *domain.SizingSet, setSize int) (bool, error) {
	if !sizingSet.IsCleanArmIk || (sizingSet.IsCleanArmIk && sizingSet.CompletedCleanArmIk) {
		return false, nil
	}

	if !isValidCleanArmIk(sizingSet) {
		return false, nil
	}

	originalModel := sizingSet.OriginalPmx
	originalMotion := sizingSet.OriginalVmd
	sizingMotion := sizingSet.OutputVmd

	// 腕IKに相当するボーンがあるか取得
	armIkLeftBone, armIkRightBone := getArmIkBones(originalModel)

	if armIkLeftBone == nil && armIkRightBone == nil {
		return false, nil
	}

	if !(sizingMotion.BoneFrames.ContainsActive(armIkLeftBone.Name()) ||
		sizingMotion.BoneFrames.ContainsActive(armIkRightBone.Name())) {
		return false, nil
	}

	mlog.I(mi18n.T("腕IK最適化開始", map[string]interface{}{"No": sizingSet.Index + 1,
		"LeftBoneName": armIkLeftBone.Name(), "RightBoneName": armIkRightBone.Name()}))

	allFrames := make([][]int, 2)
	allRelativeBoneNames := make([][]string, 2)
	allBlockSizes := make([]int, 2)
	armRotations := make([][]*mmath.MQuaternion, originalModel.Bones.Len())

	for i, direction := range directions {
		var armIkBone *pmx.Bone
		switch direction {
		case "左":
			armIkBone = armIkLeftBone
		case "右":
			armIkBone = armIkRightBone
		}
		shoulderRootBone := originalModel.Bones.GetByName(pmx.SHOULDER_ROOT.StringFromDirection(direction))
		wristBone := originalModel.Bones.GetByName(pmx.WRIST.StringFromDirection(direction))

		relativeBoneNames := make([]string, 0)
		relativeBoneNames = append(relativeBoneNames, armIkBone.Name())
		relativeBoneNames = append(relativeBoneNames, wristBone.Name())
		shoulderRootBoneLayerIndex := slices.Index(originalModel.Bones.LayerSortedIndexes, shoulderRootBone.Index())
		for _, boneIndex := range wristBone.Extend.RelativeBoneIndexes {
			bone := originalModel.Bones.Get(boneIndex)
			boneLayerIndex := slices.Index(originalModel.Bones.LayerSortedIndexes, bone.Index())
			if bone != nil && boneLayerIndex > shoulderRootBoneLayerIndex {
				// 肩根元からの子のみ対象とする
				relativeBoneNames = append(relativeBoneNames, bone.Name())
			}
		}
		relativeBoneNames = append(relativeBoneNames, pmx.MIDDLE1.StringFromDirection(direction))
		frames := sizingMotion.BoneFrames.RegisteredFrames(relativeBoneNames)
		allBlockSizes[i], _ = miter.GetBlockSize(len(frames) * setSize)

		allFrames[i] = frames
		allRelativeBoneNames[i] = relativeBoneNames

		// 元モデルのデフォーム(IK ON)
		if err := miter.IterParallelByList(frames, allBlockSizes[i], func(data, index int) {
			frame := float32(data)
			vmdDeltas := delta.NewVmdDeltas(frame, originalModel.Bones, originalModel.Hash(), sizingMotion.Hash())
			vmdDeltas.Morphs = deform.DeformMorph(originalModel, sizingMotion.MorphFrames, frame, nil)
			vmdDeltas = deform.DeformBoneByPhysicsFlag(originalModel, sizingMotion, vmdDeltas, true, frame, relativeBoneNames, false)

			for _, boneDelta := range vmdDeltas.Bones.Data {
				quat := getFixRotationForArmIk(vmdDeltas, armIkBone, boneDelta)
				if quat != nil {
					if armRotations[boneDelta.Bone.Index()] == nil {
						armRotations[boneDelta.Bone.Index()] = make([]*mmath.MQuaternion, len(frames))
					}
					armRotations[boneDelta.Bone.Index()][index] = quat
				}
			}
		}, func(iterIndex, allCount int) {
			mlog.I(mi18n.T("腕IK最適化01", map[string]interface{}{"No": sizingSet.Index + 1, "BoneName": armIkBone.Name(), "IterIndex": iterIndex, "AllCount": allCount}))
		}); err != nil {
			return false, err
		}
	}

	// IK関連のボーンを削除
	for _, direction := range directions {
		var armIkBone *pmx.Bone
		switch direction {
		case "左":
			armIkBone = armIkLeftBone
		case "右":
			armIkBone = armIkRightBone
		}

		sizingMotion.BoneFrames.Delete(armIkBone.Name())
		sizingMotion.BoneFrames.Delete(originalModel.Bones.Get(armIkBone.Ik.BoneIndex).Name())
		for _, ikLink := range armIkBone.Ik.Links {
			sizingMotion.BoneFrames.Delete(originalModel.Bones.Get(ikLink.BoneIndex).Name())
		}
	}

	for i, rotations := range armRotations {
		for j, rot := range rotations {
			if rot == nil {
				continue
			}

			bone := originalModel.Bones.Get(i)
			var frame float32
			if bone.Direction() == "左" {
				frame = float32(allFrames[0][j])
			} else {
				frame = float32(allFrames[1][j])
			}

			bf := sizingMotion.BoneFrames.Get(bone.Name()).Get(frame)
			bf.Rotation = rot
			sizingMotion.InsertRegisteredBoneFrame(bone.Name(), bf)
		}
	}

	// 中間キーフレのズレをチェック
	threshold := 0.01
	errorChan := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)

	for i, direction := range directions {
		go func(i int, direction string) {
			defer func() {
				wg.Done()
				errorChan <- miter.GetError()
			}()

			var armIkBone *pmx.Bone
			switch direction {
			case "左":
				armIkBone = armIkLeftBone
			case "右":
				armIkBone = armIkRightBone
			}

			frames := allFrames[i]
			relativeBoneNames := allRelativeBoneNames[i]
			relativeArmBones := make([]*pmx.Bone, 0)
			for _, boneName := range relativeBoneNames {
				if originalModel.Bones.GetByName(boneName).IsArm() {
					relativeArmBones = append(relativeArmBones, originalModel.Bones.GetByName(boneName))
				}
			}

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
					mlog.I(mi18n.T("腕IK最適化02", map[string]interface{}{"No": sizingSet.Index + 1, "BoneName": armIkBone.Name(), "IterIndex": endFrame, "AllCount": allCount}))
					logEndFrame += 1000
				}

				for iFrame := startFrame + 1; iFrame < endFrame; iFrame++ {
					frame := float32(iFrame)

					originalVmdDeltas := delta.NewVmdDeltas(frame, originalModel.Bones, originalModel.Hash(), originalMotion.Hash())
					originalVmdDeltas.Morphs = deform.DeformMorph(originalModel, originalMotion.MorphFrames, frame, nil)
					originalVmdDeltas = deform.DeformBoneByPhysicsFlag(originalModel, originalMotion, originalVmdDeltas, true, frame, relativeBoneNames, false)

					cleanVmdDeltas := delta.NewVmdDeltas(frame, originalModel.Bones, originalModel.Hash(), sizingMotion.Hash())
					cleanVmdDeltas.Morphs = deform.DeformMorph(originalModel, sizingMotion.MorphFrames, frame, nil)
					cleanVmdDeltas = deform.DeformBoneByPhysicsFlag(originalModel, sizingMotion, cleanVmdDeltas, true, frame, relativeBoneNames, false)

					for _, bone := range relativeArmBones {
						originalDelta := originalVmdDeltas.Bones.Get(bone.Index())
						cleanDelta := cleanVmdDeltas.Bones.Get(bone.Index())

						if originalDelta.FilledGlobalPosition().Distance(cleanDelta.FilledGlobalPosition()) > threshold {
							// ボーンの位置がずれている場合、キーを追加
							for _, b := range relativeArmBones {
								quat := getFixRotationForArmIk(originalVmdDeltas, armIkBone, originalVmdDeltas.Bones.Get(b.Index()))
								if quat != nil {
									bf := sizingMotion.BoneFrames.Get(b.Name()).Get(frame)
									bf.Rotation = quat
									sizingMotion.InsertRegisteredBoneFrame(b.Name(), bf)
								}
							}
							break
						}
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

	sizingSet.CompletedCleanArmIk = true
	return true, nil
}

func getFixRotationForArmIk(
	vmdDeltas *delta.VmdDeltas,
	armIkBone *pmx.Bone,
	boneDelta *delta.BoneDelta,
) *mmath.MQuaternion {
	if boneDelta == nil {
		return nil
	}
	if !boneDelta.Bone.IsArm() {
		// 腕系ボーンのみ対象とする
		return nil
	}

	if boneDelta.Bone.Name() == pmx.WRIST.Left() || boneDelta.Bone.Name() == pmx.WRIST.Right() {
		armIkTargetDelta := vmdDeltas.Bones.Get(armIkBone.Ik.BoneIndex)
		parentQuat := mmath.NewMQuaternion()
		for _, parentIndex := range boneDelta.Bone.Extend.ParentBoneIndexes {
			parentDelta := vmdDeltas.Bones.Get(parentIndex)
			if parentDelta.Bone.Index() == armIkBone.Index() {
				break
			}
			parentQuat = parentQuat.Muled(parentDelta.FilledFrameRotation())
		}
		return parentQuat.Inverted().ToMat4().Muled(armIkTargetDelta.FilledGlobalMatrix().Inverted()).Muled(boneDelta.FilledGlobalMatrix()).Quaternion()
	}

	return boneDelta.FilledFrameRotation()
}

func isValidCleanArmIk(sizingSet *domain.SizingSet) bool {
	originalModel := sizingSet.OriginalPmx

	for _, direction := range directions {
		if !originalModel.Bones.ContainsByName(pmx.ARM.StringFromDirection(direction)) {
			mlog.WT(mi18n.T("ボーン不足"), mi18n.T("腕IK最適化ボーン不足", map[string]interface{}{
				"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.ARM.StringFromDirection(direction)}))
			return false
		}

		if !originalModel.Bones.ContainsByName(pmx.ELBOW.StringFromDirection(direction)) {
			mlog.WT(mi18n.T("ボーン不足"), mi18n.T("腕IK最適化ボーン不足", map[string]interface{}{
				"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.ELBOW.StringFromDirection(direction)}))
			return false
		}

		if !originalModel.Bones.ContainsByName(pmx.WRIST.StringFromDirection(direction)) {
			mlog.WT(mi18n.T("ボーン不足"), mi18n.T("腕IK最適化ボーン不足", map[string]interface{}{
				"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.WRIST.StringFromDirection(direction)}))
			return false
		}
	}

	return true
}

func getArmIkBones(model *pmx.PmxModel) (armIkLeftBone, armIkRightBone *pmx.Bone) {
	for _, direction := range directions {
		var armIkBone *pmx.Bone

		for _, standardBoneName := range []pmx.StandardBoneNames{
			pmx.ARM, pmx.ARM_TWIST, pmx.ELBOW, pmx.WRIST_TWIST, pmx.WRIST} {
			// 腕・腕捩・ひじ・手捩・手首のいずれかのボーンがリンクもしくはターゲットになっているボーン
			bone := model.Bones.GetByName(standardBoneName.StringFromDirection(direction))

			for _, boneIndex := range bone.Extend.IkTargetBoneIndexes {
				armIkBone = model.Bones.Get(boneIndex)
				if armIkBone != nil && armIkBone.IsIK() {
					break
				}
			}

			if armIkBone != nil {
				switch direction {
				case "左":
					armIkLeftBone = armIkBone
				case "右":
					armIkRightBone = armIkBone
				}
				break
			}

			for _, boneIndex := range bone.Extend.IkLinkBoneIndexes {
				armIkBone = model.Bones.Get(boneIndex)
				if armIkBone != nil && armIkBone.IsIK() {
					break
				}
			}

			if armIkBone != nil {
				switch direction {
				case "左":
					armIkLeftBone = armIkBone
				case "右":
					armIkRightBone = armIkBone
				}
				break
			}
		}
	}

	return armIkLeftBone, armIkRightBone
}
