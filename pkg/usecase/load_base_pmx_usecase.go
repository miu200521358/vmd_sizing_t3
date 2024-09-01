package usecase

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/miu200521358/mlib_go/pkg/domain/mmath"
	"github.com/miu200521358/mlib_go/pkg/domain/pmx"
	"github.com/miu200521358/mlib_go/pkg/domain/vmd"
	"github.com/miu200521358/mlib_go/pkg/infrastructure/deform"
	"github.com/miu200521358/mlib_go/pkg/infrastructure/repository"
	"github.com/miu200521358/mlib_go/pkg/mutils/mlog"
	"github.com/miu200521358/vmd_sizing_t3/pkg/model"
)

//go:embed base_model/*.pmx
//go:embed base_model/tex/*.png
var modelFs embed.FS

// FitBoneモーフ名
var fit_morph_name = fmt.Sprintf("%s_%s", pmx.MLIB_PREFIX, "FitBone")

func LoadOriginalPmx(jsonModel *pmx.PmxModel) (*pmx.PmxModel, error) {
	// 素体PMXモデルを読み込む
	model, err := loadMannequinPmx()
	if err != nil {
		return nil, err
	}

	// テクスチャをTempディレクトリに読み込んでおく
	loadOriginalPmxTextures(model)

	// 足りないボーンを追加
	addNonExistBones(model, jsonModel)

	// ボーン設定を補正
	fixBaseBones(model, jsonModel)

	jsonModel.Setup()
	model.Setup()
	// 強制更新用にハッシュ上書き
	model.SetRandHash()

	// フィットボーンモーフを作成
	createFitMorph(model, jsonModel, fit_morph_name)

	return model, nil
}

func AddFitMorph(motion *vmd.VmdMotion) *vmd.VmdMotion {
	// フィットボーンモーフを適用
	mf := vmd.NewMorphFrame(float32(0))
	mf.Ratio = 1.0
	motion.AppendMorphFrame(fit_morph_name, mf)
	return motion
}

func RemakeFitMorph(model, jsonModel *pmx.PmxModel, sizingSet *model.SizingSet) *pmx.PmxModel {
	model.Morphs.RemoveByName(fit_morph_name)

	// 足りないボーンを追加
	addNonExistBones(model, jsonModel)

	// ボーン設定を補正
	fixBaseBones(model, jsonModel)

	// jsonモデルをリサイズ
	resizeJsonModel(jsonModel, sizingSet)

	// フィットボーンモーフを再度作成
	createFitMorph(model, jsonModel, fit_morph_name)

	return model
}

