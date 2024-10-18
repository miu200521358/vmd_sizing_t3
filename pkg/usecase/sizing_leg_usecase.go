package usecase

import (
	"fmt"
	"slices"

	"github.com/miu200521358/mlib_go/pkg/domain/delta"
	"github.com/miu200521358/mlib_go/pkg/domain/miter"
	"github.com/miu200521358/mlib_go/pkg/domain/mmath"
	"github.com/miu200521358/mlib_go/pkg/domain/pmx"
	"github.com/miu200521358/mlib_go/pkg/domain/vmd"
	"github.com/miu200521358/mlib_go/pkg/infrastructure/deform"
	"github.com/miu200521358/mlib_go/pkg/infrastructure/repository"
	"github.com/miu200521358/mlib_go/pkg/mutils"
	"github.com/miu200521358/mlib_go/pkg/mutils/mi18n"
	"github.com/miu200521358/mlib_go/pkg/mutils/mlog"
	"github.com/miu200521358/vmd_sizing_t3/pkg/domain"
)

func SizingLeg(sizingSet *domain.SizingSet, scale *mmath.MVec3) {
	if !sizingSet.IsSizingLeg || (sizingSet.IsSizingLeg && sizingSet.CompletedSizingLeg) {
		return
	}

	if !isValidSizingLower(sizingSet) {
		return
	}

	mlog.I(mi18n.T("足補正開始", map[string]interface{}{"No": sizingSet.Index + 1}))

	originalModel := sizingSet.OriginalPmx
	originalMotion := sizingSet.OriginalVmd
	sizingModel := sizingSet.SizingPmx
	sizingMotion := sizingSet.OutputVmd

	// ------------

	originalLowerRootBone := originalModel.Bones.GetByName(pmx.LOWER_ROOT.String())
	// originalLowerBone := originalModel.Bones.GetByName(pmx.LOWER.String())
	// originalLegCenterBone := originalModel.Bones.GetByName(pmx.LEG_CENTER.String())

	// originalLeftLegIkBone := originalModel.Bones.GetByName(pmx.LEG_IK.Left())
	originalLeftLegBone := originalModel.Bones.GetByName(pmx.LEG.Left())
	originalLeftKneeBone := originalModel.Bones.GetByName(pmx.KNEE.Left())
	originalLeftAnkleBone := originalModel.Bones.GetIkTarget(pmx.LEG_IK.Left())
	// originalLeftHeelBone := originalModel.Bones.GetByName(pmx.HEEL.Left())
	// originalLeftToeIkBone := originalModel.Bones.GetByName(pmx.TOE_IK.Left())
	// originalLeftToeBone := originalModel.Bones.GetIkTarget(pmx.TOE_IK.Left())
	originalLeftToeTailBone := originalModel.Bones.GetByName(pmx.TOE_T.Left())

	// originalRightLegIkBone := originalModel.Bones.GetByName(pmx.LEG_IK.Right())
	originalRightLegBone := originalModel.Bones.GetByName(pmx.LEG.Right())
	originalRightKneeBone := originalModel.Bones.GetByName(pmx.KNEE.Right())
	originalRightAnkleBone := originalModel.Bones.GetIkTarget(pmx.LEG_IK.Right())
	// originalRightHeelBone := originalModel.Bones.GetByName(pmx.HEEL.Right())
	// originalRightToeIkBone := originalModel.Bones.GetByName(pmx.TOE_IK.Right())
	// originalRightToeBone := originalModel.Bones.GetIkTarget(pmx.TOE_IK.Right())
	originalRightToeTailBone := originalModel.Bones.GetByName(pmx.TOE_T.Right())

	// ------------

	sizingCenterBone := sizingModel.Bones.GetByName(pmx.CENTER.String())
	sizingGrooveBone := sizingModel.Bones.GetByName(pmx.GROOVE.String())

	sizingLowerRootBone := sizingModel.Bones.GetByName(pmx.LOWER_ROOT.String())
	sizingLowerBone := sizingModel.Bones.GetByName(pmx.LOWER.String())
	sizingLegCenterBone := sizingModel.Bones.GetByName(pmx.LEG_CENTER.String())

	// sizingLeftLegIkParentBone := sizingModel.Bones.GetByName(pmx.LEG_IK_PARENT.Left())
	sizingLeftLegIkBone := sizingModel.Bones.GetByName(pmx.LEG_IK.Left())
	sizingLeftLegBone := sizingModel.Bones.GetByName(pmx.LEG.Left())
	sizingLeftKneeBone := sizingModel.Bones.GetByName(pmx.KNEE.Left())
	sizingLeftAnkleBone := sizingModel.Bones.GetIkTarget(pmx.LEG_IK.Left())
	// sizingLeftHeelBone := sizingModel.Bones.GetByName(pmx.HEEL.Left())
	sizingLeftToeIkBone := sizingModel.Bones.GetByName(pmx.TOE_IK.Left())
	sizingLeftToeBone := sizingModel.Bones.GetIkTarget(pmx.TOE_IK.Left())
	sizingLeftToeTailBone := sizingModel.Bones.GetByName(pmx.TOE_T.Left())

	// sizingRightLegIkParentBone := sizingModel.Bones.GetByName(pmx.LEG_IK_PARENT.Right())
	sizingRightLegIkBone := sizingModel.Bones.GetByName(pmx.LEG_IK.Right())
	sizingRightLegBone := sizingModel.Bones.GetByName(pmx.LEG.Right())
	sizingRightKneeBone := sizingModel.Bones.GetByName(pmx.KNEE.Right())
	sizingRightAnkleBone := sizingModel.Bones.GetIkTarget(pmx.LEG_IK.Right())
	// sizingRightHeelBone := sizingModel.Bones.GetByName(pmx.HEEL.Right())
	sizingRightToeIkBone := sizingModel.Bones.GetByName(pmx.TOE_IK.Right())
	sizingRightToeBone := sizingModel.Bones.GetIkTarget(pmx.TOE_IK.Right())
	sizingRightToeTailBone := sizingModel.Bones.GetByName(pmx.TOE_T.Right())

	// 下半身IK
	lowerIkBone := pmx.NewBoneByName(fmt.Sprintf("%s%sIk", pmx.MLIB_PREFIX, sizingLowerBone.Name()))
	lowerIkBone.Position = sizingLegCenterBone.Position
	lowerIkBone.Ik = pmx.NewIk()
	lowerIkBone.Ik.BoneIndex = sizingLegCenterBone.Index()
	lowerIkBone.Ik.LoopCount = 10
	lowerIkBone.Ik.UnitRotation = mmath.NewMRotationFromDegrees(&mmath.MVec3{X: 180, Y: 0, Z: 0})
	lowerIkBone.Ik.Links = make([]*pmx.IkLink, 1)
	lowerIkBone.Ik.Links[0] = pmx.NewIkLink()
	lowerIkBone.Ik.Links[0].BoneIndex = sizingLowerBone.Index()

	// 左足IK
	leftLegIkBone := pmx.NewBoneByName(fmt.Sprintf("%s%sIk", pmx.MLIB_PREFIX, sizingLeftLegBone.Name()))
	leftLegIkBone.Position = sizingLeftKneeBone.Position
	leftLegIkBone.Ik = pmx.NewIk()
	leftLegIkBone.Ik.BoneIndex = sizingLeftKneeBone.Index()
	leftLegIkBone.Ik.LoopCount = 10
	leftLegIkBone.Ik.UnitRotation = mmath.NewMRotationFromDegrees(&mmath.MVec3{X: 180, Y: 0, Z: 0})
	leftLegIkBone.Ik.Links = make([]*pmx.IkLink, 1)
	leftLegIkBone.Ik.Links[0] = pmx.NewIkLink()
	leftLegIkBone.Ik.Links[0].BoneIndex = sizingLeftLegBone.Index()

	// 左ひざIK（この時点では角度制限なし）
	leftKneeIkBone := pmx.NewBoneByName(fmt.Sprintf("%s%sIk", pmx.MLIB_PREFIX, sizingLeftKneeBone.Name()))
	leftKneeIkBone.Position = sizingLeftAnkleBone.Position
	leftKneeIkBone.Ik = pmx.NewIk()
	leftKneeIkBone.Ik.BoneIndex = sizingLeftAnkleBone.Index()
	leftKneeIkBone.Ik.LoopCount = 10
	leftKneeIkBone.Ik.UnitRotation = mmath.NewMRotationFromDegrees(&mmath.MVec3{X: 180, Y: 0, Z: 0})
	leftKneeIkBone.Ik.Links = make([]*pmx.IkLink, 1)
	leftKneeIkBone.Ik.Links[0] = pmx.NewIkLink()
	leftKneeIkBone.Ik.Links[0].BoneIndex = sizingLeftKneeBone.Index()

	// 左足首IK
	leftAnkleIkBone := pmx.NewBoneByName(fmt.Sprintf("%s%sIk", pmx.MLIB_PREFIX, sizingLeftAnkleBone.Name()))
	leftAnkleIkBone.Position = sizingLeftToeTailBone.Position
	leftAnkleIkBone.Ik = pmx.NewIk()
	leftAnkleIkBone.Ik.BoneIndex = sizingLeftToeTailBone.Index()
	leftAnkleIkBone.Ik.LoopCount = 10
	leftAnkleIkBone.Ik.UnitRotation = mmath.NewMRotationFromDegrees(&mmath.MVec3{X: 180, Y: 0, Z: 0})
	leftAnkleIkBone.Ik.Links = make([]*pmx.IkLink, 1)
	leftAnkleIkBone.Ik.Links[0] = pmx.NewIkLink()
	leftAnkleIkBone.Ik.Links[0].BoneIndex = sizingLeftAnkleBone.Index()

	// 右足IK
	rightLegIkBone := pmx.NewBoneByName(fmt.Sprintf("%s%sIk", pmx.MLIB_PREFIX, sizingRightLegBone.Name()))
	rightLegIkBone.Position = sizingRightKneeBone.Position
	rightLegIkBone.Ik = pmx.NewIk()
	rightLegIkBone.Ik.BoneIndex = sizingRightKneeBone.Index()
	rightLegIkBone.Ik.LoopCount = 10
	rightLegIkBone.Ik.UnitRotation = mmath.NewMRotationFromDegrees(&mmath.MVec3{X: 180, Y: 0, Z: 0})
	rightLegIkBone.Ik.Links = make([]*pmx.IkLink, 1)
	rightLegIkBone.Ik.Links[0] = pmx.NewIkLink()
	rightLegIkBone.Ik.Links[0].BoneIndex = sizingRightLegBone.Index()

	// 右ひざIK（この時点では角度制限なし）
	rightKneeIkBone := pmx.NewBoneByName(fmt.Sprintf("%s%sIk", pmx.MLIB_PREFIX, sizingRightKneeBone.Name()))
	rightKneeIkBone.Position = sizingRightAnkleBone.Position
	rightKneeIkBone.Ik = pmx.NewIk()
	rightKneeIkBone.Ik.BoneIndex = sizingRightAnkleBone.Index()
	rightKneeIkBone.Ik.LoopCount = 10
	rightKneeIkBone.Ik.UnitRotation = mmath.NewMRotationFromDegrees(&mmath.MVec3{X: 180, Y: 0, Z: 0})
	rightKneeIkBone.Ik.Links = make([]*pmx.IkLink, 1)
	rightKneeIkBone.Ik.Links[0] = pmx.NewIkLink()
	rightKneeIkBone.Ik.Links[0].BoneIndex = sizingRightKneeBone.Index()

	// 右足首IK
	rightAnkleIkBone := pmx.NewBoneByName(fmt.Sprintf("%s%sIk", pmx.MLIB_PREFIX, sizingRightAnkleBone.Name()))
	rightAnkleIkBone.Position = sizingRightToeTailBone.Position
	rightAnkleIkBone.Ik = pmx.NewIk()
	rightAnkleIkBone.Ik.BoneIndex = sizingRightToeTailBone.Index()
	rightAnkleIkBone.Ik.LoopCount = 10
	rightAnkleIkBone.Ik.UnitRotation = mmath.NewMRotationFromDegrees(&mmath.MVec3{X: 180, Y: 0, Z: 0})
	rightAnkleIkBone.Ik.Links = make([]*pmx.IkLink, 1)
	rightAnkleIkBone.Ik.Links[0] = pmx.NewIkLink()
	rightAnkleIkBone.Ik.Links[0].BoneIndex = sizingRightAnkleBone.Index()

	// // 下半身の傾き度合いの補正
	// originalLowerDirection := originalLegCenterBone.Position.Subed(originalLowerRootBone.Position).Normalized()
	// sizingLowerDirection := sizingLegCenterBone.Position.Subed(sizingLowerRootBone.Position).Normalized()
	// sizingLowerSlopeMat := sizingLowerDirection.ToLocalMat().Muled(originalLowerDirection.ToLocalMat().Inverted())

	// // 下半身根元から地面の間に足中心がどの辺りに位置しているか
	// originalLegCenterRatio := originalLegCenterBone.Position.Y / originalLowerRootBone.Position.Y
	// sizingLegCenterRatio := sizingLegCenterBone.Position.Y / sizingLowerRootBone.Position.Y
	// legCenterPositionRatio := sizingLegCenterRatio / originalLegCenterRatio

	// 下半身根元から地面の間に足がどの辺りに位置しているか
	originalLegRatio := originalLeftLegBone.Position.Y / originalLowerRootBone.Position.Y
	sizingLegRatio := sizingLeftLegBone.Position.Y / sizingLowerRootBone.Position.Y
	legPositionRatio := sizingLegRatio / originalLegRatio

	// 下半身根元から地面の間にひざがどの辺りに位置しているか
	originalKneeRatio := originalLeftKneeBone.Position.Y / originalLowerRootBone.Position.Y
	sizingKneeRatio := sizingLeftKneeBone.Position.Y / sizingLowerRootBone.Position.Y
	kneePositionRatio := sizingKneeRatio / originalKneeRatio

	// 下半身根元から地面の間に足首がどの辺りに位置しているか
	originalAnkleRatio := originalLeftAnkleBone.Position.Y / originalLowerRootBone.Position.Y
	sizingAnkleRatio := sizingLeftAnkleBone.Position.Y / sizingLowerRootBone.Position.Y
	anklePositionRatio := sizingAnkleRatio / originalAnkleRatio

	originalLeftLegDirection := originalLeftLegBone.Position.Subed(originalLeftKneeBone.Position).Normalized()
	sizingLeftLegDirection := sizingLeftLegBone.Position.Subed(sizingLeftKneeBone.Position).Normalized()
	sizingLeftLegSlopeMat := mmath.NewMQuaternionRotate(originalLeftLegDirection, sizingLeftLegDirection).ToMat4()

	originalLeftKneeDirection := originalLeftKneeBone.Position.Subed(originalLeftAnkleBone.Position).Normalized()
	sizingLeftKneeDirection := sizingLeftKneeBone.Position.Subed(sizingLeftAnkleBone.Position).Normalized()
	sizingLeftKneeSlopeMat := mmath.NewMQuaternionRotate(originalLeftKneeDirection, sizingLeftKneeDirection).ToMat4()

	originalLeftAnkleDirection := originalLeftAnkleBone.Position.Subed(originalLeftToeTailBone.Position).Normalized()
	sizingLeftAnkleDirection := sizingLeftAnkleBone.Position.Subed(sizingLeftToeTailBone.Position).Normalized()
	sizingLeftAnkleSlopeMat := mmath.NewMQuaternionRotate(originalLeftAnkleDirection, sizingLeftAnkleDirection).ToMat4()

	originalRightLegDirection := originalRightLegBone.Position.Subed(originalRightKneeBone.Position).Normalized()
	sizingRightLegDirection := sizingRightLegBone.Position.Subed(sizingRightKneeBone.Position).Normalized()
	sizingRightLegSlopeMat := mmath.NewMQuaternionRotate(originalRightLegDirection, sizingRightLegDirection).ToMat4()

	originalRightKneeDirection := originalRightKneeBone.Position.Subed(originalRightAnkleBone.Position).Normalized()
	sizingRightKneeDirection := sizingRightKneeBone.Position.Subed(sizingRightAnkleBone.Position).Normalized()
	sizingRightKneeSlopeMat := mmath.NewMQuaternionRotate(originalRightKneeDirection, sizingRightKneeDirection).ToMat4()

	originalRightAnkleDirection := originalRightAnkleBone.Position.Subed(originalRightToeTailBone.Position).Normalized()
	sizingRightAnkleDirection := sizingRightAnkleBone.Position.Subed(sizingRightToeTailBone.Position).Normalized()
	sizingRightAnkleSlopeMat := mmath.NewMQuaternionRotate(originalRightAnkleDirection, sizingRightAnkleDirection).ToMat4()

	// 下半身全体のサイズ差
	originalLowerLength := originalLowerRootBone.Position.Length()
	sizingLowerLength := sizingLowerRootBone.Position.Length()
	lowerTotalRatio := sizingLowerLength / originalLowerLength

	// // 下半身スケール
	// originalLowerVector := originalLowerRootBone.Position.Subed(originalLegCenterBone.Position).Round(1e-2)
	// sizingLowerVector := sizingLowerRootBone.Position.Subed(sizingLegCenterBone.Position).Round(1e-2)
	// lowerScale := sizingLowerVector.Length() / originalLowerVector.Length() * legCenterPositionRatio * lowerTotalRatio

	// 足スケール
	originalLegVector := originalLeftLegBone.Position.Subed(originalLeftKneeBone.Position).Round(1e-2)
	sizingLegVector := sizingLeftLegBone.Position.Subed(sizingLeftKneeBone.Position).Round(1e-2)
	legScale := sizingLegVector.Length() / originalLegVector.Length() * legPositionRatio * lowerTotalRatio

	// ひざスケール
	originalKneeVector := originalLeftKneeBone.Position.Subed(originalLeftAnkleBone.Position).Round(1e-2)
	sizingKneeVector := sizingLeftKneeBone.Position.Subed(sizingLeftAnkleBone.Position).Round(1e-2)
	kneeScale := sizingKneeVector.Length() / originalKneeVector.Length() * kneePositionRatio * lowerTotalRatio

	// 足首スケール
	originalAnkleVector := originalLeftAnkleBone.Position.Subed(originalLeftToeTailBone.Position).Round(1e-2)
	sizingAnkleVector := sizingLeftAnkleBone.Position.Subed(sizingLeftToeTailBone.Position).Round(1e-2)
	ankleScale := sizingAnkleVector.Length() / originalAnkleVector.Length() * anklePositionRatio * lowerTotalRatio

	mlog.I(mi18n.T("足補正01", map[string]interface{}{"No": sizingSet.Index + 1}))

	frames := sizingMotion.BoneFrames.RegisteredFrames(all_lower_leg_bone_names)
	originalAllDeltas := make([]*delta.VmdDeltas, len(frames))

	// 元モデルのデフォーム(IK ON)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		// 最適化結果のサイジングモーションを使用する
		vmdDeltas := delta.NewVmdDeltas(frame, originalModel.Bones, originalModel.Hash(), sizingMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(originalModel, sizingMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(originalModel, sizingMotion, vmdDeltas, true, frame, all_limb_bone_names, false)
		originalAllDeltas[index] = vmdDeltas
	})

	// サイジング先にFKを焼き込み
	for _, vmdDeltas := range originalAllDeltas {
		for _, boneDelta := range vmdDeltas.Bones.Data {
			if boneDelta == nil || !boneDelta.Bone.IsLegFK() {
				continue
			}

			sizingBf := sizingMotion.BoneFrames.Get(boneDelta.Bone.Name()).Get(boneDelta.Frame)

			// 最終的な足FKを焼き込み
			sizingBf.Rotation = boneDelta.FilledFrameRotation()
			sizingMotion.InsertRegisteredBoneFrame(boneDelta.Bone.Name(), sizingBf)
		}
	}

	if mlog.IsVerbose() {
		kf := vmd.NewIkFrame(0)
		kf.Registered = true
		{
			kef := vmd.NewIkEnableFrame(0)
			kef.BoneName = sizingLeftLegIkBone.Name()
			kef.Enabled = false
			kf.IkList = append(kf.IkList, kef)
		}
		{
			kef := vmd.NewIkEnableFrame(0)
			kef.BoneName = sizingLeftToeIkBone.Name()
			kef.Enabled = false
			kf.IkList = append(kf.IkList, kef)
		}
		{
			kef := vmd.NewIkEnableFrame(0)
			kef.BoneName = sizingRightLegIkBone.Name()
			kef.Enabled = false
			kf.IkList = append(kf.IkList, kef)
		}
		{
			kef := vmd.NewIkEnableFrame(0)
			kef.BoneName = sizingRightToeIkBone.Name()
			kef.Enabled = false
			kf.IkList = append(kf.IkList, kef)
		}
		sizingMotion.InsertIkFrame(kf)

		outputPath := mutils.CreateOutputPath(sizingSet.OriginalVmdPath, "足補正01_FK焼き込み")
		repository.NewVmdRepository().Save(outputPath, sizingMotion, true)
		mlog.V("足補正01_FK焼き込み: %s", outputPath)
	}

	// sizingLowerIkAllDeltas := make([]*delta.VmdDeltas, len(frames))

	// sizingLowerRotations := make([]*mmath.MQuaternion, len(frames))
	sizingLeftLegRotations := make([]*mmath.MQuaternion, len(frames))
	sizingLeftKneeRotations := make([]*mmath.MQuaternion, len(frames))
	sizingLeftAnkleRotations := make([]*mmath.MQuaternion, len(frames))
	sizingRightLegRotations := make([]*mmath.MQuaternion, len(frames))
	sizingRightKneeRotations := make([]*mmath.MQuaternion, len(frames))
	sizingRightAnkleRotations := make([]*mmath.MQuaternion, len(frames))

	// mlog.I(mi18n.T("足補正02", map[string]interface{}{"No": sizingSet.Index + 1, "Scale": fmt.Sprintf("%.4f", lowerScale)}))

	// // 先モデルの下半身デフォーム(IK ON)
	// miter.IterParallelByList(frames, 500, func(data, index int) {
	// 	frame := float32(data)
	// 	vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
	// 	vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
	// 	vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, true, frame, trunk_lower_bone_names, false)

	// 	var diffLowerRotation *mmath.MQuaternion
	// 	sizingLowerIkAllDeltas[index], diffLowerRotation = deformLegIk(index, frame, sizingModel, sizingMotion,
	// 		originalAllDeltas, vmdDeltas, originalLowerRootBone, originalLegCenterBone,
	// 		sizingLowerRootBone, sizingLegCenterBone, lowerIkBone, sizingLowerSlopeMat, lowerScale)

	// 	sizingLowerRotations[index] = sizingLowerIkAllDeltas[index].Bones.Get(sizingLowerBone.Index()).FilledFrameRotation()

	// 	originalLeftLegRotation := originalAllDeltas[index].Bones.Get(originalLeftLegBone.Index()).FilledFrameRotation()
	// 	originalRightLegRotation := originalAllDeltas[index].Bones.Get(originalRightLegBone.Index()).FilledFrameRotation()

	// 	sizingLeftLegRotations[index] = originalLeftLegRotation.Muled(diffLowerRotation)
	// 	sizingRightLegRotations[index] = originalRightLegRotation.Muled(diffLowerRotation)
	// })

	// // 補正を登録
	// for i, iFrame := range frames {
	// 	frame := float32(iFrame)

	// 	lowerBf := sizingMotion.BoneFrames.Get(sizingLowerBone.Name()).Get(frame)
	// 	lowerBf.Rotation = sizingLowerRotations[i]
	// 	sizingMotion.InsertRegisteredBoneFrame(sizingLowerBone.Name(), lowerBf)

	// 	leftLegBf := sizingMotion.BoneFrames.Get(sizingLeftLegBone.Name()).Get(frame)
	// 	leftLegBf.Rotation = sizingLeftLegRotations[i]
	// 	sizingMotion.InsertRegisteredBoneFrame(sizingLeftLegBone.Name(), leftLegBf)

	// 	rightLegBf := sizingMotion.BoneFrames.Get(sizingRightLegBone.Name()).Get(frame)
	// 	rightLegBf.Rotation = sizingRightLegRotations[i]
	// 	sizingMotion.InsertRegisteredBoneFrame(sizingRightLegBone.Name(), rightLegBf)
	// }

	// if mlog.IsVerbose() {
	// 	title := "足補正02_下半身補正"
	// 	outputPath := mutils.CreateOutputPath(sizingSet.OriginalVmdPath, title)
	// 	repository.NewVmdRepository().Save(outputPath, sizingMotion, true)
	// 	mlog.V("%s: %s", title, outputPath)
	// }

	mlog.I(mi18n.T("足補正03", map[string]interface{}{"No": sizingSet.Index + 1, "Scale": fmt.Sprintf("%.4f", legScale)}))

	sizingLeftLegIkAllDeltas := make([]*delta.VmdDeltas, len(frames))
	sizingRightLegIkAllDeltas := make([]*delta.VmdDeltas, len(frames))

	// 先モデルの足デフォーム(IK ON)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)
		vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, true, frame, trunk_lower_bone_names, false)

		// 左足から見た左ひざの相対位置を取得
		var diffLeftLegRotation *mmath.MQuaternion
		sizingLeftLegIkAllDeltas[index], diffLeftLegRotation, _ = deformLegIk(index, frame, sizingModel, sizingMotion,
			originalAllDeltas, vmdDeltas, originalLeftLegBone, originalLeftKneeBone,
			sizingLeftLegBone, sizingLeftKneeBone, leftLegIkBone, sizingLeftLegSlopeMat, legScale)

		sizingLeftLegRotations[index] = sizingLeftLegIkAllDeltas[index].Bones.Get(sizingLeftLegBone.Index()).FilledFrameRotation()

		originalLeftKneeRotation := originalAllDeltas[index].Bones.Get(originalLeftKneeBone.Index()).FilledFrameRotation()
		sizingLeftKneeRotations[index] = originalLeftKneeRotation.Muled(diffLeftLegRotation)

		// 右足から見た右ひざの相対位置を取得
		var diffRightLegRotation *mmath.MQuaternion
		sizingRightLegIkAllDeltas[index], diffRightLegRotation, _ = deformLegIk(index, frame, sizingModel, sizingMotion,
			originalAllDeltas, vmdDeltas, originalRightLegBone, originalRightKneeBone,
			sizingRightLegBone, sizingRightKneeBone, rightLegIkBone, sizingRightLegSlopeMat, legScale)

		sizingRightLegRotations[index] = sizingRightLegIkAllDeltas[index].Bones.Get(sizingRightLegBone.Index()).FilledFrameRotation()

		originalRightKneeRotation := originalAllDeltas[index].Bones.Get(originalRightKneeBone.Index()).FilledFrameRotation()
		sizingRightKneeRotations[index] = originalRightKneeRotation.Muled(diffRightLegRotation)
	})

	// 補正を登録
	for i, iFrame := range frames {
		frame := float32(iFrame)

		leftLegBf := sizingMotion.BoneFrames.Get(sizingLeftLegBone.Name()).Get(frame)
		leftLegBf.Rotation = sizingLeftLegRotations[i]
		sizingMotion.InsertRegisteredBoneFrame(sizingLeftLegBone.Name(), leftLegBf)

		rightLegBf := sizingMotion.BoneFrames.Get(sizingRightLegBone.Name()).Get(frame)
		rightLegBf.Rotation = sizingRightLegRotations[i]
		sizingMotion.InsertRegisteredBoneFrame(sizingRightLegBone.Name(), rightLegBf)

		leftKneeBf := sizingMotion.BoneFrames.Get(sizingLeftKneeBone.Name()).Get(frame)
		leftKneeBf.Rotation = sizingLeftKneeRotations[i]
		sizingMotion.InsertRegisteredBoneFrame(sizingLeftKneeBone.Name(), leftKneeBf)

		rightKneeBf := sizingMotion.BoneFrames.Get(sizingRightKneeBone.Name()).Get(frame)
		rightKneeBf.Rotation = sizingRightKneeRotations[i]
		sizingMotion.InsertRegisteredBoneFrame(sizingRightKneeBone.Name(), rightKneeBf)
	}

	if mlog.IsVerbose() {
		title := "足補正03_足補正"
		outputPath := mutils.CreateOutputPath(sizingSet.OriginalVmdPath, title)
		repository.NewVmdRepository().Save(outputPath, sizingMotion, true)
		mlog.V("%s: %s", title, outputPath)
	}

	mlog.I(mi18n.T("足補正04", map[string]interface{}{"No": sizingSet.Index + 1, "Scale": fmt.Sprintf("%.4f", legScale)}))

	sizingLeftKneeIkAllDeltas := make([]*delta.VmdDeltas, len(frames))
	sizingRightKneeIkAllDeltas := make([]*delta.VmdDeltas, len(frames))
	sizingLeftAnkleFixGlobalPositions := make([]*mmath.MVec3, len(frames))
	sizingRightAnkleFixGlobalPositions := make([]*mmath.MVec3, len(frames))

	// 先モデルの足デフォーム(IK ON)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)

		// 左ひざから見た左足首の相対位置を取得
		var diffLeftKneeRotation *mmath.MQuaternion
		sizingLeftKneeIkAllDeltas[index], diffLeftKneeRotation, sizingLeftAnkleFixGlobalPositions[index] =
			deformLegIk(index, frame, sizingModel, sizingMotion,
				originalAllDeltas, sizingLeftLegIkAllDeltas[index], originalLeftKneeBone, originalLeftAnkleBone,
				sizingLeftKneeBone, sizingLeftAnkleBone, leftKneeIkBone, sizingLeftKneeSlopeMat, kneeScale)

		sizingLeftKneeRotations[index] = sizingLeftKneeIkAllDeltas[index].Bones.Get(sizingLeftKneeBone.Index()).FilledFrameRotation()

		originalLeftAnkleRotation := originalAllDeltas[index].Bones.Get(originalLeftAnkleBone.Index()).FilledFrameRotation()
		sizingLeftAnkleRotations[index] = originalLeftAnkleRotation.Muled(diffLeftKneeRotation)

		// 右ひざから見た右足首の相対位置を取得
		var diffRightKneeRotation *mmath.MQuaternion
		sizingRightKneeIkAllDeltas[index], diffRightKneeRotation, sizingRightAnkleFixGlobalPositions[index] =
			deformLegIk(index, frame, sizingModel, sizingMotion,
				originalAllDeltas, sizingRightLegIkAllDeltas[index], originalRightKneeBone, originalRightAnkleBone,
				sizingRightKneeBone, sizingRightAnkleBone, rightKneeIkBone, sizingRightKneeSlopeMat, kneeScale)

		sizingRightKneeRotations[index] = sizingRightKneeIkAllDeltas[index].Bones.Get(sizingRightKneeBone.Index()).FilledFrameRotation()

		originalRightAnkleRotation := originalAllDeltas[index].Bones.Get(originalRightAnkleBone.Index()).FilledFrameRotation()
		sizingRightAnkleRotations[index] = originalRightAnkleRotation.Muled(diffRightKneeRotation)
	})

	// 補正を登録
	for i, iFrame := range frames {
		frame := float32(iFrame)

		leftKneeBf := sizingMotion.BoneFrames.Get(sizingLeftKneeBone.Name()).Get(frame)
		leftKneeBf.Rotation = sizingLeftKneeRotations[i]
		sizingMotion.InsertRegisteredBoneFrame(sizingLeftKneeBone.Name(), leftKneeBf)

		rightKneeBf := sizingMotion.BoneFrames.Get(sizingRightKneeBone.Name()).Get(frame)
		rightKneeBf.Rotation = sizingRightKneeRotations[i]
		sizingMotion.InsertRegisteredBoneFrame(sizingRightKneeBone.Name(), rightKneeBf)

		leftAnkleBf := sizingMotion.BoneFrames.Get(sizingLeftAnkleBone.Name()).Get(frame)
		leftAnkleBf.Rotation = sizingLeftAnkleRotations[i]
		sizingMotion.InsertRegisteredBoneFrame(sizingLeftAnkleBone.Name(), leftAnkleBf)

		rightAnkleBf := sizingMotion.BoneFrames.Get(sizingRightAnkleBone.Name()).Get(frame)
		rightAnkleBf.Rotation = sizingRightAnkleRotations[i]
		sizingMotion.InsertRegisteredBoneFrame(sizingRightAnkleBone.Name(), rightAnkleBf)
	}

	if mlog.IsVerbose() {
		title := "足補正04_ひざ補正"
		outputPath := mutils.CreateOutputPath(sizingSet.OriginalVmdPath, title)
		repository.NewVmdRepository().Save(outputPath, sizingMotion, true)
		mlog.V("%s: %s", title, outputPath)
	}

	mlog.I(mi18n.T("足補正05", map[string]interface{}{"No": sizingSet.Index + 1, "Scale": fmt.Sprintf("%.4f", legScale)}))

	sizingLeftAnkleIkAllDeltas := make([]*delta.VmdDeltas, len(frames))
	sizingRightAnkleIkAllDeltas := make([]*delta.VmdDeltas, len(frames))
	leftLegIkPositions := make([]*mmath.MVec3, len(frames))
	rightLegIkPositions := make([]*mmath.MVec3, len(frames))

	// 先モデルの足デフォーム(IK ON)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)

		// 左足首から見た左つま先の相対位置を取得
		sizingLeftAnkleIkAllDeltas[index], _, _ = deformLegIk(index, frame, sizingModel, sizingMotion,
			originalAllDeltas, sizingLeftKneeIkAllDeltas[index], originalLeftAnkleBone, originalLeftToeTailBone,
			sizingLeftAnkleBone, sizingLeftToeTailBone, leftAnkleIkBone, sizingLeftAnkleSlopeMat, ankleScale)

		sizingLeftAnkleRotations[index] = sizingLeftAnkleIkAllDeltas[index].Bones.Get(sizingLeftAnkleBone.Index()).FilledFrameRotation()

		// 右足首から見た右つま先の相対位置を取得
		sizingRightAnkleIkAllDeltas[index], _, _ = deformLegIk(index, frame, sizingModel, sizingMotion,
			originalAllDeltas, sizingRightLegIkAllDeltas[index], originalRightAnkleBone, originalRightToeTailBone,
			sizingRightAnkleBone, sizingRightToeTailBone, rightAnkleIkBone, sizingRightAnkleSlopeMat, ankleScale)

		sizingRightAnkleRotations[index] = sizingRightAnkleIkAllDeltas[index].Bones.Get(sizingRightAnkleBone.Index()).FilledFrameRotation()

		leftAnkleDelta := sizingLeftAnkleIkAllDeltas[index].Bones.GetByName(pmx.ANKLE.Left())
		rightAnkleDelta := sizingRightAnkleIkAllDeltas[index].Bones.GetByName(pmx.ANKLE.Right())

		// 足IKから見た足首の位置
		leftLegIkPositions[index] = leftAnkleDelta.FilledGlobalPosition().Subed(sizingLeftLegIkBone.Position)
		rightLegIkPositions[index] = rightAnkleDelta.FilledGlobalPosition().Subed(sizingRightLegIkBone.Position)
	})

	// 補正を登録
	for i, iFrame := range frames {
		frame := float32(iFrame)

		leftAnkleBf := sizingMotion.BoneFrames.Get(sizingLeftAnkleBone.Name()).Get(frame)
		leftAnkleBf.Rotation = sizingLeftAnkleRotations[i]
		sizingMotion.InsertRegisteredBoneFrame(sizingLeftAnkleBone.Name(), leftAnkleBf)

		rightAnkleBf := sizingMotion.BoneFrames.Get(sizingRightAnkleBone.Name()).Get(frame)
		rightAnkleBf.Rotation = sizingRightAnkleRotations[i]
		sizingMotion.InsertRegisteredBoneFrame(sizingRightAnkleBone.Name(), rightAnkleBf)

		rightLegIkBf := sizingMotion.BoneFrames.Get(sizingRightLegIkBone.Name()).Get(frame)
		rightLegIkBf.Position = rightLegIkPositions[i]
		sizingMotion.InsertRegisteredBoneFrame(sizingRightLegIkBone.Name(), rightLegIkBf)

		leftLegIkBf := sizingMotion.BoneFrames.Get(sizingLeftLegIkBone.Name()).Get(frame)
		leftLegIkBf.Position = leftLegIkPositions[i]
		sizingMotion.InsertRegisteredBoneFrame(sizingLeftLegIkBone.Name(), leftLegIkBf)
	}

	if mlog.IsVerbose() {
		title := "足補正05_足首補正"
		outputPath := mutils.CreateOutputPath(sizingSet.OriginalVmdPath, title)
		repository.NewVmdRepository().Save(outputPath, sizingMotion, true)
		mlog.V("%s: %s", title, outputPath)
	}

	// mlog.I(mi18n.T("足補正06", map[string]interface{}{"No": sizingSet.Index + 1}))

	// sizingLeftFkAllDeltas := make([]*delta.VmdDeltas, len(frames))
	// sizingRightFkAllDeltas := make([]*delta.VmdDeltas, len(frames))

	// // 足IK再計算
	// miter.IterParallelByList(frames, 500, func(data, index int) {
	// 	frame := float32(data)

	// 	sizingLeftFkAllDeltas[index] = deform.DeformIk(
	// 		sizingModel, sizingMotion, sizingLeftAnkleIkAllDeltas[index], frame,
	// 		sizingLeftLegIkBone, sizingLeftAnkleFixGlobalPositions[index],
	// 		[]string{sizingLeftToeTailBone.Name(), sizingLeftHeelBone.Name()})

	// 	sizingRightFkAllDeltas[index] = deform.DeformIk(
	// 		sizingModel, sizingMotion, sizingRightAnkleIkAllDeltas[index], frame,
	// 		sizingRightLegIkBone, sizingRightAnkleFixGlobalPositions[index],
	// 		[]string{sizingRightToeTailBone.Name(), sizingRightHeelBone.Name()})

	// 	leftLegRotations[index] = sizingLeftFkAllDeltas[index].Bones.Get(sizingLeftLegBone.Index()).FilledFrameRotation()
	// 	leftKneeRotations[index] = sizingLeftFkAllDeltas[index].Bones.Get(sizingLeftKneeBone.Index()).FilledFrameRotation()
	// 	leftAnkleRotations[index] = sizingLeftFkAllDeltas[index].Bones.Get(sizingLeftAnkleBone.Index()).FilledFrameRotation()

	// 	rightLegRotations[index] = sizingRightFkAllDeltas[index].Bones.Get(sizingRightLegBone.Index()).FilledFrameRotation()
	// 	rightKneeRotations[index] = sizingRightFkAllDeltas[index].Bones.Get(sizingRightKneeBone.Index()).FilledFrameRotation()
	// 	rightAnkleRotations[index] = sizingRightFkAllDeltas[index].Bones.Get(sizingRightAnkleBone.Index()).FilledFrameRotation()
	// })

	// registerLegFk(frames, sizingMotion, sizingLeftLegBone, sizingLeftKneeBone, sizingLeftAnkleBone, sizingRightLegBone,
	// 	sizingRightKneeBone, sizingRightAnkleBone, leftLegRotations, leftKneeRotations, leftAnkleRotations,
	// 	rightLegRotations, rightKneeRotations, rightAnkleRotations)

	// if mlog.IsVerbose() {
	// 	title := "足補正06_足IK再計算"
	// 	outputPath := mutils.CreateOutputPath(sizingSet.OriginalVmdPath, title)
	// 	repository.NewVmdRepository().Save(outputPath, sizingMotion, true)
	// 	mlog.V("%s: %s", title, outputPath)
	// }

	mlog.I(mi18n.T("足補正07", map[string]interface{}{"No": sizingSet.Index + 1}))

	centerPositions := make([]*mmath.MVec3, len(frames))
	groovePositions := make([]*mmath.MVec3, len(frames))

	centerTargetBones := []*pmx.Bone{
		sizingModel.Bones.GetByName(pmx.UPPER.String()), sizingModel.Bones.GetByName(pmx.UPPER2.String()),
		sizingModel.Bones.GetByName(pmx.NECK_ROOT.String()), sizingModel.Bones.GetByName(pmx.NECK.String()),
		sizingModel.Bones.GetByName(pmx.HEAD.String()), sizingModel.Bones.GetByName(pmx.SHOULDER.Left()),
		sizingModel.Bones.GetByName(pmx.ARM.Left()), sizingModel.Bones.GetByName(pmx.ELBOW.Left()),
		sizingModel.Bones.GetByName(pmx.WRIST.Left()), sizingModel.Bones.GetByName(pmx.SHOULDER.Right()),
		sizingModel.Bones.GetByName(pmx.ARM.Right()), sizingModel.Bones.GetByName(pmx.ELBOW.Right()),
		sizingModel.Bones.GetByName(pmx.WRIST.Right()), sizingModel.Bones.GetByName(pmx.LOWER.String()),
		sizingModel.Bones.GetByName(pmx.LEG.Left()), sizingModel.Bones.GetByName(pmx.KNEE.Left()),
		sizingModel.Bones.GetByName(pmx.ANKLE.Left()), sizingModel.Bones.GetByName(pmx.LEG.Right()),
		sizingModel.Bones.GetByName(pmx.KNEE.Right()), sizingModel.Bones.GetByName(pmx.ANKLE.Right())}

	// 先モデルのデフォーム
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)

		// 各関節の最も地面に近い位置からセンターを計算する
		centerTargetYs := []float64{
			originalAllDeltas[index].Bones.GetByName(pmx.UPPER.String()).FilledGlobalPosition().Y,
			originalAllDeltas[index].Bones.GetByName(pmx.UPPER2.String()).FilledGlobalPosition().Y,
			originalAllDeltas[index].Bones.GetByName(pmx.NECK_ROOT.String()).FilledGlobalPosition().Y,
			originalAllDeltas[index].Bones.GetByName(pmx.NECK.String()).FilledGlobalPosition().Y,
			originalAllDeltas[index].Bones.GetByName(pmx.HEAD.String()).FilledGlobalPosition().Y,
			originalAllDeltas[index].Bones.GetByName(pmx.SHOULDER.Left()).FilledGlobalPosition().Y,
			originalAllDeltas[index].Bones.GetByName(pmx.ARM.Left()).FilledGlobalPosition().Y,
			originalAllDeltas[index].Bones.GetByName(pmx.ELBOW.Left()).FilledGlobalPosition().Y,
			originalAllDeltas[index].Bones.GetByName(pmx.WRIST.Left()).FilledGlobalPosition().Y,
			originalAllDeltas[index].Bones.GetByName(pmx.SHOULDER.Right()).FilledGlobalPosition().Y,
			originalAllDeltas[index].Bones.GetByName(pmx.ARM.Right()).FilledGlobalPosition().Y,
			originalAllDeltas[index].Bones.GetByName(pmx.ELBOW.Right()).FilledGlobalPosition().Y,
			originalAllDeltas[index].Bones.GetByName(pmx.WRIST.Right()).FilledGlobalPosition().Y,
			originalAllDeltas[index].Bones.GetByName(pmx.LOWER.String()).FilledGlobalPosition().Y,
			originalAllDeltas[index].Bones.GetByName(pmx.LEG.Left()).FilledGlobalPosition().Y,
			originalAllDeltas[index].Bones.GetByName(pmx.KNEE.Left()).FilledGlobalPosition().Y,
			originalAllDeltas[index].Bones.GetByName(pmx.ANKLE.Left()).FilledGlobalPosition().Y,
			originalAllDeltas[index].Bones.GetByName(pmx.LEG.Right()).FilledGlobalPosition().Y,
			originalAllDeltas[index].Bones.GetByName(pmx.KNEE.Right()).FilledGlobalPosition().Y,
			originalAllDeltas[index].Bones.GetByName(pmx.ANKLE.Right()).FilledGlobalPosition().Y,
		}

		// 最もY位置が低い関節を処理対象とする
		centerTargetBone := centerTargetBones[mmath.ArgMin(centerTargetYs)]
		originalCenterTargetDelta := originalAllDeltas[index].Bones.GetByName(centerTargetBone.Name())

		var sizingCenterTargetDelta *delta.BoneDelta
		if slices.Contains(all_upper_arm_bone_names, centerTargetBone.Name()) {
			// 上半身系の変形情報
			armVmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
			armVmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
			armVmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, armVmdDeltas, true, frame, all_upper_arm_bone_names, false)
			sizingCenterTargetDelta = armVmdDeltas.Bones.GetByName(centerTargetBone.Name())
		} else if centerTargetBone.Direction() == "左" {
			// 左足系
			sizingCenterTargetDelta = sizingLeftAnkleIkAllDeltas[index].Bones.GetByName(centerTargetBone.Name())
		} else {
			// 下半身か右足系
			sizingCenterTargetDelta = sizingRightAnkleIkAllDeltas[index].Bones.GetByName(centerTargetBone.Name())
		}

		originalCenterTargetY := originalCenterTargetDelta.FilledGlobalPosition().Y
		sizingCenterTargetY := sizingCenterTargetDelta.FilledGlobalPosition().Y

		if centerTargetBone.Name() == pmx.ANKLE.Left() || centerTargetBone.Name() == pmx.ANKLE.Right() {
			originalCenterTargetY -= originalModel.Bones.GetByName(centerTargetBone.Name()).Position.Y
			sizingCenterTargetY -= centerTargetBone.Position.Y
		}

		// 元モデルの対象ボーンのY位置*スケールから補正後のY位置を計算
		sizingFixCenterTargetY := originalCenterTargetY * scale.Y
		yDiff := sizingFixCenterTargetY - sizingCenterTargetY

		mlog.V("足補正07[%.0f][%s] originalY[%.4f], sizingY[%.4f], sizingFixY[%.4f], diff[%.4f]",
			frame, centerTargetBone.Name(), originalCenterTargetY, sizingCenterTargetY, sizingFixCenterTargetY, yDiff)

		// センターの位置をスケールに合わせる
		originalCenterBf := originalMotion.BoneFrames.Get(sizingCenterBone.Name()).Get(frame)
		centerPositions[index] = originalCenterBf.Position.Muled(scale)
		centerPositions[index].Y = 0

		sizingGrooveBf := sizingMotion.BoneFrames.Get(sizingGrooveBone.Name()).Get(frame)
		groovePositions[index] = sizingGrooveBf.Position.Added(&mmath.MVec3{X: 0, Y: yDiff, Z: 0})
	})

	// 補正を登録
	for i, iFrame := range frames {
		frame := float32(iFrame)

		sizingCenterBf := sizingMotion.BoneFrames.Get(sizingCenterBone.Name()).Get(frame)
		sizingCenterBf.Position = centerPositions[i]
		sizingMotion.InsertRegisteredBoneFrame(sizingCenterBone.Name(), sizingCenterBf)

		sizingGrooveBf := sizingMotion.BoneFrames.Get(sizingGrooveBone.Name()).Get(frame)
		sizingGrooveBf.Position = groovePositions[i]
		sizingMotion.InsertRegisteredBoneFrame(sizingGrooveBone.Name(), sizingGrooveBf)
	}

	if mlog.IsVerbose() {
		title := "足補正07_センター補正"
		outputPath := mutils.CreateOutputPath(sizingSet.OriginalVmdPath, title)
		repository.NewVmdRepository().Save(outputPath, sizingMotion, true)
		mlog.V("%s: %s", title, outputPath)
	}

	mlog.I(mi18n.T("足補正08", map[string]interface{}{"No": sizingSet.Index + 1}))

	leftLegIkRotations := make([]*mmath.MQuaternion, len(frames))
	rightLegIkRotations := make([]*mmath.MQuaternion, len(frames))

	// 先モデルのデフォーム(IK OFF+センター補正済み)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)

		vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, false, frame, all_lower_leg_bone_names, false)

		originalLeftAnklePosition := originalAllDeltas[index].Bones.GetByName(pmx.ANKLE.Left()).FilledGlobalPosition()
		originalRightAnklePosition := originalAllDeltas[index].Bones.GetByName(pmx.ANKLE.Right()).FilledGlobalPosition()

		// 地面に近い足底が同じ高さになるように調整
		// originalLegLeftDelta := originalAllDeltas[index].Bones.GetByName(pmx.LEG.Left())
		originalLeftHeelDelta := originalAllDeltas[index].Bones.GetByName(pmx.HEEL.Left())
		originalLeftToeTailDelta := originalAllDeltas[index].Bones.GetByName(pmx.TOE_T.Left())
		// originalLegRightDelta := originalAllDeltas[index].Bones.GetByName(pmx.LEG.Right())
		// originalAnkleRightDelta := originalAllDeltas[index].Bones.GetByName(pmx.ANKLE.Right())
		originalRightHeelDelta := originalAllDeltas[index].Bones.GetByName(pmx.HEEL.Right())
		originalRightToeTailDelta := originalAllDeltas[index].Bones.GetByName(pmx.TOE_T.Right())

		sizingLeftAnkleDelta := vmdDeltas.Bones.GetByName(pmx.ANKLE.Left())
		sizingLeftHeelDelta := vmdDeltas.Bones.GetByName(pmx.HEEL.Left())
		sizingLeftToeDelta := vmdDeltas.Bones.GetByName(sizingLeftToeBone.Name())
		sizingLeftToeTailDelta := vmdDeltas.Bones.GetByName(pmx.TOE_T.Left())

		sizingRightAnkleDelta := vmdDeltas.Bones.GetByName(pmx.ANKLE.Right())
		sizingRightHeelDelta := vmdDeltas.Bones.GetByName(pmx.HEEL.Right())
		sizingRightToeDelta := vmdDeltas.Bones.GetByName(sizingRightToeBone.Name())
		sizingRightToeTailDelta := vmdDeltas.Bones.GetByName(pmx.TOE_T.Right())

		// 足IKから見た足首の位置
		leftLegIkPositions[index] = sizingLeftAnkleDelta.FilledGlobalPosition().Subed(sizingLeftLegIkBone.Position)
		rightLegIkPositions[index] = sizingRightAnkleDelta.FilledGlobalPosition().Subed(sizingRightLegIkBone.Position)

		calcLegIkPositionY(index, frame, "左", leftLegIkPositions, originalLeftAnkleBone, originalLeftAnklePosition,
			originalLeftToeTailDelta, originalLeftHeelDelta, sizingLeftToeTailDelta, sizingLeftHeelDelta, scale)

		calcLegIkPositionY(index, frame, "右", rightLegIkPositions, originalRightAnkleBone, originalRightAnklePosition,
			originalRightToeTailDelta, originalRightHeelDelta, sizingRightToeTailDelta, sizingRightHeelDelta, scale)

		// 足首から見たつま先IKの方向
		leftLegIkMat := sizingLeftToeIkBone.Position.Subed(sizingLeftLegIkBone.Position).Normalize().ToLocalMat()
		leftLegFkMat := sizingLeftToeDelta.FilledGlobalPosition().Subed(
			sizingLeftAnkleDelta.FilledGlobalPosition()).Normalize().ToLocalMat()
		leftLegIkRotations[index] = leftLegFkMat.Muled(leftLegIkMat.Inverted()).Quaternion()

		rightLegIkMat := sizingRightToeIkBone.Position.Subed(sizingRightLegIkBone.Position).Normalize().ToLocalMat()
		rightLegFkMat := sizingRightToeDelta.FilledGlobalPosition().Subed(
			sizingRightAnkleDelta.FilledGlobalPosition()).Normalize().ToLocalMat()
		rightLegIkRotations[index] = rightLegFkMat.Muled(rightLegIkMat.Inverted()).Quaternion()
	})

	for i, iFrame := range frames {
		frame := float32(iFrame)

		originalLeftAnklePosition := originalAllDeltas[i].Bones.GetByName(pmx.ANKLE.Left()).FilledGlobalPosition()
		originalRightAnklePosition := originalAllDeltas[i].Bones.GetByName(pmx.ANKLE.Right()).FilledGlobalPosition()

		if i > 0 {
			originalLeftAnklePrevPosition := originalAllDeltas[i-1].Bones.GetByName(pmx.ANKLE.Left()).FilledGlobalPosition()
			originalRightAnklePrevPosition := originalAllDeltas[i-1].Bones.GetByName(pmx.ANKLE.Right()).FilledGlobalPosition()

			// 前と同じ位置なら同じ位置にする
			if mmath.NearEquals(originalRightAnklePrevPosition.X, originalRightAnklePosition.X, 1e-2) {
				rightLegIkPositions[i].X = rightLegIkPositions[i-1].X
			}
			if mmath.NearEquals(originalRightAnklePrevPosition.Y, originalRightAnklePosition.Y, 1e-2) {
				rightLegIkPositions[i].Y = rightLegIkPositions[i-1].Y
			}
			if mmath.NearEquals(originalRightAnklePrevPosition.Z, originalRightAnklePosition.Z, 1e-2) {
				rightLegIkPositions[i].Z = rightLegIkPositions[i-1].Z
			}

			if mmath.NearEquals(originalLeftAnklePrevPosition.X, originalLeftAnklePosition.X, 1e-2) {
				leftLegIkPositions[i].X = leftLegIkPositions[i-1].X
			}
			if mmath.NearEquals(originalLeftAnklePrevPosition.Y, originalLeftAnklePosition.Y, 1e-2) {
				leftLegIkPositions[i].Y = leftLegIkPositions[i-1].Y
			}
			if mmath.NearEquals(originalLeftAnklePrevPosition.Z, originalLeftAnklePosition.Z, 1e-2) {
				leftLegIkPositions[i].Z = leftLegIkPositions[i-1].Z
			}
		}

		rightLegIkBf := sizingMotion.BoneFrames.Get(sizingRightLegIkBone.Name()).Get(frame)
		rightLegIkBf.Position = rightLegIkPositions[i]
		rightLegIkBf.Rotation = rightLegIkRotations[i]
		sizingMotion.InsertRegisteredBoneFrame(sizingRightLegIkBone.Name(), rightLegIkBf)

		leftLegIkBf := sizingMotion.BoneFrames.Get(sizingLeftLegIkBone.Name()).Get(frame)
		leftLegIkBf.Position = leftLegIkPositions[i]
		leftLegIkBf.Rotation = leftLegIkRotations[i]
		sizingMotion.InsertRegisteredBoneFrame(sizingLeftLegIkBone.Name(), leftLegIkBf)
	}

	if mlog.IsVerbose() {
		sizingMotion.IkFrames.Delete(0)

		title := "足補正08_足IK補正"
		outputPath := mutils.CreateOutputPath(sizingSet.OriginalVmdPath, title)
		repository.NewVmdRepository().Save(outputPath, sizingMotion, true)
		mlog.V("%s: %s", title, outputPath)
	}

	mlog.I(mi18n.T("足補正09", map[string]interface{}{"No": sizingSet.Index + 1}))

	leftLegRotations := make([]*mmath.MQuaternion, len(frames))
	leftKneeRotations := make([]*mmath.MQuaternion, len(frames))
	leftAnkleRotations := make([]*mmath.MQuaternion, len(frames))
	rightLegRotations := make([]*mmath.MQuaternion, len(frames))
	rightKneeRotations := make([]*mmath.MQuaternion, len(frames))
	rightAnkleRotations := make([]*mmath.MQuaternion, len(frames))

	// 足IK再計算
	// 元モデルのデフォーム(IK ON)
	miter.IterParallelByList(frames, 500, func(data, index int) {
		frame := float32(data)

		vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
		vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
		vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, true, frame, all_lower_leg_bone_names, false)

		leftLegRotations[index] = vmdDeltas.Bones.Get(sizingLeftLegBone.Index()).FilledFrameRotation()
		leftKneeRotations[index] = vmdDeltas.Bones.Get(sizingLeftKneeBone.Index()).FilledFrameRotation()
		leftAnkleRotations[index] = vmdDeltas.Bones.Get(sizingLeftAnkleBone.Index()).FilledFrameRotation()

		rightLegRotations[index] = vmdDeltas.Bones.Get(sizingRightLegBone.Index()).FilledFrameRotation()
		rightKneeRotations[index] = vmdDeltas.Bones.Get(sizingRightKneeBone.Index()).FilledFrameRotation()
		rightAnkleRotations[index] = vmdDeltas.Bones.Get(sizingRightAnkleBone.Index()).FilledFrameRotation()
	})

	registerLegFk(frames, sizingMotion, sizingLeftLegBone, sizingLeftKneeBone, sizingLeftAnkleBone, sizingRightLegBone,
		sizingRightKneeBone, sizingRightAnkleBone, leftLegRotations, leftKneeRotations, leftAnkleRotations,
		rightLegRotations, rightKneeRotations, rightAnkleRotations)

	if mlog.IsVerbose() {
		sizingMotion.IkFrames.Delete(0)

		title := "足補正09_FK再計算"
		outputPath := mutils.CreateOutputPath(sizingSet.OriginalVmdPath, title)
		repository.NewVmdRepository().Save(outputPath, sizingMotion, true)
		mlog.V("%s: %s", title, outputPath)
	}

	sizingSet.CompletedSizingLeg = true
}

