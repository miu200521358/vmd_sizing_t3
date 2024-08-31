package model

import (
	"github.com/miu200521358/mlib_go/pkg/domain/pmx"
	"github.com/miu200521358/mlib_go/pkg/domain/vmd"
)

type SizingSet struct {
	Index int

	OriginalVmdPath string
	OriginalPmxPath string
	SizingPmxPath   string
	OutputVmdPath   string

	OriginalVmdName string
	OriginalPmxName string
	SizingPmxName   string

	OriginalVmd *vmd.VmdMotion
	OriginalPmx *pmx.PmxModel
	SizingPmx   *pmx.PmxModel
	OutputVmd   *vmd.VmdMotion

	OriginalJsonPmx        *pmx.PmxModel
	OriginalPmxRatio       float64 // 全体比率
	OriginalPmxArmStance   float64 // 腕スタンス
	OriginalPmxElbowStance float64 // ひじスタンス
}

func NewSizingSet(index int) *SizingSet {
	return &SizingSet{
		Index: index,
	}
}
