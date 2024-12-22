package usecase

import (
	"fmt"
	"slices"

	"github.com/miu200521358/mlib_go/pkg/domain/mmath"
	"github.com/miu200521358/mlib_go/pkg/domain/pmx"
	"github.com/miu200521358/mlib_go/pkg/infrastructure/repository"
)

func SaveJson(savePath string, model *pmx.PmxModel) error {
	if err := addSizingBones(model); err != nil {
		return err
	}

	if err := addRigidBodies(model); err != nil {
		return err
	}

	return repository.NewPmxJsonRepository().Save(savePath, model, false)
}

func addRigidBodies(model *pmx.PmxModel) error {
	// 頂点をボーンINDEX別に纏める
	allBoneVertices := model.Vertices.GetMapByBoneIndex(0.0)

	for _, bone := range model.Bones.Data {
		if bone.Config() == nil &&
			(bone.Extend.RigidBody == nil || bone.Extend.RigidBody.PhysicsType == pmx.PHYSICS_TYPE_STATIC) {
			// 準標準ボーンではなく、物理剛体に紐付いていない場合、親の中で最も近い準標準ボーンに頂点INDEXリストを載せ替える
			for _, parentIndex := range bone.Extend.ParentBoneIndexes {
				parentBone := model.Bones.Get(parentIndex)
				if parentBone.Config() != nil && !parentBone.IsHead() {
					// 親が準標準ボーンの場合、頂点INDEXリストを載せ替える
					allBoneVertices[parentIndex] = append(allBoneVertices[parentIndex], allBoneVertices[bone.Index()]...)
					break
				}
			}
		}
	}

	// 準標準ボーンINDEX別に頂点を覆うバウンディングボックスを計算
	for _, bone := range model.Bones.Data {
		if bone.Config() == nil || bone.Config().BoundingBoxShape == pmx.SHAPE_NONE {
			continue
		}
		if _, ok := allBoneVertices[bone.Index()]; !ok {
			continue
		}

		boneVertices := allBoneVertices[bone.Index()]
		if len(boneVertices) == 0 {
			continue
		}

		positions := make([]*mmath.MVec3, len(boneVertices))
		for i, vertex := range boneVertices {
			positions[i] = vertex.Position
		}

		spheres := mmath.AdaptiveCoverPointsWithSpheres(positions, 0.3, 100)
		for n, sphere := range spheres {
			rigidBody := pmx.NewRigidBody()
			rigidBody.SetName(fmt.Sprintf("%s%s_%04d", pmx.MLIB_PREFIX, bone.Name(), n))
			rigidBody.BoneIndex = bone.Index()
			rigidBody.PhysicsType = pmx.PHYSICS_TYPE_STATIC
			rigidBody.ShapeType = pmx.SHAPE_SPHERE
			rigidBody.Position = sphere.Center
			rigidBody.Size = &mmath.MVec3{X: sphere.Radius, Y: sphere.Radius, Z: sphere.Radius}
			rigidBody.IsSystem = false
			model.RigidBodies.Append(rigidBody)
		}
	}

	return nil
}