func calcLegIkPositionY(
	index int,
	frame float32,
	direction string,
	legIkPositions []*mmath.MVec3,
	originalAnkleBone *pmx.Bone,
	originalAnklePosition *mmath.MVec3,
	originalToeTailDelta, originalHeelDelta, sizingToeTailDelta, sizingHeelDelta *delta.BoneDelta,
	scale *mmath.MVec3,
) {

	// 左足IK-Yの位置を調整
	if mmath.NearEquals(originalAnklePosition.Y, 0, 1e-2) {
		legIkPositions[index].Y = 0
		return
	}

	if originalToeTailDelta.FilledGlobalPosition().Y <= originalHeelDelta.FilledGlobalPosition().Y {
		// つま先の方がかかとより低い場合
		originalLeftToeTailY := originalToeTailDelta.FilledGlobalPosition().Y

		// つま先のY座標を元モデルのつま先のY座標*スケールに合わせる
		sizingLeftToeTailY := originalLeftToeTailY * scale.Y

		// 現時点のつま先のY座標
		actualLeftToeTailY := sizingToeTailDelta.FilledGlobalPosition().Y

		leftToeDiff := sizingLeftToeTailY - actualLeftToeTailY
		lerpLeftToeDiff := mmath.LerpFloat(leftToeDiff, 0,
			originalToeTailDelta.FilledGlobalPosition().Y/originalAnkleBone.Position.Y)
		// 足首Y位置に近付くにつれて補正を弱める
		legIkPositions[index].Y += lerpLeftToeDiff
		mlog.V("足補正08[%.0f][%sつま先] originalLeftY[%.4f], sizingLeftY[%.4f], actualLeftY[%.4f], diff[%.4f], lerp[%.4f]",
			frame, direction, originalLeftToeTailY, sizingLeftToeTailY, actualLeftToeTailY, leftToeDiff, lerpLeftToeDiff)

		return
	}

	// かかとの方がつま先より低い場合
	originalLeftHeelY := originalHeelDelta.FilledGlobalPosition().Y

	// かかとのY座標を元モデルのかかとのY座標*スケールに合わせる
	sizingLeftHeelY := originalLeftHeelY * scale.Y

	// 現時点のかかとのY座標
	actualLeftHeelY := sizingHeelDelta.FilledGlobalPosition().Y

	leftHeelDiff := sizingLeftHeelY - actualLeftHeelY
	lerpLeftHeelDiff := mmath.LerpFloat(leftHeelDiff, 0,
		originalHeelDelta.FilledGlobalPosition().Y/originalAnkleBone.Position.Y)
	// 足首Y位置に近付くにつれて補正を弱める
	legIkPositions[index].Y += lerpLeftHeelDiff

	mlog.V("足補正04[%.0f][%sかかと] originalLeftY[%.4f], sizingLeftY[%.4f], actualLeftY[%.4f], diff[%.4f], lerp[%.4f]",
		frame, direction, originalLeftHeelY, sizingLeftHeelY, actualLeftHeelY, leftHeelDiff, lerpLeftHeelDiff)
}

