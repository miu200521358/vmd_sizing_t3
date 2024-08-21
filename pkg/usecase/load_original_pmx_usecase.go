package usecase

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/miu200521358/mlib_go/pkg/domain/mmath"
	"github.com/miu200521358/mlib_go/pkg/domain/pmx"
	"github.com/miu200521358/mlib_go/pkg/domain/vmd"
	"github.com/miu200521358/mlib_go/pkg/infrastructure/repository"
	"github.com/miu200521358/mlib_go/pkg/mutils/mlog"
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
	// 矯正更新用にハッシュ上書き
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

func getRootRatio(model, jsonModel *pmx.PmxModel) float64 {
	if !jsonModel.Bones.ContainsByName(pmx.ARM.Left()) || !jsonModel.Bones.ContainsByName(pmx.ARM.Right()) ||
		!model.Bones.ContainsByName(pmx.ARM.Left()) || !model.Bones.ContainsByName(pmx.ARM.Right()) {
		return 1.0
	}

	// 両腕の中央を首根元として、両モデルの比率を取得
	neckRootPos := model.Bones.GetByName(pmx.ARM.Left()).Position.Added(
		model.Bones.GetByName(pmx.ARM.Right()).Position).MuledScalar(0.5)
	jsonNeckRootPos := jsonModel.Bones.GetByName(pmx.ARM.Left()).Position.Added(
		jsonModel.Bones.GetByName(pmx.ARM.Right()).Position).MuledScalar(0.5)
	ratio := jsonNeckRootPos.Length() / neckRootPos.Length()

	return ratio
}