func resizeJsonModel(jsonModel *pmx.PmxModel, sizingSet *model.SizingSet) {
	// リサイズ
	resizeMotion := vmd.NewVmdMotion("")
	{
		// 全体比率
		bf := vmd.NewBoneFrame(0)
		bf.Scale = &mmath.MVec3{X: sizingSet.OriginalPmxRatio,
			Y: sizingSet.OriginalPmxRatio, Z: sizingSet.OriginalPmxRatio}
		resizeMotion.AppendRegisteredBoneFrame(pmx.ROOT.String(), bf)
	}
	{
		// 上半身
		bf := vmd.NewBoneFrame(0)
		bf.CancelableRotation, bf.CancelableScale = getResizeParams(
			jsonModel, pmx.UPPER.String(),
			sizingSet.OriginalPmxUpperLength, sizingSet.OriginalPmxUpperAngle, 0, 0)
		resizeMotion.AppendRegisteredBoneFrame(pmx.UPPER.String(), bf)
	}
	{
		// 上半身2
		bf := vmd.NewBoneFrame(0)
		bf.CancelableRotation, bf.CancelableScale = getResizeParams(
			jsonModel, pmx.UPPER2.String(),
			sizingSet.OriginalPmxUpper2Length, sizingSet.OriginalPmxUpper2Angle, 0, 0)
		resizeMotion.AppendRegisteredBoneFrame(pmx.UPPER2.String(), bf)
	}
	{
		// 首
		bf := vmd.NewBoneFrame(0)
		bf.CancelableRotation, bf.CancelableScale = getResizeParams(
			jsonModel, pmx.NECK.String(),
			sizingSet.OriginalPmxNeckLength, sizingSet.OriginalPmxNeckAngle, 0, 0)
		resizeMotion.AppendRegisteredBoneFrame(pmx.NECK.String(), bf)
	}
	{
		// 右肩
		bf := vmd.NewBoneFrame(0)
		bf.CancelableRotation, bf.CancelableScale = getResizeParams(
			jsonModel, pmx.SHOULDER.Right(),
			sizingSet.OriginalPmxShoulderLength, 0, 0, sizingSet.OriginalPmxShoulderAngle)
		resizeMotion.AppendRegisteredBoneFrame(pmx.SHOULDER.Right(), bf)
	}
	{
		// 左肩
		bf := vmd.NewBoneFrame(0)
		bf.CancelableRotation, bf.CancelableScale = getResizeParams(
			jsonModel, pmx.SHOULDER.Left(),
			sizingSet.OriginalPmxShoulderLength, 0, 0, -sizingSet.OriginalPmxShoulderAngle)
		resizeMotion.AppendRegisteredBoneFrame(pmx.SHOULDER.Left(), bf)
	}
	{
		// 右腕(角度補正は子ども全てに適用)
		bf := vmd.NewBoneFrame(0)
		bf.Rotation, bf.CancelableScale = getResizeParams(
			jsonModel, pmx.ARM.Right(),
			sizingSet.OriginalPmxArmLength, 0, 0, sizingSet.OriginalPmxArmAngle)
		resizeMotion.AppendRegisteredBoneFrame(pmx.ARM.Right(), bf)
	}
	{
		// 左腕(角度補正は子ども全てに適用)
		bf := vmd.NewBoneFrame(0)
		bf.Rotation, bf.CancelableScale = getResizeParams(
			jsonModel, pmx.ARM.Left(),
			sizingSet.OriginalPmxArmLength, 0, 0, -sizingSet.OriginalPmxArmAngle)
		resizeMotion.AppendRegisteredBoneFrame(pmx.ARM.Left(), bf)
	}
	{
		// 右ひじ(角度補正は子ども全てに適用)
		bf := vmd.NewBoneFrame(0)
		bf.Rotation, bf.CancelableScale = getResizeParams(
			jsonModel, pmx.ELBOW.Right(),
			sizingSet.OriginalPmxElbowLength, 0, 0, sizingSet.OriginalPmxElbowAngle)
		resizeMotion.AppendRegisteredBoneFrame(pmx.ELBOW.Right(), bf)
	}
	{
		// 左ひじ(角度補正は子ども全てに適用)
		bf := vmd.NewBoneFrame(0)
		bf.Rotation, bf.CancelableScale = getResizeParams(
			jsonModel, pmx.ELBOW.Left(),
			sizingSet.OriginalPmxElbowLength, 0, 0, -sizingSet.OriginalPmxElbowAngle)
		resizeMotion.AppendRegisteredBoneFrame(pmx.ELBOW.Left(), bf)
	}
	{
		// 右手首(角度補正は子ども全てに適用)
		bf := vmd.NewBoneFrame(0)
		bf.Rotation, bf.CancelableScale = getResizeParams(
			jsonModel, pmx.WRIST.Right(),
			sizingSet.OriginalPmxWristLength, 0, 0, sizingSet.OriginalPmxWristAngle)
		resizeMotion.AppendRegisteredBoneFrame(pmx.WRIST.Right(), bf)
	}
	{
		// 左手首(角度補正は子ども全てに適用)
		bf := vmd.NewBoneFrame(0)
		bf.Rotation, bf.CancelableScale = getResizeParams(
			jsonModel, pmx.WRIST.Left(),
			sizingSet.OriginalPmxWristLength, 0, 0, -sizingSet.OriginalPmxWristAngle)
		resizeMotion.AppendRegisteredBoneFrame(pmx.WRIST.Left(), bf)
	}
	{
		// 下半身
		bf := vmd.NewBoneFrame(0)
		bf.CancelableRotation, bf.CancelableScale = getResizeParams(
			jsonModel, pmx.LOWER.String(),
			sizingSet.OriginalPmxLowerLength, sizingSet.OriginalPmxLowerAngle, 0, 0)
		resizeMotion.AppendRegisteredBoneFrame(pmx.LOWER.String(), bf)
	}
	{
		// 右足根元
		bf := vmd.NewBoneFrame(0)
		bf.CancelableRotation, bf.CancelableScale = getResizeParams(
			jsonModel, pmx.LEG_ROOT.Right(), sizingSet.OriginalPmxLegWidth, 0, 0, 0)
		resizeMotion.AppendRegisteredBoneFrame(pmx.LEG_ROOT.Right(), bf)
	}
	{
		// 左足根元
		bf := vmd.NewBoneFrame(0)
		bf.CancelableRotation, bf.CancelableScale = getResizeParams(
			jsonModel, pmx.LEG_ROOT.Left(), sizingSet.OriginalPmxLegWidth, 0, 0, 0)
		resizeMotion.AppendRegisteredBoneFrame(pmx.LEG_ROOT.Left(), bf)
	}
	{
		// 右足
		for _, boneName := range []string{pmx.LEG.Right(), pmx.LEG_D.Right()} {
			bf := vmd.NewBoneFrame(0)
			bf.CancelableRotation, bf.CancelableScale = getResizeParams(
				jsonModel, boneName,
				sizingSet.OriginalPmxLegLength, sizingSet.OriginalPmxLegAngle, 0, 0)
			resizeMotion.AppendRegisteredBoneFrame(boneName, bf)
		}
	}
	{
		// 左足
		for _, boneName := range []string{pmx.LEG.Left(), pmx.LEG_D.Left()} {
			bf := vmd.NewBoneFrame(0)
			bf.CancelableRotation, bf.CancelableScale = getResizeParams(
				jsonModel, boneName,
				sizingSet.OriginalPmxLegLength, sizingSet.OriginalPmxLegAngle, 0, 0)
			resizeMotion.AppendRegisteredBoneFrame(boneName, bf)
		}
	}
	{
		// 右ひざ
		for _, boneName := range []string{pmx.KNEE.Right(), pmx.KNEE_D.Right()} {
			bf := vmd.NewBoneFrame(0)
			bf.CancelableRotation, bf.CancelableScale = getResizeParams(
				jsonModel, boneName,
				sizingSet.OriginalPmxKneeLength, sizingSet.OriginalPmxKneeAngle, 0, 0)
			resizeMotion.AppendRegisteredBoneFrame(boneName, bf)
		}
	}
	{
		// 左ひざ
		for _, boneName := range []string{pmx.KNEE.Left(), pmx.KNEE_D.Left()} {
			bf := vmd.NewBoneFrame(0)
			bf.CancelableRotation, bf.CancelableScale = getResizeParams(
				jsonModel, boneName,
				sizingSet.OriginalPmxKneeLength, sizingSet.OriginalPmxKneeAngle, 0, 0)
			resizeMotion.AppendRegisteredBoneFrame(boneName, bf)
		}
	}
	{
		// 右足首
		for _, boneName := range []string{pmx.ANKLE.Right(), pmx.ANKLE_D.Right()} {
			bf := vmd.NewBoneFrame(0)
			bf.CancelableRotation, bf.CancelableScale = getResizeParams(
				jsonModel, boneName,
				sizingSet.OriginalPmxAnkleLength, 0, 0, 0)
			resizeMotion.AppendRegisteredBoneFrame(boneName, bf)
		}
	}
	{
		// 左足首
		for _, boneName := range []string{pmx.ANKLE.Left(), pmx.ANKLE_D.Left()} {
			bf := vmd.NewBoneFrame(0)
			bf.CancelableRotation, bf.CancelableScale = getResizeParams(
				jsonModel, boneName,
				sizingSet.OriginalPmxAnkleLength, 0, 0, 0)
			resizeMotion.AppendRegisteredBoneFrame(boneName, bf)
		}
	}

	{
		// リサイズモーションを適用
		boneDeltas := deform.DeformBone(jsonModel, resizeMotion, false, 0, nil)
		for _, boneDelta := range boneDeltas.Data {
			if boneDelta == nil {
				continue
			}
			jsonModel.Bones.Get(boneDelta.Bone.Index()).Position = boneDelta.FilledGlobalPosition()
		}

		heelY := (boneDeltas.Bones.GetByName(pmx.HEEL.Right()).Position.Y +
			boneDeltas.Bones.GetByName(pmx.HEEL.Left()).Position.Y) / 2

		// 体幹中心
		{
			bf := vmd.NewBoneFrame(0)
			bf.Position = &mmath.MVec3{X: 0, Y: -heelY, Z: 0}
			resizeMotion.AppendRegisteredBoneFrame(pmx.TRUNK_ROOT.String(), bf)
		}
	}

	// リサイズモーションを再適用
	boneDeltas := deform.DeformBone(jsonModel, resizeMotion, false, 0, nil)
	for _, boneDelta := range boneDeltas.Data {
		if boneDelta == nil {
			continue
		}
		jsonModel.Bones.Get(boneDelta.Bone.Index()).Position = boneDelta.FilledGlobalPosition()
	}

	// 足IK親
	jsonModel.Bones.GetByName(pmx.LEG_IK_PARENT.Right()).Position =
		boneDeltas.Bones.GetByName(pmx.ANKLE.Right()).Position.Copy()
	jsonModel.Bones.GetByName(pmx.LEG_IK_PARENT.Right()).Position.Y = 0
	jsonModel.Bones.GetByName(pmx.LEG_IK_PARENT.Left()).Position =
		boneDeltas.Bones.GetByName(pmx.ANKLE.Left()).Position.Copy()
	jsonModel.Bones.GetByName(pmx.LEG_IK_PARENT.Left()).Position.Y = 0

	// 足IK
	jsonModel.Bones.GetByName(pmx.LEG_IK.Right()).Position =
		boneDeltas.Bones.GetByName(pmx.ANKLE.Right()).Position.Copy()
	jsonModel.Bones.GetByName(pmx.LEG_IK.Left()).Position =
		boneDeltas.Bones.GetByName(pmx.ANKLE.Left()).Position.Copy()

	// つま先IK
	jsonModel.Bones.GetByName(pmx.TOE_IK.Right()).Position =
		boneDeltas.Bones.GetByName(pmx.TOE.Right()).Position.Copy()
	jsonModel.Bones.GetByName(pmx.TOE_IK.Left()).Position =
		boneDeltas.Bones.GetByName(pmx.TOE.Left()).Position.Copy()

	jsonModel.Setup()
}

