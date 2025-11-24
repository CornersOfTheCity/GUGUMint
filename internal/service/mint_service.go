package service

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/common"
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

	// 允许 status 为 "" 或 "unused" 的记录使用；
	// 其余非空状态（如 pending/success/failed）视为已使用，防止重复使用
	if mr.Status != "" && mr.Status != "unused" {
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
	// 由于实际交易由前端钱包发起，这里在签名发放时就标记为 success，
	// 表示该 hash 已经被使用，防止重复使用同一个 hash
	mr.Status = "success"
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
