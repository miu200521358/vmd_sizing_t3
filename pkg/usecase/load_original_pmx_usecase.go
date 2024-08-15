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

//go:embed model/*
var modelFs embed.FS

// FitBoneモーフ名
var fit_morph_name = fmt.Sprintf("%s_%s", pmx.MLIB_PREFIX, "FitBone")

func LoadOriginalPmx(jsonModel *pmx.PmxModel) (*pmx.PmxModel, error) {
	var model *pmx.PmxModel

	// JSONファイルが指定されている場合、embedからPMXモデルの素体を読み込む
	if f, err := modelFs.Open("model/mannequin.pmx"); err != nil {
		return nil, err
	} else if pmxData, err := repository.NewPmxRepository().LoadByFile(f); err != nil {
		return nil, err
	} else {
		model = pmxData.(*pmx.PmxModel)
	}

	// テクスチャをTempディレクトリに読み込んでおく
	loadOriginalPmxTextures(model)

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

func createFitMorph(model, jsonModel *pmx.PmxModel, fitMorphName string) {
	offsets := make([]pmx.IMorphOffset, 0)
	// nonExistBones := make([]*pmx.Bone, 0)

	for _, bone := range model.Bones.Data {
		if jsonBone := jsonModel.Bones.GetByName(bone.Name()); jsonBone != nil {
			if bone.CanTranslate() {
				// 移動系
				bonePosDiff := jsonBone.Position.Subed(bone.Position)
				offset := pmx.NewBoneMorphOffset(bone.Index(), bonePosDiff, mmath.NewMRotation())
				offsets = append(offsets, offset)
			} else {
				// 回転系
				boneQuatMat := bone.Extend.NormalizedLocalAxisX.ToLocalMatrix4x4()
				jsonBoneQuatMat := jsonBone.Extend.NormalizedLocalAxisX.ToLocalMatrix4x4()

				// ボーンの傾き補正
				offsetQuat := jsonBoneQuatMat.Mul(boneQuatMat.Inverse()).Quaternion()
				offsetRot := mmath.NewMRotationFromQuaternion(offsetQuat)
				// ボーンの長さ補正
				offsetScale := jsonBone.Extend.ChildRelativePosition.Length() / bone.Extend.ChildRelativePosition.Length()

				offset := pmx.NewBoneMorphOffset(bone.Index(), mmath.MVec3Zero, mmath.NewMRotation())
				offset.Extend.LocalRotation = offsetRot
				offset.Extend.LocalScale = &mmath.MVec3{X: offsetScale, Y: offsetScale, Z: offsetScale}

				offsets = append(offsets, offset)
			}
			// } else {
			// 	// 準標準ボーンが無い場合、後でボーン位置を調整する
			// 	nonExistBones = append(nonExistBones, bone)
		}
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
