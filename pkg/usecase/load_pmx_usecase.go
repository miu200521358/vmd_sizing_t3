package usecase

import (
	"embed"
	"fmt"
	"math"
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

func AdjustPmxForSizing(model *pmx.PmxModel) (*pmx.PmxModel, error) {
	// 素体PMXモデルを読み込む
	baseModel, err := loadMannequinPmx()
	if err != nil {
		return nil, err
	}

	// 足りないボーンを追加
	addNonExistBones(baseModel, model, false)

	model.Setup()
	// 強制更新用にハッシュ上書き
	model.SetRandHash()

	return model, nil
}

func LoadOriginalPmxByJson(jsonModel *pmx.PmxModel) (*pmx.PmxModel, error) {
	// 素体PMXモデルを読み込む
	model, err := loadMannequinPmx()
	if err != nil {
		return nil, err
	}

	// テクスチャをTempディレクトリに読み込んでおく
	loadOriginalPmxTextures(model)

	// 足りないボーンを追加
	addNonExistBones(model, jsonModel, true)

	jsonModel.Setup()
	model.Setup()
	// 強制更新用にハッシュ上書き
	model.SetRandHash()

	// フィットボーンモーフを作成
	createFitMorph(model, jsonModel, fit_morph_name)
	model.Setup()

	return model, nil
}

func AddFitMorph(motion *vmd.VmdMotion) *vmd.VmdMotion {
	if motion.MorphFrames != nil && motion.MorphFrames.Contains(fit_morph_name) {
		return motion
	}

	// フィットボーンモーフを適用
	mf := vmd.NewMorphFrame(float32(0))
	mf.Ratio = 1.0
	motion.AppendMorphFrame(fit_morph_name, mf)
	return motion
}

func RemakeFitMorph(model, jsonModel *pmx.PmxModel, sizingSet *model.SizingSet) *pmx.PmxModel {
	model.Morphs.RemoveByName(fit_morph_name)

	// 足りないボーンを追加
	addNonExistBones(model, jsonModel, true)

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

// baseModel にあって、 model にないボーンを追加する
func addNonExistBones(baseModel, model *pmx.PmxModel, fromJson bool) {
	if !model.Bones.ContainsByName(pmx.ARM.Left()) || !model.Bones.ContainsByName(pmx.ARM.Right()) {
		return
	}

	ratio := getBaseScale(baseModel, model)
	nonExistBoneNames := make([]string, 0)

	for _, baseBoneIndex := range baseModel.Bones.LayerSortedIndexes {
		baseBone := baseModel.Bones.Get(baseBoneIndex)

		if slices.Contains([]string{"両目光", "左目光", "右目光", "舌1", "舌2", "舌3", "舌4"}, baseBone.Name()) {
			continue
		}

		// 存在するボーンの場合
		if model.Bones.ContainsByName(baseBone.Name()) {
			bone := model.Bones.GetByName(baseBone.Name())

			if baseBone.CanTranslate() {
				bone.BoneFlag |= pmx.BONE_FLAG_CAN_TRANSLATE
			}
			if baseBone.CanRotate() {
				bone.BoneFlag |= pmx.BONE_FLAG_CAN_ROTATE
			}
			if baseBone.CanManipulate() {
				bone.BoneFlag |= pmx.BONE_FLAG_CAN_MANIPULATE
			}
			if baseBone.IsVisible() {
				bone.BoneFlag |= pmx.BONE_FLAG_IS_VISIBLE
			}

			if baseBone.Name() == pmx.ROOT.String() {
				// 全ての親はそのまま
				continue
			}
			parentIndex := bone.ParentIndex
			if bone.ParentIndex < 0 {
				// 親が存在しない場合、ROOTを親にする
				parentIndex = model.Bones.GetByName(pmx.ROOT.String()).Index()
			}

			parentName := model.Bones.Get(parentIndex).Name()
			baseParentName := baseModel.Bones.Get(baseBone.ParentIndex).Name()

			// 必要に応じて親を切り替える
			var parentBone *pmx.Bone
			if !baseModel.Bones.ContainsByName(parentName) {
				// 素体モデルに存在しないボーン名が親の場合、その親を使用する
				parentBone = model.Bones.GetByName(parentName)
			} else {
				// それ以外は素体モデルの親を使用する
				parentBone = model.Bones.GetByName(baseParentName)
			}

			if parentBone == nil {
				// 親が存在しない場合、ROOTを親にする
				bone.ParentIndex = -1
			} else {
				bone.ParentIndex = parentBone.Index()
			}

			continue
		}

		// 存在しないボーンは追加
		newBone := pmx.NewBone()
		// 最後に追加
		newBone.SetIndex(model.Bones.Len())
		newBone.SetName(baseBone.Name())
		newBone.SetEnglishName(baseBone.EnglishName())
		newBone.BoneFlag = baseBone.BoneFlag
		if baseBone.ParentIndex < 0 {
			if newBone.Name() == pmx.ROOT.String() {
				newBone.ParentIndex = -1
			} else {
				newBone.ParentIndex = model.Bones.GetByName(pmx.ROOT.String()).Index()
			}
		} else {
			baseParentBone := baseModel.Bones.Get(baseBone.ParentIndex)
			parentBone := model.Bones.GetByName(baseParentBone.Name())
			if parentBone == nil {
				continue
			}

			newBone.ParentIndex = parentBone.Index()

			// 親からの相対位置から比率で求める
			newBone.Position = parentBone.Position.Added(baseBone.Extend.ParentRelativePosition.MuledScalar(ratio))

			if baseBone.Name() == pmx.WAIST.String() {
				// 腰は上下が揃ってたら
				upperBone := model.Bones.GetByName(pmx.UPPER.String())
				lowerBone := model.Bones.GetByName(pmx.LOWER.String())
				if upperBone == nil || lowerBone == nil {
					continue
				}
			} else if baseBone.Name() == pmx.UPPER2.String() {
				// 上半身2の場合、首根元と上半身の間に置く
				baseNeckRootBone := baseModel.Bones.GetByName(pmx.NECK_ROOT.String())
				baseUpperBone := baseModel.Bones.GetByName(pmx.UPPER.String())
				baseUpper2Bone := baseModel.Bones.GetByName(pmx.UPPER2.String())

				upperBone := model.Bones.GetByName(baseUpperBone.Name())
				neckBone := model.Bones.GetByName(pmx.NECK.String())
				if upperBone == nil || neckBone == nil {
					continue
				}

				neckRootPosition := model.Bones.GetByName(pmx.ARM.Left()).Position.Added(
					model.Bones.GetByName(pmx.ARM.Right()).Position).MuledScalar(0.5)

				// 上半身の長さを上半身と首根元の距離で求める
				baseUpperLength := baseUpperBone.Position.Distance(baseNeckRootBone.Position)
				baseUpper2Length := baseUpper2Bone.Position.Distance(baseUpperBone.Position) * 1.5
				upperRatio := baseUpper2Length / baseUpperLength

				newBone.Position = upperBone.Position.Added(
					neckRootPosition.Subed(upperBone.Position).MuledScalar(upperRatio))
				upperBone.TailIndex = newBone.Index()
				upperBone.BoneFlag |= pmx.BONE_FLAG_TAIL_IS_BONE
			} else if strings.Contains(baseBone.Name(), "腕捩") {
				if !((slices.Contains(nonExistBoneNames, pmx.ARM_TWIST.Left()) &&
					newBone.Name() == pmx.ARM_TWIST1.Left()) &&
					(slices.Contains(nonExistBoneNames, pmx.ARM_TWIST.Left()) &&
						newBone.Name() == pmx.ARM_TWIST2.Left()) &&
					(slices.Contains(nonExistBoneNames, pmx.ARM_TWIST.Left()) &&
						newBone.Name() == pmx.ARM_TWIST3.Left()) &&
					(slices.Contains(nonExistBoneNames, pmx.ARM_TWIST.Right()) &&
						newBone.Name() == pmx.ARM_TWIST1.Right()) &&
					(slices.Contains(nonExistBoneNames, pmx.ARM_TWIST.Right()) &&
						newBone.Name() == pmx.ARM_TWIST2.Right()) &&
					(slices.Contains(nonExistBoneNames, pmx.ARM_TWIST.Right()) &&
						newBone.Name() == pmx.ARM_TWIST3.Right())) {
					// 分割ボーンは元の捩りボーンが追加されている時だけにする
					continue
				}

				// 腕捩の場合、腕とひじの間に置く
				baseArmBone := baseModel.Bones.GetByName(
					strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(
						baseBone.Name(), "腕捩1", "腕"), "腕捩2", "腕"), "腕捩3", "腕"), "腕捩", "腕"))
				baseElbowBone := baseModel.Bones.GetByName(
					strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(
						baseBone.Name(), "腕捩1", "ひじ"), "腕捩2", "ひじ"), "腕捩3", "ひじ"), "腕捩", "ひじ"))

				twistRatio := baseBone.Position.Subed(baseArmBone.Position).Length() / baseElbowBone.Position.Subed(baseArmBone.Position).Length()

				armBone := model.Bones.GetByName(baseArmBone.Name())
				elbowBone := model.Bones.GetByName(baseElbowBone.Name())

				if armBone == nil || elbowBone == nil {
					continue
				}

				newBone.Position = armBone.Position.Lerp(elbowBone.Position, twistRatio)
				newBone.FixedAxis = elbowBone.Position.Subed(armBone.Position).Normalized()
			} else if strings.Contains(baseBone.Name(), "手捩") {
				if !((slices.Contains(nonExistBoneNames, pmx.WRIST_TWIST.Left()) &&
					newBone.Name() == pmx.WRIST_TWIST1.Left()) &&
					(slices.Contains(nonExistBoneNames, pmx.WRIST_TWIST.Left()) &&
						newBone.Name() == pmx.WRIST_TWIST2.Left()) &&
					(slices.Contains(nonExistBoneNames, pmx.WRIST_TWIST.Left()) &&
						newBone.Name() == pmx.WRIST_TWIST3.Left()) &&
					(slices.Contains(nonExistBoneNames, pmx.WRIST_TWIST.Right()) &&
						newBone.Name() == pmx.WRIST_TWIST1.Right()) &&
					(slices.Contains(nonExistBoneNames, pmx.WRIST_TWIST.Right()) &&
						newBone.Name() == pmx.WRIST_TWIST2.Right()) &&
					(slices.Contains(nonExistBoneNames, pmx.WRIST_TWIST.Right()) &&
						newBone.Name() == pmx.WRIST_TWIST3.Right())) {
					// 分割ボーンは元の捩りボーンが追加されている時だけにする
					continue
				}

				baseElbowBone := baseModel.Bones.GetByName(
					strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(
						baseBone.Name(), "手捩1", "ひじ"), "手捩2", "ひじ"), "手捩3", "ひじ"), "手捩", "ひじ"))
				baseWristBone := baseModel.Bones.GetByName(
					strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(
						baseBone.Name(), "手捩1", "手首"), "手捩2", "手首"), "手捩3", "手首"), "手捩", "手首"))

				twistRatio := baseBone.Position.Subed(baseElbowBone.Position).Length() / baseWristBone.Position.Subed(baseElbowBone.Position).Length()

				elbowBone := model.Bones.GetByName(baseElbowBone.Name())
				wristBone := model.Bones.GetByName(baseWristBone.Name())

				if elbowBone == nil || wristBone == nil {
					continue
				}

				newBone.Position = elbowBone.Position.Lerp(wristBone.Position, twistRatio)
				newBone.FixedAxis = wristBone.Position.Subed(elbowBone.Position).Normalized()
			} else if baseBone.Name() == pmx.SHOULDER_P.Left() || baseBone.Name() == pmx.SHOULDER_P.Right() {
				// 肩Pの場合、肩と同じ位置に置く
				baseShoulderBone := baseModel.Bones.GetByName(strings.ReplaceAll(baseBone.Name(), "肩P", "肩"))
				shoulderBone := model.Bones.GetByName(baseShoulderBone.Name())

				if shoulderBone == nil {
					continue
				}

				newBone.Position = shoulderBone.Position.Copy()
			} else if slices.Contains([]string{pmx.SHOULDER_C.Left(), pmx.SHOULDER_C.Right()}, baseBone.Name()) {
				// 肩Cの場合、腕と同じ位置に置く
				baseArmBone := baseModel.Bones.GetByName(strings.ReplaceAll(baseBone.Name(), "肩C", "腕"))
				armBone := model.Bones.GetByName(baseArmBone.Name())

				if armBone == nil {
					continue
				}

				newBone.Position = armBone.Position.Copy()
			} else if slices.Contains([]string{pmx.NECK_ROOT.String(), pmx.SHOULDER_ROOT.Left(), pmx.SHOULDER_ROOT.Right()}, baseBone.Name()) {
				// 首根元・肩根元は首根元の位置
				newBone.Position = model.Bones.GetByName(pmx.ARM.Left()).Position.Added(
					model.Bones.GetByName(pmx.ARM.Right()).Position).MuledScalar(0.5)
				if baseBone.Name() == pmx.NECK_ROOT.String() {
					// 首根元の場合、上半身2ボーンの表示先として設定
					upper2Bone := model.Bones.GetByName(pmx.UPPER2.String())

					if upper2Bone == nil {
						continue
					}

					upper2Bone.TailIndex = newBone.Index()
					upper2Bone.BoneFlag |= pmx.BONE_FLAG_TAIL_IS_BONE
				}
				newBone.IsSystem = true
			} else if slices.Contains([]string{pmx.THUMB0.Left(), pmx.THUMB0.Right()}, baseBone.Name()) {
				// 親指０は手首と親指１の間
				baseWristBone := baseModel.Bones.GetByName(strings.ReplaceAll(baseBone.Name(), "親指０", "手首"))
				baseThumbBone := baseModel.Bones.GetByName(strings.ReplaceAll(baseBone.Name(), "親指０", "親指１"))
				thumbRatio := baseBone.Position.Subed(baseWristBone.Position).Length() / baseThumbBone.Position.Subed(baseWristBone.Position).Length()

				wristBone := model.Bones.GetByName(baseWristBone.Name())
				thumbBone := model.Bones.GetByName(baseThumbBone.Name())

				if wristBone == nil || thumbBone == nil {
					continue
				}

				newBone.Position = wristBone.Position.Lerp(thumbBone.Position, min(0.8, thumbRatio*2))
				newBone.TailIndex = thumbBone.Index()
				newBone.BoneFlag |= pmx.BONE_FLAG_TAIL_IS_BONE
			} else if slices.Contains([]string{pmx.LEG_CENTER.String(), pmx.LEG_ROOT.Left(), pmx.LEG_ROOT.Right()}, baseBone.Name()) {
				// 足中心は足の中心
				newBone.Position = model.Bones.GetByName(pmx.LEG.Left()).Position.Added(
					model.Bones.GetByName(pmx.LEG.Right()).Position).MuledScalar(0.5)
				newBone.IsSystem = true
			} else if strings.Contains(baseBone.Name(), "腰キャンセル") {
				// 腰キャンセルは足と同じ位置
				baseLegBoneName := fmt.Sprintf("%s足", baseBone.Direction())
				legBone := model.Bones.GetByName(baseLegBoneName)

				if legBone == nil {
					continue
				}

				newBone.Position = legBone.Position.Copy()
			} else if slices.Contains([]string{pmx.TRUNK_ROOT.String(), pmx.UPPER_ROOT.String(), pmx.LOWER_ROOT.String()}, baseBone.Name()) {
				// 体幹中心・上半身根元・下半身根元は上半身と下半身の間
				upperBone := model.Bones.GetByName(pmx.UPPER.String())
				lowerBone := model.Bones.GetByName(pmx.LOWER.String())
				if upperBone == nil || lowerBone == nil {
					continue
				}

				newBone.Position = upperBone.Position.Lerp(lowerBone.Position, 0.5)
				newBone.IsSystem = true
			} else if slices.Contains([]string{pmx.LEG_IK_PARENT.Left(), pmx.LEG_IK_PARENT.Right()}, baseBone.Name()) {
				// 足IK親 は 足IKのYを0にした位置
				baseLegIkBone := baseModel.Bones.GetByName(strings.ReplaceAll(baseBone.Name(), "足IK親", "足ＩＫ"))
				legIkBone := model.Bones.GetByName(baseLegIkBone.Name())

				if legIkBone == nil {
					continue
				}

				newBone.Position = legIkBone.Position.Copy()
				newBone.Position.Y = 0
			} else if slices.Contains([]string{pmx.LEG_D.Left(), pmx.LEG_D.Right()}, baseBone.Name()) {
				// 足D は 足の位置
				baseLegBone := baseModel.Bones.GetByName(strings.ReplaceAll(baseBone.Name(), "足D", "足"))
				legBone := model.Bones.GetByName(baseLegBone.Name())

				if legBone == nil {
					continue
				}

				newBone.Position = legBone.Position.Copy()
			} else if slices.Contains([]string{pmx.KNEE_D.Left(), pmx.KNEE_D.Right()}, baseBone.Name()) {
				// ひざD は ひざの位置
				baseKneeBone := baseModel.Bones.GetByName(strings.ReplaceAll(baseBone.Name(), "ひざD", "ひざ"))
				kneeBone := model.Bones.GetByName(baseKneeBone.Name())

				if kneeBone == nil {
					continue
				}

				newBone.Position = kneeBone.Position.Copy()
			} else if slices.Contains([]string{pmx.ANKLE_D.Left(), pmx.ANKLE_D.Right()}, baseBone.Name()) {
				// 足首D は 足首の位置
				baseAnkleBone := baseModel.Bones.GetByName(strings.ReplaceAll(baseBone.Name(), "足首D", "足首"))
				ankleBone := model.Bones.GetByName(baseAnkleBone.Name())

				if ankleBone == nil {
					continue
				}

				newBone.Position = ankleBone.Position.Copy()
			} else if slices.Contains([]string{pmx.TOE_EX.Left(), pmx.TOE_EX.Right()}, baseBone.Name()) {
				// 足先EXは 足首とつま先の間
				baseAnkleBone := baseModel.Bones.GetByName(strings.ReplaceAll(baseBone.Name(), "足先EX", "足首"))
				// つま先のボーン名は標準ではないので、つま先ＩＫのターゲットから取る
				baseToeBone := baseModel.Bones.Get(baseModel.Bones.GetByName(strings.ReplaceAll(baseBone.Name(), "足先EX", "つま先ＩＫ")).Ik.BoneIndex)
				toeRatio := baseBone.Position.Subed(baseAnkleBone.Position).Length() / baseToeBone.Position.Subed(baseAnkleBone.Position).Length()

				ankleBone := model.Bones.GetByName(baseAnkleBone.Name())
				toeBone := model.Bones.GetByName(baseToeBone.Name())

				if ankleBone == nil || toeBone == nil {
					continue
				}

				newBone.Position = ankleBone.Position.Lerp(toeBone.Position, toeRatio)
			} else if slices.Contains([]string{pmx.HEEL.Left(), pmx.HEEL.Right(), pmx.HEEL_D.Left(), pmx.HEEL_D.Right()}, baseBone.Name()) {
				// かかとXは足首Dと同じ
				baseAnkleBone := baseModel.Bones.GetByName(strings.ReplaceAll(baseBone.Name(), "かかと", "足首"))
				ankleBone := model.Bones.GetByName(baseAnkleBone.Name())

				if ankleBone == nil {
					continue
				}

				newBone.Position.X = ankleBone.Position.X
				newBone.Position.Y = 0
				newBone.IsSystem = true
			} else if strings.Contains(baseBone.Name(), "指先") {
				// 指先ボーンは親ボーンの相対表示先位置
				baseParentBoneName := baseBone.ConfigParentBoneNames()[0]
				parentBone := model.Bones.GetByName(baseParentBoneName)
				if parentBone != nil {
					newBone.Position = parentBone.Position.Added(parentBone.Extend.ChildRelativePosition)
				}
				newBone.IsSystem = true
			} else if slices.Contains([]string{pmx.TOE.Left(), pmx.TOE.Right(), pmx.TOE_D.Left(), pmx.TOE_D.Right()}, baseBone.Name()) {
				// つま先ボーンはつま先IKの位置と同じ
				toeIkBoneName := fmt.Sprintf("%sつま先ＩＫ", baseBone.Direction())
				toeIkBone := model.Bones.GetByName(toeIkBoneName)
				if toeIkBone == nil {
					continue
				}
				newBone.Position = toeIkBone.Position.Copy()
				newBone.IsSystem = true
			} else if slices.Contains([]string{pmx.TOE_C_D.Left(), pmx.TOE_C_D.Right()}, baseBone.Name()) {
				// つま先子Dはつま先子の位置
				toeCBoneName := fmt.Sprintf("%sつま先子", baseBone.Direction())
				toeCBone := model.Bones.GetByName(toeCBoneName)
				if toeCBone == nil {
					continue
				}
				newBone.Position = toeCBone.Position.Copy()
				newBone.IsSystem = true
			} else if slices.Contains([]string{pmx.TOE_P_D.Left(), pmx.TOE_P_D.Right()}, baseBone.Name()) {
				// つま先親Dはつま先親の位置
				toePBoneName := fmt.Sprintf("%sつま先親", baseBone.Direction())
				toePBone := model.Bones.GetByName(toePBoneName)
				if toePBone == nil {
					continue
				}
				newBone.Position = toePBone.Position.Copy()
				newBone.IsSystem = true
			}
		}

		afterIndex := newBone.ParentIndex

		// 付与親がある場合、付与親のINDEXを変更
		if (baseBone.IsEffectorTranslation() || baseBone.IsEffectorRotation()) && baseBone.EffectIndex >= 0 {
			effectBone := model.Bones.GetByName(baseModel.Bones.Get(baseBone.EffectIndex).Name())
			newBone.EffectIndex = effectBone.Index()
			newBone.EffectFactor = baseBone.EffectFactor

			parentLayerIndex := slices.Index(model.Bones.LayerSortedIndexes, newBone.ParentIndex)
			effectLayerIndex := slices.Index(model.Bones.LayerSortedIndexes, effectBone.Index())
			if parentLayerIndex < effectLayerIndex {
				afterIndex = effectBone.Index()
			}
		}

		// // 表示先は位置に変更
		// if baseBone.IsTailBone() {
		// 	// BONE_FLAG_TAIL_IS_BONE を削除
		// 	newBone.BoneFlag = newBone.BoneFlag &^ pmx.BONE_FLAG_TAIL_IS_BONE
		// 	newBone.TailPosition = baseBone.Extend.ChildRelativePosition.Copy()
		// }

		// ボーン追加
		model.Bones.Insert(newBone, afterIndex)
		nonExistBoneNames = append(nonExistBoneNames, newBone.Name())
	}

	// ボーン設定を補正
	fixBaseBones(baseModel, model, fromJson, nonExistBoneNames)

	// ウェイト調整
	fixDeformWeights(model, nonExistBoneNames)
}