func addSizingBones(model *pmx.PmxModel) error {
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

		if bone.Name() == pmx.ROOT.String() {
			// 全ての親を追加した場合、表示枠を切り替える
			rootDisplaySlot := model.DisplaySlots.GetRootDisplaySlot()

			// 前のルートボーンを次の表示枠に移動
			if model.DisplaySlots.Len() > 2 {
				moveToDisplaySlot := model.DisplaySlots.Get(2)
				for _, reference := range rootDisplaySlot.References {
					prevRootReference := pmx.NewDisplaySlotReferenceByValues(
						pmx.DISPLAY_TYPE_BONE, reference.DisplayIndex)
					moveToDisplaySlot.References = slices.Insert(
						moveToDisplaySlot.References, 0, prevRootReference)
				}
			}

			// ルート表示枠を入替
			reference := pmx.NewDisplaySlotReferenceByValues(pmx.DISPLAY_TYPE_BONE, bone.Index())
			rootDisplaySlot.References = []*pmx.Reference{reference}
		} else if bone.IsVisible() {
			// 表示枠に追加
			appendDisplaySlot(model, bone)
		}

		model.Bones.Setup()
	}

	for _, funcs := range [][]func(direction pmx.BoneDirection) *pmx.Bone{
		{model.Bones.GetEye, model.Bones.CreateEye},
		{model.Bones.GetShoulderRoot, model.Bones.CreateShoulderRoot},
		{model.Bones.GetShoulderP, model.Bones.CreateShoulderP},
		{model.Bones.GetShoulderC, model.Bones.CreateShoulderC},
		{model.Bones.GetArmTwist, model.Bones.CreateArmTwist},
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

			// 子ボーンの親INDEX調整
			adjustChildBoneParentIndex(model, bone, direction)

			// 表示枠に追加
			appendDisplaySlot(model, bone)

			model.Bones.Setup()
		}
	}

	// 腕捩・手捩
	for _, funcs := range [][]func(direction pmx.BoneDirection, idx int) *pmx.Bone{
		{model.Bones.GetArmTwistChild, model.Bones.CreateArmTwistChild},
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

	// 親指0
	{
		for _, direction := range []pmx.BoneDirection{pmx.BONE_DIRECTION_LEFT, pmx.BONE_DIRECTION_RIGHT} {
			if model.Bones.GetThumb(direction, 0) != nil {
				continue
			}

			bone := model.Bones.CreateThumb0(direction)

			model.Bones.Insert(bone)

			// 子ボーンの親INDEX調整
			adjustChildBoneParentIndex(model, bone, direction)

			// 表示枠に追加
			appendDisplaySlot(model, bone)

			model.Bones.Setup()
		}
	}

	// 頂点をボーンINDEX別に纏める
	allBoneVertices := model.Vertices.GetMapByBoneIndex(0.0)

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

			if bone.Name() == pmx.THUMB_TAIL.Left() || bone.Name() == pmx.INDEX_TAIL.Left() ||
				bone.Name() == pmx.MIDDLE_TAIL.Left() || bone.Name() == pmx.RING_TAIL.Left() ||
				bone.Name() == pmx.PINKY_TAIL.Left() {
				// 指先は指の親ボーンの最もX方向に＋の頂点を取得
				parentBone := model.Bones.Get(bone.ParentIndex)
				vertices := allBoneVertices[parentBone.Index()]
				if len(vertices) > 0 {
					maxX := vertices[0].Position.X
					maxY := vertices[0].Position.Y
					maxZ := vertices[0].Position.Z
					for _, vertex := range vertices {
						if vertex.Position.X > maxX {
							maxX = vertex.Position.X
							maxY = vertex.Position.Y
							maxZ = vertex.Position.Z
						}
					}
					bone.Position.X = maxX
					bone.Position.Y = maxY
					bone.Position.Z = maxZ
				}
			}

			if bone.Name() == pmx.THUMB_TAIL.Right() || bone.Name() == pmx.INDEX_TAIL.Right() ||
				bone.Name() == pmx.MIDDLE_TAIL.Right() || bone.Name() == pmx.RING_TAIL.Right() ||
				bone.Name() == pmx.PINKY_TAIL.Right() {
				// 指先は指の親ボーンの最もX方向にーの頂点を取得
				parentBone := model.Bones.Get(bone.ParentIndex)
				vertices := allBoneVertices[parentBone.Index()]
				if len(vertices) > 0 {
					minX := vertices[0].Position.X
					minY := vertices[0].Position.Y
					minZ := vertices[0].Position.Z
					for _, vertex := range vertices {
						if vertex.Position.X < minX {
							minX = vertex.Position.X
							minY = vertex.Position.Y
							minZ = vertex.Position.Z
						}
					}
					bone.Position.X = minX
					bone.Position.Y = minY
					bone.Position.Z = minZ
				}
			}

			if bone.Name() == pmx.HEEL.Left() || bone.Name() == pmx.HEEL.Right() {
				// かかとは足首・足首Dの最もZ方向に＋の頂点を取得
				ankleBone := model.Bones.GetAnkle(direction)
				ankleDBone := model.Bones.GetAnkleD(direction)
				var vertices []*pmx.Vertex
				if ankleDBone != nil {
					vertices = allBoneVertices[ankleDBone.Index()]
				}
				if vertices == nil && ankleBone != nil {
					vertices = allBoneVertices[ankleBone.Index()]
				}
				if len(vertices) > 0 {
					// 接地している頂点のうち、最もZ方向に＋の頂点を取得
					minY := vertices[0].Position.Y
					for _, vertex := range vertices {
						if vertex.Position.Y < minY {
							minY = vertex.Position.Y
						}
					}

					maxX := vertices[0].Position.X
					maxY := vertices[0].Position.Y
					maxZ := vertices[0].Position.Z
					for _, vertex := range vertices {
						if mmath.NearEquals(vertex.Position.Y, minY, 1e-1) && vertex.Position.Z > maxZ {
							maxX = vertex.Position.X
							maxY = vertex.Position.Y
							maxZ = vertex.Position.Z
						}
					}

					bone.Position.X = maxX
					bone.Position.Y = maxY
					bone.Position.Z = maxZ
				}
			}

			if bone.Name() == pmx.TOE_T.Left() || bone.Name() == pmx.TOE_T.Right() {
				// つま先は足首・足首D・足先EXの最もZ方向にーの頂点を取得
				ankleBone := model.Bones.GetAnkle(direction)
				ankleDBone := model.Bones.GetAnkleD(direction)
				toeExBone := model.Bones.GetToeEx(direction)
				var vertices []*pmx.Vertex
				if toeExBone != nil {
					vertices = allBoneVertices[toeExBone.Index()]
				}
				if vertices == nil && ankleDBone != nil {
					vertices = allBoneVertices[ankleDBone.Index()]
				}
				if vertices == nil && ankleBone != nil {
					vertices = allBoneVertices[ankleBone.Index()]
				}
				if len(vertices) > 0 {
					minX := vertices[0].Position.X
					minY := vertices[0].Position.Y
					minZ := vertices[0].Position.Z
					for _, vertex := range vertices {
						if vertex.Position.Z < minZ {
							minX = vertex.Position.X
							minY = vertex.Position.Y
							minZ = vertex.Position.Z
						}
					}
					bone.Position.X = minX
					bone.Position.Y = minY
					bone.Position.Z = minZ
				}
			}

			model.Bones.Insert(bone)

			// 子ボーンの親INDEX調整
			adjustChildBoneParentIndex(model, bone, direction)

			// 表示枠に追加
			appendDisplaySlot(model, bone)

			model.Bones.Setup()
		}
	}

	return nil
}

