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

	originalUpperRootBone := originalModel.Bones.GetByName(pmx.UPPER_ROOT.String())
	originalUpperBone := originalModel.Bones.GetByName(pmx.UPPER.String())
	originalUpper2Bone := originalModel.Bones.GetByName(pmx.UPPER2.String())
	originalNeckRootBone := originalModel.Bones.GetByName(pmx.NECK_ROOT.String())

	upperRootBone := sizingModel.Bones.GetByName(pmx.UPPER_ROOT.String())
	upperBone := sizingModel.Bones.GetByName(pmx.UPPER.String())
	upper2Bone := sizingModel.Bones.GetByName(pmx.UPPER2.String())
	neckRootBone := sizingModel.Bones.GetByName(pmx.NECK_ROOT.String())

	// 体幹中心から首根元のベクトル
	originalUpperVector := originalUpper2Bone.Position.Subed(originalUpperBone.Position).Round(1e-2)
	sizingUpperVector := upper2Bone.Position.Subed(upperBone.Position).Round(1e-2)

	// 体幹中心から上半身2のベクトル
	originalUpper2Vector := originalNeckRootBone.Position.Subed(originalUpper2Bone.Position).Round(1e-2)
	sizingUpper2Vector := neckRootBone.Position.Subed(upper2Bone.Position).Round(1e-2)

	// 上半身2の位置比率
	upperRatio := sizingUpperVector.Length() / originalUpperVector.Length()
	upper2Ratio := sizingUpper2Vector.Length() / originalUpper2Vector.Length()

	// 上半身IK
	upperIkBone := pmx.NewBoneByName(fmt.Sprintf("%s%sIk", pmx.MLIB_PREFIX, upperBone.Name()))
	upperIkBone.Position = upper2Bone.Position
	upperIkBone.Ik = pmx.NewIk()
	upperIkBone.Ik.BoneIndex = upper2Bone.Index()
	upperIkBone.Ik.LoopCount = 20
	upperIkBone.Ik.UnitRotation = mmath.NewMRotationFromRadians(&mmath.MVec3{X: 2, Y: 0, Z: 0})
	upperIkBone.Ik.Links = make([]*pmx.IkLink, 2)
	// 体幹中心（動かさない）
	upperIkBone.Ik.Links[0] = pmx.NewIkLink()
	upperIkBone.Ik.Links[0].BoneIndex = upperRootBone.Index()
	upperIkBone.Ik.Links[0].AngleLimit = true
	// 上半身
	upperIkBone.Ik.Links[1] = pmx.NewIkLink()
	upperIkBone.Ik.Links[1].BoneIndex = upperBone.Index()

	// 上半身2IK
	upper2IkBone := pmx.NewBoneByName(fmt.Sprintf("%s%sIk", pmx.MLIB_PREFIX, upper2Bone.Name()))
	upper2IkBone.Position = neckRootBone.Position
	upper2IkBone.Ik = pmx.NewIk()
	upper2IkBone.Ik.BoneIndex = neckRootBone.Index()
	upper2IkBone.Ik.LoopCount = 20
	upper2IkBone.Ik.UnitRotation = mmath.NewMRotationFromRadians(&mmath.MVec3{X: 2, Y: 0, Z: 0})
	upper2IkBone.Ik.Links = make([]*pmx.IkLink, 1)
	// 上半身2
	upper2IkBone.Ik.Links[0] = pmx.NewIkLink()
	upper2IkBone.Ik.Links[0].BoneIndex = upper2Bone.Index()

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
	upper2Positions := make([]*mmath.MVec3, len(frames))
	neckRootPositions := make([]*mmath.MVec3, len(frames))

	// 先モデルのデフォーム(IK OFF)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, false, frame, trunk_upper_bone_names, false)
		sizingOffDeltas[index] = vmdDeltas

		// 体幹中心から見た首根元の相対位置を取得
		originalNeckRootDelta := originalAllDeltas[index].Bones.Get(originalNeckRootBone.Index())
		originalUpper2Delta := originalAllDeltas[index].Bones.Get(originalUpper2Bone.Index())
		originalUpperRootDelta := originalAllDeltas[index].Bones.Get(originalUpperRootBone.Index())
		originalUpper2LocalPosition := originalUpper2Delta.FilledGlobalPosition().Subed(
			originalUpperRootDelta.FilledGlobalPosition())
		originalNeckRootLocalPosition := originalNeckRootDelta.FilledGlobalPosition().Subed(
			originalUpper2Delta.FilledGlobalPosition())

		sizingUpperRootDelta := vmdDeltas.Bones.Get(upperRootBone.Index())
		upper2Positions[index] = sizingUpperRootDelta.FilledGlobalPosition().Added(
			originalUpper2LocalPosition.MuledScalar(upperRatio))
		sizingUpper2Delta := vmdDeltas.Bones.Get(upper2Bone.Index())
		neckRootPositions[index] = sizingUpper2Delta.FilledGlobalPosition().Added(
			originalNeckRootLocalPosition.MuledScalar(upper2Ratio))
	})

	sizingUpperIkDeltas := make([]*delta.VmdDeltas, len(frames))

	// 先モデルの上半身追加補正(IK ON)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		sizingUpperIkDeltas[index] = deform.DeformIk(
			sizingModel, sizingMotion, frame, upperIkBone, upper2Positions[index])
	})

	// 補正を登録
	for i, iFrame := range frames {
		frame := float32(iFrame)

		upperBf := sizingMotion.BoneFrames.Get(upperBone.Name()).Get(frame)
		upperBf.Rotation = sizingUpperIkDeltas[i].Bones.Get(upperBone.Index()).FilledFrameRotation()
		sizingMotion.InsertRegisteredBoneFrame(upperBone.Name(), upperBf)
	}

	sizingUpper2IkDeltas := make([]*delta.VmdDeltas, len(frames))

	// 先モデルの上半身2追加補正(IK ON)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		sizingUpper2IkDeltas[index] = deform.DeformIk(
			sizingModel, sizingMotion, frame, upper2IkBone, neckRootPositions[index])
	})

	// 補正を登録
	for i, iFrame := range frames {
		frame := float32(iFrame)

		upper2Bf := sizingMotion.BoneFrames.Get(upper2Bone.Name()).Get(frame)
		upper2Bf.Rotation = sizingUpper2IkDeltas[i].Bones.Get(upper2Bone.Index()).FilledFrameRotation()
		sizingMotion.InsertRegisteredBoneFrame(upper2Bone.Name(), upper2Bf)
	}

	sizingSet.CompletedSizingUpper = true
}

func isValidSizingUpper(sizingSet *domain.SizingSet) bool {
	originalModel := sizingSet.OriginalPmx
	sizingModel := sizingSet.SizingPmx
	sizingMotion := sizingSet.OutputVmd

	// 上半身、上半身2、首根元が存在するか

	if !originalModel.Bones.ContainsByName(pmx.UPPER_ROOT.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("上半身補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.UPPER_ROOT.String()}))
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

	if !sizingModel.Bones.ContainsByName(pmx.UPPER_ROOT.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("上半身補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.UPPER_ROOT.String()}))
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
