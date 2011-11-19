package track

import (
	"fmt"
	//	"github.com/cmars/replican-sync/replican/fs"
	"github.com/cmars/replican-sync/replican/sync"
)

type TrackerReq interface {
	Checkpoint() string
	RespChan() chan TrackerResp
}

type ReqIndex struct {
	ckpt     string
	respChan chan TrackerResp
}

func NewReqIndex(ckpt string, respChan chan TrackerResp) *ReqIndex {
	return &ReqIndex{ckpt: ckpt, respChan: respChan}
}

func (req *ReqIndex) Checkpoint() string { return req.ckpt }

func (req *ReqIndex) RespChan() chan TrackerResp { return req.respChan }

type ReqPatchBlocks struct {
	ckpt      string
	respChan  chan TrackerResp
	patchPlan *sync.PatchPlan
}

func NewReqPatchBlocks(ckpt string, respChan chan TrackerResp, patchPlan *sync.PatchPlan) *ReqPatchBlocks {
	return &ReqPatchBlocks{ckpt: ckpt, respChan: respChan, patchPlan: patchPlan}
}

func (req *ReqPatchBlocks) Checkpoint() string { return req.ckpt }

func (req *ReqPatchBlocks) RespChan() chan TrackerResp { return req.respChan }

func (req *ReqPatchBlocks) PatchPlan() *sync.PatchPlan { return req.patchPlan }

type TrackerResp interface {
	Checkpoint() string
}

func StartTracker(path string, requestChan chan TrackerReq) {
	//	store, _ := fs.NewLocalStore(path)
	//	dirStore := store.(*fs.LocalDirStore)

	go func() {
		poller := NewPoller(path, 60, nil)
		for {
			select {
			case path := <-poller.Paths:
				fmt.Printf("%v\n", path)
				// update this part of current head
				// commit if changed
				//				log.Commit(scannerUpdate.Root)
			case request := <-requestChan:
				switch request.(type) {
				case *ReqIndex:

				case *ReqPatchBlocks:

				}
			}
		}
		poller.Stop()
	}()
}
