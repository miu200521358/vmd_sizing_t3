package usecase

import (
	"fmt"
	"runtime"
	"sync"

	"github.com/miu200521358/mlib_go/pkg/domain/miter"
	"github.com/miu200521358/mlib_go/pkg/domain/vmd"
	"github.com/miu200521358/mlib_go/pkg/mutils/mi18n"
	"github.com/miu200521358/mlib_go/pkg/mutils/mlog"
	"github.com/miu200521358/vmd_sizing_t3/pkg/domain"
)

func SizingReduction(sizingSet *domain.SizingSet, setSize, completedProcessCount, totalProcessCount int) (bool, error) {
	if !sizingSet.IsSizingReduction || (sizingSet.IsSizingReduction && sizingSet.CompletedSizingReduction) {
		return false, nil
	}

	sizingMotion := sizingSet.OutputVmd
	sizingMotion.Processing = true

	mlog.I(mi18n.T("不要キー間引き開始", map[string]interface{}{"No": sizingSet.Index + 1, "CompletedProcessCount": fmt.Sprintf("%02d", completedProcessCount), "TotalProcessCount": fmt.Sprintf("%02d", totalProcessCount)}))
	sizingMotion.Processing = true

	numCPU := runtime.NumCPU()
	runtime.GOMAXPROCS(numCPU)
	defer runtime.GOMAXPROCS(int(numCPU / 4))

	reduced := vmd.NewBoneFrames()
	allCount := len(sizingMotion.BoneFrames.Data)
	allBoneNames := sizingMotion.BoneFrames.Names()

	blockSize, _ := miter.GetBlockSize(allCount)
	errorChan := make(chan error, numCPU)

	// ブロックサイズが全件数より小さい場合は並列処理
	var wg sync.WaitGroup
	var mu sync.Mutex

	iterIndex := 0
	for startIndex := 0; startIndex < allCount; startIndex += blockSize {
		wg.Add(1)
		go func(startIndex int) {
			defer func() {
				if err := miter.GetError(); err != nil {
					errorChan <- err
				}
				wg.Done()
			}()

			endIndex := startIndex + blockSize
			if endIndex > allCount-1 {
				endIndex = allCount - 1
			}

			for j := startIndex; j < endIndex; j++ {
				mu.Lock()
				bnfs := sizingMotion.BoneFrames.Get(allBoneNames[j])

				mlog.I(mi18n.T("不要キー間引き01", map[string]interface{}{"No": sizingSet.Index + 1, "CompletedProcessCount": fmt.Sprintf("%02d", completedProcessCount), "TotalProcessCount": fmt.Sprintf("%02d", totalProcessCount), "Name": bnfs.Name, "IterIndex": fmt.Sprintf("%04d", iterIndex), "AllCount": fmt.Sprintf("%04d", allCount)}))
				iterIndex++
				mu.Unlock()

				reduced.Append(bnfs.Reduce())
			}
		}(startIndex)
	}

	go func() {
		wg.Wait()
		close(errorChan)
	}()

	// チャネルからエラーを受け取る
	for err := range errorChan {
		if err != nil {
			return false, err
		}
	}

	sizingMotion.BoneFrames = reduced

	return true, nil
}
