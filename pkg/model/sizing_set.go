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

	OriginalJsonPmx           *pmx.PmxModel
	OriginalPmxRatio          float64 // 全体比率
	OriginalPmxUpperLength    float64 // 上半身長さ
	OriginalPmxUpperAngle     float64 // 上半身角度
	OriginalPmxUpper2Length   float64 // 上半身2長さ
	OriginalPmxUpper2Angle    float64 // 上半身2角度
	OriginalPmxNeckLength     float64 // 首長さ
	OriginalPmxNeckAngle      float64 // 首角度
	OriginalPmxShoulderLength float64 // 肩長さ
	OriginalPmxShoulderAngle  float64 // 肩角度
	OriginalPmxArmLength      float64 // 腕長さ
	OriginalPmxArmAngle       float64 // 腕角度
	OriginalPmxElbowLength    float64 // ひじ長さ
	OriginalPmxElbowAngle     float64 // ひじ角度
	OriginalPmxWristLength    float64 // 手首長さ
	OriginalPmxWristAngle     float64 // 手首角度
	OriginalPmxLowerLength    float64 // 下半身長さ
	OriginalPmxLowerAngle     float64 // 下半身角度
	OriginalPmxLegLength      float64 // 足長さ
	OriginalPmxLegAngle       float64 // 足角度
	OriginalPmxKneeLength     float64 // ひざ長さ
	OriginalPmxKneeAngle      float64 // ひざ角度
}

func NewSizingSet(index int) *SizingSet {
	return &SizingSet{
		Index: index,
	}
}