func getResizeParams(
	jsonModel *pmx.PmxModel, boneName string, length, xPitch, yHead, zRoll float64,
) (*mmath.MQuaternion, *mmath.MVec3) {
	rot := mmath.NewMQuaternionFromDegrees(xPitch, yHead, zRoll)

	var scale *mmath.MVec3
	if strings.Contains(boneName, "足首") {
		localMat := jsonModel.Bones.GetByName(boneName).Extend.LocalAxis.ToLocalMat()
		localMat = localMat.Muled(rot.ToMat4())

		scales := &mmath.MVec3{X: length, Y: 1, Z: 1}
		scale = localMat.Muled(scales.ToScaleMat4()).Muled(localMat.Inverted()).Scaling()
	} else {
		scale = &mmath.MVec3{X: length, Y: length, Z: length}
	}

	return rot, scale
}

func loadMannequinPmx() (*pmx.PmxModel, error) {
	var model *pmx.PmxModel

	// JSONファイルが指定されている場合、embedからPMXモデルの素体を読み込む
	if f, err := modelFs.Open("base_model/model.pmx"); err != nil {
		return nil, err
	} else if pmxData, err := repository.NewPmxRepository().LoadByFile(f); err != nil {
		return nil, err
	} else {
		model = pmxData.(*pmx.PmxModel)
	}

	return model, nil
}

func getBaseScale(model, jsonModel *pmx.PmxModel) float64 {
	if !jsonModel.Bones.ContainsByName(pmx.ARM.Left()) || !jsonModel.Bones.ContainsByName(pmx.ARM.Right()) ||
		!model.Bones.ContainsByName(pmx.ARM.Left()) || !model.Bones.ContainsByName(pmx.ARM.Right()) ||
		!jsonModel.Bones.ContainsByName(pmx.LEG.Left()) || !jsonModel.Bones.ContainsByName(pmx.LEG.Right()) ||
		!model.Bones.ContainsByName(pmx.LEG.Left()) || !model.Bones.ContainsByName(pmx.LEG.Right()) {
		return 1.0
	}

	// 両腕の中央を首根元とする
	neckRootPos := model.Bones.GetByName(pmx.ARM.Left()).Position.Added(
		model.Bones.GetByName(pmx.ARM.Right()).Position).MuledScalar(0.5)
	jsonNeckRootPos := jsonModel.Bones.GetByName(pmx.ARM.Left()).Position.Added(
		jsonModel.Bones.GetByName(pmx.ARM.Right()).Position).MuledScalar(0.5)

	// 両足の中央を足中央とする
	legRootPos := model.Bones.GetByName(pmx.LEG.Left()).Position.Added(
		model.Bones.GetByName(pmx.LEG.Right()).Position).MuledScalar(0.5)
	jsonLegRootPos := jsonModel.Bones.GetByName(pmx.LEG.Left()).Position.Added(
		jsonModel.Bones.GetByName(pmx.LEG.Right()).Position).MuledScalar(0.5)

	upperRatio := jsonNeckRootPos.Distance(jsonLegRootPos) / neckRootPos.Distance(legRootPos)

	return upperRatio
}

