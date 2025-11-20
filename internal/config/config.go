package config

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	// 运行时使用的完整 DSN，由下面拆开的字段拼出来
	DBDSN string `yaml:"-"`

	// 拆开的数据库配置字段
	DBHost     string `yaml:"db_host"`
	DBUser     string `yaml:"db_user"`
	DBPassword string `yaml:"db_password"`
	DBName     string `yaml:"db_name"`
	DBPort     int    `yaml:"db_port"`
	DBSSLMode  string `yaml:"db_sslmode"`

	BSCRPCURL       string `yaml:"bsc_rpc_url"`
	ContractAddress string `yaml:"contract_address"`
	PrivateKey      string `yaml:"private_key"`
	ChainID         int64  `yaml:"chain_id"`
}

func Load() Config {
	var cfg Config

	// 1. 从项目根目录的 config.yaml 读取（如果存在）
	wd, err := os.Getwd()
	if err == nil {
		path := filepath.Join(wd, "config.yaml")
		if data, err := ioutil.ReadFile(path); err == nil {
			if err := yaml.Unmarshal(data, &cfg); err != nil {
				log.Fatalf("failed to parse config.yaml: %v", err)
			}
		}
	}

	// 2. 环境变量覆盖（如果设置了的话）
	if v := os.Getenv("DB_DSN"); v != "" {
		cfg.DBDSN = v
	}
	if v := os.Getenv("BSC_RPC_URL"); v != "" {
		cfg.BSCRPCURL = v
	}
	if v := os.Getenv("CONTRACT_ADDRESS"); v != "" {
		cfg.ContractAddress = v
	}
	if v := os.Getenv("PRIVATE_KEY"); v != "" {
		cfg.PrivateKey = v
	}
	if v := os.Getenv("CHAIN_ID"); v != "" {
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			log.Fatalf("invalid CHAIN_ID: %v", err)
		}
		cfg.ChainID = parsed
	}

	// 3. 默认值：若未设置 chain_id，则默认 BSC testnet 97
	if cfg.ChainID == 0 {
		cfg.ChainID = 97
	}

	// 3.1 如果没有通过环境变量直接提供 DB_DSN，则用拆开的字段拼接
	if cfg.DBDSN == "" {
		// 给一些合理默认值，避免完全为空
		if cfg.DBHost == "" {
			cfg.DBHost = "127.0.0.1"
		}
		if cfg.DBPort == 0 {
			cfg.DBPort = 5432
		}
		if cfg.DBSSLMode == "" {
			cfg.DBSSLMode = "disable"
		}
		// 必须提供的：user / password / dbname
		if cfg.DBUser == "" || cfg.DBPassword == "" || cfg.DBName == "" {
			log.Fatalf("missing required db config fields: db_user, db_password, db_name")
		}
		cfg.DBDSN =
			"host=" + cfg.DBHost +
				" user=" + cfg.DBUser +
				" password=" + cfg.DBPassword +
				" dbname=" + cfg.DBName +
				" port=" + strconv.Itoa(cfg.DBPort) +
				" sslmode=" + cfg.DBSSLMode
	}

	// 4. 基础校验
	if cfg.DBDSN == "" || cfg.BSCRPCURL == "" || cfg.ContractAddress == "" || cfg.PrivateKey == "" {
		log.Fatalf("missing required config fields: db connection info, bsc_rpc_url, contract_address, private_key")
	}

	_ = time.Now() // 占位防止未使用导入，如后续需要可以加入时间相关配置

	return cfg
}