func fixDeformWeights(model *pmx.PmxModel, nonExistBoneNames []string) {
	var allBoneVertices map[int][]*pmx.Vertex

	// ウェイト調整が必要なボーンを追加した場合
	if slices.Contains(nonExistBoneNames, pmx.UPPER2.String()) ||
		slices.Contains(nonExistBoneNames, pmx.LEG_D.Right()) ||
		slices.Contains(nonExistBoneNames, pmx.KNEE_D.Right()) ||
		slices.Contains(nonExistBoneNames, pmx.ANKLE_D.Right()) ||
		slices.Contains(nonExistBoneNames, pmx.LEG_D.Left()) ||
		slices.Contains(nonExistBoneNames, pmx.KNEE_D.Left()) ||
		slices.Contains(nonExistBoneNames, pmx.ANKLE_D.Left()) ||
		slices.Contains(nonExistBoneNames, pmx.TOE_EX.Right()) ||
		slices.Contains(nonExistBoneNames, pmx.TOE_EX.Left()) ||
		slices.Contains(nonExistBoneNames, pmx.THUMB0.Right()) ||
		slices.Contains(nonExistBoneNames, pmx.THUMB0.Left()) ||
		slices.Contains(nonExistBoneNames, pmx.ARM_TWIST.Right()) ||
		slices.Contains(nonExistBoneNames, pmx.ARM_TWIST.Left()) ||
		slices.Contains(nonExistBoneNames, pmx.WRIST_TWIST.Right()) ||
		slices.Contains(nonExistBoneNames, pmx.WRIST_TWIST.Left()) {
		// 頂点をボーンINDEX別に纏める
		allBoneVertices = model.Vertices.GetMapByBoneIndex(0.0)
	}

	if allBoneVertices == nil {
		// ウェイト調整が不要な場合、終了
		return
	}

	// D系の置き換え
	for _, boneNames := range [][]string{
		{pmx.LEG.Right(), pmx.LEG_D.Right()},
		{pmx.KNEE.Right(), pmx.KNEE_D.Right()},
		{pmx.ANKLE.Right(), pmx.ANKLE_D.Right()},
		{pmx.LEG.Left(), pmx.LEG_D.Left()},
		{pmx.KNEE.Left(), pmx.KNEE_D.Left()},
		{pmx.ANKLE.Left(), pmx.ANKLE_D.Left()},
	} {
		if !slices.Contains(nonExistBoneNames, boneNames[1]) {
			continue
		}
		fkBone := model.Bones.GetByName(boneNames[0])
		dBone := model.Bones.GetByName(boneNames[1])

		for _, vertex := range allBoneVertices[fkBone.Index()] {
			deformIndex := vertex.Deform.Index(fkBone.Index())
			if deformIndex < 0 {
				continue
			}
			vertex.Deform.AllIndexes()[deformIndex] = dBone.Index()
		}
	}

	// 足先EXの置き換え
	for _, boneNames := range [][]string{
		{pmx.ANKLE.Right(), pmx.TOE_EX.Right(), pmx.TOE.Right(), pmx.ANKLE_D.Right()},
		{pmx.ANKLE.Left(), pmx.TOE_EX.Left(), pmx.TOE.Left(), pmx.ANKLE_D.Left()},
	} {
		if !slices.Contains(nonExistBoneNames, boneNames[1]) {
			continue
		}

		ankleBone := model.Bones.GetByName(boneNames[0])
		toeExBone := model.Bones.GetByName(boneNames[1])
		toeBone := model.Bones.GetByName(boneNames[2])
		ankleDBone := model.Bones.GetByName(boneNames[3])
		overlap := (toeBone.Position.Z - toeExBone.Position.Z) * 0.3

		for _, vertex := range allBoneVertices[ankleBone.Index()] {
			switch vertex.Deform.(type) {
			case *pmx.Sdef:
				continue
			}

			vertexRatio := 1.0
			if vertex.Position.Z > toeExBone.Position.Z-overlap {
				continue
			} else if vertex.Position.Z > toeExBone.Position.Z+overlap {
				vertexRatio = (math.Abs(toeBone.Position.Z-toeExBone.Position.Z) - math.Abs(overlap)) /
					(math.Abs(toeBone.Position.Z-toeExBone.Position.Z) + math.Abs(overlap*2))
			}
			vertex.Deform.Add(toeExBone.Index(), ankleDBone.Index(), vertexRatio)

			switch len(vertex.Deform.AllIndexes()) {
			case 1:
				vertex.Deform = pmx.NewBdef1(vertex.Deform.AllIndexes()[0])
			case 2:
				vertex.Deform = pmx.NewBdef2(vertex.Deform.AllIndexes()[0], vertex.Deform.AllIndexes()[1],
					vertex.Deform.AllWeights()[0])
			case 4:
				vertex.Deform = pmx.NewBdef4(
					vertex.Deform.AllIndexes()[0], vertex.Deform.AllIndexes()[1],
					vertex.Deform.AllIndexes()[2], vertex.Deform.AllIndexes()[3],
					vertex.Deform.AllWeights()[0], vertex.Deform.AllWeights()[1],
					vertex.Deform.AllWeights()[2], vertex.Deform.AllWeights()[3])
			}
		}
	}

	// 上半身2の置き換え
	if slices.Contains(nonExistBoneNames, pmx.UPPER2.String()) {
		upperBone := model.Bones.GetByName(pmx.UPPER.String())
		upper2Bone := model.Bones.GetByName(pmx.UPPER2.String())
		overlap := (upper2Bone.Position.Y - upperBone.Position.Y) * 0.3

		for _, vertex := range allBoneVertices[upperBone.Index()] {
			switch vertex.Deform.(type) {
			case *pmx.Sdef:
				continue
			}

			vertexRatio := 1.0
			if vertex.Position.Y < upperBone.Position.Y+overlap {
				continue
			} else if vertex.Position.Y < upper2Bone.Position.Y+overlap {
				vertexRatio = (vertex.Position.Y - overlap - upperBone.Position.Y) /
					(upper2Bone.Position.Y + overlap*2 - upperBone.Position.Y)
			}
			vertex.Deform.Add(upper2Bone.Index(), upperBone.Index(), vertexRatio)

			switch len(vertex.Deform.AllIndexes()) {
			case 1:
				vertex.Deform = pmx.NewBdef1(vertex.Deform.AllIndexes()[0])
			case 2:
				vertex.Deform = pmx.NewBdef2(vertex.Deform.AllIndexes()[0], vertex.Deform.AllIndexes()[1],
					vertex.Deform.AllWeights()[0])
			case 4:
				vertex.Deform = pmx.NewBdef4(
					vertex.Deform.AllIndexes()[0], vertex.Deform.AllIndexes()[1],
					vertex.Deform.AllIndexes()[2], vertex.Deform.AllIndexes()[3],
					vertex.Deform.AllWeights()[0], vertex.Deform.AllWeights()[1],
					vertex.Deform.AllWeights()[2], vertex.Deform.AllWeights()[3])
			}
		}
	}

	// 親指0の置き換え
	for _, boneNames := range [][]string{
		{pmx.WRIST.Right(), pmx.THUMB0.Right(), pmx.THUMB1.Right(), pmx.INDEX1.Right()},
		{pmx.WRIST.Left(), pmx.THUMB0.Left(), pmx.THUMB1.Left(), pmx.INDEX1.Left()},
	} {
		if !slices.Contains(nonExistBoneNames, boneNames[1]) {
			continue
		}

		wristBone := model.Bones.GetByName(boneNames[0])
		thumb0Bone := model.Bones.GetByName(boneNames[1])
		thumb1Bone := model.Bones.GetByName(boneNames[2])
		index1Bone := model.Bones.GetByName(boneNames[3])
		overlap := thumb1Bone.Position.Distance(wristBone.Position) * 0.5

		for _, vertex := range allBoneVertices[wristBone.Index()] {
			switch vertex.Deform.(type) {
			case *pmx.Sdef:
				continue
			}

			// 手首から指0へのベクトルと頂点位置の直交地点
			thumb0Orthogonal := mmath.IntersectLinePoint(wristBone.Position, thumb0Bone.Position, vertex.Position)

			vertexRatio := 1.0
			if vertex.Position.Z < (index1Bone.Position.Z+thumb0Bone.Position.Z)/2 &&
				vertex.Position.Distance(thumb0Bone.Position) < vertex.Position.Distance(index1Bone.Position) &&
				(vertex.Position.Distance(thumb0Bone.Position) < vertex.Position.Distance(thumb1Bone.Position) ||
					vertex.Position.Distance(thumb0Bone.Position) < vertex.Position.Distance(wristBone.Position)) &&
				(vertex.Position.Distance(thumb0Bone.Position) < overlap ||
					vertex.Position.Z < thumb0Bone.Position.Z) {
				vertexRatio = (thumb0Orthogonal.Distance(thumb0Bone.Position) + overlap) /
					(wristBone.Position.Distance(thumb0Bone.Position))
			} else {
				continue
			}
			vertex.Deform.Add(thumb0Bone.Index(), wristBone.Index(), vertexRatio)

			switch len(vertex.Deform.AllIndexes()) {
			case 1:
				vertex.Deform = pmx.NewBdef1(vertex.Deform.AllIndexes()[0])
			case 2:
				vertex.Deform = pmx.NewBdef2(vertex.Deform.AllIndexes()[0], vertex.Deform.AllIndexes()[1],
					vertex.Deform.AllWeights()[0])
			case 4:
				vertex.Deform = pmx.NewBdef4(
					vertex.Deform.AllIndexes()[0], vertex.Deform.AllIndexes()[1],
					vertex.Deform.AllIndexes()[2], vertex.Deform.AllIndexes()[3],
					vertex.Deform.AllWeights()[0], vertex.Deform.AllWeights()[1],
					vertex.Deform.AllWeights()[2], vertex.Deform.AllWeights()[3])
			}
		}
	}

	// 捩の置き換え
	for _, boneNames := range [][]string{
		{pmx.ARM.Right(), pmx.ARM_TWIST.Right(), pmx.ARM_TWIST1.Right(), pmx.ARM_TWIST2.Right(), pmx.ARM_TWIST3.Right(), pmx.ELBOW.Right()},
		{pmx.ARM.Left(), pmx.ARM_TWIST.Left(), pmx.ARM_TWIST1.Left(), pmx.ARM_TWIST2.Left(), pmx.ARM_TWIST3.Left(), pmx.ELBOW.Left()},
		{pmx.ELBOW.Right(), pmx.WRIST_TWIST.Right(), pmx.WRIST_TWIST1.Right(), pmx.WRIST_TWIST2.Right(), pmx.WRIST_TWIST3.Right(), pmx.WRIST.Right()},
		{pmx.ELBOW.Left(), pmx.WRIST_TWIST.Left(), pmx.WRIST_TWIST1.Left(), pmx.WRIST_TWIST2.Left(), pmx.WRIST_TWIST3.Left(), pmx.WRIST.Left()},
	} {
		if !slices.Contains(nonExistBoneNames, boneNames[1]) {
			continue
		}

		parentBone := model.Bones.GetByName(boneNames[0])
		twistBone := model.Bones.GetByName(boneNames[1])
		twist1Bone := model.Bones.GetByName(boneNames[2])
		twist2Bone := model.Bones.GetByName(boneNames[3])
		twist3Bone := model.Bones.GetByName(boneNames[4])
		childBone := model.Bones.GetByName(boneNames[5])

		for _, vertex := range allBoneVertices[parentBone.Index()] {
			switch vertex.Deform.(type) {
			case *pmx.Sdef:
				continue
			}

			parentDistance := parentBone.Position.Distance(vertex.Position)
			twist1Distance := twist1Bone.Position.Distance(vertex.Position)
			twist2Distance := twist2Bone.Position.Distance(vertex.Position)
			twist3Distance := twist3Bone.Position.Distance(vertex.Position)
			childDistance := childBone.Position.Distance(vertex.Position)

			nearestIndex := mmath.ArgSort(mmath.Float64Slice{parentDistance, twist1Distance, twist2Distance, twist3Distance, childDistance})[0]
			// ひじに最も近い頂点は捩りに割り当てる
			nearestBone := []*pmx.Bone{parentBone, twist1Bone, twist2Bone, twist3Bone, twistBone}[nearestIndex]
			nearestPositionBone := []*pmx.Bone{parentBone, twist1Bone, twist2Bone, twist3Bone, childBone}[nearestIndex]
			nearestParentBone := []*pmx.Bone{parentBone, parentBone, twist1Bone, twist2Bone, twist3Bone}[nearestIndex]
			nearestChildBone := []*pmx.Bone{twist1Bone, twist2Bone, twist3Bone, childBone, childBone}[nearestIndex]

			// 腕ベクトルと頂点位置の直交地点
			twistOrthogonal := mmath.IntersectLinePoint(parentBone.Position, childBone.Position, vertex.Position)

			rangeDistance := nearestParentBone.Position.Distance(nearestChildBone.Position)
			nearestDistance := nearestPositionBone.Position.Distance(twistOrthogonal)
			vertexRatio := (rangeDistance - nearestDistance) / rangeDistance
			vertex.Deform.Add(nearestBone.Index(), parentBone.Index(), vertexRatio)

			// 腕とかの残りを親に割り当てる
			parentDeformIndex := vertex.Deform.Index(parentBone.Index())
			if parentDeformIndex >= 0 {
				vertex.Deform.AllIndexes()[parentDeformIndex] = nearestParentBone.Index()
				vertex.Deform.Normalize(true)
			}

			switch len(vertex.Deform.AllIndexes()) {
			case 1:
				vertex.Deform = pmx.NewBdef1(vertex.Deform.AllIndexes()[0])
			case 2:
				vertex.Deform = pmx.NewBdef2(vertex.Deform.AllIndexes()[0], vertex.Deform.AllIndexes()[1],
					vertex.Deform.AllWeights()[0])
			case 4:
				vertex.Deform = pmx.NewBdef4(
					vertex.Deform.AllIndexes()[0], vertex.Deform.AllIndexes()[1],
					vertex.Deform.AllIndexes()[2], vertex.Deform.AllIndexes()[3],
					vertex.Deform.AllWeights()[0], vertex.Deform.AllWeights()[1],
					vertex.Deform.AllWeights()[2], vertex.Deform.AllWeights()[3])
			}
		}
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
			boneMat.Mul(offsetPosition.ToMat4())

			offsets = append(offsets, offset)
			offsetMats[bone.Index()] = offset.LocalMat

			jsonRigidBody := jsonModel.RigidBodies.GetByName(bone.Name())
			if jsonRigidBody != nil {
				rigidBody := &pmx.RigidBody{}
				rigidBody.SetIndex(model.RigidBodies.Len())
				rigidBody.SetName(jsonRigidBody.Name())
				rigidBody.SetEnglishName(jsonRigidBody.EnglishName())
				rigidBody.BoneIndex = model.Bones.GetByName(jsonRigidBody.Bone.Name()).Index()
				rigidBody.CollisionGroup = jsonRigidBody.CollisionGroup
				rigidBody.CollisionGroupMask = jsonRigidBody.CollisionGroupMask
				rigidBody.CollisionGroupMaskValue = jsonRigidBody.CollisionGroupMaskValue
				rigidBody.Size = jsonRigidBody.Size
				rigidBody.ShapeType = jsonRigidBody.ShapeType
				rigidBody.RigidBodyParam = jsonRigidBody.RigidBodyParam

				jsonRigidBodyOffsetPosition := boneMat.Inverted().MulVec3(jsonRigidBody.Position)
				rigidBody.Position = bone.Position.Added(jsonRigidBodyOffsetPosition)
				rigidBody.Rotation = jsonRigidBody.Rotation

				model.RigidBodies.Append(rigidBody)
			}
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
func fixBaseBones(model, baseModel *pmx.PmxModel, fromJson bool, nonExistBoneNames []string) {
	for _, bone := range model.Bones.Data {
		if !slices.Contains(nonExistBoneNames, bone.Name()) {
			continue
		}

		if baseBone := baseModel.Bones.GetByName(bone.Name()); baseBone != nil {
			// bone.TailPosition = baseBone.TailPosition
			bone.FixedAxis = baseBone.FixedAxis
			bone.LocalAxisX = baseBone.LocalAxisX
			bone.LocalAxisZ = baseBone.LocalAxisZ

			if (bone.IsEffectorRotation() || bone.IsEffectorTranslation()) && model.Bones.Contains(bone.EffectIndex) {
				if baseEffectorBone := baseModel.Bones.GetByName(
					model.Bones.Get(bone.EffectIndex).Name()); baseEffectorBone != nil {
					bone.EffectFactor = baseBone.EffectFactor
				}
			}

			if bone.Ik != nil && baseBone.Ik != nil {
				if baseIkBone := baseModel.Bones.GetByName(
					model.Bones.Get(bone.Ik.BoneIndex).Name()); baseIkBone != nil {
					bone.Ik.LoopCount = baseBone.Ik.LoopCount
					bone.Ik.UnitRotation = baseBone.Ik.UnitRotation
					for i, baseLink := range baseBone.Ik.Links {
						if linkBone := model.Bones.GetByName(
							baseModel.Bones.Get(baseLink.BoneIndex).Name()); linkBone != nil {
							bone.Ik.Links[i].AngleLimit = baseLink.AngleLimit
							bone.Ik.Links[i].MaxAngleLimit = baseLink.MaxAngleLimit
							bone.Ik.Links[i].MinAngleLimit = baseLink.MinAngleLimit
							bone.Ik.Links[i].LocalAngleLimit = baseLink.LocalAngleLimit
							bone.Ik.Links[i].LocalMaxAngleLimit = baseLink.LocalMaxAngleLimit
							bone.Ik.Links[i].LocalMinAngleLimit = baseLink.LocalMinAngleLimit
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

	if fromJson {
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

			baseShoulderBone := baseModel.Bones.GetByName(shoulderBoneName)
			baseArmBone := baseModel.Bones.GetByName(armBoneName)
			baseNeckRootBone := baseModel.Bones.GetByName(neckRootBoneName)
			baseNeckBone := baseModel.Bones.GetByName(neckBoneName)

			// baseモデルの首根元から見た肩と腕と首の相対位置
			baseArmByNeckRoot := baseArmBone.Position.Subed(baseNeckRootBone.Position)
			baseShoulderByNeckRoot := baseShoulderBone.Position.Subed(baseNeckRootBone.Position)
			baseNeckByNeckRoot := baseNeckBone.Position.Subed(baseNeckRootBone.Position)

			// 素体モデルの首根元から見た腕の相対位置
			armByNeckRoot := armBone.Position.Subed(neckRootBone.Position)

			// スケーリング係数を計算
			shoulderXZRatio := armByNeckRoot.Length() / baseArmByNeckRoot.Length()

			// 素体モデルの首根元から見た首の相対位置
			neckByNeckRoot := neckBone.Position.Subed(neckRootBone.Position)

			// スケーリング係数を計算
			shoulderYRatio := neckByNeckRoot.Length() / baseNeckByNeckRoot.Length()

			// 素体モデルの首根元から見た肩の位置に相当する位置を求める
			shoulderOffset := baseShoulderByNeckRoot.Muled(
				&mmath.MVec3{X: shoulderXZRatio, Y: shoulderYRatio, Z: shoulderXZRatio})

			// 素体モデルの肩ボーンの位置を求める
			shoulderBone.Position = neckRootBone.Position.Added(shoulderOffset)
			shoulderPBone.Position = neckRootBone.Position.Added(shoulderOffset)
		}
	}

	// 腕捩ボーンの位置を、腕とひじの間に合わせる
	for _, twistBoneName := range []string{pmx.ARM_TWIST.Left(), pmx.ARM_TWIST.Right(),
		pmx.WRIST_TWIST.Left(), pmx.WRIST_TWIST.Right()} {
		if !slices.Contains(nonExistBoneNames, twistBoneName) {
			continue
		}

		baseTwistBone := baseModel.Bones.GetByName(twistBoneName)
		baseTwistParentBone := baseModel.Bones.GetByName(baseTwistBone.ConfigParentBoneNames()[0])
		baseTwistChildBone := baseModel.Bones.GetByName(baseTwistBone.ConfigChildBoneNames()[0])
		for n := range 3 {
			twistBonePartName := fmt.Sprintf("%s%d", twistBoneName, n+1)
			baseTwistPartBone := baseModel.Bones.GetByName(twistBonePartName)
			twistBoneFactor := 0.25 * float64(n+1)
			twistBonePosition := baseTwistParentBone.Position.Lerp(baseTwistChildBone.Position, twistBoneFactor)
			baseTwistPartBone.Position = twistBonePosition
			if n == 1 {
				baseTwistBone.Position = twistBonePosition.Copy()
			}
		}
	}

	if fromJson {
		// 素体のつま先IKの位置を、元モデルのつま先IKの位置に合わせる
		for _, toeIkBoneName := range []string{pmx.TOE_IK.Left(), pmx.TOE_IK.Right()} {
			toeIkBone := model.Bones.GetByName(toeIkBoneName)
			baseToeIkBone := baseModel.Bones.GetByName(toeIkBoneName)

			ankleBone := model.Bones.GetByName(fmt.Sprintf("%s足首", toeIkBone.Direction()))
			baseAnkleBone := baseModel.Bones.GetByName(ankleBone.Name())

			ankleYRatio := ankleBone.Position.Y / baseAnkleBone.Position.Y
			toeIkZRatio := (toeIkBone.Position.Z - ankleBone.Position.Z) / (baseToeIkBone.Position.Z - baseAnkleBone.Position.Z)

			toeIkBone.Position.Y = baseToeIkBone.Position.Y * ankleYRatio
			toeIkBone.Position.Z = baseAnkleBone.Position.Z + (toeIkBone.Position.Z-ankleBone.Position.Z)*toeIkZRatio

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
