package usecase

import (
	"fmt"
	"math"
	"runtime/debug"
	"sync"

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

func SizingArmTwist(sizingSet *domain.SizingSet, setSize int) (bool, error) {
	if !sizingSet.IsSizingArmTwist || (sizingSet.IsSizingArmTwist && sizingSet.CompletedSizingArmTwist) {
		return false, nil
	}

	if !isValidCleanArmTwist(sizingSet) {
		return false, nil
	}

	sizingModel := sizingSet.SizingPmx
	sizingMotion := sizingSet.OutputVmd

	armIkBones := make([]*pmx.Bone, 2)
	armTwistIkBones := make([]*pmx.Bone, 2)
	wristTwistIkBones := make([]*pmx.Bone, 2)
	// wristIkBones := make([]*pmx.Bone, 2)

	mlog.I(mi18n.T("捩り補正開始", map[string]interface{}{"No": sizingSet.Index + 1}))

	for i, direction := range directions {
		sizingArmBone := sizingModel.Bones.GetByName(pmx.ARM.StringFromDirection(direction))
		sizingArmTwistBone := sizingModel.Bones.GetByName(pmx.ARM_TWIST.StringFromDirection(direction))
		sizingElbowBone := sizingModel.Bones.GetByName(pmx.ELBOW.StringFromDirection(direction))
		sizingWristTwistBone := sizingModel.Bones.GetByName(pmx.WRIST_TWIST.StringFromDirection(direction))
		sizingWristBone := sizingModel.Bones.GetByName(pmx.WRIST.StringFromDirection(direction))
		sizingWristTailBone := sizingModel.Bones.GetByName(pmx.WRIST_TAIL.StringFromDirection(direction))

		// 腕IK
		armIkBone := pmx.NewBoneByName(fmt.Sprintf("%s%sIk", pmx.MLIB_PREFIX, sizingArmBone.Name()))
		armIkBone.Position = sizingElbowBone.Position
		armIkBone.Ik = pmx.NewIk()
		armIkBone.Ik.BoneIndex = sizingElbowBone.Index()
		armIkBone.Ik.LoopCount = 10
		armIkBone.Ik.UnitRotation = mmath.NewMRotationFromDegrees(&mmath.MVec3{X: 180, Y: 0, Z: 0})
		armIkBone.Ik.Links = make([]*pmx.IkLink, 1)
		armIkBone.Ik.Links[0] = pmx.NewIkLink()
		armIkBone.Ik.Links[0].BoneIndex = sizingArmBone.Index()
		armIkBones[i] = armIkBone

		// 腕捩IK
		armTwistIkBone := pmx.NewBoneByName(fmt.Sprintf("%s%sIk", pmx.MLIB_PREFIX, sizingArmTwistBone.Name()))
		armTwistIkBone.Position = sizingWristBone.Position
		armTwistIkBone.Ik = pmx.NewIk()
		armTwistIkBone.Ik.BoneIndex = sizingWristBone.Index()
		armTwistIkBone.Ik.LoopCount = 100
		armTwistIkBone.Ik.UnitRotation = mmath.NewMRotationFromDegrees(&mmath.MVec3{X: 180, Y: 0, Z: 0})
		armTwistIkBone.Ik.Links = make([]*pmx.IkLink, 1)
		armTwistIkBone.Ik.Links[0] = pmx.NewIkLink()
		armTwistIkBone.Ik.Links[0].BoneIndex = sizingArmTwistBone.Index()
		armTwistIkBones[i] = armTwistIkBone

		// 手捩IK
		wristTwistIkBone := pmx.NewBoneByName(fmt.Sprintf("%s%sIk", pmx.MLIB_PREFIX, sizingWristTwistBone.Name()))
		wristTwistIkBone.Position = sizingWristTailBone.Position
		wristTwistIkBone.Ik = pmx.NewIk()
		wristTwistIkBone.Ik.BoneIndex = sizingWristTailBone.Index()
		wristTwistIkBone.Ik.LoopCount = 100
		wristTwistIkBone.Ik.UnitRotation = mmath.NewMRotationFromDegrees(&mmath.MVec3{X: 180, Y: 0, Z: 0})
		wristTwistIkBone.Ik.Links = make([]*pmx.IkLink, 1)
		wristTwistIkBone.Ik.Links[0] = pmx.NewIkLink()
		wristTwistIkBone.Ik.Links[0].BoneIndex = sizingWristTwistBone.Index()
		wristTwistIkBones[i] = wristTwistIkBone

		// // 手首IK
		// wristIkBone := pmx.NewBoneByName(fmt.Sprintf("%s%sIk", pmx.MLIB_PREFIX, sizingWristBone.Name()))
		// wristIkBone.Position = sizingWristTailBone.Position
		// wristIkBone.Ik = pmx.NewIk()
		// wristIkBone.Ik.BoneIndex = sizingWristTailBone.Index()
		// wristIkBone.Ik.LoopCount = 10
		// wristIkBone.Ik.UnitRotation = mmath.NewMRotationFromDegrees(&mmath.MVec3{X: 180, Y: 0, Z: 0})
		// wristIkBone.Ik.Links = make([]*pmx.IkLink, 1)
		// wristIkBone.Ik.Links[0] = pmx.NewIkLink()
		// wristIkBone.Ik.Links[0].BoneIndex = sizingWristBone.Index()
		// wristIkBones[i] = wristIkBone
	}

	sizingOriginalAllDeltas := make([][]*delta.VmdDeltas, 2)

	sizingArmRotations := make([][]*mmath.MQuaternion, 2)
	sizingElbowRotations := make([][]*mmath.MQuaternion, 2)
	sizingWristRotations := make([][]*mmath.MQuaternion, 2)

	allFrames := make([][]int, 2)
	allBlockSizes := make([]int, 2)

	errorChan := make(chan error, 2)

	var wg sync.WaitGroup
	wg.Add(2)
	for i, direction := range directions {
		go func(i int, direction string) {
			defer wg.Done()

			sizingArmBone := sizingModel.Bones.GetByName(pmx.ARM.StringFromDirection(direction))
			sizingElbowBone := sizingModel.Bones.GetByName(pmx.ELBOW.StringFromDirection(direction))
			sizingWristBone := sizingModel.Bones.GetByName(pmx.WRIST.StringFromDirection(direction))

			frames := sizingMotion.BoneFrames.RegisteredFrames(arm_direction_bone_names[i])
			allFrames[i] = frames
			allBlockSizes[i] = miter.GetBlockSize(len(frames) * setSize)

			mlog.I(mi18n.T("捩り補正01", map[string]interface{}{"No": sizingSet.Index + 1, "Direction": direction}))

			sizingOriginalAllDeltas[i] = make([]*delta.VmdDeltas, len(frames))

			sizingArmRotations[i] = make([]*mmath.MQuaternion, len(frames))
			sizingElbowRotations[i] = make([]*mmath.MQuaternion, len(frames))
			sizingWristRotations[i] = make([]*mmath.MQuaternion, len(frames))

			// 元モデルのデフォーム(IK ON)
			if err := miter.IterParallelByList(frames, allBlockSizes[i], func(data, index int) {
				frame := float32(data)
				vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
				vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
				vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, true, frame, arm_direction_bone_names[i], false)
				sizingOriginalAllDeltas[i][index] = vmdDeltas

				nowArmRot := vmdDeltas.Bones.Get(sizingArmBone.Index()).FilledFrameRotation()
				_, sizingArmRotations[i][index] = nowArmRot.SeparateTwistByAxis(sizingArmBone.Extend.NormalizedLocalAxisX)

				nowElbowRot := vmdDeltas.Bones.Get(sizingElbowBone.Index()).FilledFrameRotation()
				angle := math.Abs(nowElbowRot.ToRadian())
				if nowElbowRot.XYZ().Dot(sizingElbowBone.Extend.NormalizedLocalAxisY) < 0 {
					angle = -angle
				}
				sizingElbowRotations[i][index] = mmath.NewMQuaternionFromAxisAngles(sizingElbowBone.Extend.NormalizedLocalAxisY, angle)

				nowWristRot := vmdDeltas.Bones.Get(sizingWristBone.Index()).FilledFrameRotation()
				_, sizingWristRotations[i][index] = nowWristRot.SeparateTwistByAxis(sizingWristBone.Extend.NormalizedLocalAxisX)
			}); err != nil {
				errorChan <- err
			}
		}(i, direction)
	}

	// すべてのゴルーチンの完了を待つ
	wg.Wait()
	close(errorChan) // 全てのゴルーチンが終了したらチャネルを閉じる

	// チャネルからエラーを受け取る
	for err := range errorChan {
		if err != nil {
			return false, err
		}
	}

	// 補正を登録
	for i, frames := range allFrames {
		armBoneName := pmx.ARM.StringFromDirection(directions[i])
		elbowBoneName := pmx.ELBOW.StringFromDirection(directions[i])
		wristBoneName := pmx.WRIST.StringFromDirection(directions[i])

		// sizingMotion.BoneFrames.Delete(pmx.ARM_TWIST.StringFromDirection(directions[i]))
		// sizingMotion.BoneFrames.Delete(pmx.WRIST_TWIST.StringFromDirection(directions[i]))

		for j, iFrame := range frames {
			frame := float32(iFrame)

			armBf := sizingMotion.BoneFrames.Get(armBoneName).Get(frame)
			armBf.Rotation = sizingArmRotations[i][j]
			sizingMotion.InsertRegisteredBoneFrame(armBoneName, armBf)

			elbowBf := sizingMotion.BoneFrames.Get(elbowBoneName).Get(frame)
			elbowBf.Rotation = sizingElbowRotations[i][j]
			sizingMotion.InsertRegisteredBoneFrame(elbowBoneName, elbowBf)

			wristBf := sizingMotion.BoneFrames.Get(wristBoneName).Get(frame)
			wristBf.Rotation = sizingWristRotations[i][j]
			sizingMotion.InsertRegisteredBoneFrame(wristBoneName, wristBf)
		}
	}

	if mlog.IsVerbose() {
		title := "捩り補正01_クリーニング"
		outputPath := mutils.CreateOutputPath(sizingSet.OriginalVmdPath, title)
		repository.NewVmdRepository().Save(outputPath, sizingMotion, true)
		mlog.V("%s: %s", title, outputPath)
	}

	// // 腕補正 -----------------------------------------------------
	// errorChan = make(chan error, 2)
	// wg.Add(2)
	// for i, direction := range directions {
	// 	frames := allFrames[i]

	// 	go func(i int, direction string, bfs *vmd.BoneNameFrames) {
	// 		defer wg.Done()
	// 		defer func() {
	// 			// recoverによるpanicキャッチ
	// 			if r := recover(); r != nil {
	// 				stackTrace := debug.Stack()

	// 				var errMsg string
	// 				// パニックの値がerror型である場合、エラーメッセージを取得
	// 				if err, ok := r.(error); ok {
	// 					errMsg = err.Error()
	// 				} else {
	// 					// それ以外の型の場合は、文字列に変換
	// 					errMsg = fmt.Sprintf("%v", r)
	// 				}

	// 				errorChan <- fmt.Errorf("panic: %s\n%s", errMsg, stackTrace)
	// 			}
	// 		}()

	// 		mlog.I(mi18n.T("捩り補正02", map[string]interface{}{"No": sizingSet.Index + 1, "Direction": direction}))

	// 		sizingArmBone := sizingModel.Bones.GetByName(pmx.ARM.StringFromDirection(direction))
	// 		sizingElbowBone := sizingModel.Bones.GetByName(pmx.ELBOW.StringFromDirection(direction))

	// 		// 先モデルの腕デフォーム(IK ON)
	// 		for j, iFrame := range frames {
	// 			frame := float32(iFrame)

	// 			vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
	// 			vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
	// 			vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, true, frame, arm_direction_bone_names[i], false)

	// 			elbowGlobalPosition := sizingOriginalAllDeltas[i][j].Bones.Get(sizingElbowBone.Index()).FilledGlobalPosition()

	// 			sizingArmIkDeltas := deform.DeformIk(sizingModel, sizingMotion, vmdDeltas, frame, armIkBones[i], elbowGlobalPosition, arm_direction_bone_names[i])

	// 			bf := bfs.Get(frame)
	// 			bf.Rotation = sizingArmIkDeltas.Bones.Get(sizingArmBone.Index()).FilledFrameRotation()
	// 			bf.Registered = true
	// 			bfs.Insert(bf)

	// 			if j < len(frames)-1 {
	// 				nextFrame := float32(frames[j+1])
	// 				nextBf := bfs.Get(nextFrame)
	// 				nextBf.Rotation = bf.Rotation.Copy()
	// 				nextBf.Registered = true
	// 				bfs.Insert(nextBf)
	// 			}
	// 		}
	// 	}(i, direction, sizingMotion.BoneFrames.Get(pmx.ARM.StringFromDirection(direction)))
	// }

	// // すべてのゴルーチンの完了を待つ
	// wg.Wait()
	// close(errorChan) // 全てのゴルーチンが終了したらチャネルを閉じる

	// // チャネルからエラーを受け取る
	// for err := range errorChan {
	// 	if err != nil {
	// 		return false, err
	// 	}
	// }

	// if mlog.IsVerbose() {
	// 	title := "捩り補正02_腕"
	// 	outputPath := mutils.CreateOutputPath(sizingSet.OriginalVmdPath, title)
	// 	repository.NewVmdRepository().Save(outputPath, sizingMotion, true)
	// 	mlog.V("%s: %s", title, outputPath)
	// }

	// 腕捩り補正 -----------------------------------------------------
	errorChan = make(chan error, 2)
	wg.Add(2)
	for i, direction := range directions {
		frames := allFrames[i]

		go func(i int, direction string, bfs *vmd.BoneNameFrames) {
			defer wg.Done()
			defer func() {
				// recoverによるpanicキャッチ
				if r := recover(); r != nil {
					stackTrace := debug.Stack()

					var errMsg string
					// パニックの値がerror型である場合、エラーメッセージを取得
					if err, ok := r.(error); ok {
						errMsg = err.Error()
					} else {
						// それ以外の型の場合は、文字列に変換
						errMsg = fmt.Sprintf("%v", r)
					}

					errorChan <- fmt.Errorf("panic: %s\n%s", errMsg, stackTrace)
				}
			}()

			mlog.I(mi18n.T("捩り補正03", map[string]interface{}{"No": sizingSet.Index + 1, "Direction": direction}))

			sizingArmTwistBone := sizingModel.Bones.GetByName(pmx.ARM_TWIST.StringFromDirection(direction))
			sizingWristBone := sizingModel.Bones.GetByName(pmx.WRIST.StringFromDirection(direction))

			// 先モデルの腕捩デフォーム(IK ON)
			for j, iFrame := range frames {
				frame := float32(iFrame)

				vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
				vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
				vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, true, frame, arm_direction_bone_names[i], false)

				wristGlobalPosition := sizingOriginalAllDeltas[i][j].Bones.Get(sizingWristBone.Index()).FilledGlobalPosition()

				sizingArmTwistIkDeltas := deform.DeformIk(sizingModel, sizingMotion, vmdDeltas, frame, armTwistIkBones[i], wristGlobalPosition, arm_direction_bone_names[i])

				bf := bfs.Get(frame)
				bf.Rotation = sizingArmTwistIkDeltas.Bones.Get(sizingArmTwistBone.Index()).FilledFrameRotation()
				bf.Registered = true
				bfs.Insert(bf)

				if j < len(frames)-1 {
					nextFrame := float32(frames[j+1])
					nextBf := bfs.Get(nextFrame)
					nextBf.Rotation = bf.Rotation.Copy()
					nextBf.Registered = true
					bfs.Insert(nextBf)
				}
			}
		}(i, direction, sizingMotion.BoneFrames.Get(pmx.ARM_TWIST.StringFromDirection(direction)))
	}

	// すべてのゴルーチンの完了を待つ
	wg.Wait()
	close(errorChan) // 全てのゴルーチンが終了したらチャネルを閉じる

	// チャネルからエラーを受け取る
	for err := range errorChan {
		if err != nil {
			return false, err
		}
	}

	if mlog.IsVerbose() {
		title := "捩り補正03_腕捩"
		outputPath := mutils.CreateOutputPath(sizingSet.OriginalVmdPath, title)
		repository.NewVmdRepository().Save(outputPath, sizingMotion, true)
		mlog.V("%s: %s", title, outputPath)
	}

	// 手捩り補正 -----------------------------------------------------
	errorChan = make(chan error, 2)
	wg.Add(2)
	for i, direction := range directions {
		frames := allFrames[i]

		go func(i int, direction string, bfs *vmd.BoneNameFrames) {
			defer wg.Done()
			defer func() {
				// recoverによるpanicキャッチ
				if r := recover(); r != nil {
					stackTrace := debug.Stack()

					var errMsg string
					// パニックの値がerror型である場合、エラーメッセージを取得
					if err, ok := r.(error); ok {
						errMsg = err.Error()
					} else {
						// それ以外の型の場合は、文字列に変換
						errMsg = fmt.Sprintf("%v", r)
					}

					errorChan <- fmt.Errorf("panic: %s\n%s", errMsg, stackTrace)
				}
			}()

			mlog.I(mi18n.T("捩り補正04", map[string]interface{}{"No": sizingSet.Index + 1, "Direction": direction}))

			sizingWristTwistBone := sizingModel.Bones.GetByName(pmx.WRIST_TWIST.StringFromDirection(direction))
			sizingWristTailBone := sizingModel.Bones.GetByName(pmx.WRIST_TAIL.StringFromDirection(direction))

			// 先モデルの手捩デフォーム(IK ON)
			for j, iFrame := range frames {
				frame := float32(iFrame)

				vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
				vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
				vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, true, frame, arm_direction_bone_names[i], false)

				wristTailGlobalPosition := sizingOriginalAllDeltas[i][j].Bones.Get(sizingWristTailBone.Index()).FilledGlobalPosition()

				sizingWristTwistIkDeltas := deform.DeformIk(sizingModel, sizingMotion, vmdDeltas, frame, wristTwistIkBones[i], wristTailGlobalPosition, arm_direction_bone_names[i])

				bf := bfs.Get(frame)
				bf.Rotation = sizingWristTwistIkDeltas.Bones.Get(sizingWristTwistBone.Index()).FilledFrameRotation()
				bf.Registered = true
				bfs.Insert(bf)

				if j < len(frames)-1 {
					nextFrame := float32(frames[j+1])
					nextBf := bfs.Get(nextFrame)
					nextBf.Rotation = bf.Rotation.Copy()
					nextBf.Registered = true
					bfs.Insert(nextBf)
				}
			}
		}(i, direction, sizingMotion.BoneFrames.Get(pmx.WRIST_TWIST.StringFromDirection(direction)))
	}

	// すべてのゴルーチンの完了を待つ
	wg.Wait()
	close(errorChan) // 全てのゴルーチンが終了したらチャネルを閉じる

	// チャネルからエラーを受け取る
	for err := range errorChan {
		if err != nil {
			return false, err
		}
	}

	if mlog.IsVerbose() {
		title := "捩り補正04_手捩"
		outputPath := mutils.CreateOutputPath(sizingSet.OriginalVmdPath, title)
		repository.NewVmdRepository().Save(outputPath, sizingMotion, true)
		mlog.V("%s: %s", title, outputPath)
	}

	// // 手首補正 -----------------------------------------------------
	// errorChan = make(chan error, 2)
	// wg.Add(2)
	// for i, direction := range directions {
	// 	frames := allFrames[i]

	// 	go func(i int, direction string, bfs *vmd.BoneNameFrames) {
	// 		defer wg.Done()
	// 		defer func() {
	// 			// recoverによるpanicキャッチ
	// 			if r := recover(); r != nil {
	// 				stackTrace := debug.Stack()

	// 				var errMsg string
	// 				// パニックの値がerror型である場合、エラーメッセージを取得
	// 				if err, ok := r.(error); ok {
	// 					errMsg = err.Error()
	// 				} else {
	// 					// それ以外の型の場合は、文字列に変換
	// 					errMsg = fmt.Sprintf("%v", r)
	// 				}

	// 				errorChan <- fmt.Errorf("panic: %s\n%s", errMsg, stackTrace)
	// 			}
	// 		}()

	// 		mlog.I(mi18n.T("捩り補正05", map[string]interface{}{"No": sizingSet.Index + 1, "Direction": direction}))

	// 		sizingWristBone := sizingModel.Bones.GetByName(pmx.WRIST.StringFromDirection(direction))
	// 		sizingWristTailBone := sizingModel.Bones.GetByName(pmx.WRIST_TAIL.StringFromDirection(direction))

	// 		// 先モデルの腕デフォーム(IK ON)
	// 		for j, iFrame := range frames {
	// 			frame := float32(iFrame)

	// 			vmdDeltas := delta.NewVmdDeltas(frame, sizingModel.Bones, sizingModel.Hash(), sizingMotion.Hash())
	// 			vmdDeltas.Morphs = deform.DeformMorph(sizingModel, sizingMotion.MorphFrames, frame, nil)
	// 			vmdDeltas = deform.DeformBoneByPhysicsFlag(sizingModel, sizingMotion, vmdDeltas, true, frame, arm_direction_bone_names[i], false)

	// 			wristTailGlobalPosition := sizingOriginalAllDeltas[i][j].Bones.Get(sizingWristTailBone.Index()).FilledGlobalPosition()

	// 			sizingWristIkDeltas := deform.DeformIk(sizingModel, sizingMotion, vmdDeltas, frame, wristIkBones[i], wristTailGlobalPosition, arm_direction_bone_names[i])

	// 			bf := bfs.Get(frame)
	// 			bf.Rotation = sizingWristIkDeltas.Bones.Get(sizingWristBone.Index()).FilledFrameRotation()
	// 			bf.Registered = true
	// 			bfs.Insert(bf)

	// 			if j < len(frames)-1 {
	// 				nextFrame := float32(frames[j+1])
	// 				nextBf := bfs.Get(nextFrame)
	// 				nextBf.Rotation = bf.Rotation.Copy()
	// 				nextBf.Registered = true
	// 				bfs.Insert(nextBf)
	// 			}
	// 		}
	// 	}(i, direction, sizingMotion.BoneFrames.Get(pmx.WRIST_TWIST.StringFromDirection(direction)))
	// }

	// // すべてのゴルーチンの完了を待つ
	// wg.Wait()
	// close(errorChan) // 全てのゴルーチンが終了したらチャネルを閉じる

	// // チャネルからエラーを受け取る
	// for err := range errorChan {
	// 	if err != nil {
	// 		return false, err
	// 	}
	// }

	// if mlog.IsVerbose() {
	// 	title := "捩り補正05_手首"
	// 	outputPath := mutils.CreateOutputPath(sizingSet.OriginalVmdPath, title)
	// 	repository.NewVmdRepository().Save(outputPath, sizingMotion, true)
	// 	mlog.V("%s: %s", title, outputPath)
	// }

	sizingSet.CompletedSizingArmTwist = true

	return true, nil
}

