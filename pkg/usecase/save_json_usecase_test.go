package usecase

import (
	"strings"
	"testing"

	"github.com/miu200521358/mlib_go/pkg/domain/pmx"
	"github.com/miu200521358/mlib_go/pkg/infrastructure/repository"
)

func TestSaveJson(t *testing.T) {
	// pmxPath := "D:/MMD/MikuMikuDance_v926x64/UserFile/Model/_あにまさ式/カイト.pmx"
	// pmxPath := "D:/MMD/MikuMikuDance_v926x64/UserFile/Model/刀剣乱舞/003_三日月宗近/三日月宗近 わち式 （刀ミュインナーβ）/わち式三日月宗近（刀ミュインナーβ）.pmx"
	// pmxPath := "D:/MMD/MikuMikuDance_v926x64/UserFile/Model/VOCALOID/初音ミク/Lat式ミクVer2.31/Lat式ミクVer2.31_Normal.pmx"
	pmxPath := "D:/MMD/MikuMikuDance_v926x64/UserFile/Model/_VMDサイジング/wa_129cm 20240628/wa_129cm.pmx"

	jsonSavePath := strings.ReplaceAll(pmxPath, ".pmx", ".json")

	pmxRep := repository.NewPmxRepository()
	data, err := pmxRep.Load(pmxPath)
	if err != nil {
		t.Errorf("Expected error to be nil, got %q", err)
	}
	model := data.(*pmx.PmxModel)

	if err := addBones(model); err != nil {
		t.Errorf("Expected error to be nil, got %q", err)
	}

	if err := addRigidBodies(model); err != nil {
		t.Errorf("Expected error to be nil, got %q", err)
	}

	pmxJsonPath := strings.ReplaceAll(jsonSavePath, ".json", "_json.pmx")
	pmxRep.Save(pmxJsonPath, model, false)

	jsonRep := repository.NewPmxJsonRepository()

	if err := jsonRep.Save(jsonSavePath, model, false); err != nil {
		t.Errorf("Expected error to be nil, got %q", err)
	}
}