// model にあって、 jsonModel にないボーンを追加する
func addNonExistBones(model, jsonModel *pmx.PmxModel) {
	if !jsonModel.Bones.ContainsByName(pmx.ARM.Left()) || !jsonModel.Bones.ContainsByName(pmx.ARM.Right()) {
		return
	}

	ratio := getBaseScale(model, jsonModel)

	for i, boneIndex := range model.Bones.LayerSortedIndexes {
		bone := model.Bones.Get(boneIndex)
		// 存在するボーンの場合
		if jsonModel.Bones.ContainsByName(bone.Name()) {
			jsonBone := jsonModel.Bones.GetByName(bone.Name())
			if jsonBone.ParentIndex < 0 && jsonBone.Name() != pmx.ROOT.String() {
				// センターがルートなどの場合に、全ての親を親に切り替える
				jsonBone.ParentIndex = jsonModel.Bones.GetByName(pmx.ROOT.String()).Index()
			} else if bone.ParentIndex >= 0 {
				// それ以外も親を切り替える
				jsonBone.ParentIndex = jsonModel.Bones.GetByName(model.Bones.Get(bone.ParentIndex).Name()).Index()
			}

			continue
		}

		// 存在しないボーンは追加
		newBone := bone.Copy().(*pmx.Bone)
		// 最後に追加
		newBone.SetIndex(jsonModel.Bones.Len())
		if bone.ParentIndex < 0 {
			if newBone.Name() == pmx.ROOT.String() {
				newBone.ParentIndex = -1
			} else {
				newBone.ParentIndex = jsonModel.Bones.GetByName(pmx.ROOT.String()).Index()
			}
			newBone.Layer = 0
		} else {
			parentBone := model.Bones.Get(bone.ParentIndex)
			jsonParentBone := jsonModel.Bones.GetByName(parentBone.Name())
			newBone.ParentIndex = jsonParentBone.Index()
			newBone.Layer = jsonParentBone.Layer
			newBone.IsSystem = true

			// 親からの相対位置から比率で求める
			newBone.Position = jsonParentBone.Position.Added(bone.Extend.ParentRelativePosition.MuledScalar(ratio))

			if bone.Name() == pmx.UPPER2.String() {
				// 上半身2の場合、首根元と上半身の間に置く
				neckRootBone := model.Bones.GetByName(pmx.NECK_ROOT.String())
				upperBone := model.Bones.GetByName(pmx.UPPER.String())
				upper2Bone := model.Bones.GetByName(pmx.UPPER2.String())

				jsonUpperBone := jsonModel.Bones.GetByName(upperBone.Name())
				jsonNeckRootPosition := jsonModel.Bones.GetByName(pmx.ARM.Left()).Position.Added(
					jsonModel.Bones.GetByName(pmx.ARM.Right()).Position).MuledScalar(0.5)

				// 上半身の長さを上半身と首根元の距離で求める
				upperLength := upperBone.Position.Distance(neckRootBone.Position)
				jsonUpperLength := jsonUpperBone.Position.Distance(jsonNeckRootPosition)
				upperRatio := upperLength / jsonUpperLength

				upper2Offset := upper2Bone.Position.Subed(upperBone.Position).MuledScalar(upperRatio)
				newBone.Position = jsonUpperBone.Position.Added(upper2Offset)
			} else if strings.Contains(bone.Name(), "腕捩") {
				// 腕捩の場合、腕とひじの間に置く
				armBone := model.Bones.GetByName(
					strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(
						bone.Name(), "腕捩1", "腕"), "腕捩2", "腕"), "腕捩3", "腕"), "腕捩", "腕"))
				elbowBone := model.Bones.GetByName(
					strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(
						bone.Name(), "腕捩1", "ひじ"), "腕捩2", "ひじ"), "腕捩3", "ひじ"), "腕捩", "ひじ"))

				twistRatio := bone.Position.Subed(armBone.Position).Length() / elbowBone.Position.Subed(armBone.Position).Length()

				jsonArmBone := jsonModel.Bones.GetByName(armBone.Name())
				jsonElbowBone := jsonModel.Bones.GetByName(elbowBone.Name())
				newBone.Position = jsonArmBone.Position.Lerp(jsonElbowBone.Position, twistRatio)
				newBone.FixedAxis = jsonElbowBone.Position.Subed(jsonArmBone.Position).Normalized()
			} else if strings.Contains(bone.Name(), "手捩") {
				elbowBone := model.Bones.GetByName(
					strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(
						bone.Name(), "手捩1", "ひじ"), "手捩2", "ひじ"), "手捩3", "ひじ"), "手捩", "ひじ"))
				wristBone := model.Bones.GetByName(
					strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(
						bone.Name(), "手捩1", "手首"), "手捩2", "手首"), "手捩3", "手首"), "手捩", "手首"))

				twistRatio := bone.Position.Subed(wristBone.Position).Length() / elbowBone.Position.Subed(wristBone.Position).Length()

				jsonElbowBone := jsonModel.Bones.GetByName(elbowBone.Name())
				jsonWristBone := jsonModel.Bones.GetByName(wristBone.Name())
				newBone.Position = jsonElbowBone.Position.Lerp(jsonWristBone.Position, twistRatio)
				newBone.FixedAxis = jsonWristBone.Position.Subed(jsonElbowBone.Position).Normalized()
			} else if bone.Name() == pmx.SHOULDER_P.Left() || bone.Name() == pmx.SHOULDER_P.Right() {
				// 肩Pの場合、肩と同じ位置に置く
				shoulderBone := model.Bones.GetByName(strings.ReplaceAll(bone.Name(), "肩P", "肩"))
				jsonShoulderBone := jsonModel.Bones.GetByName(shoulderBone.Name())
				newBone.Position = jsonShoulderBone.Position.Copy()
			} else if slices.Contains([]string{pmx.SHOULDER_C.Left(), pmx.SHOULDER_C.Right()}, bone.Name()) {
				// 肩Cの場合、腕と同じ位置に置く
				armBone := model.Bones.GetByName(strings.ReplaceAll(bone.Name(), "肩C", "腕"))
				jsonArmBone := jsonModel.Bones.GetByName(armBone.Name())
				newBone.Position = jsonArmBone.Position.Copy()
			} else if slices.Contains([]string{pmx.NECK_ROOT.String(), pmx.SHOULDER_ROOT.Left(), pmx.SHOULDER_ROOT.Right()}, bone.Name()) {
				// 首根元・肩根元は首根元の位置
				newBone.Position = jsonModel.Bones.GetByName(pmx.ARM.Left()).Position.Added(
					jsonModel.Bones.GetByName(pmx.ARM.Right()).Position).MuledScalar(0.5)
			} else if slices.Contains([]string{pmx.THUMB0.Left(), pmx.THUMB0.Right()}, bone.Name()) {
				// 親指０は手首と親指１の間
				wristBone := model.Bones.GetByName(strings.ReplaceAll(bone.Name(), "親指０", "手首"))
				thumbBone := model.Bones.GetByName(strings.ReplaceAll(bone.Name(), "親指０", "親指１"))
				thumbRatio := bone.Position.Subed(wristBone.Position).Length() / thumbBone.Position.Subed(wristBone.Position).Length()

				jsonWristBone := jsonModel.Bones.GetByName(wristBone.Name())
				jsonThumbBone := jsonModel.Bones.GetByName(thumbBone.Name())
				newBone.Position = jsonWristBone.Position.Lerp(jsonThumbBone.Position, thumbRatio)
			} else if slices.Contains([]string{pmx.LEG_CENTER.String(), pmx.LEG_ROOT.Left(), pmx.LEG_ROOT.Right()}, bone.Name()) {
				// 足中心は足の中心
				newBone.Position = jsonModel.Bones.GetByName(pmx.LEG.Left()).Position.Added(
					jsonModel.Bones.GetByName(pmx.LEG.Right()).Position).MuledScalar(0.5)
			} else if strings.Contains(bone.Name(), "腰キャンセル") {
				// 腰キャンセルは足と同じ位置
				legBoneName := fmt.Sprintf("%s足", bone.Direction())
				jsonLegBone := jsonModel.Bones.GetByName(legBoneName)
				newBone.Position = jsonLegBone.Position.Copy()
			} else if slices.Contains([]string{pmx.TRUNK_ROOT.String(), pmx.UPPER_ROOT.String(), pmx.LOWER_ROOT.String()}, bone.Name()) {
				// 体幹中心・上半身根元・下半身根元は上半身と下半身の間
				jsonUpperBone := jsonModel.Bones.GetByName(pmx.UPPER.String())
				jsonLowerBone := jsonModel.Bones.GetByName(pmx.LOWER.String())
				newBone.Position = jsonUpperBone.Position.Lerp(jsonLowerBone.Position, 0.5)
			} else if slices.Contains([]string{pmx.LEG_IK_PARENT.Left(), pmx.LEG_IK_PARENT.Right()}, bone.Name()) {
				// 足IK親 は 足IKのYを0にした位置
				legIkBone := model.Bones.GetByName(strings.ReplaceAll(bone.Name(), "足IK親", "足ＩＫ"))
				jsonLegIkBone := jsonModel.Bones.GetByName(legIkBone.Name())
				newBone.Position = jsonLegIkBone.Position.Copy()
				newBone.Position.Y = 0
			} else if slices.Contains([]string{pmx.LEG_D.Left(), pmx.LEG_D.Right()}, bone.Name()) {
				// 足D は 足の位置
				legBone := model.Bones.GetByName(strings.ReplaceAll(bone.Name(), "足D", "足"))
				jsonLegBone := jsonModel.Bones.GetByName(legBone.Name())
				newBone.Position = jsonLegBone.Position.Copy()
			} else if slices.Contains([]string{pmx.KNEE_D.Left(), pmx.KNEE_D.Right()}, bone.Name()) {
				// ひざD は ひざの位置
				kneeBone := model.Bones.GetByName(strings.ReplaceAll(bone.Name(), "ひざD", "ひざ"))
				jsonKneeBone := jsonModel.Bones.GetByName(kneeBone.Name())
				newBone.Position = jsonKneeBone.Position.Copy()
			} else if slices.Contains([]string{pmx.ANKLE_D.Left(), pmx.ANKLE_D.Right()}, bone.Name()) {
				// 足首D は 足首の位置
				ankleBone := model.Bones.GetByName(strings.ReplaceAll(bone.Name(), "足首D", "足首"))
				jsonAnkleBone := jsonModel.Bones.GetByName(ankleBone.Name())
				newBone.Position = jsonAnkleBone.Position.Copy()
			} else if slices.Contains([]string{pmx.TOE_EX.Left(), pmx.TOE_EX.Right()}, bone.Name()) {
				// 足先EXは 足首とつま先の間
				ankleBone := model.Bones.GetByName(strings.ReplaceAll(bone.Name(), "足先EX", "足首"))
				// つま先のボーン名は標準ではないので、つま先ＩＫのターゲットから取る
				toeBone := model.Bones.Get(model.Bones.GetByName(strings.ReplaceAll(bone.Name(), "足先EX", "つま先ＩＫ")).Ik.BoneIndex)
				toeRatio := bone.Position.Subed(ankleBone.Position).Length() / toeBone.Position.Subed(ankleBone.Position).Length()

				jsonAnkleBone := jsonModel.Bones.GetByName(ankleBone.Name())
				jsonToeBone := jsonModel.Bones.GetByName(toeBone.Name())
				newBone.Position = jsonAnkleBone.Position.Lerp(jsonToeBone.Position, toeRatio)
			} else if slices.Contains([]string{pmx.HEEL.Left(), pmx.HEEL.Right(), pmx.HEEL_D.Left(), pmx.HEEL_D.Right()}, bone.Name()) {
				// かかとXは足首Dと同じ
				ankleBone := model.Bones.GetByName(strings.ReplaceAll(bone.Name(), "かかと", "足首"))
				jsonAnkleBone := jsonModel.Bones.GetByName(ankleBone.Name())
				newBone.Position.X = jsonAnkleBone.Position.X
				newBone.Position.Y = 0
			} else if strings.Contains(bone.Name(), "指先") {
				// 指先ボーンは親ボーンの相対表示先位置
				parentBoneName := bone.ConfigParentBoneNames()[0]
				jsonParentBone := jsonModel.Bones.GetByName(parentBoneName)
				if jsonParentBone != nil {
					newBone.Position = jsonParentBone.Position.Added(jsonParentBone.Extend.ChildRelativePosition)
				}
			} else if slices.Contains([]string{pmx.TOE.Left(), pmx.TOE.Right(), pmx.TOE_D.Left(), pmx.TOE_D.Right()}, bone.Name()) {
				// つま先ボーンはつま先IKの位置と同じ
				toeIkBoneName := fmt.Sprintf("%sつま先ＩＫ", bone.Direction())
				toeIkBone := jsonModel.Bones.GetByName(toeIkBoneName)
				if toeIkBone != nil {
					newBone.Position = toeIkBone.Position.Copy()
				}
			}
		}

		// 付与親がある場合、付与親のINDEXを変更
		if (bone.IsEffectorTranslation() || bone.IsEffectorRotation()) && bone.EffectIndex >= 0 {
			jsonEffectBone := jsonModel.Bones.GetByName(model.Bones.Get(bone.EffectIndex).Name())
			newBone.EffectIndex = jsonEffectBone.Index()
			newBone.Layer = max(jsonEffectBone.Layer+1, newBone.Layer)
		}

		// 表示先は位置に変更
		if bone.IsTailBone() {
			// BONE_FLAG_TAIL_IS_BONE を削除
			newBone.BoneFlag = newBone.BoneFlag &^ pmx.BONE_FLAG_TAIL_IS_BONE
			newBone.TailPosition = bone.Extend.ChildRelativePosition.Copy()
		}

		// jsonモデルの後続のボーンの変形階層をひとつ後ろにずらす
		for j := i + 1; j < len(model.Bones.LayerSortedIndexes); j++ {
			childBone := model.Bones.Get(model.Bones.LayerSortedIndexes[j])
			if jsonModel.Bones.ContainsByName(childBone.Name()) {
				jsonChildBone := jsonModel.Bones.GetByName(childBone.Name())
				jsonChildBone.Layer++
				if childBone.ParentIndex >= 0 &&
					jsonModel.Bones.ContainsByName(model.Bones.Get(childBone.ParentIndex).Name()) {
					jsonParentBone := jsonModel.Bones.GetByName(model.Bones.Get(childBone.ParentIndex).Name())
					if jsonParentBone.Layer > jsonChildBone.Layer {
						// 親が子よりも後の階層にある場合、子の階層に合わせる
						jsonParentBone.Layer = jsonChildBone.Layer
					}
				}
			}
		}

		// ボーン追加
		jsonModel.Bones.Append(newBone)
	}
}

