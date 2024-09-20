package explorer

import (
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum/common"
)

type BlockscoutAPI struct {
}

func (e *BlockscoutAPI) DownloadContract(w *cmn.Wallet, b *cmn.Blockchain, contract common.Address) error {
	return nil
}
