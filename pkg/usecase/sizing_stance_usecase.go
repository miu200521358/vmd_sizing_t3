package usecase

import (
	"slices"
	"sync"

	"github.com/miu200521358/mlib_go/pkg/domain/mmath"
	"github.com/miu200521358/mlib_go/pkg/domain/pmx"
	"github.com/miu200521358/mlib_go/pkg/domain/vmd"
	"github.com/miu200521358/vmd_sizing_t3/pkg/model"
)

func createStanceQuats(
	originalModel, sizingModel *pmx.PmxModel, isArmStance, isFingerStance bool,
) map[int][]*mmath.MMat4 {
	stanceQuats := make(map[int][]*mmath.MMat4)

	for _, direction := range []string{"左", "右"} {
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

func SizingStance(sizingSet *model.SizingSet) {
	originalModel := sizingSet.OriginalPmx
	originalMotion := sizingSet.OriginalVmd
	sizingModel := sizingSet.SizingPmx
	sizingMotion := sizingSet.OutputVmd

	if originalModel == nil || sizingModel == nil || originalMotion == nil || sizingMotion == nil {
		return
	}

	var scales *mmath.MVec3
	if sizingSet.IsSizingMove {
		if sizingModel.Bones.GetByName(pmx.LEG.Left()) == nil ||
			sizingModel.Bones.GetByName(pmx.KNEE.Left()) == nil ||
			sizingModel.Bones.GetByName(pmx.ANKLE.Left()) == nil ||
			sizingModel.Bones.GetByName(pmx.LEG_IK.Left()) == nil ||
			originalModel.Bones.GetByName(pmx.LEG.Left()) == nil ||
			originalModel.Bones.GetByName(pmx.KNEE.Left()) == nil ||
			originalModel.Bones.GetByName(pmx.ANKLE.Left()) == nil ||
			originalModel.Bones.GetByName(pmx.LEG_IK.Left()) == nil {

			if sizingModel.Bones.GetByName(pmx.UPPER.String()) == nil ||
				originalModel.Bones.GetByName(pmx.UPPER.String()) == nil {
				scales = mmath.MVec3One
			} else {
				// 足が無い場合、上半身までの長さで比較する
				// 上半身の長さ比率
				upperBodyLengthRatio := sizingModel.Bones.GetByName(pmx.UPPER.String()).Position.Y /
					originalModel.Bones.GetByName(pmx.UPPER.String()).Position.Y
				scales = &mmath.MVec3{X: upperBodyLengthRatio, Y: upperBodyLengthRatio, Z: upperBodyLengthRatio}
			}
		} else {
			// 足の長さ比率(XZ)
			legLengthRatio := (sizingModel.Bones.GetByName(pmx.LEG.Left()).Position.Distance(
				sizingModel.Bones.GetByName(pmx.KNEE.Left()).Position) +
				sizingModel.Bones.GetByName(pmx.KNEE.Left()).Position.Distance(
					sizingModel.Bones.GetByName(pmx.ANKLE.Left()).Position)) /
				(originalModel.Bones.GetByName(pmx.LEG.Left()).Position.Distance(
					originalModel.Bones.GetByName(pmx.KNEE.Left()).Position) +
					originalModel.Bones.GetByName(pmx.KNEE.Left()).Position.Distance(
						originalModel.Bones.GetByName(pmx.ANKLE.Left()).Position))
			// 足の長さ比率(Y)
			legHeightRatio := sizingModel.Bones.GetByName(pmx.LEG.Left()).Position.Distance(
				sizingModel.Bones.GetByName(pmx.LEG_IK.Left()).Position) /
				originalModel.Bones.GetByName(pmx.LEG.Left()).Position.Distance(
					originalModel.Bones.GetByName(pmx.LEG_IK.Left()).Position)

			scales = &mmath.MVec3{X: legLengthRatio, Y: legHeightRatio, Z: legLengthRatio}
		}
		// mlog.I("legHeightRatio: %.5f", legLengthRatio)
	}

	stanceQuats := createStanceQuats(originalModel, sizingModel, sizingSet.IsSizingArmStance, sizingSet.IsSizingFingerStance)

	var wg sync.WaitGroup
	for _, boneName := range originalMotion.BoneFrames.Names() {
		wg.Add(1)

		go func(originalBfs, sizingBfs *vmd.BoneNameFrames) {
			defer wg.Done()
			for _, frame := range originalBfs.Indexes.List() {
				originalBf := originalMotion.BoneFrames.Get(boneName).Get(frame)
				sizingBf := sizingBfs.Get(frame)
				if originalBf == nil || sizingBf == nil {
					continue
				}

				// 移動補正
				if scales != nil && slices.Contains(move_bone_names, originalBfs.Name) {
					sizingBf.Position = originalBf.Position.Muled(scales)
					sizingBfs.Update(sizingBf)
				}

				// 回転補正
				bone := sizingModel.Bones.GetByName(boneName)
				if bone != nil {
					if _, ok := stanceQuats[bone.Index()]; ok {
						sizingBf.Rotation = stanceQuats[bone.Index()][0].Muled(originalBf.Rotation.ToMat4()).Muled(stanceQuats[bone.Index()][1]).Quaternion()
						sizingBfs.Update(sizingBf)
					} else if bone.IsTwist() {
						// 捩系は軸に合わせて回転を修正する
						sizingBf.Rotation = originalBf.Rotation.ToFixedAxisRotation(bone.Extend.NormalizedFixedAxis)
						sizingBfs.Update(sizingBf)
					}
				}
			}
		}(originalMotion.BoneFrames.Get(boneName), sizingMotion.BoneFrames.Get(boneName))
	}

	wg.Wait()

	sizingSet.CompletedSizingMove = true
	sizingSet.CompletedSizingArmStance = true
	sizingSet.CompletedSizingFingerStance = true
}
