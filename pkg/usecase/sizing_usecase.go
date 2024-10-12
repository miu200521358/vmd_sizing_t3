package usecase

import (
	"fmt"

	"github.com/miu200521358/mlib_go/pkg/domain/mmath"
	"github.com/miu200521358/mlib_go/pkg/domain/pmx"
	"github.com/miu200521358/mlib_go/pkg/mutils/mi18n"
	"github.com/miu200521358/mlib_go/pkg/mutils/mlog"
	"github.com/miu200521358/vmd_sizing_t3/pkg/domain"
)

var trunk_upper_bone_names = []string{
	pmx.TRUNK_ROOT.String(), pmx.UPPER_ROOT.String(), pmx.UPPER.String(), pmx.UPPER2.String(), pmx.NECK_ROOT.String(),
	pmx.SHOULDER.Left(), pmx.SHOULDER.Right()}
var trunk_lower_bone_names = []string{
	pmx.ROOT.String(), pmx.CENTER.String(), pmx.GROOVE.String(), pmx.LOWER_ROOT.String(),
	pmx.LOWER.String(), pmx.LEG_CENTER.String()}
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
var leg_all_bone_names = append(trunk_lower_bone_names, leg_all_direction_bone_names...)

var arm_bone_names = []string{
	pmx.ARM.Left(), pmx.ELBOW.Left(), pmx.WRIST.Left(), pmx.ARM.Right(), pmx.ELBOW.Right(), pmx.WRIST.Right()}
var finger_bone_names = []string{
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
