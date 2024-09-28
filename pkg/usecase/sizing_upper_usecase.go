package usecase

import (
	"fmt"

	"github.com/miu200521358/mlib_go/pkg/domain/delta"
	"github.com/miu200521358/mlib_go/pkg/domain/miter"
	"github.com/miu200521358/mlib_go/pkg/domain/mmath"
	"github.com/miu200521358/mlib_go/pkg/domain/pmx"
	"github.com/miu200521358/mlib_go/pkg/infrastructure/deform"
	"github.com/miu200521358/mlib_go/pkg/mutils/mi18n"
	"github.com/miu200521358/mlib_go/pkg/mutils/mlog"
	"github.com/miu200521358/vmd_sizing_t3/pkg/domain"
)

func SizingUpper(sizingSet *domain.SizingSet) {
	if !sizingSet.IsSizingUpper || (sizingSet.IsSizingUpper && sizingSet.CompletedSizingUpper) {
		return
	}

	if !isValidSizingUpper(sizingSet) {
		return
	}

	originalModel := sizingSet.OriginalPmx
	originalMotion := sizingSet.OriginalVmd
	sizingModel := sizingSet.SizingPmx
	sizingMotion := sizingSet.OutputVmd

	originalTrunkRoot := originalModel.Bones.GetByName(pmx.TRUNK_ROOT.String())
	originalNeckRootBone := originalModel.Bones.GetByName(pmx.NECK_ROOT.String())

	trunkRootBone := sizingModel.Bones.GetByName(pmx.TRUNK_ROOT.String())
	upperBone := sizingModel.Bones.GetByName(pmx.UPPER.String())
	upper2Bone := sizingModel.Bones.GetByName(pmx.UPPER2.String())
	neckRootBone := sizingModel.Bones.GetByName(pmx.NECK_ROOT.String())

	// 体幹中心から首根元のベクトル比率
	originalUpperVector := originalTrunkRoot.Position.Subed(originalNeckRootBone.Position).Round(1e-2)
	sizingUpperVector := trunkRootBone.Position.Subed(neckRootBone.Position).Round(1e-2)
	upperRatio := sizingUpperVector.Dived(originalUpperVector).Effective().One()

	// 上半身IK
	upperIkBone := pmx.NewBoneByName(fmt.Sprintf("%s%sIk", pmx.MLIB_PREFIX, upperBone.Name()))
	upperIkBone.Ik = pmx.NewIk()
	upperIkBone.Ik.BoneIndex = neckRootBone.Index()
	upperIkBone.Ik.LoopCount = 20
	upperIkBone.Ik.UnitRotation = mmath.NewMRotationFromRadians(&mmath.MVec3{X: 2, Y: 0, Z: 0})
	upperIkBone.Ik.Links = make([]*pmx.IkLink, 3)
	// 体幹中心（動かさない）
	upperIkBone.Ik.Links[0] = pmx.NewIkLink()
	upperIkBone.Ik.Links[0].BoneIndex = trunkRootBone.Index()
	upperIkBone.Ik.Links[0].AngleLimit = true
	// 上半身
	upperIkBone.Ik.Links[1] = pmx.NewIkLink()
	upperIkBone.Ik.Links[1].BoneIndex = upperBone.Index()
	// 上半身2
	upperIkBone.Ik.Links[2] = pmx.NewIkLink()
	upperIkBone.Ik.Links[2].BoneIndex = upper2Bone.Index()

	frames := originalMotion.BoneFrames.RegisteredFrames(trunk_upper_bone_names)

	originalAllDeltas := make([]*delta.VmdDeltas, len(frames))

	mlog.I(mi18n.T("上半身補正開始", map[string]interface{}{"No": sizingSet.Index + 1}))

	// 元モデルのデフォーム(IK ON)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		vmdDeltas := delta.NewVmdDeltas(frame, originalModel.Bones, originalModel.Hash(), originalMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(originalModel, originalMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(originalModel, originalMotion, vmdDeltas, true, frame, trunk_upper_bone_names, false)
		originalAllDeltas[index] = vmdDeltas
	})

	sizingOffDeltas := make([]*delta.VmdDeltas, len(frames))
	neckRootPositions := make([]*mmath.MVec3, len(frames))

	// 先モデルのデフォーム(IK OFF)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, false, frame, leg_all_bone_names, false)
		sizingOffDeltas[index] = vmdDeltas

		// 体幹中心から見た首根元の相対位置を取得
		originalNeckRootDelta := originalAllDeltas[index].Bones.Get(originalNeckRootBone.Index())
		originalTrunkRootDelta := originalAllDeltas[index].Bones.Get(originalTrunkRoot.Index())
		originalNeckRootLocalPosition := originalNeckRootDelta.FilledGlobalPosition().Subed(
			originalTrunkRootDelta.FilledGlobalPosition())

		sizingTrunkRootDelta := vmdDeltas.Bones.Get(trunkRootBone.Index())
		neckRootPositions[index] = sizingTrunkRootDelta.FilledGlobalPosition().Added(
			originalNeckRootLocalPosition.Muled(upperRatio))
	})

	sizingIkOnDeltas := make([]*delta.VmdDeltas, len(frames))

	// 先モデルの上半身追加補正(IK ON)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		sizingIkOnDeltas[index] = deform.DeformIk(
			sizingModel, sizingMotion, frame, upperIkBone, neckRootPositions[index])
	})

	// 補正を登録
	for i, iFrame := range frames {
		frame := float32(iFrame)

		upperBf := sizingMotion.BoneFrames.Get(upperBone.Name()).Get(frame)
		upperBf.Rotation = sizingIkOnDeltas[i].Bones.Get(upperBone.Index()).FilledFrameRotation()
		sizingMotion.InsertRegisteredBoneFrame(upperBone.Name(), upperBf)

		upper2Bf := sizingMotion.BoneFrames.Get(upper2Bone.Name()).Get(frame)
		upper2Bf.Rotation = sizingIkOnDeltas[i].Bones.Get(upper2Bone.Index()).FilledFrameRotation()
		sizingMotion.InsertRegisteredBoneFrame(upper2Bone.Name(), upper2Bf)
	}

	sizingSet.CompletedSizingUpper = true
}

func isValidSizingUpper(sizingSet *domain.SizingSet) bool {
	originalModel := sizingSet.OriginalPmx
	sizingModel := sizingSet.SizingPmx
	sizingMotion := sizingSet.OutputVmd

	// 上半身、上半身2、首根元が存在するか

	if !originalModel.Bones.ContainsByName(pmx.TRUNK_ROOT.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("上半身補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.TRUNK_ROOT.String()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.UPPER.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("上半身補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.UPPER.String()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.UPPER2.String()) &&
		sizingMotion.BoneFrames.Contains(pmx.UPPER2.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("上半身ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.UPPER2.String()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.NECK_ROOT.String()) &&
		sizingMotion.BoneFrames.Contains(pmx.NECK_ROOT.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("上半身ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.NECK_ROOT.String()}))
		return false
	}

	// ------------------------------

	if !sizingModel.Bones.ContainsByName(pmx.TRUNK_ROOT.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("上半身補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.TRUNK_ROOT.String()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.UPPER.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("上半身補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.UPPER.String()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.UPPER2.String()) &&
		sizingMotion.BoneFrames.Contains(pmx.UPPER2.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("上半身ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.UPPER2.String()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.NECK_ROOT.String()) &&
		sizingMotion.BoneFrames.Contains(pmx.NECK_ROOT.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("上半身ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.NECK_ROOT.String()}))
		return false
	}

	return true
}
