package usecase

import (
	"sync"

	"github.com/miu200521358/mlib_go/pkg/domain/mmath"
	"github.com/miu200521358/mlib_go/pkg/domain/pmx"
	"github.com/miu200521358/mlib_go/pkg/domain/vmd"
	"github.com/miu200521358/mlib_go/pkg/mutils/mi18n"
	"github.com/miu200521358/mlib_go/pkg/mutils/mlog"
	"github.com/miu200521358/vmd_sizing_t3/pkg/domain"
)

func SizingArmFingerStance(sizingSet *domain.SizingSet, setSize int) (bool, error) {
	originalModel := sizingSet.OriginalPmx
	sizingModel := sizingSet.SizingPmx
	sizingMotion := sizingSet.OutputVmd

	if (!sizingSet.IsSizingArmStance || (sizingSet.IsSizingArmStance && sizingSet.CompletedSizingArmStance)) &&
		(!sizingSet.IsSizingFingerStance || (sizingSet.IsSizingFingerStance && sizingSet.CompletedSizingFingerStance)) {
		return false, nil
	}

	if !isValidSizingArm(sizingSet) {
		return false, nil
	}

	mlog.I(mi18n.T("腕指スタンス補正開始", map[string]interface{}{"No": sizingSet.Index + 1}))
	sizingMotion.Processing = true

	stanceQuats := createArmFingerStanceQuats(
		originalModel, sizingModel, sizingSet.IsSizingArmStance, sizingSet.IsSizingFingerStance)

	var wg sync.WaitGroup
	for i, boneNames := range [][]string{all_arm_bone_names, all_finger_bone_names} {
		if i == 0 && (!sizingSet.IsSizingArmStance ||
			(sizingSet.IsSizingArmStance && sizingSet.CompletedSizingArmStance)) {
			continue
		}
		if i == 1 && (!sizingSet.IsSizingFingerStance ||
			(sizingSet.IsSizingFingerStance && sizingSet.CompletedSizingFingerStance)) {
			continue
		}

		for _, boneName := range boneNames {
			wg.Add(1)

			go func(sizingBfs *vmd.BoneNameFrames) {
				defer wg.Done()
				for _, frame := range sizingBfs.Indexes.List() {
					sizingBf := sizingBfs.Get(frame)
					if sizingBf == nil {
						continue
					}

					// 回転補正
					bone := sizingModel.Bones.GetByName(boneName)
					if bone != nil {
						if _, ok := stanceQuats[bone.Index()]; ok {
							sizingRotation := sizingBf.Rotation
							if sizingRotation == nil {
								sizingRotation = mmath.MQuaternionIdent
							}
							sizingBf.Rotation = stanceQuats[bone.Index()][0].Muled(sizingRotation.ToMat4()).Muled(stanceQuats[bone.Index()][1]).Quaternion()
							sizingBfs.Update(sizingBf)
						}
					}
				}
			}(sizingMotion.BoneFrames.Get(boneName))
		}
	}

	wg.Wait()

	// 腕スタンス補正だけしているときとかあるので、Completeは補正対象のフラグを受け継ぐ
	sizingSet.CompletedSizingArmStance = sizingSet.IsSizingArmStance
	sizingSet.CompletedSizingFingerStance = sizingSet.IsSizingFingerStance
	sizingMotion.Processing = false

	return true, nil
}

