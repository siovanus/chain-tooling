package plan

import (
	"errors"
	"github.com/binance-chain/chain-tooling/airdrop/config"
	"github.com/binance-chain/go-sdk/client"
	"github.com/binance-chain/go-sdk/keys"
	"log"
	"time"
)

type ExecuteContext struct {
	Config *config.Conf

	KeyManager keys.KeyManager

	Sender string

	Tasks []*ExecuteTask

	StartTime    time.Time
	CompleteTime time.Time
}

func (ex *ExecuteContext) GetDexClient() (client.DexClient, error) {
	return client.NewDexClient(ex.Config.BaseUrl, ex.Config.Network, ex.KeyManager)
}

type ExecuteTask struct {
	Token          string
	Txs            []*config.Tx
	TxHash         string
	Affirmed       bool
	Exception      error
	ValidException error
}

type PlanMaker struct {
	Context *ExecuteContext
}

func (pm *PlanMaker) InitializeContext() error {

	var context = ExecuteContext{}
	context.Config = config.RawConf

	km, error := keys.NewMnemonicKeyManager(context.Config.Mnemonic)

	if error != nil {
		log.Fatal(error)
		return error
	}

	context.KeyManager = km

	context.Sender = km.GetAddr().String()

	pm.Context = &context
	return nil
}

func (pm *PlanMaker) MakeExecutePlan() error {
	var context = pm.Context

	client, error := context.GetDexClient()

	if error != nil {
		return error
	}

	account, error := client.GetAccount(context.Sender)

	if error != nil {
		return error
	}

	var balanceAmount = int64(0)
	for _, balance := range account.Balances {
		if balance.Symbol == context.Config.Token {
			balanceAmount = balance.Free.ToInt64()
		}
	}

	if balanceAmount < context.Config.Sum {
		return errors.New("Your balance is not enough for this airdrop ")
	}

	batchSize := context.Config.BatchSize
	taskCount := ((context.Config.ReceiversCount - 1) / batchSize) + 1

	context.Tasks = make([]*ExecuteTask, taskCount)

	for index, task := range context.Tasks {
		task = &ExecuteTask{}

		task.Token = context.Config.Token

		var start = index * batchSize
		var end = (index + 1) * batchSize

		if end >= context.Config.ReceiversCount {
			end = context.Config.ReceiversCount
		}

		task.Txs = context.Config.Txs[start:end]
		context.Tasks[index] = task
	}

	return nil
}
