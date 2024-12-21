package usecase

import (
	"strings"
	"testing"

	"github.com/miu200521358/mlib_go/pkg/domain/pmx"
	"github.com/miu200521358/mlib_go/pkg/infrastructure/repository"
)

func TestSaveJson(t *testing.T) {
	// pmxPath := "D:/MMD/MikuMikuDance_v926x64/UserFile/Model/_あにまさ式/カイト.pmx"
	pmxPath := "D:/MMD/MikuMikuDance_v926x64/UserFile/Model/刀剣乱舞/003_三日月宗近/三日月宗近 わち式 （刀ミュインナーβ）/わち式三日月宗近（刀ミュインナーβ）.pmx"
	// pmxPath := "D:/MMD/MikuMikuDance_v926x64/UserFile/Model/VOCALOID/初音ミク/Lat式ミクVer2.31/Lat式ミクVer2.31_Normal.pmx"

	jsonSavePath := strings.ReplaceAll(pmxPath, ".pmx", ".json")

	pmxRep := repository.NewPmxRepository()
	data, err := pmxRep.Load(pmxPath)
	if err != nil {
		t.Errorf("Expected error to be nil, got %q", err)
	}

	// Call the function to be tested
	if err := SaveJson(jsonSavePath, data.(*pmx.PmxModel)); err != nil {
		t.Errorf("Expected error to be nil, got %q", err)
	}
}
