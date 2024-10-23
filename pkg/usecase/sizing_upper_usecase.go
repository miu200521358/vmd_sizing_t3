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

func SizingUpper(sizingSet *domain.SizingSet, setSize int) (bool, error) {
	if !sizingSet.IsSizingUpper || (sizingSet.IsSizingUpper && sizingSet.CompletedSizingUpper) {
		return false, nil
	}

	if !isValidSizingUpper(sizingSet) {
		return false, nil
	}

	originalModel := sizingSet.OriginalPmx
	originalMotion := sizingSet.OriginalVmd
	sizingModel := sizingSet.SizingPmx
	sizingMotion := sizingSet.OutputVmd

	originalUpperRootBone := originalModel.Bones.GetByName(pmx.UPPER_ROOT.String())
	originalUpperBone := originalModel.Bones.GetByName(pmx.UPPER.String())
	// originalUpper2Bone := originalModel.Bones.GetByName(pmx.UPPER2.String())
	// originalLeftShoulderBone := originalModel.Bones.GetByName(pmx.SHOULDER.Left())
	// originalRightShoulderBone := originalModel.Bones.GetByName(pmx.SHOULDER.Right())
	originalNeckRootBone := originalModel.Bones.GetByName(pmx.NECK_ROOT.String())
	// originalNeckBone := originalModel.Bones.GetByName(pmx.NECK.String())
	// originalHeadBone := originalModel.Bones.GetByName(pmx.HEAD.String())
	// originalHeadTailBone := originalModel.Bones.GetByName(pmx.HEAD_TAIL.String())

	sizingUpperRootBone := sizingModel.Bones.GetByName(pmx.UPPER_ROOT.String())
	sizingUpperBone := sizingModel.Bones.GetByName(pmx.UPPER.String())
	// sizingUpper2Bone := sizingModel.Bones.GetByName(pmx.UPPER2.String())
	sizingNeckRootBone := sizingModel.Bones.GetByName(pmx.NECK_ROOT.String())
	// sizingLeftShoulderBone := sizingModel.Bones.GetByName(pmx.SHOULDER.Left())
	// sizingRightShoulderBone := sizingModel.Bones.GetByName(pmx.SHOULDER.Right())
	sizingNeckBone := sizingModel.Bones.GetByName(pmx.NECK.String())
	// sizingHeadBone := sizingModel.Bones.GetByName(pmx.HEAD.String())
	// sizingHeadTailBone := sizingModel.Bones.GetByName(pmx.HEAD_TAIL.String())

	// upperScaleY := math.Abs(sizingNeckRootLocalPosition.Y / originalNeckRootLocalPosition.Y)
	// upperScaleZ := math.Abs(sizingNeckRootLocalPosition.Z / originalNeckRootLocalPosition.Z)

	// // 上半身全体のサイズ差
	// originalUpperLength2 := originalNeckRootLocalPosition.Length()
	// sizingUpperLength2 := sizingNeckRootLocalPosition.Length()
	// upperScale2 := sizingUpperLength2 / originalUpperLength2
	// mlog.V("upperScale2: %.4f", upperScale2)

	// // 上半身全体のサイズ差
	// originalUpperLength := 0.0
	// for _, boneIndex := range originalNeckRootBone.Extend.ParentBoneIndexes {
	// 	bone := originalModel.Bones.Get(boneIndex)
	// 	if bone.Name() == pmx.UPPER_ROOT.String() {
	// 		break
	// 	}
	// 	originalUpperLength += math.Abs(bone.Extend.ParentRelativePosition.Y) + math.Abs(bone.Extend.ParentRelativePosition.Z)
	// }

	// upperScale := 1.0
	// sizingUpperLength := 0.0
	upperBoneNames := make([]string, 0)
	upperBones := make([]*pmx.Bone, 0)
	for _, boneIndex := range sizingNeckRootBone.Extend.ParentBoneIndexes {
		bone := sizingModel.Bones.Get(boneIndex)
		if bone.Name() == pmx.UPPER_ROOT.String() {
			break
		}
		// sizingUpperLength += math.Abs(bone.Extend.ParentRelativePosition.Y) + math.Abs(bone.Extend.ParentRelativePosition.Z)
		if bone.IsEffectorRotation() || bone.IsEffectorTranslation() || bone.IsIK() || !bone.CanManipulate() {
			// 付与親あり・Ik・操作不可は無視
			continue
		}
		upperBones = append(upperBones, bone)
		upperBoneNames = append(upperBoneNames, bone.Name())

		// if originalModel.Bones.ContainsByName(bone.Name()) && bone.Extend.ParentRelativePosition.Y > 0 &&
		// 	originalModel.Bones.GetByName(bone.Name()).Extend.ParentRelativePosition.Y > 0 {
		// 	upperScale *= bone.Extend.ParentRelativePosition.Y / originalModel.Bones.GetByName(bone.Name()).Extend.ParentRelativePosition.Y
		// }
	}
	upperBoneNames = append(upperBoneNames, sizingNeckRootBone.Name())
	// upperScale *= upperTotalScale

	// 上半身根元から見た首根元の相対位置
	originalNeckRootLocalPosition := originalNeckRootBone.Position.Subed(originalUpperRootBone.Position)
	sizingNeckRootLocalPosition := sizingNeckRootBone.Position.Subed(sizingUpperRootBone.Position)
	upperScales := &mmath.MVec3{
		X: 1.0,
		Y: sizingNeckRootLocalPosition.Length() / originalNeckRootLocalPosition.Length(),
		Z: 1.0,
	}

	// 上半身根元から見た首根元の角度差
	originalUpperDirection := originalNeckRootLocalPosition.Normalized()
	sizingUpperDirection := sizingNeckRootLocalPosition.Normalized()
	sizingUpperSlopeMat := mmath.NewMQuaternionRotate(originalUpperDirection, sizingUpperDirection).ToMat4()

	// 上半身IK
	upperIkBone := pmx.NewBoneByName(fmt.Sprintf("%s%sIk", pmx.MLIB_PREFIX, pmx.UPPER.String()))
	upperIkBone.Position = sizingNeckRootBone.Position
	upperIkBone.Ik = pmx.NewIk()
	upperIkBone.Ik.BoneIndex = sizingNeckRootBone.Index()
	upperIkBone.Ik.LoopCount = 100
	upperIkBone.Ik.UnitRotation = mmath.NewMRotationFromDegrees(&mmath.MVec3{X: 1, Y: 0, Z: 0})
	upperIkBone.Ik.Links = make([]*pmx.IkLink, len(upperBones))
	for i, bone := range upperBones {
		upperIkBone.Ik.Links[i] = pmx.NewIkLink()
		upperIkBone.Ik.Links[i].BoneIndex = bone.Index()
	}

	frames := sizingMotion.BoneFrames.RegisteredFrames(trunk_upper_bone_names)
	blockSize, _ := miter.GetBlockSize(len(frames) * setSize)

	if len(frames) == 0 {
		return false, nil
	}

	mlog.I(mi18n.T("上半身補正開始", map[string]interface{}{"No": sizingSet.Index + 1}))
	sizingMotion.Processing = true

	originalAllDeltas := make([]*delta.VmdDeltas, len(frames))

	// 元モデルのデフォーム(IK ON)
	if err := miter.IterParallelByList(frames, blockSize, func(data, index int) {
		frame := float32(data)
		vmdDeltas := delta.NewVmdDeltas(frame, originalModel.Bones, originalModel.Hash(), originalMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(originalModel, originalMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(originalModel, originalMotion, vmdDeltas, true, frame, trunk_upper_bone_names, false)
		originalAllDeltas[index] = vmdDeltas
	}, func(iterIndex, allCount int) {
		mlog.I(mi18n.T("上半身補正01", map[string]interface{}{"No": sizingSet.Index + 1, "IterIndex": fmt.Sprintf("%02d", iterIndex), "AllCount": fmt.Sprintf("%02d", allCount)}))
	}); err != nil {
		sizingMotion.Processing = false
		return false, err
	}

	sizingUpperRotations := make([][]*mmath.MQuaternion, len(upperBones))
	for i := range sizingUpperRotations {
		sizingUpperRotations[i] = make([]*mmath.MQuaternion, len(frames))
	}

	sizingLeftShoulderRotations := make([]*mmath.MQuaternion, len(frames))
	sizingRightShoulderRotations := make([]*mmath.MQuaternion, len(frames))
	sizingNeckRotations := make([]*mmath.MQuaternion, len(frames))

	// 先モデルの上半身デフォーム(IK ON)
	if err := miter.IterParallelByList(frames, blockSize, func(data, index int) {
		frame := float32(data)
		vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, true, frame, trunk_upper_bone_names, false)

		// 上半身から見た首根元の相対位置を取得
		originalUpperDelta := originalAllDeltas[index].Bones.Get(originalUpperBone.Index())
		originalNeckRootDelta := originalAllDeltas[index].Bones.Get(originalNeckRootBone.Index())

		originalUpperLocalPosition := originalUpperDelta.FilledGlobalMatrix().Inverted().MulVec3(originalNeckRootDelta.FilledGlobalPosition())
		// sizingUpperLocalPosition := &mmath.MVec3{
		// 	X: originalUpperLocalPosition.X,
		// 	Y: originalUpperLocalPosition.Y * upperScaleY,
		// 	Z: originalUpperLocalPosition.Z * upperScaleZ,
		// }
		sizingUpperLocalPosition := originalUpperLocalPosition.Muled(upperScales)
		sizingUpperSlopeLocalPosition := sizingUpperSlopeMat.MulVec3(sizingUpperLocalPosition)

		sizingUpperDelta := vmdDeltas.Bones.Get(sizingUpperBone.Index())
		neckRootFixGlobalPosition := sizingUpperDelta.FilledGlobalMatrix().MulVec3(sizingUpperSlopeLocalPosition)

		sizingUpperIkDeltas := deform.DeformIk(sizingModel, sizingMotion, vmdDeltas, frame, upperIkBone, neckRootFixGlobalPosition, upperBoneNames)

		nowLeftShoulderBf := sizingMotion.BoneFrames.Get(pmx.SHOULDER.Left()).Get(frame)
		nowRightShoulderBf := sizingMotion.BoneFrames.Get(pmx.SHOULDER.Right()).Get(frame)
		nowNeckBf := sizingMotion.BoneFrames.Get(sizingNeckBone.Name()).Get(frame)

		sizingLeftShoulderRotations[index] = nowLeftShoulderBf.Rotation
		if sizingLeftShoulderRotations[index] == nil {
			sizingLeftShoulderRotations[index] = mmath.NewMQuaternion()
		}

		sizingRightShoulderRotations[index] = nowRightShoulderBf.Rotation
		if sizingRightShoulderRotations[index] == nil {
			sizingRightShoulderRotations[index] = mmath.NewMQuaternion()
		}

		sizingNeckRotations[index] = nowNeckBf.Rotation
		if sizingNeckRotations[index] == nil {
			sizingNeckRotations[index] = mmath.NewMQuaternion()
		}

		for n, bone := range upperBones {
			sizingUpperRotations[n][index] = sizingUpperIkDeltas.Bones.Get(bone.Index()).FilledFrameRotation()

			nowUpperBf := sizingMotion.BoneFrames.Get(bone.Name()).Get(frame)

			// 首・肩は逆補正をかける
			var upperDiffRotation *mmath.MQuaternion
			if nowUpperBf.Rotation != nil {
				upperDiffRotation = nowUpperBf.Rotation.Inverted().Muled(sizingUpperRotations[n][index]).Inverted()
			} else {
				upperDiffRotation = mmath.MQuaternionIdent.Inverted().Muled(sizingUpperRotations[n][index]).Inverted()
			}

			sizingLeftShoulderRotations[index] = upperDiffRotation.Muled(sizingLeftShoulderRotations[index])
			sizingRightShoulderRotations[index] = upperDiffRotation.Muled(sizingRightShoulderRotations[index])
			sizingNeckRotations[index] = upperDiffRotation.Muled(sizingNeckRotations[index])
		}
	}, func(iterIndex, allCount int) {
		mlog.I(mi18n.T("上半身補正02", map[string]interface{}{"No": sizingSet.Index + 1, "Scale": fmt.Sprintf("%.4f", upperScales.Y), "IterIndex": fmt.Sprintf("%02d", iterIndex), "AllCount": fmt.Sprintf("%02d", allCount)}))
	}); err != nil {
		sizingMotion.Processing = false
		return false, err
	}

	// 補正を登録
	for i, iFrame := range frames {
		frame := float32(iFrame)

		for n, bone := range upperBones {
			upperBf := sizingMotion.BoneFrames.Get(bone.Name()).Get(frame)
			upperBf.Rotation = sizingUpperRotations[n][i]
			sizingMotion.InsertRegisteredBoneFrame(bone.Name(), upperBf)
		}

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
	sizingMotion.Processing = false
	return true, nil
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
