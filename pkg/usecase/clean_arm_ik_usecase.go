package usecase

import (
	"slices"

	"github.com/miu200521358/mlib_go/pkg/domain/delta"
	"github.com/miu200521358/mlib_go/pkg/domain/miter"
	"github.com/miu200521358/mlib_go/pkg/domain/pmx"
	"github.com/miu200521358/mlib_go/pkg/infrastructure/deform"
	"github.com/miu200521358/mlib_go/pkg/mutils/mi18n"
	"github.com/miu200521358/mlib_go/pkg/mutils/mlog"
	"github.com/miu200521358/vmd_sizing_t3/pkg/domain"
)

func CleanArmIk(sizingSet *domain.SizingSet) {
	if !sizingSet.IsCleanArmIk || (sizingSet.IsCleanArmIk && sizingSet.CompletedCleanArmIk) {
		return
	}

	if !isValidCleanArmIk(sizingSet) {
		return
	}

	originalModel := sizingSet.OriginalPmx
	// originalMotion := sizingSet.OriginalVmd
	sizingMotion := sizingSet.OutputVmd

	neckRootBone := originalModel.Bones.GetByName(pmx.NECK.String())

	// 腕IKに相当するボーンがあるか取得
	armIkLeftBone, armIkRightBone := getArmIkBones(originalModel)

	if armIkLeftBone == nil && armIkRightBone == nil {
		return
	}

	if !(sizingMotion.BoneFrames.ContainsActive(armIkLeftBone.Name()) ||
		sizingMotion.BoneFrames.ContainsActive(armIkRightBone.Name())) {
		return
	}

	mlog.I(mi18n.T("腕IK最適化開始", map[string]interface{}{"No": sizingSet.Index + 1,
		"LeftBoneName": armIkLeftBone.Name(), "RightBoneName": armIkRightBone.Name()}))

	mlog.I(mi18n.T("腕IK最適化01", map[string]interface{}{"No": sizingSet.Index + 1}))

	allVmdDeltas := make([][]*delta.VmdDeltas, 2)
	allRelativeBoneNames := make([][]string, 2)

	for i, direction := range []string{"左", "右"} {
		var armIkBone *pmx.Bone
		switch direction {
		case "左":
			armIkBone = armIkLeftBone
		case "右":
			armIkBone = armIkRightBone
		}

		relativeBoneNames := make([]string, 0)
		for _, boneIndex := range armIkBone.Extend.RelativeBoneIndexes {
			bone := originalModel.Bones.Get(boneIndex)
			if bone != nil {
				relativeBoneNames = append(relativeBoneNames, bone.Name())
			}
		}
		frames := sizingMotion.BoneFrames.RegisteredFrames(relativeBoneNames)

		allVmdDeltas[i] = make([]*delta.VmdDeltas, len(frames))
		allRelativeBoneNames[i] = relativeBoneNames

		// 元モデルのデフォーム(IK ON)
		miter.IterParallelByList(frames, 500, func(data, index int) {
			frame := float32(data)
			vmdDeltas := delta.NewVmdDeltas(frame, originalModel.Bones, originalModel.Hash(), sizingMotion.Hash())
			vmdDeltas.Morphs = deform.DeformMorph(originalModel, sizingMotion.MorphFrames, frame, nil)
			vmdDeltas = deform.DeformBoneByPhysicsFlag(originalModel, sizingMotion, vmdDeltas, true, frame, relativeBoneNames, false)

			allVmdDeltas[i][index] = vmdDeltas
		})
	}

	for i, directionVmdDeltas := range allVmdDeltas {
		for _, vmdDeltas := range directionVmdDeltas {
			for _, boneDelta := range vmdDeltas.Bones.Data {
				if boneDelta == nil {
					continue
				}
				if !slices.Contains(boneDelta.Bone.Extend.ParentBoneIndexes, neckRootBone.Index()) {
					// 首根元の子のみ対象とする
					continue
				}
				if !boneDelta.Bone.IsStandard() {
					// 準標準までを対象とする
					continue
				}

				bf := sizingMotion.BoneFrames.Get(boneDelta.Bone.Name()).Get(boneDelta.Frame)
				bf.Rotation = boneDelta.FilledFrameRotation()
				sizingMotion.InsertRegisteredBoneFrame(boneDelta.Bone.Name(), bf)
			}
		}

		for _, relativeBoneName := range allRelativeBoneNames[i] {
			bone := originalModel.Bones.GetByName(relativeBoneName)
			if bone == nil {
				continue
			}
			if !slices.Contains(bone.Extend.ParentBoneIndexes, neckRootBone.Index()) {
				// 首根元の子のみ対象とする
				continue
			}
			if !bone.IsStandard() {
				// 準標準ではないボーンのキーフレを削除する
				sizingMotion.BoneFrames.Delete(bone.Name())
			}
		}
	}

	mlog.I(mi18n.T("腕IK最適化02", map[string]interface{}{"No": sizingSet.Index + 1}))

	// // 中間キーフレのズレをチェック
	// threshold := 0.02

	// for i, endFrame := range frames {
	// 	if i == 0 {
	// 		continue
	// 	}
	// 	startFrame := frames[i-1] + 1

	// 	if endFrame-startFrame-1 <= 0 {
	// 		continue
	// 	}

	// 	miter.IterParallelByCount(endFrame-startFrame-1, 500, func(index int) {
	// 		frame := float32(startFrame + index + 1)

	// 		wg.Add(2)
	// 		var originalVmdDeltas, cleanVmdDeltas *delta.VmdDeltas

	// 		go func() {
	// 			defer wg.Done()
	// 			originalVmdDeltas = delta.NewVmdDeltas(frame, originalModel.Bones, originalModel.Hash(), originalMotion.Hash())
	// 			originalVmdDeltas.Morphs = deform.DeformMorph(originalModel, originalMotion.MorphFrames, frame, nil)
	// 			originalVmdDeltas = deform.DeformBoneByPhysicsFlag(originalModel, originalMotion, originalVmdDeltas, false, frame, centerRelativeBoneNames, false)
	// 		}()

	// 		go func() {
	// 			defer wg.Done()
	// 			cleanVmdDeltas = delta.NewVmdDeltas(frame, originalModel.Bones, originalModel.Hash(), sizingMotion.Hash())
	// 			cleanVmdDeltas.Morphs = deform.DeformMorph(originalModel, sizingMotion.MorphFrames, frame, nil)
	// 			cleanVmdDeltas = deform.DeformBoneByPhysicsFlag(originalModel, sizingMotion, cleanVmdDeltas, false, frame, centerRelativeBoneNames, false)
	// 		}()

	// 		wg.Wait()

	// 		bone := originalModel.Bones.GetByName(pmx.UPPER.String())
	// 		originalDelta := originalVmdDeltas.Bones.Get(bone.Index())
	// 		cleanDelta := cleanVmdDeltas.Bones.Get(bone.Index())

	// 		if originalDelta.FilledGlobalPosition().Distance(cleanDelta.FilledGlobalPosition()) > threshold {
	// 			// ボーンの位置がずれている場合、キーを追加
	// 			localPosition := originalDelta.FilledGlobalPosition().Subed(bone.Position)

	// 			{
	// 				bf := sizingMotion.BoneFrames.Get(pmx.CENTER.String()).Get(frame)
	// 				bf.Position = &mmath.MVec3{X: localPosition.X, Y: 0, Z: localPosition.Z}
	// 				sizingMotion.InsertRegisteredBoneFrame(pmx.CENTER.String(), bf)
	// 			}
	// 			{
	// 				bf := sizingMotion.BoneFrames.Get(pmx.GROOVE.String()).Get(frame)
	// 				bf.Position = &mmath.MVec3{X: 0, Y: localPosition.Y, Z: 0}
	// 				sizingMotion.InsertRegisteredBoneFrame(pmx.GROOVE.String(), bf)
	// 			}
	// 		}
	// 	})
	// }

	sizingSet.CompletedCleanArmIk = true
}

