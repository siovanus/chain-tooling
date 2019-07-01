package config

import (
	"encoding/json"
	"errors"
	"github.com/binance-chain/go-sdk/common/types"
	"io/ioutil"
	"log"
	"math/big"
	"strconv"
	"strings"
	"time"
)

type Conf struct {
	Env            string
	BaseUrl        string
	Token          string
	Decimal        uint64
	Txs            []*Tx
	Sum            int64
	ReceiversCount int
	Mnemonic       string
	BatchSize      int
	BatchInterval  int
	ReportFile     string
	Network        types.ChainNetwork
}

var RawConf *Conf

func init() {

	rawConf, error := parseConfig("airdrop.conf")
	if error != nil {
		log.Fatal(error)
		panic(error)
	}
	rawTxs, err := ReadRawTx("input.json")
	if err != nil {
		log.Fatal(error)
		panic(error)
	}
	rawConf.Txs, rawConf.Sum = RawTxsToTxs(rawTxs, rawConf.Decimal)
	rawConf.ReceiversCount = len(rawConf.Txs)
	error2 := validateConfig(rawConf)

	if error2 != nil {
		log.Fatal(error2)
		panic(error2)
	}

	logConf(rawConf)

	RawConf = rawConf
}

func logConf(conf *Conf) {
	log.Println("env:" + conf.Env)
	log.Println("token:" + conf.Token)
	log.Println("decimal:" + strconv.FormatUint(conf.Decimal, 10))
	log.Println("batch size:" + strconv.Itoa(conf.BatchSize))
	log.Println("batch interval (s):" + strconv.Itoa(conf.BatchInterval))
}

func parseConfig(configFile string) (*Conf, error) {
	content, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	configs := strings.Split(string(content), "\n")
	result := Conf{}
	for _, config := range configs {

		kv := strings.Split(config, "=")
		if strings.HasPrefix(config, "#") || len(kv) < 2 {
			continue
		}

		key := kv[0]
		value := kv[1]
		switch key {
		case "env":
			result.Env = value
		case "token":
			result.Token = value
		case "decimal":
			result.Decimal, _ = strconv.ParseUint(value, 10, 64)
		case "mnemonic":
			result.Mnemonic = value
		case "batchsize":
			result.BatchSize, _ = strconv.Atoi(value)
		case "batchinterval":
			result.BatchInterval, _ = strconv.Atoi(value)
		case "reportfile":
			result.ReportFile = value
		}
	}
	return &result, nil
}

func validateConfig(conf *Conf) error {

	conf.Env = strings.ToLower(conf.Env)
	if conf.Env == "testnet" {
		conf.BaseUrl = "testnet-dex.binance.org"
		conf.Network = types.TestNetwork
		types.Network = types.TestNetwork
	} else if conf.Env == "prod" {
		conf.BaseUrl = "dex.binance.org"
		conf.Network = types.ProdNetwork
		types.Network = types.ProdNetwork
	} else {
		return errors.New("env must be testnet or prod ")
	}

	if strings.TrimSpace(conf.Token) == "" {
		return errors.New("token must be specified ")
	}

	if conf.Decimal < 0 {
		return errors.New("amount can not be less than zero ")
	}

	if conf.ReceiversCount == 0 {
		return errors.New("there must be at least one receiver ")
	}

	if conf.BatchSize <= 0 || conf.BatchSize > 1000 {
		return errors.New("batchsize must be greater than 0 and less than 1001")
	}

	if len(conf.ReportFile) == 0 {
		conf.ReportFile = "report." + strconv.FormatInt(time.Now().UnixNano()/int64(1000000000), 10)
	}

	if conf.BatchInterval <= 0 {
		conf.BatchInterval = 5
	}

	return nil
}

type RawTx struct {
	BlockNumber string
	Hash        string
	To          string
	Amount      string
}

func ReadRawTx(path string) ([]*RawTx, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	txs := make([]*RawTx, 0)
	err = json.Unmarshal(data, &txs)
	if err != nil {
		return nil, err
	}
	return txs, nil
}

type Tx struct {
	To     string
	Amount int64
}

func RawTxsToTxs(rawTxs []*RawTx, decimal uint64) ([]*Tx, int64) {
	var sum int64 = 0
	txs := make([]*Tx, 0)
	for _, rawTx := range rawTxs {
		amount := ToIntByPrecise(rawTx.Amount, decimal)
		txs = append(txs, &Tx{
			To:     rawTx.To,
			Amount: amount.Int64(),
		})
		sum = sum + amount.Int64()
	}
	return txs, sum
}

func ToIntByPrecise(str string, decimal uint64) *big.Int {
	result := new(big.Int)
	splits := strings.Split(str, ".")
	if len(splits) == 1 { // doesn't contain "."
		var i uint64 = 0
		for ; i < decimal; i++ {
			str += "0"
		}
		intValue, ok := new(big.Int).SetString(str, 10)
		if ok {
			result.Set(intValue)
		}
	} else if len(splits) == 2 {
		value := new(big.Int)
		ok := false
		floatLen := uint64(len(splits[1]))
		if floatLen <= decimal { // add "0" at last of str
			parseString := strings.Replace(str, ".", "", 1)
			var i uint64 = 0
			for ; i < decimal-floatLen; i++ {
				parseString += "0"
			}
			value, ok = value.SetString(parseString, 10)
		} else { // remove redundant digits after "."
			splits[1] = splits[1][:decimal]
			parseString := splits[0] + splits[1]
			value, ok = value.SetString(parseString, 10)
		}
		if ok {
			result.Set(value)
		}
	}

	return result
}
