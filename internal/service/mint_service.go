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

func (s *MintService) ProcessMint(ctx context.Context, hashHex, addrHex string) (string, error) {
	if !common.IsHexAddress(addrHex) {
		return "", errors.New("invalid address")
	}

	var mr db.MintRequest
	if err := s.db.WithContext(ctx).Where("hash = ?", hashHex).First(&mr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", errors.New("hash not found")
		}
		return "", err
	}

	if mr.Status == "success" {
		return mr.TxHash, nil
	}

	to := common.HexToAddress(addrHex)
	hash32, err := eth.HexToBytes32(hashHex)
	if err != nil {
		return "", errors.New("invalid hash bytes32")
	}

	// message = keccak256(abi.encodePacked(to, hash))
	msgHash := crypto.Keccak256Hash(common.LeftPadBytes(to.Bytes(), 32), hash32[:])

	sig, err := crypto.Sign(msgHash.Bytes(), s.contract.PrivKey)
	if err != nil {
		return "", err
	}

	if len(sig) != 65 {
		return "", errors.New("invalid signature length")
	}

	var r, sBytes [32]byte
	copy(r[:], sig[0:32])
	copy(sBytes[:], sig[32:64])
	v := sig[64]
	if v < 27 {
		v += 27
	}

	tx, err := s.contract.Mint(to, hash32, v, r, sBytes)
	if err != nil {
		return "", err
	}

	mr.Address = addrHex
	mr.Status = "pending"
	mr.TxHash = tx.Hash().Hex()
	mr.UpdatedAt = time.Now().Unix()

	if err := s.db.WithContext(ctx).Save(&mr).Error; err != nil {
		return "", err
	}

	return tx.Hash().Hex(), nil
}
