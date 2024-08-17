package usecase

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/miu200521358/mlib_go/pkg/domain/pmx"
	"github.com/miu200521358/mlib_go/pkg/domain/vmd"
	"github.com/miu200521358/mlib_go/pkg/infrastructure/repository"
	"github.com/miu200521358/mlib_go/pkg/mutils/mlog"
)

//go:embed model/*
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
	jsonModel = addNonExistBones(model, jsonModel)

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
	if f, err := modelFs.Open("model/mannequin.pmx"); err != nil {
		return nil, err
	} else if pmxData, err := repository.NewPmxRepository().LoadByFile(f); err != nil {
		return nil, err
	} else {
		model = pmxData.(*pmx.PmxModel)
	}

	return model, nil
}

// model にあって、 jsonModel にないボーンを追加する
func addNonExistBones(model, jsonModel *pmx.PmxModel) *pmx.PmxModel {
	if !jsonModel.Bones.ContainsByName(pmx.ARM.Left()) || !jsonModel.Bones.ContainsByName(pmx.ARM.Right()) {
		return jsonModel
	}

	// 両腕の中央を首根元として、両モデルの比率を取得
	neckRootPos := model.Bones.GetByName(pmx.ARM.Left()).Position.Added(
		model.Bones.GetByName(pmx.ARM.Right()).Position).MuledScalar(0.5)
	jsonNeckRootPos := jsonModel.Bones.GetByName(pmx.ARM.Left()).Position.Added(
		jsonModel.Bones.GetByName(pmx.ARM.Right()).Position).MuledScalar(0.5)
	ratio := jsonNeckRootPos.Length() / neckRootPos.Length()

	for i, boneIndex := range model.Bones.LayerSortedIndexes {
		bone := model.Bones.Get(boneIndex)
		// 存在するボーンはスキップ
		if jsonModel.Bones.ContainsByName(bone.Name()) {
			jsonBone := jsonModel.Bones.GetByName(bone.Name())
			if jsonBone.ParentIndex < 0 && jsonBone.Name() != pmx.ROOT.String() {
				// センターがルートなどの場合に、全ての親を親に切り替える
				jsonBone.ParentIndex = jsonModel.Bones.GetByName(pmx.ROOT.String()).Index()
			} else {
				// それ以外も親を切り替える
				jsonBone.ParentIndex = jsonModel.Bones.GetByName(model.Bones.Get(bone.ParentIndex).Name()).Index()
			}

			continue
		}
		// 存在しないボーンは追加
		newBone := bone.Copy().(*pmx.Bone)
		// 最後に追加
		newBone.SetIndex(jsonModel.Bones.Len())
		// 変形階層は親ボーンと一緒
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

	jsonModel.Setup()

	return jsonModel
}

func createFitMorph(model, jsonModel *pmx.PmxModel, fitMorphName string) {
	offsets := make([]pmx.IMorphOffset, 0)
	// nonExistBones := make([]*pmx.Bone, 0)

	// 対象のボーン名をスライスにまとめる
	ignoredBones := []string{"頭", "左目", "右目", "両目"}

	// {
	// 	bone := model.Bones.GetByName("上半身")
	// 	offset := pmx.NewBoneMorphOffset(bone.Index(), mmath.MVec3Zero, mmath.NewMRotation())
	// 	offset.Extend.LocalScale = &mmath.MVec3{X: 2, Y: 1, Z: 1}
	// 	offsets = append(offsets, offset)
	// }

	// {
	// 	bone := model.Bones.GetByName("右足")
	// 	offset := pmx.NewBoneMorphOffset(bone.Index(), mmath.MVec3Zero, mmath.NewMRotation())
	// 	offset.Extend.LocalScale = &mmath.MVec3{X: 2, Y: 2, Z: 2}
	// 	offsets = append(offsets, offset)
	// }

	// {
	// 	bone := model.Bones.GetByName("左腕")
	// 	offset := pmx.NewBoneMorphOffset(bone.Index(), mmath.MVec3Zero, mmath.NewMRotation())
	// 	offset.Extend.LocalScale = &mmath.MVec3{X: 2, Y: 1, Z: 1}
	// 	offsets = append(offsets, offset)
	// }

	for _, bone := range model.Bones.Data {
		// 頭系のボーンは一括処理
		if slices.Contains(ignoredBones, bone.Name()) {
			continue
		}
		// if jsonBone := jsonModel.Bones.GetByName(bone.Name()); jsonBone != nil {
		// 	// 移動系
		// 	parentBone := model.Bones.Get(bone.ParentIndex)
		// 	boneParentRelativePosition := bone.Position.Sub(parentBone.Position)

		// 	bonePosDiff := jsonBone.Extend.ParentRelativePosition.Subed()
		// 	offset := pmx.NewBoneMorphOffset(bone.Index(), bonePosDiff, mmath.NewMRotation())
		// 	offsets = append(offsets, offset)

		// 	// } else {
		// 	// 	// 回転系
		// 	// 	boneQuatMat := bone.Extend.NormalizedLocalAxisX.ToLocalMat()
		// 	// 	jsonBoneQuatMat := jsonBone.Extend.NormalizedLocalAxisX.ToLocalMat()

		// 	// 	// ボーンの傾き補正
		// 	// 	offsetQuat := jsonBoneQuatMat.Mul(boneQuatMat.Inverse()).Quaternion()
		// 	// 	offsetRot := mmath.NewMRotationFromQuaternion(offsetQuat)
		// 	// 	// ボーンの長さ補正
		// 	// 	// offsetScale := jsonBone.Extend.ChildRelativePosition.Length() / bone.Extend.ChildRelativePosition.Length()

		// 	// 	offset := pmx.NewBoneMorphOffset(bone.Index(), mmath.MVec3Zero, mmath.NewMRotation())
		// 	// 	offset.Extend.LocalRotation = offsetRot
		// 	// 	// offset.Extend.LocalScale = &mmath.MVec3{X: offsetScale, Y: offsetScale, Z: offsetScale}

		// 	// 	offsets = append(offsets, offset)
		// 	// }
		// 	// } else {
		// 	// 	// 準標準ボーンが無い場合、後でボーン位置を調整する
		// 	// 	nonExistBones = append(nonExistBones, bone)
		// }
	}

	// // 存在しなかったボーンを補正
	// for _, bone := range nonExistBones {
	// 	parentBone := model.Bones.Get(bone.ParentIndex)
	// 	var childBone *pmx.Bone
	// 	if len(bone.Extend.ChildBoneIndexes) > 0 {
	// 		childBone = model.Bones.Get(bone.Extend.ChildBoneIndexes[0])
	// 	}
	// 	if parentBone != nil && childBone != nil {
	// 		// 親子ボーンの中間点を求める
	// 	}
	// }

	morph := pmx.NewMorph()
	morph.SetIndex(model.Morphs.Len())
	morph.SetName(fitMorphName)
	morph.Offsets = offsets
	morph.MorphType = pmx.MORPH_TYPE_BONE
	morph.Panel = pmx.MORPH_PANEL_OTHER_LOWER_RIGHT
	morph.IsSystem = true
	model.Morphs.Append(morph)
}

func fixBones(model, jsonModel *pmx.PmxModel) {
	for _, bone := range model.Bones.Data {
		if jsonBone := jsonModel.Bones.GetByName(bone.Name()); jsonBone != nil {
			bone.Position = jsonBone.Position
			bone.BoneFlag = jsonBone.BoneFlag
			bone.TailPosition = jsonBone.TailPosition
			bone.FixedAxis = jsonBone.FixedAxis
			bone.LocalAxisX = jsonBone.LocalAxisX
			bone.LocalAxisZ = jsonBone.LocalAxisZ

			if bone.IsEffectorRotation() || bone.IsEffectorTranslation() {
				if jsonEffectorBone := jsonModel.Bones.GetByName(
					model.Bones.Get(bone.EffectIndex).Name()); jsonEffectorBone != nil {
					bone.EffectIndex = jsonEffectorBone.Index()
					bone.EffectFactor = jsonBone.EffectFactor
				}
			}

			if bone.Ik != nil && jsonBone.Ik != nil {
				if jsonIkBone := jsonModel.Bones.GetByName(
					model.Bones.Get(bone.Ik.BoneIndex).Name()); jsonIkBone != nil {
					bone.Ik.BoneIndex = jsonIkBone.Index()
					bone.Ik.LoopCount = jsonBone.Ik.LoopCount
					bone.Ik.UnitRotation = jsonBone.Ik.UnitRotation
					bone.Ik.Links = make([]*pmx.IkLink, len(jsonBone.Ik.Links))
					for i, jsonLink := range jsonBone.Ik.Links {
						if linkBone := model.Bones.GetByName(
							jsonModel.Bones.Get(jsonLink.BoneIndex).Name()); linkBone != nil {
							jsonLink.BoneIndex = linkBone.Index()
							bone.Ik.Links[i] = jsonLink
						}
					}
				}
			}
		}
	}
}

func loadOriginalPmxTextures(model *pmx.PmxModel) {
	model.SetPath(filepath.Join(os.TempDir(), "model"))
	for _, tex := range model.Textures.Data {
		texPath := filepath.Join("model", tex.Name())
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
