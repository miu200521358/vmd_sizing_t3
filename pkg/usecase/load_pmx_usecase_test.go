package usecase

import (
	"testing"

	"github.com/miu200521358/mlib_go/pkg/domain/pmx"
	"github.com/miu200521358/mlib_go/pkg/domain/vmd"
	"github.com/miu200521358/mlib_go/pkg/infrastructure/deform"
	"github.com/miu200521358/mlib_go/pkg/infrastructure/repository"
	"github.com/miu200521358/vmd_sizing_t3/pkg/model"
)

func TestUsecase_LoadOriginalPmxByJson(t *testing.T) {
	// Save the model
	// jsonPath := "D:/MMD/MikuMikuDance_v926x64/UserFile/Model/刀剣乱舞/003_三日月宗近/三日月宗近 わち式 （刀ミュインナーβ）/わち式三日月宗近（刀ミュインナーβ）.json"
	// jsonPath := "D:/MMD/MikuMikuDance_v926x64/UserFile/Model/_あにまさ式/カイト.json"
	// jsonPath := "D:/MMD/MikuMikuDance_v926x64/UserFile/Model/_VMDサイジング/wa_129cm 20240628/wa_129cm.json"
	// jsonPath := "D:/MMD/MikuMikuDance_v926x64/UserFile/Model/刀剣乱舞/055_鶯丸/鶯丸 さとく式 ver0.90/さとく式鶯丸ver0.90.json"
	jsonPath := "C:/MMD/vmd_sizing_t3/archive/sizing_model.json"

	data, err := repository.NewPmxJsonRepository().Load(jsonPath)
	if err != nil {
		t.Errorf("Expected error to be nil, got %q", err)
	}
	jsonModel := data.(*pmx.PmxModel)

	{
		model, err := loadMannequinPmx()
		if err != nil {
			t.Errorf("Expected error to be nil, got %q", err)
		}

		rep := repository.NewPmxRepository()

		addNonExistBones(model, jsonModel, true)
		rep.Save("C:/MMD/vmd_sizing_t3/test_resources/sizing_model_debug_add.pmx", jsonModel, true)
	}

	{
		model, err := LoadOriginalPmxByJson(jsonModel)
		if err != nil {
			t.Errorf("Expected error to be nil, got %q", err)
		}

		motion := vmd.NewVmdMotion("")
		motion = AddFitMorph(motion)

		deformModel := deform.DeformModel(model, motion, 0)
		repository.NewPmxRepository().Save(
			"C:/MMD/vmd_sizing_t3/test_resources/sizing_model_debug_fit.pmx", deformModel, true)

		for _, bone := range deformModel.Bones.Data {
			if !jsonModel.Bones.ContainsByName(bone.Name()) {
				t.Errorf("Expected bone %s to be contained", bone.Name())
			}
			if !bone.Position.NearEquals(jsonModel.Bones.GetByName(bone.Name()).Position, 1e-4) {
				t.Errorf("Expected bone %s to be near equals, got %v (%v)", bone.Name(),
					bone.Position, jsonModel.Bones.GetByName(bone.Name()).Position)
			}
		}
	}

	{
		originalModel, err := LoadOriginalPmxByJson(jsonModel)
		if err != nil {
			t.Errorf("Expected error to be nil, got %q", err)
		}

		sizingSet := model.NewSizingSet(0)
		sizingSet.OriginalPmxRatio = 1.0
		sizingSet.OriginalPmxShoulderLength = 5.0
		sizingSet.OriginalPmxShoulderAngle = 30.0
		jsonModel = RemakeFitMorph(originalModel, jsonModel, sizingSet)

		repository.NewPmxRepository().Save(
			"C:/MMD/vmd_sizing_t3/test_resources/sizing_model_debug_remake.pmx", jsonModel, true)

		for _, bone := range jsonModel.Bones.Data {
			if !jsonModel.Bones.ContainsByName(bone.Name()) {
				t.Errorf("Expected bone %s to be contained", bone.Name())
			}
		}
	}
}

func TestUsecase_AdjustPmxForSizing(t *testing.T) {
	// Save the model
	// originalPath := "D:/MMD/MikuMikuDance_v926x64/UserFile/Model/刀剣乱舞/136_日本号/ちびにほ A4式/ちびにほ.pmx"
	originalPath := "D:/MMD/MikuMikuDance_v926x64/UserFile/Model/刀剣乱舞/118_へし切長谷部/ちびはせ A4式/ちびはせ_ボーン修正.pmx"
	// originalPath := "D:/MMD/MikuMikuDance_v926x64/UserFile/Model/刀剣乱舞/003_三日月宗近/三日月宗近 わんぱく風 ちゃむ/wp_三日月.pmx"
	// originalPath := "D:/MMD/MikuMikuDance_v926x64/UserFile/Model/刀剣乱舞/003_三日月宗近/三日月宗近 わち式 （刀ミュインナーβ）/わち式三日月宗近（刀ミュインナーβ）.pmx"
	// originalPath := "D:/MMD/MikuMikuDance_v926x64/UserFile/Model/_あにまさ式/カイト.pmx"
	// originalPath := "D:/MMD/MikuMikuDance_v926x64/UserFile/Model/ゲーム/アイドルマスター/SDTYSシーズン3(Dモデル改変) あおうさぎ（P）/SDロイヤルスターレットまつり.pmx"
	// originalPath := "D:/MMD/MikuMikuDance_v926x64/UserFile/Model/ヘタリア/はなから牛乳Ｐ式 ドイツ ver1.01/ヘタリア・ドイツver1.00.pmx"
	// originalPath := "D:/MMD/MikuMikuDance_v926x64/UserFile/Model/ヘタリア/おりんぴぃ もちD/ドイツもち.pmx"
	// originalPath := "D:/MMD/MikuMikuDance_v926x64/UserFile/Model/_VMDサイジング/wa_129cm 20240628/wa_129cm.pmx"
	// originalPath := "D:/MMD/MikuMikuDance_v926x64/UserFile/Model/刀剣乱舞/055_鶯丸/鶯丸 さとく式 ver0.90/さとく式鶯丸ver0.90.pmx"
	// originalPath := "D:/MMD/MikuMikuDance_v926x64/UserFile/Model/VOCALOID/初音ミク/Tda式初音ミク・アペンドVer1.10/Tda式初音ミク・アペンド_Ver1.10.pmx"
	// originalPath := "D:/MMD/MikuMikuDance_v926x64/UserFile/Model/Vtuber/オリバー・エバンスモデル_ver1.03/オリバーエバンス.pmx"
	// originalPath := "D:/MMD/MikuMikuDance_v926x64/UserFile/Model/オリジナル/折岸みつ つみだんご/折岸みつ.pmx"

	rep := repository.NewPmxRepository()

	data, err := rep.Load(originalPath)
	if err != nil {
		t.Errorf("Expected error to be nil, got %q", err)
	}
	originalModel := data.(*pmx.PmxModel)

	{
		model, err := AdjustPmxForSizing(originalModel)
		if err != nil {
			t.Errorf("Expected error to be nil, got %q", err)
		}

		rep.Save("C:/MMD/vmd_sizing_t3/test_resources/sizing_model_adjust.pmx", model, true)
	}
}
