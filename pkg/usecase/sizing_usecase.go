package usecase

import (
	"fmt"

	"github.com/miu200521358/mlib_go/pkg/domain/mmath"
	"github.com/miu200521358/mlib_go/pkg/domain/pmx"
	"github.com/miu200521358/mlib_go/pkg/mutils/mi18n"
	"github.com/miu200521358/mlib_go/pkg/mutils/mlog"
	"github.com/miu200521358/vmd_sizing_t3/pkg/domain"
)

var directions = []string{"左", "右"}

var trunk_upper_bone_names = []string{
	pmx.ROOT.String(), pmx.TRUNK_ROOT.String(), pmx.CENTER.String(), pmx.GROOVE.String(), pmx.WAIST.String(),
	pmx.UPPER_ROOT.String(), pmx.UPPER.String(), pmx.UPPER2.String(), pmx.NECK_ROOT.String(),
	pmx.SHOULDER.Left(), pmx.SHOULDER.Right(), pmx.NECK.String()}
var trunk_lower_bone_names = []string{
	pmx.ROOT.String(), pmx.TRUNK_ROOT.String(), pmx.CENTER.String(), pmx.GROOVE.String(), pmx.WAIST.String(),
	pmx.LOWER_ROOT.String(), pmx.LOWER.String(), pmx.LEG_CENTER.String(), pmx.LEG.Left(), pmx.LEG.Right()}
var leg_direction_bone_names = [][]string{
	{pmx.LEG.Left(), pmx.KNEE.Left(), pmx.HEEL.Left(), pmx.ANKLE.Left(), pmx.TOE_T.Left(), pmx.TOE_P.Left(),
		pmx.TOE_C.Left(), pmx.LEG_D.Left(), pmx.KNEE_D.Left(), pmx.HEEL_D.Left(), pmx.ANKLE_D.Left(),
		pmx.TOE_T_D.Left(), pmx.TOE_P_D.Left(), pmx.TOE_C_D.Left(), pmx.TOE_EX.Left(),
		pmx.LEG_IK_PARENT.Left(), pmx.LEG_IK.Left(), pmx.TOE_IK.Left()},
	{pmx.LEG.Right(), pmx.KNEE.Right(), pmx.HEEL.Right(), pmx.ANKLE.Right(), pmx.TOE_T.Right(), pmx.TOE_P.Right(),
		pmx.TOE_C.Right(), pmx.LEG_D.Right(), pmx.KNEE_D.Right(), pmx.HEEL_D.Right(), pmx.ANKLE_D.Right(),
		pmx.TOE_T_D.Right(), pmx.TOE_P_D.Right(), pmx.TOE_C_D.Right(), pmx.TOE_EX.Right(),
		pmx.LEG_IK_PARENT.Right(), pmx.LEG_IK.Right(), pmx.TOE_IK.Right()},
}
var leg_all_direction_bone_names = append(leg_direction_bone_names[0], leg_direction_bone_names[1]...)
var all_lower_leg_bone_names = append(trunk_lower_bone_names, leg_all_direction_bone_names...)

var shoulder_direction_bone_names = [][]string{
	{
		pmx.NECK_ROOT.String(), pmx.SHOULDER_P.Left(), pmx.SHOULDER.Left(), pmx.ARM.Left(),
	},
	{
		pmx.NECK_ROOT.String(), pmx.SHOULDER_P.Right(), pmx.SHOULDER.Right(), pmx.ARM.Right(),
	},
}

var arm_direction_bone_names = [][]string{
	{
		pmx.ARM.Left(), pmx.ARM_TWIST.Left(), pmx.ELBOW.Left(), pmx.WRIST_TWIST.Left(), pmx.WRIST.Left(), pmx.WRIST_TAIL.Left(),
	},
	{
		pmx.ARM.Right(), pmx.ARM_TWIST.Right(), pmx.ELBOW.Right(), pmx.WRIST_TWIST.Right(), pmx.WRIST.Right(), pmx.WRIST_TAIL.Right(),
	},
}

var all_arm_bone_names = []string{
	pmx.ARM.Left(), pmx.ELBOW.Left(), pmx.WRIST.Left(), pmx.ARM.Right(), pmx.ELBOW.Right(), pmx.WRIST.Right()}
