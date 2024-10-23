package usecase

import (
	"fmt"
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

func SizingShoulder(sizingSet *domain.SizingSet, setSize int) (bool, error) {
	if !sizingSet.IsSizingShoulder || (sizingSet.IsSizingShoulder && sizingSet.CompletedSizingShoulder) {
		return false, nil
	}

	if !isValidSizingShoulder(sizingSet) {
		return false, nil
	}

	originalModel := sizingSet.OriginalPmx
	originalMotion := sizingSet.OriginalVmd
	sizingModel := sizingSet.SizingPmx
	sizingMotion := sizingSet.OutputVmd

	armScales := make([]float64, 2)
	shoulderIkBones := make([]*pmx.Bone, 2)

	mlog.I(mi18n.T("肩補正開始", map[string]interface{}{"No": sizingSet.Index + 1}))

	for i, direction := range directions {
		originalNeckRootBone := originalModel.Bones.GetByName(pmx.NECK_ROOT.String())
		originalArmBone := originalModel.Bones.GetByName(pmx.ARM.StringFromDirection(direction))

		sizingNeckRootBone := sizingModel.Bones.GetByName(pmx.NECK_ROOT.String())
		sizingShoulderBone := sizingModel.Bones.GetByName(pmx.SHOULDER.StringFromDirection(direction))
		sizingArmBone := sizingModel.Bones.GetByName(pmx.ARM.StringFromDirection(direction))

		// 肩スケール
		originalShoulderVector := originalArmBone.Position.Subed(originalNeckRootBone.Position).Round(1e-2)
		sizingShoulderVector := sizingArmBone.Position.Subed(sizingNeckRootBone.Position).Round(1e-2)
		armScales[i] = sizingShoulderVector.Length() / originalShoulderVector.Length()

		// 肩IK
		shoulderIkBone := pmx.NewBoneByName(fmt.Sprintf("%s%sIk", pmx.MLIB_PREFIX, sizingShoulderBone.Name()))
		shoulderIkBone.Position = sizingArmBone.Position
		shoulderIkBone.Ik = pmx.NewIk()
		shoulderIkBone.Ik.BoneIndex = sizingArmBone.Index()
		shoulderIkBone.Ik.LoopCount = 10
		shoulderIkBone.Ik.UnitRotation = mmath.NewMRotationFromDegrees(&mmath.MVec3{X: 180, Y: 0, Z: 0})
		shoulderIkBone.Ik.Links = make([]*pmx.IkLink, 1)
		shoulderIkBone.Ik.Links[0] = pmx.NewIkLink()
		shoulderIkBone.Ik.Links[0].BoneIndex = sizingShoulderBone.Index()
		shoulderIkBones[i] = shoulderIkBone
	}

	sizingShoulderRotations := make([][]*mmath.MQuaternion, 2)
	sizingArmRotations := make([][]*mmath.MQuaternion, 2)
	allFrames := make([][]int, 2)
	allBlockSizes := make([]int, 2)
	allBlockCounts := make([]int, 2)

	errorChan := make(chan error, 2)

	var wg sync.WaitGroup
	wg.Add(2)
	for i, direction := range directions {
		go func(i int, direction string) {
			defer wg.Done()

			originalNeckRootBone := originalModel.Bones.GetByName(pmx.NECK_ROOT.String())
			originalArmBone := originalModel.Bones.GetByName(pmx.ARM.StringFromDirection(direction))

			sizingNeckRootBone := sizingModel.Bones.GetByName(pmx.NECK_ROOT.String())
			sizingShoulderBone := sizingModel.Bones.GetByName(pmx.SHOULDER.StringFromDirection(direction))
			sizingArmBone := sizingModel.Bones.GetByName(pmx.ARM.StringFromDirection(direction))

			frames := sizingMotion.BoneFrames.RegisteredFrames(shoulder_direction_bone_names[i])
			allFrames[i] = frames
			allBlockSizes[i], allBlockCounts[i] = miter.GetBlockSize(len(frames) * setSize)

			originalAllDeltas := make([]*delta.VmdDeltas, len(frames))

			// 元モデルのデフォーム(IK ON)
			miter.IterParallelByList(frames, allBlockSizes[i], func(data, index int) {
				frame := float32(data)
				vmdDeltas := delta.NewVmdDeltas(frame, originalModel.Bones, originalModel.Hash(), originalMotion.Hash())
				vmdDeltas.Morphs = deform.DeformMorph(originalModel, originalMotion.MorphFrames, frame, nil)
				vmdDeltas = deform.DeformBoneByPhysicsFlag(originalModel, originalMotion, vmdDeltas, true, frame, shoulder_direction_bone_names[i], false)
				originalAllDeltas[index] = vmdDeltas
			}, func(iterIndex, allCount int) {
				mlog.I(mi18n.T("肩補正01", map[string]interface{}{"No": sizingSet.Index + 1, "Direction": direction, "IterIndex": iterIndex, "AllCount": allCount}))
			})

			sizingShoulderRotations[i] = make([]*mmath.MQuaternion, len(frames))
			sizingArmRotations[i] = make([]*mmath.MQuaternion, len(frames))

			// 先モデルの上半身デフォーム(IK ON)
			if err := miter.IterParallelByList(frames, allBlockSizes[i], func(data, index int) {
				frame := float32(data)
				vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
				vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
				vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, true, frame, shoulder_direction_bone_names[i], false)

				// 首根元から見た肩の相対位置を取得
				originalNeckRootDelta := originalAllDeltas[index].Bones.Get(originalNeckRootBone.Index())
				originalArmDelta := originalAllDeltas[index].Bones.Get(originalArmBone.Index())

				originalArmLocalPosition := originalArmDelta.FilledGlobalPosition().Subed(originalNeckRootDelta.FilledGlobalPosition())
				sizingShoulderLocalPosition := originalArmLocalPosition.MuledScalar(armScales[i])

				sizingNeckRootDelta := vmdDeltas.Bones.Get(sizingNeckRootBone.Index())
				armFixGlobalPosition := sizingNeckRootDelta.FilledGlobalPosition().Added(sizingShoulderLocalPosition)

				sizingShoulderIkDeltas := deform.DeformIk(sizingModel, sizingMotion, vmdDeltas, frame, shoulderIkBones[i], armFixGlobalPosition, []string{sizingArmBone.Name()})
				sizingShoulderRotations[i][index] = sizingShoulderIkDeltas.Bones.Get(sizingShoulderBone.Index()).FilledFrameRotation()

				nowShoulderBf := sizingMotion.BoneFrames.Get(sizingShoulderBone.Name()).Get(frame)
				nowArmBf := sizingMotion.BoneFrames.Get(sizingArmBone.Name()).Get(frame)

				// 腕は逆補正をかける
				upperDiffRotation := nowShoulderBf.Rotation.Inverted().Muled(sizingShoulderRotations[i][index]).Inverted()
				sizingArmRotations[i][index] = upperDiffRotation.Muled(nowArmBf.Rotation)
			}, func(iterIndex, allCount int) {
				mlog.I(mi18n.T("肩補正02", map[string]interface{}{"No": sizingSet.Index + 1, "Direction": direction, "Scale": fmt.Sprintf("%.4f", armScales[i]), "IterIndex": iterIndex, "AllCount": allCount}))
			}); err != nil {
				errorChan <- err
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

	// 補正を登録
	for i, frames := range allFrames {
		shoulderBoneName := pmx.SHOULDER.StringFromDirection(directions[i])
		armBoneName := pmx.ARM.StringFromDirection(directions[i])

		for j, iFrame := range frames {
			frame := float32(iFrame)

			shoulderBf := sizingMotion.BoneFrames.Get(shoulderBoneName).Get(frame)
			shoulderBf.Rotation = sizingShoulderRotations[i][j]
			sizingMotion.InsertRegisteredBoneFrame(shoulderBoneName, shoulderBf)

			armBf := sizingMotion.BoneFrames.Get(armBoneName).Get(frame)
			armBf.Rotation = sizingArmRotations[i][j]
			sizingMotion.InsertRegisteredBoneFrame(armBoneName, armBf)
		}
	}

	sizingSet.CompletedSizingShoulder = true

	return true, nil
}

func isValidSizingShoulder(sizingSet *domain.SizingSet) bool {
	originalModel := sizingSet.OriginalPmx
	sizingModel := sizingSet.SizingPmx

	if !originalModel.Bones.ContainsByName(pmx.NECK_ROOT.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("肩補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.NECK_ROOT.String()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.SHOULDER.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("肩ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.SHOULDER.Left()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.SHOULDER.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("肩ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.SHOULDER.Right()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.ARM.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("肩ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.ARM.Left()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.ARM.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("肩ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.ARM.Right()}))
		return false
	}

	// ------------------------------

	if !sizingModel.Bones.ContainsByName(pmx.NECK_ROOT.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("肩補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.NECK_ROOT.String()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.SHOULDER.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("肩ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.SHOULDER.Left()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.SHOULDER.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("肩ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.SHOULDER.Right()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.ARM.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("肩ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.ARM.Left()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.ARM.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("肩ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.ARM.Right()}))
		return false
	}

	return true
}
