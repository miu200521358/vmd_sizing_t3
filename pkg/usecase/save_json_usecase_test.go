package usecase

import (
	"strings"
	"testing"

	"github.com/miu200521358/mlib_go/pkg/domain/pmx"
	"github.com/miu200521358/mlib_go/pkg/infrastructure/repository"
)

func TestSaveJson(t *testing.T) {
	pmxPath := "D:/MMD/MikuMikuDance_v926x64/UserFile/Model/_あにまさ式/カイト.pmx"
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
