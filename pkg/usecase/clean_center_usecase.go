package usecase

import (
	"github.com/miu200521358/mlib_go/pkg/domain/delta"
	"github.com/miu200521358/mlib_go/pkg/domain/miter"
	"github.com/miu200521358/mlib_go/pkg/domain/mmath"
	"github.com/miu200521358/mlib_go/pkg/domain/pmx"
	"github.com/miu200521358/mlib_go/pkg/domain/vmd"
	"github.com/miu200521358/mlib_go/pkg/infrastructure/deform"
	"github.com/miu200521358/mlib_go/pkg/mutils/mi18n"
	"github.com/miu200521358/mlib_go/pkg/mutils/mlog"
	"github.com/miu200521358/vmd_sizing_t3/pkg/domain"
)

func CleanCenter(sizingSet *domain.SizingSet) {
	if !sizingSet.IsCleanCenter || (sizingSet.IsCleanCenter && sizingSet.CompletedCleanCenter) {
		return
	}

	if !isValidCleanCenter(sizingSet) {
		return
	}

	mlog.I(mi18n.T("センター最適化開始", map[string]interface{}{"No": sizingSet.Index + 1}))

	sizingModel := sizingSet.SizingPmx
	sizingMotion := sizingSet.OutputVmd

	centerBone := sizingModel.Bones.GetByName(pmx.CENTER.String())
	grooveBone := sizingModel.Bones.GetByName(pmx.GROOVE.String())
	upperBone := sizingModel.Bones.GetByName(pmx.UPPER.String())
	lowerBone := sizingModel.Bones.GetByName(pmx.LOWER.String())

	centerRelativeBoneNames := []string{pmx.CENTER.String(), pmx.GROOVE.String()}
	frames := sizingMotion.BoneFrames.RegisteredFrames(centerRelativeBoneNames)

	centerPositions := make([]*mmath.MVec3, len(frames))
	groovePositions := make([]*mmath.MVec3, len(frames))
	centerCurves := make([]*vmd.BoneCurves, len(frames))
	grooveCurves := make([]*vmd.BoneCurves, len(frames))

	if !sizingMotion.BoneFrames.ContainsActive(grooveBone.Name()) {
		// グルーブが無い場合、センターのY移動補間曲線をグルーブにコピー
		for i, frame := range sizingMotion.BoneFrames.Get(centerBone.Name()).Indexes.List() {
			centerBf := sizingMotion.BoneFrames.Get(centerBone.Name()).Get(frame)
			centerPositions[i] = &mmath.MVec3{X: centerBf.Position.X, Y: 0, Z: centerBf.Position.Z}
			centerCurves[i] = &vmd.BoneCurves{
				TranslateX: centerBf.Curves.TranslateX.Copy(),
				TranslateY: mmath.LINER_CURVE,
				TranslateZ: centerBf.Curves.TranslateZ.Copy(),
				Rotate:     mmath.LINER_CURVE,
			}

			grooveBf := vmd.NewBoneFrame(frame)
			groovePositions[i] = &mmath.MVec3{X: 0, Y: centerBf.Position.Y, Z: 0}
			grooveCurves[i] = &vmd.BoneCurves{
				TranslateX: mmath.LINER_CURVE,
				TranslateY: centerBf.Curves.TranslateY.Copy(),
				TranslateZ: mmath.LINER_CURVE,
				Rotate:     mmath.LINER_CURVE,
			}

			sizingMotion.InsertRegisteredBoneFrame(grooveBone.Name(), grooveBf)
		}

		for i, frame := range sizingMotion.BoneFrames.Get(centerBone.Name()).Indexes.List() {
			centerBf := sizingMotion.BoneFrames.Get(centerBone.Name()).Get(frame)
			centerBf.Position = centerPositions[i]
			centerBf.Curves = centerCurves[i]
			sizingMotion.InsertRegisteredBoneFrame(centerBone.Name(), centerBf)

			grooveBf := sizingMotion.BoneFrames.Get(grooveBone.Name()).Get(frame)
			grooveBf.Position = groovePositions[i]
			grooveBf.Curves = grooveCurves[i]
			sizingMotion.InsertRegisteredBoneFrame(grooveBone.Name(), grooveBf)
		}
	}

	upperRotations := make([]*mmath.MQuaternion, len(frames))
	lowerRotations := make([]*mmath.MQuaternion, len(frames))

	// 元モデルのデフォーム(IK ON)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, true, frame, centerRelativeBoneNames, false)

		centerLocalPosition := vmdDeltas.Bones.Get(centerBone.Index()).FilledGlobalPosition().Subed(centerBone.Position)
		grooveLocalPosition := vmdDeltas.Bones.Get(grooveBone.Index()).FilledGlobalPosition().Subed(
			vmdDeltas.Bones.Get(centerBone.Index()).FilledGlobalPosition()).Subed(
			grooveBone.Position.Subed(centerBone.Position))

		centerPositions[index] = &mmath.MVec3{X: centerLocalPosition.X + grooveLocalPosition.X,
			Y: 0, Z: centerLocalPosition.Z + grooveLocalPosition.Z}
		groovePositions[index] = &mmath.MVec3{X: 0, Y: centerLocalPosition.Y + grooveLocalPosition.Y, Z: 0}

		upperLocalRotation := sizingMotion.BoneFrames.Get(upperBone.Name()).Get(frame).Rotation
		for _, boneIndex := range upperBone.Extend.ParentBoneIndexes {
			boneDelta := vmdDeltas.Bones.Get(boneIndex)
			if boneDelta == nil {
				continue
			}
			upperLocalRotation = boneDelta.FilledFrameRotation().Muled(upperLocalRotation)
		}
		upperRotations[index] = upperLocalRotation

		lowerLocalRotation := sizingMotion.BoneFrames.Get(lowerBone.Name()).Get(frame).Rotation
		for _, boneIndex := range lowerBone.Extend.ParentBoneIndexes {
			boneDelta := vmdDeltas.Bones.Get(boneIndex)
			if boneDelta == nil {
				continue
			}
			lowerLocalRotation = boneDelta.FilledFrameRotation().Muled(lowerLocalRotation)
		}
		lowerRotations[index] = lowerLocalRotation
	})

	for j, iFrame := range frames {
		frame := float32(iFrame)

		centerBf := sizingMotion.BoneFrames.Get(pmx.CENTER.String()).Get(frame)
		centerBf.Position = centerPositions[j]
		centerBf.Rotation = mmath.NewMQuaternion()
		sizingMotion.InsertRegisteredBoneFrame(pmx.CENTER.String(), centerBf)

		grooveBf := sizingMotion.BoneFrames.Get(pmx.GROOVE.String()).Get(frame)
		grooveBf.Position = groovePositions[j]
		grooveBf.Rotation = mmath.NewMQuaternion()
		sizingMotion.InsertRegisteredBoneFrame(pmx.GROOVE.String(), grooveBf)

		upperBf := sizingMotion.BoneFrames.Get(pmx.UPPER.String()).Get(frame)
		upperBf.Rotation = upperRotations[j]
		sizingMotion.InsertRegisteredBoneFrame(pmx.UPPER.String(), upperBf)

		lowerBf := sizingMotion.BoneFrames.Get(pmx.LOWER.String()).Get(frame)
		lowerBf.Rotation = lowerRotations[j]
		sizingMotion.InsertRegisteredBoneFrame(pmx.LOWER.String(), lowerBf)
	}

	sizingSet.CompletedCleanCenter = true
}

func isValidCleanCenter(sizingSet *domain.SizingSet) bool {
	sizingModel := sizingSet.SizingPmx

	if !sizingModel.Bones.ContainsByName(pmx.CENTER.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("センター最適化ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.CENTER.String()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.GROOVE.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("センター最適化ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.GROOVE.String()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.UPPER.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("センター最適化ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.UPPER.String()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.LOWER.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("センター最適化ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.LOWER.String()}))
		return false
	}

	return true
}