func deformLegIk(
	index int,
	frame float32,
	sizingModel *pmx.PmxModel,
	sizingMotion *vmd.VmdMotion,
	originalAllDeltas []*delta.VmdDeltas,
	sizingDeltas *delta.VmdDeltas,
	originalSrcBone *pmx.Bone,
	originalDstBone *pmx.Bone,
	sizingSrcBone *pmx.Bone,
	sizingDstBone *pmx.Bone,
	sizingIkBone *pmx.Bone,
	sizingSlopeMat *mmath.MMat4,
	scale float64,
) (dstIkDeltas *delta.VmdDeltas, diffSrcRotation *mmath.MQuaternion, sizingFixDstGlobalPosition *mmath.MVec3) {
	// 元から見た先の相対位置を取得
	originalSrcDelta := originalAllDeltas[index].Bones.Get(originalSrcBone.Index())
	originalDstDelta := originalAllDeltas[index].Bones.Get(originalDstBone.Index())

	// 元から見た先の相対位置をスケールに合わせる
	originalSrcLocalPosition := originalDstDelta.FilledGlobalPosition().Subed(originalSrcDelta.FilledGlobalPosition())
	sizingDstLocalPosition := originalSrcLocalPosition.MuledScalar(scale)
	sizingDstSlopeLocalPosition := sizingSlopeMat.MulVec3(sizingDstLocalPosition)

	// Fixさせた新しい先のグローバル位置を取得
	sizingSrcDelta := sizingDeltas.Bones.Get(sizingSrcBone.Index())
	sizingFixDstGlobalPosition = sizingSrcDelta.FilledGlobalPosition().Added(sizingDstSlopeLocalPosition)

	// IK結果を返す
	dstIkDeltas = deform.DeformIk(sizingModel, sizingMotion, sizingDeltas, frame, sizingIkBone,
		sizingFixDstGlobalPosition, []string{sizingSrcBone.Name(), sizingDstBone.Name()})

	originalSrcRotation := originalAllDeltas[index].Bones.Get(originalSrcBone.Index()).FilledFrameRotation()
	sizingSrcRotation := dstIkDeltas.Bones.Get(sizingSrcBone.Index()).FilledFrameRotation()

	// IK結果の回転差分
	diffSrcRotation = sizingSrcRotation.Muled(originalSrcRotation.Inverted()).Inverted()

	return dstIkDeltas, diffSrcRotation, sizingFixDstGlobalPosition
}