var all_finger_bone_names = []string{
	pmx.THUMB0.Left(), pmx.THUMB1.Left(), pmx.THUMB2.Left(),
	pmx.INDEX1.Left(), pmx.INDEX2.Left(), pmx.INDEX3.Left(),
	pmx.MIDDLE1.Left(), pmx.MIDDLE2.Left(), pmx.MIDDLE3.Left(),
	pmx.RING1.Left(), pmx.RING2.Left(), pmx.RING3.Left(),
	pmx.PINKY1.Left(), pmx.PINKY2.Left(), pmx.PINKY3.Left(),
	pmx.THUMB0.Right(), pmx.THUMB1.Right(), pmx.THUMB2.Right(),
	pmx.INDEX1.Right(), pmx.INDEX2.Right(), pmx.INDEX3.Right(),
	pmx.MIDDLE1.Right(), pmx.MIDDLE2.Right(), pmx.MIDDLE3.Right(),
	pmx.RING1.Right(), pmx.RING2.Right(), pmx.RING3.Right(),
	pmx.PINKY1.Right(), pmx.PINKY2.Right(), pmx.PINKY3.Right(),
}

// var all_upper_arm_bone_names = append(all_arm_bone_names, trunk_upper_bone_names...)

// // 四肢ボーン名（つま先とか指先は入っていない）
// var all_limb_bone_names = append(all_lower_leg_bone_names, all_upper_arm_bone_names...)

var finger_direction_bone_names = [][]string{
	{pmx.THUMB0.Left(), pmx.THUMB1.Left(), pmx.THUMB2.Left(), pmx.THUMB_TAIL.Left(),
		pmx.INDEX1.Left(), pmx.INDEX2.Left(), pmx.INDEX3.Left(), pmx.INDEX_TAIL.Left(),
		pmx.MIDDLE1.Left(), pmx.MIDDLE2.Left(), pmx.MIDDLE3.Left(), pmx.MIDDLE_TAIL.Left(),
		pmx.RING1.Left(), pmx.RING2.Left(), pmx.RING3.Left(), pmx.RING_TAIL.Left(),
		pmx.PINKY1.Left(), pmx.PINKY2.Left(), pmx.PINKY3.Left(), pmx.PINKY_TAIL.Left()},
	{pmx.THUMB0.Right(), pmx.THUMB1.Right(), pmx.THUMB2.Right(), pmx.THUMB_TAIL.Right(),
		pmx.INDEX1.Right(), pmx.INDEX2.Right(), pmx.INDEX3.Right(), pmx.INDEX_TAIL.Right(),
		pmx.MIDDLE1.Right(), pmx.MIDDLE2.Right(), pmx.MIDDLE3.Right(), pmx.MIDDLE_TAIL.Right(),
		pmx.RING1.Right(), pmx.RING2.Right(), pmx.RING3.Right(), pmx.RING_TAIL.Right(),
		pmx.PINKY1.Right(), pmx.PINKY2.Right(), pmx.PINKY3.Right(), pmx.PINKY_TAIL.Right()},
}

