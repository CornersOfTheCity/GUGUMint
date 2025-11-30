package service

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"gorm.io/gorm"

	"GUGUMint/internal/db"
	"GUGUMint/internal/eth"
)

type MintService struct {
	db       *gorm.DB
	contract *eth.Contract
}

func NewMintService(dbConn *gorm.DB, contract *eth.Contract) *MintService {
	return &MintService{db: dbConn, contract: contract}
}

type MintSignature struct {
	Hash    string `json:"hash"`
	Address string `json:"address"`
	V       uint8  `json:"v"`
	R       string `json:"r"` // 0x...
	S       string `json:"s"` // 0x...
}

func (s *MintService) ProcessMint(ctx context.Context, hashHex, addrHex string) (*MintSignature, error) {
	if !common.IsHexAddress(addrHex) {
		return nil, errors.New("invalid address")
	}

	var mr db.MintRequest
	if err := s.db.WithContext(ctx).Where("hash = ?", hashHex).First(&mr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("hash not found")
		}
		return nil, err
	}

	// 仅当状态为 pending 或 success 时视为已使用，防止重复使用；
	// failed 等状态允许重新使用该 hash
	if mr.Status == "pending" || mr.Status == "success" {
		return nil, errors.New("hash already used")
	}

	to := common.HexToAddress(addrHex)
	hash32, err := eth.HexToBytes32(hashHex)
	if err != nil {
		return nil, errors.New("invalid hash bytes32")
	}

	// message = keccak256(abi.encodePacked(to, hash))
	// Solidity abi.encodePacked(address,uint256) 对 address 使用 20 字节，不做 32 字节左填充
	// 因此这里直接使用 to.Bytes()，而不是 LeftPadBytes
	msgHash := crypto.Keccak256Hash(to.Bytes(), hash32[:])

	sig, err := crypto.Sign(msgHash.Bytes(), s.contract.PrivKey)
	if err != nil {
		return nil, err
	}

	if len(sig) != 65 {
		return nil, errors.New("invalid signature length")
	}

	var r, sBytes [32]byte
	copy(r[:], sig[0:32])
	copy(sBytes[:], sig[32:64])
	v := sig[64]
	if v < 27 {
		v += 27
	}

	mr.Address = addrHex
	// 在签名发放阶段仅标记为 signed，真正 success 由链上交易结果决定
	mr.Status = "signed"
	mr.UpdatedAt = time.Now().Unix()

	if err := s.db.WithContext(ctx).Save(&mr).Error; err != nil {
		return nil, err
	}

	rHex := common.BytesToHash(r[:]).Hex()
	sHex := common.BytesToHash(sBytes[:]).Hex()

	return &MintSignature{
		Hash:    hashHex,
		Address: addrHex,
		V:       v,
		R:       rHex,
		S:       sHex,
	}, nil
}

// SaveTxHash 将前端上报的 txHash 记录到对应的 MintRequest，并标记为 pending
func (s *MintService) SaveTxHash(ctx context.Context, hashHex, addrHex, txHash string) (*db.MintRequest, error) {
	var mr db.MintRequest
	if err := s.db.WithContext(ctx).Where("hash = ?", hashHex).First(&mr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("hash not found")
		}
		return nil, err
	}

	if mr.Address != "" && mr.Address != addrHex {
		return nil, errors.New("address mismatch")
	}

	if mr.Status != "signed" && mr.Status != "pending" {
		return nil, errors.New("invalid status for tx hash binding")
	}

	mr.Address = addrHex
	mr.TxHash = txHash
	mr.Status = "pending"
	mr.UpdatedAt = time.Now().Unix()

	if err := s.db.WithContext(ctx).Save(&mr).Error; err != nil {
		return nil, err
	}

	return &mr, nil
}

// GetStatusByTxHash 根据 txHash 查询当前 mint 请求状态
func (s *MintService) GetStatusByTxHash(ctx context.Context, txHash string) (*db.MintRequest, error) {
	var mr db.MintRequest
	if err := s.db.WithContext(ctx).Where("tx_hash = ?", txHash).First(&mr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("tx hash not found")
		}
		return nil, err
	}
	return &mr, nil
}

// StartTxWatcher 启动一个简单的轮询任务，根据链上交易收据更新 pending 状态
func (s *MintService) StartTxWatcher(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.updatePendingTx(ctx)
			}
		}
	}()
}

func (s *MintService) updatePendingTx(ctx context.Context) {
	var list []db.MintRequest
	if err := s.db.WithContext(ctx).Where("status = ? AND tx_hash <> ''", "pending").Find(&list).Error; err != nil {
		return
	}

	for _, mr := range list {
		// 防御性校验
		if mr.TxHash == "" {
			continue
		}

		receipt, err := s.contract.Client.TransactionReceipt(ctx, common.HexToHash(mr.TxHash))
		if err != nil {
			// 这里不区分未打包/真正错误，简单跳过等待下次轮询
			continue
		}
		if receipt == nil {
			continue
		}

		if receipt.Status == types.ReceiptStatusSuccessful {
			mr.Status = "success"
		} else {
			mr.Status = "failed"
		}
		mr.UpdatedAt = time.Now().Unix()
		_ = s.db.WithContext(ctx).Save(&mr).Error
	}
}
