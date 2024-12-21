package usecase

import (
	"strings"

	"github.com/miu200521358/mlib_go/pkg/domain/pmx"
	"github.com/miu200521358/mlib_go/pkg/infrastructure/repository"
)

func SaveJson(savePath string, model *pmx.PmxModel) error {
	if err := addBones(model); err != nil {
		return err
	}

	pmxPath := strings.ReplaceAll(savePath, ".json", "_json.pmx")
	pmxRep := repository.NewPmxRepository()
	pmxRep.Save(pmxPath, model, false)

	rep := repository.NewPmxJsonRepository()

	return rep.Save(savePath, model, false)
}

func addBones(model *pmx.PmxModel) error {
	for _, funcs := range [][]func() *pmx.Bone{
		{model.Bones.GetRoot, model.Bones.CreateRoot},
		{model.Bones.GetGroove, model.Bones.CreateGroove},
		{model.Bones.GetWaist, model.Bones.CreateWaist},
		{model.Bones.GetTrunkRoot, model.Bones.CreateTrunkRoot},
		{model.Bones.GetLowerRoot, model.Bones.CreateLowerRoot},
		{model.Bones.GetLegCenter, model.Bones.CreateLegCenter},
		{model.Bones.GetUpperRoot, model.Bones.CreateUpperRoot},
		{model.Bones.GetUpper2, model.Bones.CreateUpper2},
		{model.Bones.GetNeckRoot, model.Bones.CreateNeckRoot},
		{model.Bones.GetNeck, model.Bones.CreateNeck},
		{model.Bones.GetHead, model.Bones.CreateHead},
		{model.Bones.GetHeadTail, model.Bones.CreateHeadTail},
		{model.Bones.GetEyes, model.Bones.CreateEyes},
	} {
		funcGet := funcs[0]
		funcCreate := funcs[1]

		if funcGet() != nil {
			continue
		}

		bone := funcCreate()

		model.Bones.Insert(bone)

		// ボーン定義
		config := bone.Config()
		if config != nil && config.ChildBoneNames != nil {
			for _, childBoneName := range config.ChildBoneNames {
				childBone := model.Bones.GetByName(childBoneName.String())
				if childBone != nil {
					childBone.ParentIndex = bone.Index()
					break
				}
				childLeftBone := model.Bones.GetByName(childBoneName.Left())
				childRightBone := model.Bones.GetByName(childBoneName.Right())
				if childLeftBone != nil && childRightBone != nil {
					childLeftBone.ParentIndex = bone.Index()
					childRightBone.ParentIndex = bone.Index()
					break
				}
			}
		}

		model.Bones.Setup()
	}

	for _, funcs := range [][]func(direction pmx.BoneDirection) *pmx.Bone{
		{model.Bones.GetEye, model.Bones.CreateEye},
		{model.Bones.GetShoulderRoot, model.Bones.CreateShoulderRoot},
		{model.Bones.GetShoulderP, model.Bones.CreateShoulderP},
		{model.Bones.GetShoulderC, model.Bones.CreateShoulderC},
		{model.Bones.GetArmTwist, model.Bones.CreateArmTwist},
	} {
		for _, direction := range []pmx.BoneDirection{pmx.BONE_DIRECTION_LEFT, pmx.BONE_DIRECTION_RIGHT} {
			funcGet := funcs[0]
			funcCreate := funcs[1]

			if funcGet(direction) != nil {
				continue
			}

			bone := funcCreate(direction)

			model.Bones.Insert(bone)

			// ボーン定義
			config := bone.Config()
			if config != nil && config.ChildBoneNames != nil {
				for _, childBoneName := range config.ChildBoneNames {
					childBone := model.Bones.GetByName(childBoneName.StringFromDirection(string(direction)))
					if childBone != nil {
						childBone.ParentIndex = bone.Index()
						break
					}
				}
			}

			model.Bones.Setup()
		}
	}

	for _, funcs := range [][]func(direction pmx.BoneDirection, idx int) *pmx.Bone{
		{model.Bones.GetArmTwistChild, model.Bones.CreateArmTwistChild},
	} {
		for _, direction := range []pmx.BoneDirection{pmx.BONE_DIRECTION_LEFT, pmx.BONE_DIRECTION_RIGHT} {
			for idx := range 3 {
				funcGet := funcs[0]
				funcCreate := funcs[1]

				if funcGet(direction, idx) != nil {
					continue
				}

				bone := funcCreate(direction, idx)

				model.Bones.Insert(bone)
				model.Bones.Setup()
			}
		}
	}

	for _, funcs := range [][]func(direction pmx.BoneDirection) *pmx.Bone{
		{model.Bones.GetWristTwist, model.Bones.CreateWristTwist},
	} {
		for _, direction := range []pmx.BoneDirection{pmx.BONE_DIRECTION_LEFT, pmx.BONE_DIRECTION_RIGHT} {
			funcGet := funcs[0]
			funcCreate := funcs[1]

			if funcGet(direction) != nil {
				continue
			}

			bone := funcCreate(direction)

			model.Bones.Insert(bone)

			// ボーン定義
			config := bone.Config()
			if config != nil && config.ChildBoneNames != nil {
				for _, childBoneName := range config.ChildBoneNames {
					childBone := model.Bones.GetByName(childBoneName.StringFromDirection(string(direction)))
					if childBone != nil {
						childBone.ParentIndex = bone.Index()
						break
					}
				}
			}

			model.Bones.Setup()
		}
	}

	for _, funcs := range [][]func(direction pmx.BoneDirection, idx int) *pmx.Bone{
		{model.Bones.GetWristTwistChild, model.Bones.CreateWristTwistChild},
	} {
		for _, direction := range []pmx.BoneDirection{pmx.BONE_DIRECTION_LEFT, pmx.BONE_DIRECTION_RIGHT} {
			for idx := range 3 {
				funcGet := funcs[0]
				funcCreate := funcs[1]

				if funcGet(direction, idx) != nil {
					continue
				}

				bone := funcCreate(direction, idx)

				model.Bones.Insert(bone)
				model.Bones.Setup()
			}
		}
	}

	{
		for _, direction := range []pmx.BoneDirection{pmx.BONE_DIRECTION_LEFT, pmx.BONE_DIRECTION_RIGHT} {
			if model.Bones.GetThumb(direction, 0) != nil {
				continue
			}

			bone := model.Bones.CreateThumb0(direction)

			model.Bones.Insert(bone)

			// ボーン定義
			config := bone.Config()
			if config != nil && config.ChildBoneNames != nil {
				for _, childBoneName := range config.ChildBoneNames {
					childBone := model.Bones.GetByName(childBoneName.StringFromDirection(string(direction)))
					if childBone != nil {
						childBone.ParentIndex = bone.Index()
						break
					}
				}
			}

			model.Bones.Setup()
		}
	}

	for _, funcs := range [][]func(direction pmx.BoneDirection) *pmx.Bone{
		{model.Bones.GetWristTail, model.Bones.CreateWristTail},
		{model.Bones.GetThumbTail, model.Bones.CreateThumbTail},
		{model.Bones.GetIndexTail, model.Bones.CreateIndexTail},
		{model.Bones.GetMiddleTail, model.Bones.CreateMiddleTail},
		{model.Bones.GetRingTail, model.Bones.CreateRingTail},
		{model.Bones.GetPinkyTail, model.Bones.CreatePinkyTail},
		{model.Bones.GetLegRoot, model.Bones.CreateLegRoot},
		{model.Bones.GetWaistCancel, model.Bones.CreateWaistCancel},
		{model.Bones.GetHeel, model.Bones.CreateHeel},
		{model.Bones.GetToeT, model.Bones.CreateToeT},
		{model.Bones.GetToeP, model.Bones.CreateToeP},
		{model.Bones.GetToeC, model.Bones.CreateToeC},
		{model.Bones.GetLegD, model.Bones.CreateLegD},
		{model.Bones.GetKneeD, model.Bones.CreateKneeD},
		{model.Bones.GetAnkleD, model.Bones.CreateAnkleD},
		{model.Bones.GetHeelD, model.Bones.CreateHeelD},
		{model.Bones.GetToeEx, model.Bones.CreateToeEx},
		{model.Bones.GetToeTD, model.Bones.CreateToeTD},
		{model.Bones.GetToePD, model.Bones.CreateToePD},
		{model.Bones.GetToeCD, model.Bones.CreateToeCD},
		{model.Bones.GetLegIkParent, model.Bones.CreateLegIkParent},
	} {
		for _, direction := range []pmx.BoneDirection{pmx.BONE_DIRECTION_LEFT, pmx.BONE_DIRECTION_RIGHT} {
			funcGet := funcs[0]
			funcCreate := funcs[1]

			if funcGet(direction) != nil {
				continue
			}

			bone := funcCreate(direction)

			model.Bones.Insert(bone)

			// ボーン定義
			config := bone.Config()
			if config != nil && config.ChildBoneNames != nil {
				for _, childBoneName := range config.ChildBoneNames {
					childBone := model.Bones.GetByName(childBoneName.StringFromDirection(string(direction)))
					if childBone != nil {
						childBone.ParentIndex = bone.Index()
						break
					}
				}
			}

			model.Bones.Setup()
		}
	}

	return nil
}