func createFitMorph(model, jsonModel *pmx.PmxModel, fitMorphName string) {
	offsets := make([]pmx.IMorphOffset, 0)
	offsetMats := make(map[int]*mmath.MMat4)
	offsetQuats := make(map[int]*mmath.MQuaternion)
	offsetScaleMats := make(map[int]*mmath.MMat4)
	baseScale := getBaseScale(model, jsonModel)

	for _, bone := range model.Bones.Data {
		if jsonBone := jsonModel.Bones.GetByName(bone.Name()); jsonBone != nil {
			config := bone.Config()
			if config == nil {
				continue
			}

			offset := pmx.NewBoneMorphOffset(bone.Index())
			offset.LocalMat = mmath.NewMMat4()

			var jsonChildBone *pmx.Bone
			var childBone *pmx.Bone
			for _, childBoneName := range bone.ConfigChildBoneNames() {
				if model.Bones.ContainsByName(childBoneName) && jsonModel.Bones.ContainsByName(childBoneName) {
					childBone = model.Bones.GetByName(childBoneName)
					jsonChildBone = jsonModel.Bones.GetByName(childBoneName)
					break
				}
			}

			mlog.V("bone: %s", bone.Name())

			// 回転
			if !bone.CanFitOnlyMove() && !bone.IsHead() && childBone != nil && jsonChildBone != nil {
				boneDirection := childBone.Position.Subed(bone.Position).Normalized()
				jsonBoneDirection := jsonChildBone.Position.Subed(jsonBone.Position).Normalized()
				offsetQuat := mmath.NewMQuaternionRotate(boneDirection, jsonBoneDirection)

				offset.LocalMat.Rotate(offsetQuat)
				offsetQuats[bone.Index()] = offsetQuat
				mlog.V("        degrees: %v", offsetQuat.ToMMDDegrees())
			}

			// スケール
			if !bone.CanFitOnlyMove() {
				boneScale := baseScale
				if childBone != nil && jsonChildBone != nil {
					boneDistance := bone.Position.Distance(childBone.Position)
					jsonBoneDistance := jsonBone.Position.Distance(jsonChildBone.Position)
					boneScale = mmath.Effective(jsonBoneDistance / boneDistance)
				}

				if !mmath.NearEquals(boneScale, 0, 1e-4) {
					var scales *mmath.MVec3
					var jsonScaleMat *mmath.MMat4
					if bone.IsHead() {
						scales = &mmath.MVec3{X: baseScale, Y: baseScale, Z: baseScale}
						jsonScaleMat = scales.ToScaleMat4()
					} else {
						scales = &mmath.MVec3{X: boneScale, Y: baseScale, Z: baseScale}
						jsonScaleMat = jsonBone.Extend.LocalAxis.ToScaleLocalMat(scales)
					}

					offset.LocalMat = jsonScaleMat.Muled(offset.LocalMat)
					offsetScaleMats[bone.Index()] = jsonScaleMat
					mlog.V("        scale: %v", scales)
				}
			}

			for _, parentIndex := range bone.Extend.ParentBoneIndexes {
				if _, ok := offsetMats[parentIndex]; ok {
					offset.LocalMat.Mul(offsetMats[parentIndex].Inverted())
				}
			}

			// 移動
			{
				parentMat := mmath.NewMMat4()
				for _, parentIndex := range bone.Extend.ParentBoneIndexes {
					// ルートから自分の親までをかける
					if _, ok := offsetMats[parentIndex]; ok {
						parentUnitMat := model.Bones.Get(parentIndex).Extend.RevertOffsetMatrix.Muled(
							offsetMats[parentIndex])
						parentMat = parentUnitMat.Mul(parentMat)
					}
				}

				unitMat := model.Bones.Get(bone.Index()).Extend.RevertOffsetMatrix.Muled(offset.LocalMat)
				boneMat := parentMat.Muled(unitMat)
				offsetPosition := boneMat.Inverted().MulVec3(jsonBone.Position)

				mlog.V("        trans: %v json: %v)", offsetPosition, jsonBone.Position)
				offset.LocalMat.Mul(offsetPosition.ToMat4())
			}

			offsets = append(offsets, offset)
			offsetMats[bone.Index()] = offset.LocalMat
		}
	}

	morph := pmx.NewMorph()
	morph.SetIndex(model.Morphs.Len())
	morph.SetName(fitMorphName)
	morph.Offsets = offsets
	morph.MorphType = pmx.MORPH_TYPE_BONE
	morph.Panel = pmx.MORPH_PANEL_OTHER_LOWER_RIGHT
	morph.IsSystem = true
	model.Morphs.Append(morph)
}

