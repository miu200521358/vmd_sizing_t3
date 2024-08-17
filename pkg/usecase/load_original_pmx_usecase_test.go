package usecase

import (
	"testing"

	"github.com/miu200521358/mlib_go/pkg/domain/pmx"
	"github.com/miu200521358/mlib_go/pkg/infrastructure/repository"
)

func TestUsecase_LoadOriginalPmx(t *testing.T) {
	// Save the model
	// jsonPath := "D:/MMD/MikuMikuDance_v926x64/UserFile/Model/刀剣乱舞/003_三日月宗近/三日月宗近 わち式 （刀ミュインナーβ）/わち式三日月宗近（刀ミュインナーβ）.json"
	jsonPath := "D:/MMD/MikuMikuDance_v926x64/UserFile/Model/_VMDサイジング/wa_129cm 20240628/wa_129cm.json"
	data, err := repository.NewPmxJsonRepository().Load(jsonPath)
	if err != nil {
		t.Errorf("Expected error to be nil, got %q", err)
	}
	jsonModel := data.(*pmx.PmxModel)

	model, err := LoadOriginalPmx(jsonModel)
	if err != nil {
		t.Errorf("Expected error to be nil, got %q", err)
	}

	outputPath := "C:/MMD/vmd_sizing_t3/test_resources/sizing_model_debug.pmx"
	repository.NewPmxRepository().Save(outputPath, model, true)
}

func TestUsecase_addNonExistBones(t *testing.T) {
	// Save the model
	jsonPath := "D:/MMD/MikuMikuDance_v926x64/UserFile/Model/_あにまさ式/カイト.json"
	data, err := repository.NewPmxJsonRepository().Load(jsonPath)
	if err != nil {
		t.Errorf("Expected error to be nil, got %q", err)
	}
	jsonModel := data.(*pmx.PmxModel)

	model, err := loadMannequinPmx()
	if err != nil {
		t.Errorf("Expected error to be nil, got %q", err)
	}

	jsonModel = addNonExistBones(model, jsonModel)

	outputPath := "C:/MMD/vmd_sizing_t3/test_resources/sizing_model_debug.pmx"
	repository.NewPmxRepository().Save(outputPath, jsonModel, true)
}
