package main

import (
	"context"
	"flag"
	"fmt"
	"math"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	_ "unsafe"

	"github.com/holiman/uint256"
	"github.com/ledgerwatch/erigon-lib/kv/memdb"
	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/core/state"
	"github.com/ledgerwatch/erigon/core/vm/runtime"
	"github.com/ledgerwatch/erigon/crypto"
	"github.com/ledgerwatch/erigon/params"
)

var result []byte
var bytecode []byte
var bytecodeStore string

func BenchmarkBytecodeExecution(b *testing.B) {
	var calldata []byte

	cfg := new(runtime.Config)
	setDefaults(cfg)

	db := memdb.New()
	b.Cleanup(db.Close)
	tx, err := db.BeginRw(context.Background())
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(tx.Rollback)

	cfg.State = state.New(state.NewDbStateReader(tx))

	// Initialize some constant calldata of 128KB, 2^17 bytes.
	// This means, if we offset between 0th and 2^16th byte, we can fetch between 0 and 2^16 bytes (64KB)
	// In consequence, we need args to memory-copying OPCODEs to be between 0 and 2^16, 2^16 fits in a PUSH3,
	// which we'll be using to generate arguments for those OPCODEs.
	calldata = []byte(strings.Repeat("{", 1<<17))

	cfg.EVMConfig.Debug = false

	_, _, errWarmUp := runtime.Execute(bytecode, calldata, cfg, 0)

	if errWarmUp != nil {
		b.Error(errWarmUp)
		return
	}

	b.ResetTimer()

	var exBytes []byte

	for i := 0; i < b.N; i++ {
		exBytes2, _, err := runtime.Execute(bytecode, calldata, cfg, 0)
		exBytes = exBytes2

		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		cfg.State.Reset()
	}

	//anti-optimization precaution: point to external variable
	result = exBytes
}

// // copied directly from github.com/ethereum/go-ethereum/core/vm/runtime/runtime.go
// // so that we skip this in measured code
// sets defaults on the config
func setDefaults(cfg *runtime.Config) {
	if cfg.ChainConfig == nil {
		cfg.ChainConfig = &params.ChainConfig{
			ChainID:               big.NewInt(1),
			HomesteadBlock:        new(big.Int),
			DAOForkBlock:          new(big.Int),
			DAOForkSupport:        false,
			TangerineWhistleBlock: new(big.Int),
			TangerineWhistleHash:  common.Hash{},
			SpuriousDragonBlock:   new(big.Int),
			ByzantiumBlock:        new(big.Int),
			ConstantinopleBlock:   new(big.Int),
			PetersburgBlock:       new(big.Int),
			IstanbulBlock:         new(big.Int),
			MuirGlacierBlock:      new(big.Int),
			BerlinBlock:           new(big.Int),
			LondonBlock:           new(big.Int),
			ArrowGlacierBlock:     new(big.Int),
		}
	}

	if cfg.Difficulty == nil {
		cfg.Difficulty = new(big.Int)
	}
	if cfg.Time == nil {
		cfg.Time = big.NewInt(time.Now().Unix())
	}
	if cfg.GasLimit == 0 {
		cfg.GasLimit = math.MaxUint64
	}
	if cfg.GasPrice == nil {
		cfg.GasPrice = new(big.Int)
	}
	if cfg.Value == nil {
		cfg.Value = new(uint256.Int)
	}
	if cfg.BlockNumber == nil {
		cfg.BlockNumber = new(big.Int)
	}
	if cfg.GetHashFn == nil {
		cfg.GetHashFn = func(n uint64) common.Hash {
			return common.BytesToHash(crypto.Keccak256([]byte(new(big.Int).SetUint64(n).String())))
		}
	}
}

func runOverheadBenchmark(sampleSize int) {
	for i := 1; i <= sampleSize; i++ {

		bytecode = common.Hex2Bytes("00" + bytecodeStore)
		rEmpty := testing.Benchmark(BenchmarkBytecodeExecution)

		bytecode = common.Hex2Bytes(bytecodeStore)
		rActual := testing.Benchmark(BenchmarkBytecodeExecution)

		outputOverheadResults(i, rEmpty, rActual)
	}
}

func outputOverheadResults(sampleId int, rEmpty testing.BenchmarkResult, rActual testing.BenchmarkResult) {
	overheadTime := rEmpty.NsPerOp()
	var executionLoopTime int64 = 0
	var totalTime int64 = rEmpty.NsPerOp()

	if rActual.NsPerOp() > rEmpty.NsPerOp() {
		executionLoopTime = rActual.NsPerOp() - rEmpty.NsPerOp()
		totalTime = rActual.NsPerOp()
	}

	fmt.Printf("%v,%v,%v,%v,%v,%v,%v\n", sampleId, rActual.N, overheadTime, executionLoopTime, totalTime, rActual.AllocsPerOp(), rActual.AllocedBytesPerOp())
}

func main() {
	bytecodePtr := flag.String("bytecode", "", "EVM bytecode to execute and measure")
	sampleSizePtr := flag.Int("sampleSize", 1, "Size of the sample - number of measured repetitions of execution")

	flag.Parse()

	bytecodeStore = *bytecodePtr
	sampleSize := *sampleSizePtr

	runOverheadBenchmark(sampleSize)
}