// 素体モデルのボーン設定を、素体モデルのボーン設定に合わせる
func fixBaseBones(model, jsonModel *pmx.PmxModel) {
	for _, bone := range model.Bones.Data {
		if jsonBone := jsonModel.Bones.GetByName(bone.Name()); jsonBone != nil {
			// bone.TailPosition = jsonBone.TailPosition
			bone.FixedAxis = jsonBone.FixedAxis
			bone.LocalAxisX = jsonBone.LocalAxisX
			bone.LocalAxisZ = jsonBone.LocalAxisZ

			if (bone.IsEffectorRotation() || bone.IsEffectorTranslation()) && model.Bones.Contains(bone.EffectIndex) {
				if jsonEffectorBone := jsonModel.Bones.GetByName(
					model.Bones.Get(bone.EffectIndex).Name()); jsonEffectorBone != nil {
					bone.EffectFactor = jsonBone.EffectFactor
				}
			}

			if bone.Ik != nil && jsonBone.Ik != nil {
				if jsonIkBone := jsonModel.Bones.GetByName(
					model.Bones.Get(bone.Ik.BoneIndex).Name()); jsonIkBone != nil {
					bone.Ik.LoopCount = jsonBone.Ik.LoopCount
					bone.Ik.UnitRotation = jsonBone.Ik.UnitRotation
					for i, jsonLink := range jsonBone.Ik.Links {
						if linkBone := model.Bones.GetByName(
							jsonModel.Bones.Get(jsonLink.BoneIndex).Name()); linkBone != nil {
							bone.Ik.Links[i].AngleLimit = jsonLink.AngleLimit
							bone.Ik.Links[i].MaxAngleLimit = jsonLink.MaxAngleLimit
							bone.Ik.Links[i].MinAngleLimit = jsonLink.MinAngleLimit
							bone.Ik.Links[i].LocalAngleLimit = jsonLink.LocalAngleLimit
							bone.Ik.Links[i].LocalMaxAngleLimit = jsonLink.LocalMaxAngleLimit
							bone.Ik.Links[i].LocalMinAngleLimit = jsonLink.LocalMinAngleLimit
						}
					}
				}
			}
		}
	}

	neckRootPosition := model.Bones.GetByName(pmx.ARM.Left()).Position.Added(
		model.Bones.GetByName(pmx.ARM.Right()).Position).MuledScalar(0.5)

	// 根元ボーンの位置を腕の中央に合わせる
	for _, rootBoneName := range []string{pmx.NECK_ROOT.String(), pmx.SHOULDER_ROOT.Right(), pmx.SHOULDER_ROOT.Left()} {
		neckRootBone := model.Bones.GetByName(rootBoneName)
		neckRootBone.Position = neckRootPosition.Copy()
	}

	// 肩ボーンの位置を、肩・腕・首根元・首から求める
	for _, shoulderBoneName := range []string{pmx.SHOULDER.Left(), pmx.SHOULDER.Right()} {
		armBoneName := strings.ReplaceAll(shoulderBoneName, "肩", "腕")
		shoulderPBoneName := strings.ReplaceAll(shoulderBoneName, "肩", "肩P")
		neckRootBoneName := pmx.NECK_ROOT.String()
		neckBoneName := pmx.NECK.String()

		shoulderBone := model.Bones.GetByName(shoulderBoneName)
		armBone := model.Bones.GetByName(armBoneName)
		shoulderPBone := model.Bones.GetByName(shoulderPBoneName)
		neckRootBone := model.Bones.GetByName(neckRootBoneName)
		neckBone := model.Bones.GetByName(neckBoneName)

		jsonShoulderBone := jsonModel.Bones.GetByName(shoulderBoneName)
		jsonArmBone := jsonModel.Bones.GetByName(armBoneName)
		jsonNeckRootBone := jsonModel.Bones.GetByName(neckRootBoneName)
		jsonNeckBone := jsonModel.Bones.GetByName(neckBoneName)

		// jsonモデルの首根元から見た肩と腕と首の相対位置
		jsonArmByNeckRoot := jsonArmBone.Position.Subed(jsonNeckRootBone.Position)
		jsonShoulderByNeckRoot := jsonShoulderBone.Position.Subed(jsonNeckRootBone.Position)
		jsonNeckByNeckRoot := jsonNeckBone.Position.Subed(jsonNeckRootBone.Position)

		// 素体モデルの首根元から見た腕の相対位置
		armByNeckRoot := armBone.Position.Subed(neckRootBone.Position)

		// スケーリング係数を計算
		shoulderXZRatio := armByNeckRoot.Length() / jsonArmByNeckRoot.Length()

		// 素体モデルの首根元から見た首の相対位置
		neckByNeckRoot := neckBone.Position.Subed(neckRootBone.Position)

		// スケーリング係数を計算
		shoulderYRatio := neckByNeckRoot.Length() / jsonNeckByNeckRoot.Length()

		// 素体モデルの首根元から見た肩の位置に相当する位置を求める
		shoulderOffset := jsonShoulderByNeckRoot.Muled(
			&mmath.MVec3{X: shoulderXZRatio, Y: shoulderYRatio, Z: shoulderXZRatio})

		// 素体モデルの肩ボーンの位置を求める
		shoulderBone.Position = neckRootBone.Position.Added(shoulderOffset)
		shoulderPBone.Position = neckRootBone.Position.Added(shoulderOffset)
	}

	// 腕捩ボーンの位置を、腕とひじの間に合わせる
	for _, twistBoneName := range []string{pmx.ARM_TWIST.Left(), pmx.ARM_TWIST.Right(),
		pmx.WRIST_TWIST.Left(), pmx.WRIST_TWIST.Right()} {
		jsonTwistBone := jsonModel.Bones.GetByName(twistBoneName)
		jsonTwistParentBone := jsonModel.Bones.GetByName(jsonTwistBone.ConfigParentBoneNames()[0])
		jsonTwistChildBone := jsonModel.Bones.GetByName(jsonTwistBone.ConfigChildBoneNames()[0])
		for n := range 3 {
			twistBonePartName := fmt.Sprintf("%s%d", twistBoneName, n+1)
			jsonTwistPartBone := jsonModel.Bones.GetByName(twistBonePartName)
			twistBoneFactor := 0.25 * float64(n+1)
			twistBonePosition := jsonTwistParentBone.Position.Lerp(jsonTwistChildBone.Position, twistBoneFactor)
			jsonTwistPartBone.Position = twistBonePosition
			if n == 1 {
				jsonTwistBone.Position = twistBonePosition.Copy()
			}
		}
	}

	// 素体のつま先IKの位置を、元モデルのつま先IKの位置に合わせる
	for _, toeIkBoneName := range []string{pmx.TOE_IK.Left(), pmx.TOE_IK.Right()} {
		toeIkBone := model.Bones.GetByName(toeIkBoneName)
		jsonToeIkBone := jsonModel.Bones.GetByName(toeIkBoneName)

		ankleBone := model.Bones.GetByName(fmt.Sprintf("%s足首", toeIkBone.Direction()))
		jsonAnkleBone := jsonModel.Bones.GetByName(ankleBone.Name())

		ankleYRatio := ankleBone.Position.Y / jsonAnkleBone.Position.Y
		toeIkZRatio := (toeIkBone.Position.Z - ankleBone.Position.Z) / (jsonToeIkBone.Position.Z - jsonAnkleBone.Position.Z)

		toeIkBone.Position.Y = jsonToeIkBone.Position.Y * ankleYRatio
		toeIkBone.Position.Z = jsonAnkleBone.Position.Z + (toeIkBone.Position.Z-ankleBone.Position.Z)*toeIkZRatio

		toeBone := model.Bones.GetByName(fmt.Sprintf("%sつま先", toeIkBone.Direction()))
		toeBone.Position.Y = toeIkBone.Position.Y
		toeBone.Position.Z = toeIkBone.Position.Z
		toeDBone := model.Bones.GetByName(fmt.Sprintf("%sつま先D", toeIkBone.Direction()))
		toeDBone.Position.Y = toeIkBone.Position.Y
		toeDBone.Position.Z = toeIkBone.Position.Z

		heelBone := model.Bones.GetByName(fmt.Sprintf("%sかかと", toeIkBone.Direction()))
		heelBone.Position.Y = toeIkBone.Position.Y
		heelDBone := model.Bones.GetByName(fmt.Sprintf("%sかかとD", toeIkBone.Direction()))
		heelDBone.Position.Y = toeIkBone.Position.Y
	}
}

