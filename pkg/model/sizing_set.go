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
	OutputPmxPath   string
	OutputVmdPath   string

	OriginalVmdName string
	OriginalPmxName string
	SizingPmxName   string

	OriginalVmd *vmd.VmdMotion
	OriginalPmx *pmx.PmxModel
	SizingPmx   *pmx.PmxModel
	OutputPmx   *pmx.PmxModel
	OutputVmd   *vmd.VmdMotion
}

func NewSizingSet(index int) *SizingSet {
	return &SizingSet{
		Index: index,
	}
}
