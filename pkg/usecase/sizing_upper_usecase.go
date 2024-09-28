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

	originalUpperBone := originalModel.Bones.GetByName(pmx.UPPER.String())
	originalUpper2Bone := originalModel.Bones.GetByName(pmx.UPPER2.String())
	originalNeckBone := originalModel.Bones.GetByName(pmx.NECK.String())
	originalHeadBone := originalModel.Bones.GetByName(pmx.HEAD.String())
	originalHeadTailBone := originalModel.Bones.GetByName(pmx.HEAD_TAIL.String())

	upperRootBone := sizingModel.Bones.GetByName(pmx.UPPER_ROOT.String())
	upperBone := sizingModel.Bones.GetByName(pmx.UPPER.String())
	upper2Bone := sizingModel.Bones.GetByName(pmx.UPPER2.String())
	neckBone := sizingModel.Bones.GetByName(pmx.NECK.String())
	headBone := sizingModel.Bones.GetByName(pmx.HEAD.String())
	headTailBone := sizingModel.Bones.GetByName(pmx.HEAD_TAIL.String())

	// 体幹中心 - 上半身2
	originalUpperVector := originalUpper2Bone.Position.Subed(originalUpperBone.Position).Round(1e-2)
	sizingUpperVector := upper2Bone.Position.Subed(upperBone.Position).Round(1e-2)

	// 上半身2 - 首
	originalUpper2Vector := originalNeckBone.Position.Subed(originalUpper2Bone.Position).Round(1e-2)
	sizingUpper2Vector := neckBone.Position.Subed(upper2Bone.Position).Round(1e-2)

	// 首 - 頭
	originalNeckVector := originalHeadBone.Position.Subed(originalNeckBone.Position).Round(1e-2)
	sizingNeckVector := headBone.Position.Subed(neckBone.Position).Round(1e-2)

	// 頭 - 頭先
	originalHeadVector := originalHeadTailBone.Position.Subed(originalHeadBone.Position).Round(1e-2).Normalized()
	sizingHeadVector := headTailBone.Position.Subed(headBone.Position).Round(1e-2).Normalized()

	// 位置比率
	upperRatio := sizingUpperVector.Dived(originalUpperVector).Round(1e-2).Effective().One()
	upper2Ratio := sizingUpper2Vector.Dived(originalUpper2Vector).Round(1e-2).Effective().One()
	neckRatio := sizingNeckVector.Dived(originalNeckVector).Round(1e-2).Effective().One()
	headRatio := sizingHeadVector.Dived(originalHeadVector).Round(1e-2).Effective().One()

	// 上半身IK
	upperIkBone := pmx.NewBoneByName(fmt.Sprintf("%s%sIk", pmx.MLIB_PREFIX, upperBone.Name()))
	upperIkBone.Position = upper2Bone.Position
	upperIkBone.Ik = pmx.NewIk()
	upperIkBone.Ik.BoneIndex = upper2Bone.Index()
	upperIkBone.Ik.LoopCount = 20
	upperIkBone.Ik.UnitRotation = mmath.NewMRotationFromRadians(&mmath.MVec3{X: 2, Y: 0, Z: 0})
	upperIkBone.Ik.Links = make([]*pmx.IkLink, 1)
	upperIkBone.Ik.Links[0] = pmx.NewIkLink()
	upperIkBone.Ik.Links[0].BoneIndex = upperBone.Index()

	// 上半身2IK
	upper2IkBone := pmx.NewBoneByName(fmt.Sprintf("%s%sIk", pmx.MLIB_PREFIX, upper2Bone.Name()))
	upper2IkBone.Position = neckBone.Position
	upper2IkBone.Ik = pmx.NewIk()
	upper2IkBone.Ik.BoneIndex = neckBone.Index()
	upper2IkBone.Ik.LoopCount = 20
	upper2IkBone.Ik.UnitRotation = mmath.NewMRotationFromRadians(&mmath.MVec3{X: 2, Y: 0, Z: 0})
	upper2IkBone.Ik.Links = make([]*pmx.IkLink, 1)
	upper2IkBone.Ik.Links[0] = pmx.NewIkLink()
	upper2IkBone.Ik.Links[0].BoneIndex = upper2Bone.Index()

	// 首IK
	neckIkBone := pmx.NewBoneByName(fmt.Sprintf("%s%sIk", pmx.MLIB_PREFIX, neckBone.Name()))
	neckIkBone.Position = headBone.Position
	neckIkBone.Ik = pmx.NewIk()
	neckIkBone.Ik.BoneIndex = headBone.Index()
	neckIkBone.Ik.LoopCount = 20
	neckIkBone.Ik.UnitRotation = mmath.NewMRotationFromRadians(&mmath.MVec3{X: 2, Y: 0, Z: 0})
	neckIkBone.Ik.Links = make([]*pmx.IkLink, 1)
	neckIkBone.Ik.Links[0] = pmx.NewIkLink()
	neckIkBone.Ik.Links[0].BoneIndex = neckBone.Index()

	// 頭IK
	headIkBone := pmx.NewBoneByName(fmt.Sprintf("%s%sIk", pmx.MLIB_PREFIX, headBone.Name()))
	headIkBone.Position = headTailBone.Position
	headIkBone.Ik = pmx.NewIk()
	headIkBone.Ik.BoneIndex = headTailBone.Index()
	headIkBone.Ik.LoopCount = 20
	headIkBone.Ik.UnitRotation = mmath.NewMRotationFromRadians(&mmath.MVec3{X: 2, Y: 0, Z: 0})
	headIkBone.Ik.Links = make([]*pmx.IkLink, 1)
	headIkBone.Ik.Links[0] = pmx.NewIkLink()
	headIkBone.Ik.Links[0].BoneIndex = headBone.Index()

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
	sizingUpperIkDeltas := make([]*delta.VmdDeltas, len(frames))

	upper2Positions := make([]*mmath.MVec3, len(frames))
	neckPositions := make([]*mmath.MVec3, len(frames))
	headPositions := make([]*mmath.MVec3, len(frames))

	// 先モデルのデフォーム(IK OFF)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, false, frame, trunk_upper_bone_names, false)
		sizingOffDeltas[index] = vmdDeltas

		// 体幹中心から見た上半身2の相対位置を取得
		originalUpperDelta := originalAllDeltas[index].Bones.Get(originalUpperBone.Index())
		originalUpper2Delta := originalAllDeltas[index].Bones.Get(originalUpper2Bone.Index())

		originalUpper2LocalPosition := originalUpper2Delta.FilledGlobalPosition().Subed(
			originalUpperDelta.FilledGlobalPosition())

		// 体幹中心から見た首根元の相対位置を取得
		sizingUpperRootDelta := sizingOffDeltas[index].Bones.Get(upperRootBone.Index())

		upper2Positions[index] = sizingUpperRootDelta.FilledGlobalPosition().Added(
			originalUpper2LocalPosition.Muled(upperRatio))
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

	// ------------------------------
	sizingUpper2IkDeltas := make([]*delta.VmdDeltas, len(frames))

	// 先モデルの上半身2追加補正(IK ON)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, false, frame, trunk_upper_bone_names, false)
		sizingUpper2IkDeltas[index] = vmdDeltas

		// 上半身から見た上半身2の相対位置を取得
		originalUpper2Delta := originalAllDeltas[index].Bones.Get(originalUpper2Bone.Index())
		originalNeckDelta := originalAllDeltas[index].Bones.Get(originalNeckBone.Index())

		originalNeckLocalPosition := originalNeckDelta.FilledGlobalPosition().Subed(
			originalUpper2Delta.FilledGlobalPosition())

		sizingUpper2Delta := sizingUpper2IkDeltas[index].Bones.Get(upper2Bone.Index())
		neckPositions[index] = sizingUpper2Delta.FilledGlobalPosition().Added(
			originalNeckLocalPosition.Muled(upper2Ratio))
		sizingUpper2IkDeltas[index] = deform.DeformIk(
			sizingModel, sizingMotion, frame, upperIkBone, neckPositions[index])
	})

	// 補正を登録
	for i, iFrame := range frames {
		frame := float32(iFrame)

		upper2Bf := sizingMotion.BoneFrames.Get(upper2Bone.Name()).Get(frame)
		upper2Bf.Rotation = sizingUpper2IkDeltas[i].Bones.Get(upper2Bone.Index()).FilledFrameRotation()
		sizingMotion.InsertRegisteredBoneFrame(upper2Bone.Name(), upper2Bf)
	}

	// ------------------------------
	sizingNeckIkDeltas := make([]*delta.VmdDeltas, len(frames))

	// 先モデルの首追加補正(IK ON)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, false, frame, trunk_upper_bone_names, false)
		sizingNeckIkDeltas[index] = vmdDeltas

		// 上半身から見た上半身2の相対位置を取得
		originalNeckDelta := originalAllDeltas[index].Bones.Get(originalNeckBone.Index())
		originalHeadDelta := originalAllDeltas[index].Bones.Get(originalHeadBone.Index())

		originalHeadLocalPosition := originalHeadDelta.FilledGlobalPosition().Subed(
			originalNeckDelta.FilledGlobalPosition())

		sizingNeckDelta := sizingNeckIkDeltas[index].Bones.Get(neckBone.Index())
		neckPositions[index] = sizingNeckDelta.FilledGlobalPosition().Added(
			originalHeadLocalPosition.Muled(neckRatio))
		sizingNeckIkDeltas[index] = deform.DeformIk(
			sizingModel, sizingMotion, frame, neckIkBone, neckPositions[index])
	})

	// 補正を登録
	for i, iFrame := range frames {
		frame := float32(iFrame)

		neckBf := sizingMotion.BoneFrames.Get(neckBone.Name()).Get(frame)
		neckBf.Rotation = sizingNeckIkDeltas[i].Bones.Get(neckBone.Index()).FilledFrameRotation()
		sizingMotion.InsertRegisteredBoneFrame(neckBone.Name(), neckBf)
	}

	// ------------------------------
	sizingHeadIkDeltas := make([]*delta.VmdDeltas, len(frames))

	// 先モデルの頭追加補正(IK ON)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, false, frame, trunk_upper_bone_names, false)
		sizingHeadIkDeltas[index] = vmdDeltas

		// 頭から見た頭先の相対位置を取得
		originalHeadDelta := originalAllDeltas[index].Bones.Get(originalHeadBone.Index())
		originalHeadTailDelta := originalAllDeltas[index].Bones.Get(originalHeadTailBone.Index())

		originalHeadTailLocalPosition := originalHeadTailDelta.FilledGlobalPosition().Subed(
			originalHeadDelta.FilledGlobalPosition())

		sizingHeadDelta := sizingHeadIkDeltas[index].Bones.Get(headBone.Index())
		headPositions[index] = sizingHeadDelta.FilledGlobalPosition().Added(
			originalHeadTailLocalPosition.Muled(headRatio))
		sizingHeadIkDeltas[index] = deform.DeformIk(
			sizingModel, sizingMotion, frame, headIkBone, headPositions[index])
	})

	// 補正を登録
	for i, iFrame := range frames {
		frame := float32(iFrame)

		headBf := sizingMotion.BoneFrames.Get(headBone.Name()).Get(frame)
		headBf.Rotation = sizingHeadIkDeltas[i].Bones.Get(headBone.Index()).FilledFrameRotation()
		sizingMotion.InsertRegisteredBoneFrame(headBone.Name(), headBf)
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

	return true
}
