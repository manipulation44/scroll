package sender

import (
	"math/big"
	"sync/atomic"

	"github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/log"
)

func (s *Sender) estimateLegacyGas(auth *bind.TransactOpts, contract *common.Address, value *big.Int, input []byte, fallbackGasLimit uint64) (*FeeData, error) {
	gasPrice, err := s.client.SuggestGasPrice(s.ctx)
	if err != nil {
		log.Error("estimateLegacyGas SuggestGasPrice failure", "error", err)
		return nil, err
	}
	gasLimit, err := s.estimateGasLimit(auth, contract, input, gasPrice, nil, nil, value)
	if err != nil {
		log.Error("estimateLegacyGas estimateGasLimit failure", "gas price", gasPrice, "from", auth.From.Hex(),
			"nonce", auth.Nonce.Uint64(), "contract address", contract.Hex(), "fallback gas limit", fallbackGasLimit, "error", err)
		if fallbackGasLimit == 0 {
			return nil, err
		}
		gasLimit = fallbackGasLimit
	}
	return &FeeData{
		gasPrice: gasPrice,
		gasLimit: gasLimit,
	}, nil
}

func (s *Sender) estimateDynamicGas(auth *bind.TransactOpts, contract *common.Address, value *big.Int, input []byte, fallbackGasLimit uint64) (*FeeData, error) {
	gasTipCap, err := s.client.SuggestGasTipCap(s.ctx)
	if err != nil {
		log.Error("estimateDynamicGas SuggestGasTipCap failure", "error", err)
		return nil, err
	}

	baseFee := big.NewInt(0)
	if feeGas := atomic.LoadUint64(&s.baseFeePerGas); feeGas != 0 {
		baseFee.SetUint64(feeGas)
	}
	gasFeeCap := new(big.Int).Add(
		gasTipCap,
		new(big.Int).Mul(baseFee, big.NewInt(2)),
	)
	gasLimit, err := s.estimateGasLimit(auth, contract, input, nil, gasTipCap, gasFeeCap, value)
	if err != nil {
		log.Error("estimateDynamicGas estimateGasLimit failure",
			"from", auth.From.Hex(), "nonce", auth.Nonce.Uint64(), "contract address", contract.Hex(),
			"fallback gas limit", fallbackGasLimit, "error", err)
		if fallbackGasLimit == 0 {
			return nil, err
		}
		gasLimit = fallbackGasLimit
	}
	return &FeeData{
		gasLimit:  gasLimit,
		gasTipCap: gasTipCap,
		gasFeeCap: gasFeeCap,
	}, nil
}

func (s *Sender) estimateGasLimit(opts *bind.TransactOpts, contract *common.Address, input []byte, gasPrice, gasTipCap, gasFeeCap, value *big.Int) (uint64, error) {
	msg := ethereum.CallMsg{
		From:      opts.From,
		To:        contract,
		GasPrice:  gasPrice,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Value:     value,
		Data:      input,
	}
	gasLimit, err := s.client.EstimateGas(s.ctx, msg)
	if err != nil {
		log.Error("estimateGasLimit EstimateGas failure", "error", err)
		return 0, err
	}

	gasLimit = gasLimit * 12 / 10 // 20% extra gas to avoid out of gas error

	return gasLimit, nil
}