func createArmFingerStanceQuats(
	originalModel, sizingModel *pmx.PmxModel, isArmStance, isFingerStance bool,
) map[int][]*mmath.MMat4 {
	stanceQuats := make(map[int][]*mmath.MMat4)

	for _, direction := range directions {
		stanceBoneNames := make([][]string, 0)

		if isArmStance {
			// 腕スタンス補正対象
			stanceBoneNames = append(stanceBoneNames, []string{"", pmx.ARM.StringFromDirection(direction)})
			stanceBoneNames = append(stanceBoneNames,
				[]string{pmx.ARM.StringFromDirection(direction), pmx.ELBOW.StringFromDirection(direction)})
			stanceBoneNames = append(stanceBoneNames,
				[]string{pmx.ELBOW.StringFromDirection(direction), pmx.WRIST.StringFromDirection(direction)})
		}

		if isFingerStance {
			// 指スタンス補正対象
			stanceBoneNames = append(stanceBoneNames, []string{
				pmx.WRIST.StringFromDirection(direction), pmx.THUMB1.StringFromDirection(direction)})
			stanceBoneNames = append(stanceBoneNames,
				[]string{pmx.THUMB1.StringFromDirection(direction), pmx.THUMB2.StringFromDirection(direction)})
			stanceBoneNames = append(stanceBoneNames, []string{
				pmx.WRIST.StringFromDirection(direction), pmx.INDEX1.StringFromDirection(direction)})
			stanceBoneNames = append(stanceBoneNames,
				[]string{pmx.INDEX1.StringFromDirection(direction), pmx.INDEX2.StringFromDirection(direction)})
			stanceBoneNames = append(stanceBoneNames,
				[]string{pmx.INDEX2.StringFromDirection(direction), pmx.INDEX3.StringFromDirection(direction)})
			stanceBoneNames = append(stanceBoneNames, []string{
				pmx.WRIST.StringFromDirection(direction), pmx.MIDDLE1.StringFromDirection(direction)})
			stanceBoneNames = append(stanceBoneNames,
				[]string{pmx.MIDDLE1.StringFromDirection(direction), pmx.MIDDLE2.StringFromDirection(direction)})
			stanceBoneNames = append(stanceBoneNames,
				[]string{pmx.MIDDLE2.StringFromDirection(direction), pmx.MIDDLE3.StringFromDirection(direction)})
			stanceBoneNames = append(stanceBoneNames, []string{
				pmx.WRIST.StringFromDirection(direction), pmx.RING1.StringFromDirection(direction)})
			stanceBoneNames = append(stanceBoneNames,
				[]string{pmx.RING1.StringFromDirection(direction), pmx.RING2.StringFromDirection(direction)})
			stanceBoneNames = append(stanceBoneNames,
				[]string{pmx.RING2.StringFromDirection(direction), pmx.RING3.StringFromDirection(direction)})
			stanceBoneNames = append(stanceBoneNames, []string{
				pmx.WRIST.StringFromDirection(direction), pmx.PINKY1.StringFromDirection(direction)})
			stanceBoneNames = append(stanceBoneNames,
				[]string{pmx.PINKY1.StringFromDirection(direction), pmx.PINKY2.StringFromDirection(direction)})
		}

		for _, boneNames := range stanceBoneNames {
			fromBoneName := boneNames[0]
			targetBoneName := boneNames[1]

			var sizingFromBone *pmx.Bone
			if fromBoneName != "" && sizingModel.Bones.ContainsByName(fromBoneName) {
				sizingFromBone = sizingModel.Bones.GetByName(fromBoneName)
			}
			var originalTargetBone, sizingTargetBone *pmx.Bone
			if targetBoneName != "" && originalModel.Bones.ContainsByName(targetBoneName) &&
				sizingModel.Bones.ContainsByName(targetBoneName) {
				originalTargetBone = originalModel.Bones.GetByName(targetBoneName)
				sizingTargetBone = sizingModel.Bones.GetByName(targetBoneName)
			}

			if originalTargetBone == nil || sizingTargetBone == nil {
				continue
			}

			if _, ok := stanceQuats[sizingTargetBone.Index()]; !ok {
				stanceQuats[sizingTargetBone.Index()] = make([]*mmath.MMat4, 2)
			}

			if sizingFromBone != nil {
				if _, ok := stanceQuats[sizingFromBone.Index()]; ok {
					stanceQuats[sizingTargetBone.Index()][0] = stanceQuats[sizingFromBone.Index()][1].Inverted()
				} else {
					stanceQuats[sizingTargetBone.Index()][0] = mmath.NewMMat4()
				}
			} else {
				stanceQuats[sizingTargetBone.Index()][0] = mmath.NewMMat4()
			}

			// 元モデルのボーン傾き
			originalDirection := originalTargetBone.Extend.NormalizedLocalAxisX
			originalSlopeMat := originalDirection.ToLocalMat()
			// サイジング先モデルのボーン傾き
			sizingBoneDirection := sizingTargetBone.Extend.NormalizedLocalAxisX
			sizingSlopeMat := sizingBoneDirection.ToLocalMat()
			// 傾き補正
			offsetQuat := sizingSlopeMat.Muled(originalSlopeMat.Inverted()).Inverted().Quaternion()

			if offsetQuat.IsIdent() {
				stanceQuats[sizingTargetBone.Index()][1] = mmath.NewMMat4()
			} else {
				_, yzOffsetQuat := offsetQuat.SeparateTwistByAxis(sizingBoneDirection)
				stanceQuats[sizingTargetBone.Index()][1] = yzOffsetQuat.ToMat4()
			}
		}
	}

	return stanceQuats
}

func isValidSizingArm(sizingSet *domain.SizingSet) bool {
	originalModel := sizingSet.OriginalPmx
	sizingModel := sizingSet.SizingPmx

	if !originalModel.Bones.ContainsByName(pmx.SHOULDER.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("腕スタンスボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.SHOULDER.Left()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.ARM.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("腕スタンスボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.ARM.Left()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.ELBOW.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("腕スタンスボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.ELBOW.Left()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.WRIST.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("腕スタンスボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.WRIST.Left()}))
		return false
	}

	// ------------------------------

	if !originalModel.Bones.ContainsByName(pmx.SHOULDER.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("腕スタンスボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.SHOULDER.Right()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.ARM.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("腕スタンスボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.ARM.Right()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.ELBOW.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("腕スタンスボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.ELBOW.Right()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.WRIST.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("腕スタンスボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.WRIST.Right()}))
		return false
	}

	// ------------------------------

	if !sizingModel.Bones.ContainsByName(pmx.SHOULDER.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("腕スタンスボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.SHOULDER.Left()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.ARM.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("腕スタンスボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.ARM.Left()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.ELBOW.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("腕スタンスボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.ELBOW.Left()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.WRIST.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("腕スタンスボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.WRIST.Left()}))
		return false
	}

	// ------------------------------

	if !sizingModel.Bones.ContainsByName(pmx.SHOULDER.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("腕スタンスボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.SHOULDER.Right()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.ARM.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("腕スタンスボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.ARM.Right()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.ELBOW.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("腕スタンスボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.ELBOW.Right()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.WRIST.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("腕スタンスボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.WRIST.Right()}))
		return false
	}

	return true
}