// model にあって、 jsonModel にないボーンを追加する
func addNonExistBones(model, jsonModel *pmx.PmxModel) {
	if !jsonModel.Bones.ContainsByName(pmx.ARM.Left()) || !jsonModel.Bones.ContainsByName(pmx.ARM.Right()) {
		return
	}

	ratio := getRootRatio(model, jsonModel)

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

			if strings.Contains(bone.Name(), "上半身2") {
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
			} else if strings.Contains(bone.Name(), "肩P") {
				// 肩Pの場合、肩と同じ位置に置く
				shoulderBone := model.Bones.GetByName(strings.ReplaceAll(bone.Name(), "肩P", "肩"))
				jsonShoulderBone := jsonModel.Bones.GetByName(shoulderBone.Name())
				newBone.Position = jsonShoulderBone.Position.Copy()
			} else if strings.Contains(bone.Name(), "肩C") {
				// 肩Cの場合、腕と同じ位置に置く
				armBone := model.Bones.GetByName(strings.ReplaceAll(bone.Name(), "肩C", "腕"))
				jsonArmBone := jsonModel.Bones.GetByName(armBone.Name())
				newBone.Position = jsonArmBone.Position.Copy()
			} else if strings.Contains(bone.Name(), "首根元") || strings.Contains(bone.Name(), "肩根元") {
				// 首根元・肩根元は首根元の位置
				newBone.Position = jsonModel.Bones.GetByName(pmx.ARM.Left()).Position.Added(
					jsonModel.Bones.GetByName(pmx.ARM.Right()).Position).MuledScalar(0.5)
			} else if strings.Contains(bone.Name(), "親指０") {
				// 親指０は手首と親指１の間
				wristBone := model.Bones.GetByName(strings.ReplaceAll(bone.Name(), "親指０", "手首"))
				thumbBone := model.Bones.GetByName(strings.ReplaceAll(bone.Name(), "親指０", "親指１"))
				thumbRatio := bone.Position.Subed(wristBone.Position).Length() / thumbBone.Position.Subed(wristBone.Position).Length()

				jsonWristBone := jsonModel.Bones.GetByName(wristBone.Name())
				jsonThumbBone := jsonModel.Bones.GetByName(thumbBone.Name())
				newBone.Position = jsonWristBone.Position.Lerp(jsonThumbBone.Position, thumbRatio)
			} else if strings.Contains(bone.Name(), "足中心") || strings.Contains(bone.Name(), "足根元") {
				// 足中心は足の中心
				newBone.Position = jsonModel.Bones.GetByName(pmx.LEG.Left()).Position.Added(
					jsonModel.Bones.GetByName(pmx.LEG.Right()).Position).MuledScalar(0.5)
			} else if strings.Contains(bone.Name(), "腰キャンセル") {
				// 腰キャンセルは足と同じ位置
				legBoneName := fmt.Sprintf("%s足", bone.Direction())
				jsonLegBone := jsonModel.Bones.GetByName(legBoneName)
				newBone.Position = jsonLegBone.Position.Copy()
			} else if strings.Contains(bone.Name(), "腰") {
				// 腰は上半身と下半身の間
				jsonUpperBone := jsonModel.Bones.GetByName(pmx.UPPER.String())
				jsonLowerBone := jsonModel.Bones.GetByName(pmx.LOWER.String())
				newBone.Position = jsonUpperBone.Position.Lerp(jsonLowerBone.Position, 0.5)
			} else if strings.Contains(bone.Name(), "足IK親") {
				// 足IK親 は 足IKのYを0にした位置
				legIkBone := model.Bones.GetByName(strings.ReplaceAll(bone.Name(), "足IK親", "足ＩＫ"))
				jsonLegIkBone := jsonModel.Bones.GetByName(legIkBone.Name())
				newBone.Position = jsonLegIkBone.Position.Copy()
				newBone.Position.Y = 0
			} else if strings.Contains(bone.Name(), "足D") {
				// 足D は 足の位置
				legBone := model.Bones.GetByName(strings.ReplaceAll(bone.Name(), "足D", "足"))
				jsonLegBone := jsonModel.Bones.GetByName(legBone.Name())
				newBone.Position = jsonLegBone.Position.Copy()
			} else if strings.Contains(bone.Name(), "ひざD") {
				// ひざD は ひざの位置
				kneeBone := model.Bones.GetByName(strings.ReplaceAll(bone.Name(), "ひざD", "ひざ"))
				jsonKneeBone := jsonModel.Bones.GetByName(kneeBone.Name())
				newBone.Position = jsonKneeBone.Position.Copy()
			} else if strings.Contains(bone.Name(), "足首D") {
				// 足首D は 足首の位置
				ankleBone := model.Bones.GetByName(strings.ReplaceAll(bone.Name(), "足首D", "足首"))
				jsonAnkleBone := jsonModel.Bones.GetByName(ankleBone.Name())
				newBone.Position = jsonAnkleBone.Position.Copy()
			} else if strings.Contains(bone.Name(), "足先EX") {
				// 足先EXは 足首とつま先の間
				ankleBone := model.Bones.GetByName(strings.ReplaceAll(bone.Name(), "足先EX", "足首"))
				// つま先のボーン名は標準ではないので、つま先ＩＫのターゲットから取る
				toeBone := model.Bones.Get(model.Bones.GetByName(strings.ReplaceAll(bone.Name(), "足先EX", "つま先ＩＫ")).Ik.BoneIndex)
				toeRatio := bone.Position.Subed(ankleBone.Position).Length() / toeBone.Position.Subed(ankleBone.Position).Length()

				jsonAnkleBone := jsonModel.Bones.GetByName(ankleBone.Name())
				jsonToeBone := jsonModel.Bones.GetByName(toeBone.Name())
				newBone.Position = jsonAnkleBone.Position.Lerp(jsonToeBone.Position, toeRatio)
			} else if strings.Contains(bone.Name(), "かかと") {
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
			} else if strings.Contains(bone.Name(), "つま先") {
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
	ratio := getRootRatio(model, jsonModel)

	for _, bone := range model.Bones.Data {
		if jsonBone := jsonModel.Bones.GetByName(bone.Name()); jsonBone != nil {
			var parentBone *pmx.Bone
			var jsonParentBone *pmx.Bone
			for _, parentBoneName := range bone.ConfigParentBoneNames() {
				if model.Bones.ContainsByName(parentBoneName) && jsonModel.Bones.ContainsByName(parentBoneName) {
					parentBone = model.Bones.GetByName(parentBoneName)
					jsonParentBone = jsonModel.Bones.GetByName(parentBoneName)
					break
				}
			}

			if parentBone == nil || jsonParentBone == nil {
				continue
			}

			var jsonChildBone *pmx.Bone
			var childBone *pmx.Bone
			for _, childBoneName := range bone.ConfigChildBoneNames() {
				if model.Bones.ContainsByName(childBoneName) && jsonModel.Bones.ContainsByName(childBoneName) {
					childBone = model.Bones.GetByName(childBoneName)
					jsonChildBone = jsonModel.Bones.GetByName(childBoneName)
					break
				}
			}

			offset := pmx.NewBoneMorphOffset(bone.Index())

			if bone.CanFitMove() || bone.CanFitLocalMove() {
				// 位置補正
				offsetPosition := jsonBone.Position.Subed(bone.Position)

				if bone.IsSole() {
					// 靴底はY=0に合わせる
					for _, ankleOffset := range offsets {
						ankleBone := model.Bones.Get(ankleOffset.(*pmx.BoneMorphOffset).BoneIndex)
						if strings.Contains(ankleBone.Name(), fmt.Sprintf("%s足首", bone.Direction())) {
							jsonAnkleBone := jsonModel.Bones.GetByName(ankleBone.Name())
							scaledAnklePosition := jsonAnkleBone.Position.Added(ankleBone.Extend.ChildRelativePosition)

							offsetPosition.Y = -scaledAnklePosition.Y
							break
						}
					}
				}

				if bone.CanFitMove() {
					offset.Position = offsetPosition
				} else {
					offset.Extend.LocalPosition = offsetPosition
				}
			}

			if childBone != nil && jsonChildBone != nil && (bone.CanFitRotate() || bone.CanFitLocalRotate()) {
				// 回転補正
				jsonBoneAxis := jsonChildBone.Position.Subed(jsonBone.Position).Normalized()
				boneAxis := childBone.Position.Subed(bone.Position).Normalized()

				offsetQuat := mmath.NewMQuaternionRotate(boneAxis, jsonBoneAxis)

				if bone.CanFitRotate() {
					offset.Rotation = offsetQuat
				} else {
					offset.Extend.LocalRotation = offsetQuat
				}
			}

			if bone.CanFitScale() || bone.CanFitLocalScale() {
				// スケール補正
				offsetScale := ratio

				if childBone != nil && jsonChildBone != nil {
					jsonBoneLength := jsonBone.Position.Distance(jsonChildBone.Position)
					boneLength := bone.Position.Distance(childBone.Position)
					if boneLength != 0 && jsonBoneLength != 0 {
						offsetScale = jsonBoneLength / boneLength
					}
				}

				offsetScales := &mmath.MVec3{X: offsetScale, Y: ratio, Z: ratio}
				if bone.IsHead() || bone.IsAnkle() {
					// 頭系・足首の場合、均一にスケールする
					offsetScales = &mmath.MVec3{X: offsetScale, Y: offsetScale, Z: offsetScale}
				}

				if bone.CanFitScale() {
					offset.Extend.Scale = offsetScales
				} else {
					offset.Extend.LocalScale = offsetScales
				}
			}

			offsets = append(offsets, offset)
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
			// bone.FixedAxis = jsonBone.FixedAxis
			// bone.LocalAxisX = jsonBone.LocalAxisX
			// bone.LocalAxisZ = jsonBone.LocalAxisZ

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

	// 根元ボーンの位置を腕の中央に合わせる
	for _, rootBoneName := range []string{pmx.NECK_ROOT.String(), pmx.SHOULDER_ROOT.Right(), pmx.SHOULDER_ROOT.Left()} {
		neckRootBone := model.Bones.GetByName(rootBoneName)
		neckRootBone.Position = model.Bones.GetByName(pmx.ARM.Left()).Position.Added(
			model.Bones.GetByName(pmx.ARM.Right()).Position).MuledScalar(0.5)
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
		shoulderOffset := jsonShoulderByNeckRoot.Muled(&mmath.MVec3{X: shoulderXZRatio, Y: shoulderYRatio, Z: shoulderXZRatio})

		// 素体モデルの肩ボーンの位置を求める
		shoulderBone.Position = neckRootBone.Position.Added(shoulderOffset)
		shoulderPBone.Position = neckRootBone.Position.Added(shoulderOffset)
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
