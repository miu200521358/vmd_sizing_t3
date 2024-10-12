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
	originalLeftShoulderBone := originalModel.Bones.GetByName(pmx.SHOULDER.Left())
	originalRightShoulderBone := originalModel.Bones.GetByName(pmx.SHOULDER.Right())
	originalNeckRootBone := originalModel.Bones.GetByName(pmx.NECK_ROOT.String())
	// originalNeckBone := originalModel.Bones.GetByName(pmx.NECK.String())
	// originalHeadBone := originalModel.Bones.GetByName(pmx.HEAD.String())
	// originalHeadTailBone := originalModel.Bones.GetByName(pmx.HEAD_TAIL.String())

	sizingUpperRootBone := sizingModel.Bones.GetByName(pmx.UPPER_ROOT.String())
	sizingUpperBone := sizingModel.Bones.GetByName(pmx.UPPER.String())
	sizingUpper2Bone := sizingModel.Bones.GetByName(pmx.UPPER2.String())
	sizingNeckRootBone := sizingModel.Bones.GetByName(pmx.NECK_ROOT.String())
	// sizingLeftShoulderBone := sizingModel.Bones.GetByName(pmx.SHOULDER.Left())
	// sizingRightShoulderBone := sizingModel.Bones.GetByName(pmx.SHOULDER.Right())
	// sizingNeckBone := sizingModel.Bones.GetByName(pmx.NECK.String())
	// sizingHeadBone := sizingModel.Bones.GetByName(pmx.HEAD.String())
	// sizingHeadTailBone := sizingModel.Bones.GetByName(pmx.HEAD_TAIL.String())

	// 体幹中心 - 首根元
	originalUpperVector := originalNeckRootBone.Position.Subed(originalUpperRootBone.Position).Round(1e-2)
	sizingUpperVector := sizingNeckRootBone.Position.Subed(sizingUpperRootBone.Position).Round(1e-2)

	// 上半身スケール
	upperScale := sizingUpperVector.Length() / originalUpperVector.Length()

	// 上半身IK
	upperIkBone := pmx.NewBoneByName(fmt.Sprintf("%s%sIk", pmx.MLIB_PREFIX, sizingUpperBone.Name()))
	upperIkBone.Position = sizingNeckRootBone.Position
	upperIkBone.Ik = pmx.NewIk()
	upperIkBone.Ik.BoneIndex = sizingNeckRootBone.Index()
	upperIkBone.Ik.LoopCount = 200
	upperIkBone.Ik.UnitRotation = mmath.NewMRotationFromDegrees(&mmath.MVec3{X: 1, Y: 0, Z: 0})
	upperIkBone.Ik.Links = make([]*pmx.IkLink, 2)
	// upperIkBone.Ik.Links[0] = pmx.NewIkLink()
	// upperIkBone.Ik.Links[0].BoneIndex = sizingHeadBone.Index()
	// upperIkBone.Ik.Links[1] = pmx.NewIkLink()
	// upperIkBone.Ik.Links[1].BoneIndex = sizingNeckBone.Index()
	upperIkBone.Ik.Links[0] = pmx.NewIkLink()
	upperIkBone.Ik.Links[0].BoneIndex = sizingUpper2Bone.Index()
	upperIkBone.Ik.Links[1] = pmx.NewIkLink()
	upperIkBone.Ik.Links[1].BoneIndex = sizingUpperBone.Index()

	frames := sizingMotion.BoneFrames.RegisteredFrames(trunk_upper_bone_names)

	mlog.I(mi18n.T("上半身補正開始", map[string]interface{}{"No": sizingSet.Index + 1}))

	mlog.I(mi18n.T("上半身補正01", map[string]interface{}{"No": sizingSet.Index + 1, "Scale": fmt.Sprintf("%.4f", upperScale)}))

	originalAllDeltas := make([]*delta.VmdDeltas, len(frames))

	// 元モデルのデフォーム(IK ON)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		vmdDeltas := delta.NewVmdDeltas(frame, originalModel.Bones, originalModel.Hash(), originalMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(originalModel, originalMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(originalModel, originalMotion, vmdDeltas, true, frame, trunk_upper_bone_names, false)
		originalAllDeltas[index] = vmdDeltas
	})

	sizingUpperRotations := make([]*mmath.MQuaternion, len(frames))
	sizingUpper2Rotations := make([]*mmath.MQuaternion, len(frames))
	sizingLeftShoulderRotations := make([]*mmath.MQuaternion, len(frames))
	sizingRightShoulderRotations := make([]*mmath.MQuaternion, len(frames))

	mlog.I(mi18n.T("上半身補正02", map[string]interface{}{"No": sizingSet.Index + 1}))

	// 先モデルのデフォーム(IK ON)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, true, frame, trunk_upper_bone_names, false)

		// 体幹中心から見た首根元の相対位置を取得
		originalUpperRootDelta := originalAllDeltas[index].Bones.Get(originalUpperRootBone.Index())
		originalNeckRootDelta := originalAllDeltas[index].Bones.Get(originalNeckRootBone.Index())

		originalUpperLocalPosition := originalUpperRootDelta.FilledGlobalMatrix().Inverted().MulVec3(originalNeckRootDelta.FilledGlobalPosition())
		sizingUpperLocalPosition := originalUpperLocalPosition.MuledScalar(upperScale)

		sizingUpperRootDelta := vmdDeltas.Bones.Get(sizingUpperRootBone.Index())
		sizingNeckRootDelta := vmdDeltas.Bones.Get(sizingNeckRootBone.Index())
		neckRootFixGlobalPosition := sizingUpperRootDelta.FilledGlobalMatrix().MulVec3(sizingUpperLocalPosition)

		sizingUpperIkDeltas := deform.DeformIk(sizingModel, sizingMotion, frame, upperIkBone, neckRootFixGlobalPosition)
		sizingUpperRotations[index] = sizingUpperIkDeltas.Bones.Get(sizingUpperBone.Index()).FilledFrameRotation()
		sizingUpper2Rotations[index] = sizingUpperIkDeltas.Bones.Get(sizingUpper2Bone.Index()).FilledFrameRotation()

		originalUpperRotation := originalAllDeltas[index].Bones.Get(originalUpperBone.Index()).FilledFrameRotation()
		originalUpper2Rotation := originalAllDeltas[index].Bones.Get(originalUpper2Bone.Index()).FilledFrameRotation()
		originalLeftShoulderRotation := originalAllDeltas[index].Bones.Get(originalLeftShoulderBone.Index()).FilledFrameRotation()
		originalRightShoulderRotation := originalAllDeltas[index].Bones.Get(originalRightShoulderBone.Index()).FilledFrameRotation()

		// 肩は逆補正をかける
		upperDiffRotation := sizingUpper2Rotations[index].Muled(originalUpper2Rotation.Inverted()).Muled(sizingUpperRotations[index].Muled(originalUpperRotation.Inverted())).Inverted()
		sizingLeftShoulderRotations[index] = originalLeftShoulderRotation.Muled(upperDiffRotation)
		sizingRightShoulderRotations[index] = originalRightShoulderRotation.Muled(upperDiffRotation)

		mlog.V("上半身補正02[%.0f] originalHeadTailPosition[%v], sizingUpperLocalPosition[%v], sizingHeadTailPosition[%v], headTailFixGlobalPosition[%v]", frame, originalNeckRootDelta.FilledGlobalPosition(), sizingUpperLocalPosition, sizingNeckRootDelta.FilledGlobalPosition(), neckRootFixGlobalPosition)
	})

	// 補正を登録
	for i, iFrame := range frames {
		frame := float32(iFrame)

		upperBf := sizingMotion.BoneFrames.Get(sizingUpperBone.Name()).Get(frame)
		upperBf.Rotation = sizingUpperRotations[i]
		sizingMotion.InsertRegisteredBoneFrame(sizingUpperBone.Name(), upperBf)

		upper2Bf := sizingMotion.BoneFrames.Get(sizingUpper2Bone.Name()).Get(frame)
		upper2Bf.Rotation = sizingUpper2Rotations[i]
		sizingMotion.InsertRegisteredBoneFrame(sizingUpper2Bone.Name(), upper2Bf)

		leftShoulderBf := sizingMotion.BoneFrames.Get(pmx.SHOULDER.Left()).Get(frame)
		leftShoulderBf.Rotation = sizingLeftShoulderRotations[i]
		sizingMotion.InsertRegisteredBoneFrame(pmx.SHOULDER.Left(), leftShoulderBf)

		rightShoulderBf := sizingMotion.BoneFrames.Get(pmx.SHOULDER.Right()).Get(frame)
		rightShoulderBf.Rotation = sizingRightShoulderRotations[i]
		sizingMotion.InsertRegisteredBoneFrame(pmx.SHOULDER.Right(), rightShoulderBf)
	}

	sizingSet.CompletedSizingUpper = true
}

func isValidSizingUpper(sizingSet *domain.SizingSet) bool {
	originalModel := sizingSet.OriginalPmx
	sizingModel := sizingSet.SizingPmx

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

	if !originalModel.Bones.ContainsByName(pmx.UPPER2.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("上半身ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.UPPER2.String()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.NECK.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("上半身ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.NECK.String()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.HEAD.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("上半身ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.HEAD.String()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.HEAD_TAIL.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("上半身ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.HEAD_TAIL.String()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.SHOULDER.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("上半身ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.SHOULDER.Left()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.SHOULDER.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("上半身ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.SHOULDER.Right()}))
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

	if !sizingModel.Bones.ContainsByName(pmx.UPPER2.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("上半身ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.UPPER2.String()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.NECK.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("上半身ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.NECK.String()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.HEAD.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("上半身ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.HEAD.String()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.HEAD_TAIL.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("上半身ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.HEAD_TAIL.String()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.SHOULDER.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("上半身ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.SHOULDER.Left()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.SHOULDER.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("上半身ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.SHOULDER.Right()}))
		return false
	}

	return true
}