func isValidCleanArmTwist(sizingSet *domain.SizingSet) bool {
	originalModel := sizingSet.OriginalPmx

	for _, direction := range directions {
		if !originalModel.Bones.ContainsByName(pmx.ARM.StringFromDirection(direction)) {
			mlog.WT(mi18n.T("ボーン不足"), mi18n.T("捩り最適化ボーン不足", map[string]interface{}{
				"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.ARM.StringFromDirection(direction)}))
			return false
		}

		if !originalModel.Bones.ContainsByName(pmx.ARM_TWIST.StringFromDirection(direction)) {
			mlog.WT(mi18n.T("ボーン不足"), mi18n.T("捩り最適化ボーン不足", map[string]interface{}{
				"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.ARM_TWIST.StringFromDirection(direction)}))
			return false
		}

		if !originalModel.Bones.ContainsByName(pmx.ELBOW.StringFromDirection(direction)) {
			mlog.WT(mi18n.T("ボーン不足"), mi18n.T("捩り最適化ボーン不足", map[string]interface{}{
				"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.ELBOW.StringFromDirection(direction)}))
			return false
		}

		if !originalModel.Bones.ContainsByName(pmx.WRIST_TWIST.StringFromDirection(direction)) {
			mlog.WT(mi18n.T("ボーン不足"), mi18n.T("捩り最適化ボーン不足", map[string]interface{}{
				"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.WRIST_TWIST.StringFromDirection(direction)}))
			return false
		}

		if !originalModel.Bones.ContainsByName(pmx.WRIST.StringFromDirection(direction)) {
			mlog.WT(mi18n.T("ボーン不足"), mi18n.T("捩り最適化ボーン不足", map[string]interface{}{
				"No": sizingSet.Index + 1, "ModelType": mi18n.T("元モデル"), "BoneName": pmx.WRIST.StringFromDirection(direction)}))
			return false
		}
	}

	return true
}
