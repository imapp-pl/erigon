package main

import (
	"context"
	"flag"
	"fmt"
	"math"
	"math/big"
	"os"
	go_runtime "runtime"
	"strings"
	"time"

	_ "unsafe"

	"github.com/holiman/uint256"
	"github.com/ledgerwatch/erigon-lib/kv/memdb"
	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/core/state"
	"github.com/ledgerwatch/erigon/core/vm"
	"github.com/ledgerwatch/erigon/core/vm/runtime"
	"github.com/ledgerwatch/erigon/crypto"
	"github.com/ledgerwatch/erigon/params"
)

var calldata []byte

func main() {

	bytecodePtr := flag.String("bytecode", "", "EVM bytecode to execute and measure")
	sampleSizePtr := flag.Int("sampleSize", 10, "Size of the sample - number of measured repetitions of execution")
	printEachPtr := flag.Bool("printEach", true, "If false, printing of each execution time is skipped")
	printCSVPtr := flag.Bool("printCSV", false, "If true, will print a CSV with standard results to STDOUT")
	modePtr := flag.String("mode", "total", "Measurement mode. Available options: all, total, trace")

	flag.Parse()

	bytecode := common.Hex2Bytes(*bytecodePtr)
	sampleSize := *sampleSizePtr
	printEach := *printEachPtr
	printCSV := *printCSVPtr
	mode := *modePtr

	if mode != "total" {
		fmt.Fprintln(os.Stderr, "Invalid measurement mode: ", mode)
		os.Exit(1)
	}

	fmt.Fprintln(os.Stderr, "Config: ", *bytecodePtr)
	cfg := new(runtime.Config)
	setDefaults(cfg)

	fmt.Fprintln(os.Stderr, "db")
	db := memdb.New()
	tx, err := db.BeginRw(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
	cfg.State = state.New(state.NewDbStateReader(tx))

	// Initialize some constant calldata of 128KB, 2^17 bytes.
	// This means, if we offset between 0th and 2^16th byte, we can fetch between 0 and 2^16 bytes (64KB)
	// In consequence, we need args to memory-copying OPCODEs to be between 0 and 2^16, 2^16 fits in a PUSH3,
	// which we'll be using to generate arguments for those OPCODEs.
	calldata = []byte(strings.Repeat("{", 1<<17))

	cfg.EVMConfig.Debug = false
	cfg.EVMConfig.Instrumenter = vm.NewInstrumenterLogger()
	fmt.Fprintln(os.Stderr, "warmup")
	retWarmUp, _, errWarmUp := runtime.Execute(bytecode, calldata, cfg, 0)
	// End warm-up

	fmt.Fprintln(os.Stderr, "loop count: ", sampleSize)

	sampleStart := time.Now()
	for i := 0; i < sampleSize; i++ {
		if mode == "total" {
			MeasureTotal(cfg, bytecode, printEach, printCSV, i)
		}
	}

	fmt.Fprintln(os.Stderr, "Done")

	sampleDuration := time.Since(sampleStart)

	if errWarmUp != nil {
		fmt.Fprintln(os.Stderr, errWarmUp)
	}
	fmt.Fprintln(os.Stderr, "Program: ", *bytecodePtr)
	fmt.Fprintln(os.Stderr, "Return:", retWarmUp)
	fmt.Fprintln(os.Stderr, "Sample duration:", sampleDuration)
}

func MeasureTotal(cfg *runtime.Config, bytecode []byte, printEach bool, printCSV bool, sampleId int) {
	cfg.EVMConfig.Instrumenter = vm.NewInstrumenterLogger()
	go_runtime.GC()

	_, _, err := runtime.Execute(bytecode, calldata, cfg, 0)

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	if printCSV {
		vm.WriteCSVInstrumentationTotal(os.Stdout, cfg.EVMConfig.Instrumenter, sampleId)
	}
}

// copied directly from github.com/ledgerwatch/erigon/core/vm/runtime/runtime.go
// so that we skip this in measured code
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

// runtimeNano returns the current value of the runtime clock in nanoseconds.
//go:linkname runtimeNano runtime.nanotime
func runtimeNano() int64