func isValidCleanArmIk(sizingSet *domain.SizingSet) bool {
	originalModel := sizingSet.OriginalPmx

	for _, direction := range []string{"左", "右"} {
		if !originalModel.Bones.ContainsByName(pmx.ARM.StringFromDirection(direction)) {
			mlog.WT(mi18n.T("ボーン不足"), mi18n.T("腕IK最適化ボーン不足", map[string]interface{}{
				"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.ARM.StringFromDirection(direction)}))
			return false
		}

		if !originalModel.Bones.ContainsByName(pmx.ELBOW.StringFromDirection(direction)) {
			mlog.WT(mi18n.T("ボーン不足"), mi18n.T("腕IK最適化ボーン不足", map[string]interface{}{
				"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.ARM.StringFromDirection(direction)}))
			return false
		}

		if !originalModel.Bones.ContainsByName(pmx.WRIST.StringFromDirection(direction)) {
			mlog.WT(mi18n.T("ボーン不足"), mi18n.T("腕IK最適化ボーン不足", map[string]interface{}{
				"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.ARM.StringFromDirection(direction)}))
			return false
		}
	}

	return true
}

func getArmIkBones(model *pmx.PmxModel) (armIkLeftBone, armIkRightBone *pmx.Bone) {
	for _, direction := range []string{"左", "右"} {
		var armIkBone *pmx.Bone

		for _, standardBoneName := range []pmx.StandardBoneNames{pmx.ARM, pmx.ELBOW, pmx.WRIST} {
			// 腕・ひじ・手首のいずれかのボーンがリンクもしくはターゲットになっているボーン
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