func registerLegFk(
	frames []int,
	sizingMotion *vmd.VmdMotion,
	sizingLeftLegBone, sizingLeftKneeBone, sizingLeftAnkleBone,
	sizingRightLegBone, sizingRightKneeBone, sizingRightAnkleBone *pmx.Bone,
	leftLegRotations, leftKneeRotations, leftAnkleRotations,
	rightLegRotations, rightKneeRotations, rightAnkleRotations []*mmath.MQuaternion,
) (sizingLeftFkAllDeltas, sizingRightFkAllDeltas []*delta.VmdDeltas) {

	// サイジング先にFKを焼き込み
	for i, iFrame := range frames {
		frame := float32(iFrame)

		{
			bf := sizingMotion.BoneFrames.Get(sizingLeftLegBone.Name()).Get(frame)
			bf.Rotation = leftLegRotations[i]
			sizingMotion.InsertRegisteredBoneFrame(sizingLeftLegBone.Name(), bf)
		}
		{
			bf := sizingMotion.BoneFrames.Get(sizingLeftKneeBone.Name()).Get(frame)
			bf.Rotation = leftKneeRotations[i]
			sizingMotion.InsertRegisteredBoneFrame(sizingLeftKneeBone.Name(), bf)
		}
		{
			bf := sizingMotion.BoneFrames.Get(sizingLeftAnkleBone.Name()).Get(frame)
			bf.Rotation = leftAnkleRotations[i]
			sizingMotion.InsertRegisteredBoneFrame(sizingLeftAnkleBone.Name(), bf)
		}
		{
			bf := sizingMotion.BoneFrames.Get(sizingRightLegBone.Name()).Get(frame)
			bf.Rotation = rightLegRotations[i]
			sizingMotion.InsertRegisteredBoneFrame(sizingRightLegBone.Name(), bf)
		}
		{
			bf := sizingMotion.BoneFrames.Get(sizingRightKneeBone.Name()).Get(frame)
			bf.Rotation = rightKneeRotations[i]
			sizingMotion.InsertRegisteredBoneFrame(sizingRightKneeBone.Name(), bf)
		}
		{
			bf := sizingMotion.BoneFrames.Get(sizingRightAnkleBone.Name()).Get(frame)
			bf.Rotation = rightAnkleRotations[i]
			sizingMotion.InsertRegisteredBoneFrame(sizingRightAnkleBone.Name(), bf)
		}
	}

	return sizingLeftFkAllDeltas, sizingRightFkAllDeltas
}