func loadOriginalPmxTextures(model *pmx.PmxModel) {
	model.SetPath(filepath.Join(os.TempDir(), "base_model"))
	for _, tex := range model.Textures.Data {
		texPath := filepath.Join("base_model", tex.Name())
		if loadTex(texPath) == nil {
			// 問題なくテクスチャがコピーできたら、パスを設定する
			tex.SetName(texPath)
		}
	}
}

func loadTex(texPath string) error {
	fsTexPath := strings.ReplaceAll(texPath, "\\", "/")
	texFile, err := modelFs.ReadFile(fsTexPath)
	if err != nil {
		mlog.E(fmt.Sprintf("Failed to read original pmx tex file: %s", texPath), err)
		return err
	}

	tmpTexPath := filepath.Join(os.TempDir(), texPath)

	// 仮パスのフォルダ構成を作成する
	err = os.MkdirAll(filepath.Dir(tmpTexPath), 0755)
	if err != nil {
		mlog.E(fmt.Sprintf("Failed to create original pmx tex tmp directory: %s", tmpTexPath), err)
		return err
	}

	// 作業フォルダにファイルを書き込む
	err = os.WriteFile(tmpTexPath, texFile, 0644)
	if err != nil {
		mlog.E(fmt.Sprintf("Failed to write original pmx tex tmp file: %s", tmpTexPath), err)
		return err
	}

	return nil
}
