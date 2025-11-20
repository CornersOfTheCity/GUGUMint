package eth

import (
	"crypto/ecdsa"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"gorm.io/gorm"
)

type Contract struct {
	Client  *ethclient.Client
	Auth    *bind.TransactOpts
	Address common.Address
	ChainID *big.Int
	PrivKey *ecdsa.PrivateKey
	bound   *bind.BoundContract
}

func NewClient(rpcURL string, chainID int64) (*ethclient.Client, *big.Int, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, nil, err
	}
	return client, big.NewInt(chainID), nil
}

func NewMintContract(client *ethclient.Client, contractAddr string, chainID *big.Int, privKeyHex string) (*Contract, error) {
	privKey, err := crypto.HexToECDSA(trimHexPrefix(privKeyHex))
	if err != nil {
		return nil, err
	}

	fromAddr := crypto.PubkeyToAddress(privKey.Public().(*ecdsa.PublicKey))
	auth, err := bind.NewKeyedTransactorWithChainID(privKey, chainID)
	if err != nil {
		return nil, err
	}
	auth.From = fromAddr

	parsedABI, err := abi.JSON(strings.NewReader(mintABI))
	if err != nil {
		return nil, err
	}

	addr := common.HexToAddress(contractAddr)
	bound := bind.NewBoundContract(addr, parsedABI, client, client, client)

	return &Contract{
		Client:  client,
		Auth:    auth,
		Address: addr,
		ChainID: chainID,
		PrivKey: privKey,
		bound:   bound,
	}, nil
}

func trimHexPrefix(s string) string {
	if len(s) >= 2 && (s[0:2] == "0x" || s[0:2] == "0X") {
		return s[2:]
	}
	return s
}

// helper to convert hex string bytes32 to [32]byte
func HexToBytes32(hexStr string) ([32]byte, error) {
	var out [32]byte
	b := common.FromHex(hexStr)
	if len(b) != 32 {
		return out, gorm.ErrInvalidData
	}
	copy(out[:], b)
	return out, nil
}

// minimal ABI definition for the mint function
const mintABI = `[{"inputs":[{"internalType":"address","name":"to","type":"address"},{"internalType":"bytes32","name":"hash","type":"bytes32"},{"internalType":"uint8","name":"v","type":"uint8"},{"internalType":"bytes32","name":"r","type":"bytes32"},{"internalType":"bytes32","name":"s","type":"bytes32"}],"name":"mint","outputs":[],"stateMutability":"nonpayable","type":"function"}]`

// Mint calls the contract mint function and returns the transaction
func (c *Contract) Mint(to common.Address, hash [32]byte, v uint8, r [32]byte, s [32]byte) (*types.Transaction, error) {
	return c.bound.Transact(c.Auth, "mint", to, hash, v, r, s)
}