func adjustChildBoneParentIndex(model *pmx.PmxModel, bone *pmx.Bone, direction pmx.BoneDirection) {
	config := bone.Config()
	if config != nil && config.ChildBoneNames != nil {
		for _, childBoneName := range config.ChildBoneNames {
			childBone := model.Bones.GetByName(childBoneName.StringFromDirection(direction.String()))
			if childBone != nil {
				childBone.ParentIndex = bone.Index()
				break
			}
		}
	}
}

func appendDisplaySlot(model *pmx.PmxModel, bone *pmx.Bone) {
	if !bone.IsVisible() {
		return
	}

	// 表示枠に追加
	var displaySlot *pmx.DisplaySlot

	config := bone.Config()
	if config != nil && config.ChildBoneNames != nil {
		for _, childBoneName := range config.ChildBoneNames {
			{
				childBone := model.Bones.GetByName(childBoneName.String())
				if childBone != nil {
					displaySlot = model.DisplaySlots.GetByBoneIndex(childBone.Index())
					break
				}
			}
			for _, direction := range []pmx.BoneDirection{pmx.BONE_DIRECTION_LEFT, pmx.BONE_DIRECTION_RIGHT} {
				childBone := model.Bones.GetByName(childBoneName.StringFromDirection(direction.String()))
				if childBone != nil {
					displaySlot = model.DisplaySlots.GetByBoneIndex(childBone.Index())
					break
				}
			}
		}
	}

	if displaySlot == nil {
		// 表示枠がない場合は親ボーンの表示枠を取得
		displaySlot = model.DisplaySlots.GetByBoneIndex(bone.ParentIndex)
	}
	if displaySlot == nil {
		// 表示枠がない場合は付与親ボーンの表示枠を取得
		displaySlot = model.DisplaySlots.GetByBoneIndex(bone.EffectIndex)
	}
	if displaySlot != nil {
		reference := pmx.NewDisplaySlotReference()
		reference.DisplayType = pmx.DISPLAY_TYPE_BONE
		reference.DisplayIndex = bone.Index()
		displaySlot.References = append(displaySlot.References, reference)
	}
}