func isValidSizingLower(sizingSet *domain.SizingSet) bool {
	originalModel := sizingSet.OriginalPmx
	sizingModel := sizingSet.SizingPmx

	// センター、グルーブ、下半身、右足IK、左足IKが存在するか

	if !originalModel.Bones.ContainsByName(pmx.CENTER.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.CENTER.String()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.GROOVE.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.GROOVE.String()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.LOWER.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.LOWER.String()}))
		return false
	}

	// ------------------------------

	if !originalModel.Bones.ContainsByName(pmx.LEG_IK.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.LEG_IK.Left()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.LEG.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.LEG.Left()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.KNEE.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.KNEE.Left()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.ANKLE.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.ANKLE.Left()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.TOE_IK.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.TOE_IK.Left()}))
		return false
	}

	// ------------------------------

	if !originalModel.Bones.ContainsByName(pmx.LEG_IK.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.LEG_IK.Right()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.LEG.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.LEG.Right()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.KNEE.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.KNEE.Right()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.ANKLE.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.ANKLE.Right()}))
		return false
	}

	if !originalModel.Bones.ContainsByName(pmx.TOE_IK.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.TOE_IK.Left()}))
		return false
	}

	// ------------------------------

	if !sizingModel.Bones.ContainsByName(pmx.CENTER.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.CENTER.String()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.GROOVE.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.GROOVE.String()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.LOWER.String()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.LOWER.String()}))
		return false
	}

	// ------------------------------

	if !sizingModel.Bones.ContainsByName(pmx.LEG_IK.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.LEG_IK.Left()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.LEG.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.LEG.Left()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.KNEE.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.KNEE.Left()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.ANKLE.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.ANKLE.Left()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.TOE_IK.Left()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.TOE_IK.Left()}))
		return false
	}

	// ------------------------------

	if !sizingModel.Bones.ContainsByName(pmx.LEG_IK.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.LEG_IK.Right()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.LEG.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.LEG.Right()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.KNEE.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.KNEE.Right()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.ANKLE.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.ANKLE.Right()}))
		return false
	}

	if !sizingModel.Bones.ContainsByName(pmx.TOE_IK.Right()) {
		mlog.WT(mi18n.T("ボーン不足"), mi18n.T("足補正ボーン不足", map[string]interface{}{
			"No": sizingSet.Index + 1, "ModelType": mi18n.T("先モデル"), "BoneName": pmx.TOE_IK.Left()}))
		return false
	}

	return true
}