func GenerateSizingScales(sizingSets []*domain.SizingSet) []*mmath.MVec3 {
	scales := make([]*mmath.MVec3, len(sizingSets))

	// 複数人居るときはXZは共通のスケールを使用する
	meanXZScale := 0.0

	for i, sizingSet := range sizingSets {
		originalModel := sizingSet.OriginalPmx
		sizingModel := sizingSet.SizingPmx

		if originalModel == nil || sizingModel == nil {
			scales[i] = &mmath.MVec3{X: 1.0, Y: 1.0, Z: 1.0}
			meanXZScale += 1.0
			continue
		}

		if sizingModel.Bones.GetByName(pmx.LEG.Left()) == nil ||
			sizingModel.Bones.GetByName(pmx.KNEE.Left()) == nil ||
			sizingModel.Bones.GetByName(pmx.ANKLE.Left()) == nil ||
			sizingModel.Bones.GetByName(pmx.LEG_IK.Left()) == nil ||
			originalModel.Bones.GetByName(pmx.LEG.Left()) == nil ||
			originalModel.Bones.GetByName(pmx.KNEE.Left()) == nil ||
			originalModel.Bones.GetByName(pmx.ANKLE.Left()) == nil ||
			originalModel.Bones.GetByName(pmx.LEG_IK.Left()) == nil {

			if sizingModel.Bones.GetByName(pmx.NECK_ROOT.String()) != nil &&
				originalModel.Bones.GetByName(pmx.NECK_ROOT.String()) != nil {
				// 首根元までの長さ比率
				neckLengthRatio := sizingModel.Bones.GetByName(pmx.NECK_ROOT.String()).Position.Y /
					originalModel.Bones.GetByName(pmx.NECK_ROOT.String()).Position.Y
				scales[i] = &mmath.MVec3{X: neckLengthRatio, Y: neckLengthRatio, Z: neckLengthRatio}
				meanXZScale += neckLengthRatio
			} else {
				scales[i] = &mmath.MVec3{X: 1.0, Y: 1.0, Z: 1.0}
				meanXZScale += 1.0
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

			scales[i] = &mmath.MVec3{X: legLengthRatio, Y: legHeightRatio, Z: legLengthRatio}
			meanXZScale += legLengthRatio
		}
	}

	// 複数人いるときはXZは共通のスケールを使用する
	meanXZScale /= float64(len(scales))
	newXZScale := meanXZScale
	if len(sizingSets) > 1 {
		newXZScale = min(1.2, meanXZScale)
	}

	for i, sizingSet := range sizingSets {
		if sizingSet.IsSizingLeg && !sizingSet.CompletedSizingLeg {
			mlog.I(mi18n.T("移動補正スケール", map[string]interface{}{
				"No": i + 1, "XZ": fmt.Sprintf("%.3f", newXZScale),
				"OrgXZ": fmt.Sprintf("%.3f", scales[i].X), "Y": fmt.Sprintf("%.3f", scales[i].Y)}))
		}

		scales[i].X = newXZScale
		scales[i].Z = newXZScale
	}

	return scales
}

// func deformIk(
// 	index int,
// 	frame float32,
// 	sizingModel *pmx.PmxModel,
// 	sizingMotion *vmd.VmdMotion,
// 	originalAllDeltas []*delta.VmdDeltas,
// 	sizingDeltas *delta.VmdDeltas,
// 	originalSrcBone *pmx.Bone,
// 	originalDstBone *pmx.Bone,
// 	sizingSrcBone *pmx.Bone,
// 	sizingDstBone *pmx.Bone,
// 	sizingIkBone *pmx.Bone,
// 	sizingSlopeMat *mmath.MMat4,
// 	scale float64,
// ) (dstIkDeltas *delta.VmdDeltas, diffSrcRotation *mmath.MQuaternion) {
// 	// 元から見た先の相対位置を取得
// 	originalSrcDelta := originalAllDeltas[index].Bones.Get(originalSrcBone.Index())
// 	originalDstDelta := originalAllDeltas[index].Bones.Get(originalDstBone.Index())

// 	// 元から見た先の相対位置をスケールに合わせる
// 	originalSrcLocalPosition := originalSrcDelta.FilledGlobalMatrix().Inverted().MulVec3(originalDstDelta.FilledGlobalPosition())
// 	sizingDstLocalPosition := originalSrcLocalPosition.MuledScalar(scale)
// 	sizingDstSlopeLocalPosition := sizingSlopeMat.MulVec3(sizingDstLocalPosition)

// 	// Fixさせた新しい先のグローバル位置を取得
// 	sizingSrcDelta := sizingDeltas.Bones.Get(sizingSrcBone.Index())
// 	sizingFixDstGlobalPosition := sizingSrcDelta.FilledGlobalMatrix().MulVec3(sizingDstSlopeLocalPosition)

// 	// IK結果を返す
// 	dstIkDeltas = deform.DeformIk(sizingModel, sizingMotion, sizingDeltas, frame, sizingIkBone,
// 		sizingFixDstGlobalPosition, []string{sizingSrcBone.Name(), sizingDstBone.Name()})

// 	originalSrcRotation := originalAllDeltas[index].Bones.Get(originalSrcBone.Index()).FilledFrameRotation()
// 	sizingSrcRotation := dstIkDeltas.Bones.Get(sizingSrcBone.Index()).FilledFrameRotation()

// 	// IK結果の回転差分
// 	diffSrcRotation = sizingSrcRotation.Muled(originalSrcRotation.Inverted()).Inverted()

// 	return dstIkDeltas, diffSrcRotation
// }
