package execute

import (
	"fmt"
	"github.com/binance-chain/chain-tooling/airdrop/plan"
	"github.com/binance-chain/go-sdk/common/types"
	"github.com/binance-chain/go-sdk/types/msg"
	"log"
	"time"
)

type Executor struct {
	context *plan.ExecuteContext
}

func NewExecutor(context *plan.ExecuteContext) Executor {
	return Executor{context: context}
}

func (ex *Executor) Execute() error {

	var context = ex.context
	context.StartTime = time.Now()

	client, err := ex.context.GetDexClient()

	if err != nil {
		return err
	}

	for _, task := range context.Tasks {
		time.Sleep(time.Duration(context.Config.BatchInterval) * time.Second)
		txs := task.Txs
		var transfers = make([]msg.Transfer, len(txs))

		for index, tx := range txs {
			receiverAddr, err := types.AccAddressFromBech32(tx.To)
			if err != nil {
				task.Exception = err
				break
			}
			transfers[index].ToAddr = receiverAddr
			transfers[index].Coins = types.Coins{types.Coin{task.Token, tx.Amount}}
		}

		if task.Exception != nil {
			continue
		}

		result, err := client.SendToken(transfers, true)

		if err == nil {
			task.TxHash = result.Hash
			log.Println(fmt.Sprintf("Complete with tx %s", result.Hash))
		} else {
			task.Exception = err
			log.Println(fmt.Sprintf("Failed with exception %s", err.Error()))
		}
	}
	context.CompleteTime = time.Now()

	return nil
}

func (ex *Executor) Validate() error {

	var context = ex.context
	client, err := context.GetDexClient()

	if err != nil {
		return err
	}

	for _, task := range context.Tasks {
		time.Sleep(1 * time.Second)
		if len(task.TxHash) > 0 {
			txResult, error := client.GetTx(task.TxHash)

			if error != nil {
				task.ValidException = error
			}

			if txResult != nil && len(txResult.Hash) > 0 {
				task.Affirmed = true
			}

		}
	}

	return nil
}
