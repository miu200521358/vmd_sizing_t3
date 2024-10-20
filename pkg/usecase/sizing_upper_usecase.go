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

func SizingUpper(sizingSet *domain.SizingSet) bool {
	if !sizingSet.IsSizingUpper || (sizingSet.IsSizingUpper && sizingSet.CompletedSizingUpper) {
		return false
	}

	if !isValidSizingUpper(sizingSet) {
		return false
	}

	originalModel := sizingSet.OriginalPmx
	originalMotion := sizingSet.OriginalVmd
	sizingModel := sizingSet.SizingPmx
	sizingMotion := sizingSet.OutputVmd

	originalUpperRootBone := originalModel.Bones.GetByName(pmx.UPPER_ROOT.String())
	// originalUpperBone := originalModel.Bones.GetByName(pmx.UPPER.String())
	originalUpper2Bone := originalModel.Bones.GetByName(pmx.UPPER2.String())
	// originalLeftShoulderBone := originalModel.Bones.GetByName(pmx.SHOULDER.Left())
	// originalRightShoulderBone := originalModel.Bones.GetByName(pmx.SHOULDER.Right())
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
	sizingNeckBone := sizingModel.Bones.GetByName(pmx.NECK.String())
	// sizingHeadBone := sizingModel.Bones.GetByName(pmx.HEAD.String())
	// sizingHeadTailBone := sizingModel.Bones.GetByName(pmx.HEAD_TAIL.String())

	// 上半身根元から首根元の間に上半身がどの辺りに位置しているか
	originalUpperRatio := originalUpper2Bone.Position.Subed(originalUpperRootBone.Position).Length() / originalNeckRootBone.Position.Subed(originalUpperRootBone.Position).Length()
	sizingUpperRatio := sizingUpper2Bone.Position.Subed(sizingUpperRootBone.Position).Length() / sizingNeckRootBone.Position.Subed(sizingUpperRootBone.Position).Length()
	upperPositionRatio := sizingUpperRatio / originalUpperRatio
	originalUpperDirection := originalUpper2Bone.Position.Subed(originalUpperRootBone.Position).Normalized()
	sizingUpperDirection := sizingUpper2Bone.Position.Subed(sizingUpperRootBone.Position).Normalized()
	sizingUpperSlopeMat := mmath.NewMQuaternionRotate(originalUpperDirection, sizingUpperDirection).ToMat4()

	originalUpper2Direction := originalNeckRootBone.Position.Subed(originalUpperRootBone.Position).Normalized()
	sizingUpper2Direction := sizingNeckRootBone.Position.Subed(sizingUpperRootBone.Position).Normalized()
	sizingUpper2SlopeMat := mmath.NewMQuaternionRotate(originalUpper2Direction, sizingUpper2Direction).ToMat4()

	// 上半身全体のサイズ差
	originalUpperLength := originalNeckRootBone.Position.Subed(originalUpperRootBone.Position).Length()
	sizingUpperLength := sizingNeckRootBone.Position.Subed(sizingUpperRootBone.Position).Length()
	upperTotalRatio := sizingUpperLength / originalUpperLength

	// 上半身スケール
	originalUpperVector := originalUpper2Bone.Position.Subed(originalUpperRootBone.Position).Round(1e-2)
	sizingUpperVector := sizingUpper2Bone.Position.Subed(sizingUpperRootBone.Position).Round(1e-2)
	upperScale := sizingUpperVector.Length() / originalUpperVector.Length() * upperPositionRatio * upperTotalRatio

	// 上半身IK
	upperIkBone := pmx.NewBoneByName(fmt.Sprintf("%s%sIk", pmx.MLIB_PREFIX, sizingUpperBone.Name()))
	upperIkBone.Position = sizingUpper2Bone.Position
	upperIkBone.Ik = pmx.NewIk()
	upperIkBone.Ik.BoneIndex = sizingUpper2Bone.Index()
	upperIkBone.Ik.LoopCount = 10
	upperIkBone.Ik.UnitRotation = mmath.NewMRotationFromDegrees(&mmath.MVec3{X: 180, Y: 0, Z: 0})
	upperIkBone.Ik.Links = make([]*pmx.IkLink, 1)
	upperIkBone.Ik.Links[0] = pmx.NewIkLink()
	upperIkBone.Ik.Links[0].BoneIndex = sizingUpperBone.Index()

	// 上半身2スケール
	originalUpper2Vector := originalNeckRootBone.Position.Subed(originalUpperRootBone.Position).Round(1e-2)
	sizingUpper2Vector := sizingNeckRootBone.Position.Subed(sizingUpperRootBone.Position).Round(1e-2)
	upper2Scale := sizingUpper2Vector.Length() / originalUpper2Vector.Length()

	// 上半身2IK
	upper2IkBone := pmx.NewBoneByName(fmt.Sprintf("%s%sIk", pmx.MLIB_PREFIX, sizingUpper2Bone.Name()))
	upper2IkBone.Position = sizingNeckRootBone.Position
	upper2IkBone.Ik = pmx.NewIk()
	upper2IkBone.Ik.BoneIndex = sizingNeckRootBone.Index()
	upper2IkBone.Ik.LoopCount = 10
	upper2IkBone.Ik.UnitRotation = mmath.NewMRotationFromDegrees(&mmath.MVec3{X: 180, Y: 0, Z: 0})
	upper2IkBone.Ik.Links = make([]*pmx.IkLink, 1)
	upper2IkBone.Ik.Links[0] = pmx.NewIkLink()
	upper2IkBone.Ik.Links[0].BoneIndex = sizingUpper2Bone.Index()

	frames := sizingMotion.BoneFrames.RegisteredFrames(trunk_upper_bone_names)

	if len(frames) == 0 {
		return false
	}

	mlog.I(mi18n.T("上半身補正開始", map[string]interface{}{"No": sizingSet.Index + 1}))

	mlog.I(mi18n.T("上半身補正01", map[string]interface{}{"No": sizingSet.Index + 1}))

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
	sizingNeckRotations := make([]*mmath.MQuaternion, len(frames))

	mlog.I(mi18n.T("上半身補正02", map[string]interface{}{"No": sizingSet.Index + 1, "Scale": fmt.Sprintf("%.4f", upperScale)}))

	// 先モデルの上半身デフォーム(IK ON)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, true, frame, trunk_upper_bone_names, false)

		// 上半身根元から見た上半身2の相対位置を取得
		originalUpperRootDelta := originalAllDeltas[index].Bones.Get(originalUpperRootBone.Index())
		originalUpper2Delta := originalAllDeltas[index].Bones.Get(originalUpper2Bone.Index())

		originalUpper1LocalPosition := originalUpperRootDelta.FilledGlobalMatrix().MulVec3(originalUpper2Delta.FilledGlobalPosition())
		sizingUpper2LocalPosition := originalUpper1LocalPosition.MuledScalar(upperScale)
		sizingUpper2SlopeLocalPosition := sizingUpperSlopeMat.MulVec3(sizingUpper2LocalPosition)

		sizingUpperRootDelta := vmdDeltas.Bones.Get(sizingUpperRootBone.Index())
		upper2FixGlobalPosition := sizingUpperRootDelta.FilledGlobalMatrix().MulVec3(sizingUpper2SlopeLocalPosition)

		sizingUpperIkDeltas := deform.DeformIk(sizingModel, sizingMotion, vmdDeltas, frame, upperIkBone, upper2FixGlobalPosition, []string{sizingNeckRootBone.Name()})
		sizingUpperRotations[index] = sizingUpperIkDeltas.Bones.Get(sizingUpperBone.Index()).FilledFrameRotation()

		nowUpperBf := sizingMotion.BoneFrames.Get(sizingUpperBone.Name()).Get(frame)
		nowLeftShoulderBf := sizingMotion.BoneFrames.Get(pmx.SHOULDER.Left()).Get(frame)
		nowRightShoulderBf := sizingMotion.BoneFrames.Get(pmx.SHOULDER.Right()).Get(frame)
		nowNeckBf := sizingMotion.BoneFrames.Get(sizingNeckBone.Name()).Get(frame)

		// 首・肩は逆補正をかける
		upperDiffRotation := nowUpperBf.Rotation.Inverted().Muled(sizingUpperRotations[index]).Inverted()
		sizingLeftShoulderRotations[index] = upperDiffRotation.Muled(nowLeftShoulderBf.Rotation)
		sizingRightShoulderRotations[index] = upperDiffRotation.Muled(nowRightShoulderBf.Rotation)
		sizingNeckRotations[index] = upperDiffRotation.Muled(nowNeckBf.Rotation)
	})

	// 補正を登録
	for i, iFrame := range frames {
		frame := float32(iFrame)

		upperBf := sizingMotion.BoneFrames.Get(sizingUpperBone.Name()).Get(frame)
		upperBf.Rotation = sizingUpperRotations[i]
		sizingMotion.InsertRegisteredBoneFrame(sizingUpperBone.Name(), upperBf)

		neckBf := sizingMotion.BoneFrames.Get(sizingNeckBone.Name()).Get(frame)
		neckBf.Rotation = sizingNeckRotations[i]
		sizingMotion.InsertRegisteredBoneFrame(sizingNeckBone.Name(), neckBf)

		leftShoulderBf := sizingMotion.BoneFrames.Get(pmx.SHOULDER.Left()).Get(frame)
		leftShoulderBf.Rotation = sizingLeftShoulderRotations[i]
		sizingMotion.InsertRegisteredBoneFrame(pmx.SHOULDER.Left(), leftShoulderBf)

		rightShoulderBf := sizingMotion.BoneFrames.Get(pmx.SHOULDER.Right()).Get(frame)
		rightShoulderBf.Rotation = sizingRightShoulderRotations[i]
		sizingMotion.InsertRegisteredBoneFrame(pmx.SHOULDER.Right(), rightShoulderBf)
	}

	mlog.I(mi18n.T("上半身補正03", map[string]interface{}{"No": sizingSet.Index + 1, "Scale": fmt.Sprintf("%.4f", upper2Scale)}))

	// 先モデルの上半身2デフォーム(IK ON)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, true, frame, trunk_upper_bone_names, false)

		// 上半身根元から見た首根元の相対位置を取得
		originalUpperRootDelta := originalAllDeltas[index].Bones.Get(originalUpperRootBone.Index())
		originalNeckRootDelta := originalAllDeltas[index].Bones.Get(originalNeckRootBone.Index())

		originalNeckRootLocalPosition := originalUpperRootDelta.FilledGlobalMatrix().MulVec3(originalNeckRootDelta.FilledGlobalPosition())
		sizingNeckRootLocalPosition := originalNeckRootLocalPosition.MuledScalar(upper2Scale)
		sizingNeckRootSlopeLocalPosition := sizingUpper2SlopeMat.MulVec3(sizingNeckRootLocalPosition)

		sizingUpperRootDelta := vmdDeltas.Bones.Get(sizingUpperRootBone.Index())
		neckRootFixGlobalPosition := sizingUpperRootDelta.FilledGlobalMatrix().MulVec3(sizingNeckRootSlopeLocalPosition)

		sizingUpper2IkDeltas := deform.DeformIk(sizingModel, sizingMotion, vmdDeltas, frame, upper2IkBone, neckRootFixGlobalPosition, []string{sizingNeckRootBone.Name()})
		sizingUpper2Rotations[index] = sizingUpper2IkDeltas.Bones.Get(sizingUpper2Bone.Index()).FilledFrameRotation()

		nowUpper2Bf := sizingMotion.BoneFrames.Get(sizingUpper2Bone.Name()).Get(frame)

		// 首・肩は逆補正をかける
		upper2DiffRotation := nowUpper2Bf.Rotation.Inverted().Muled(sizingUpper2Rotations[index]).Inverted()
		sizingLeftShoulderRotations[index] = upper2DiffRotation.Muled(sizingLeftShoulderRotations[index])
		sizingRightShoulderRotations[index] = upper2DiffRotation.Muled(sizingRightShoulderRotations[index])
		sizingNeckRotations[index] = upper2DiffRotation.Muled(sizingNeckRotations[index])
	})

	// 補正を登録
	for i, iFrame := range frames {
		frame := float32(iFrame)

		upper2Bf := sizingMotion.BoneFrames.Get(sizingUpper2Bone.Name()).Get(frame)
		upper2Bf.Rotation = sizingUpper2Rotations[i]
		sizingMotion.InsertRegisteredBoneFrame(sizingUpper2Bone.Name(), upper2Bf)

		neckBf := sizingMotion.BoneFrames.Get(sizingNeckBone.Name()).Get(frame)
		neckBf.Rotation = sizingNeckRotations[i]
		sizingMotion.InsertRegisteredBoneFrame(sizingNeckBone.Name(), neckBf)

		leftShoulderBf := sizingMotion.BoneFrames.Get(pmx.SHOULDER.Left()).Get(frame)
		leftShoulderBf.Rotation = sizingLeftShoulderRotations[i]
		sizingMotion.InsertRegisteredBoneFrame(pmx.SHOULDER.Left(), leftShoulderBf)

		rightShoulderBf := sizingMotion.BoneFrames.Get(pmx.SHOULDER.Right()).Get(frame)
		rightShoulderBf.Rotation = sizingRightShoulderRotations[i]
		sizingMotion.InsertRegisteredBoneFrame(pmx.SHOULDER.Right(), rightShoulderBf)
	}

	sizingSet.CompletedSizingUpper = true
	return true
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
