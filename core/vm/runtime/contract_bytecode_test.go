package runtime

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/ledgerwatch/erigon-lib/kv/memdb"
	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/core/state"
)

func TestBytecode(t *testing.T) {
	cfg := new(Config)
	setDefaults(cfg)

	cfg.ChainConfig.ShardingTime = new(big.Int)

	_, tx := memdb.NewTestTx(t)
	cfg.State = state.New(state.NewDbStateReader(tx))

	// Initialize some constant calldata of 32KB, 2^15 bytes.
	// This means, if we offset between 0th and 2^14th byte, we can fetch between 0 and 2^14 bytes (16KB)
	// In consequence, we need args to memory-copying OPCODEs to be between 0 and 2^14, 2^14 fits in a PUSH2,
	// which we'll be using to generate arguments for those OPCODEs.
	calldata := []byte(strings.Repeat("{", 1<<15))

	// ret, _, err := Execute(common.Hex2Bytes("6060604052600a8060106000396000f360606040526008565b00"), calldata, cfg, 0)
	ret, _, err := Execute(common.Hex2Bytes("7f013c03613f6fc558fb7e61e75602241ed9a2f04e36d8670aadd286e71b5ca9cc610000527f4200000000000000000000000000000000000000000000000000000000000000610020527f31e5a2356cbc2ef6a733eae8d54bf48719ae3d990017ca787c419c7d369f8e3c610040527f83fac17c3f237fc51f90e2c660eb202a438bc2025baded5cd193c1a018c5885b610060527fc9281ba704d5566082e851235c7be763b2a99adff965e0a121ee972ebc472d02610080527f944a74f5c6243e14052e105124b70bf65faf85ad3a494325e269fad097842cba6100a0526020600060c06000600060145af15000"), calldata, cfg, 0)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(ret)
	// Output:
	// [96 96 96 64 82 96 8 86 91 0]
}

// bytecodePrecompiledTest defines the input/output pairs for precompiled contract tests.
type bytecodePrecompiledTest struct {
	Bytecode, ExpectedResult string
	Type                     string
	Name                     string
	NominalGasCost           uint64
}

func loadJson(name string) ([]bytecodePrecompiledTest, error) {
	data, err := os.ReadFile(fmt.Sprintf("testdata/%v.json", name))
	if err != nil {
		return nil, err
	}
	var testcases []bytecodePrecompiledTest
	err = json.Unmarshal(data, &testcases)
	return testcases, err
}

func benchmarkPrecompiled(test bytecodePrecompiledTest, bench *testing.B) {
	bench.Run(fmt.Sprintf("%s/%s", test.Type, test.Name), func(bench *testing.B) {
		bench.ReportAllocs()
		cfg := new(Config)
		setDefaults(cfg)

		setDefaults(cfg)

		cfg.ChainConfig.ShardingTime = new(big.Int)

		_, tx := memdb.NewTestTx(bench)
		cfg.State = state.New(state.NewDbStateReader(tx))

		// Initialize some constant calldata of 32KB, 2^15 bytes.
		// This means, if we offset between 0th and 2^14th byte, we can fetch between 0 and 2^14 bytes (16KB)
		// In consequence, we need args to memory-copying OPCODEs to be between 0 and 2^14, 2^14 fits in a PUSH2,
		// which we'll be using to generate arguments for those OPCODEs.
		calldata := []byte(strings.Repeat("{", 1<<15))
		bench.ResetTimer()
		for i := 0; i < bench.N; i++ {
			bench.StopTimer()
			cfg.State.Reset()
			bench.StartTimer()

			_, _, err := Execute(common.Hex2Bytes(test.Bytecode), calldata, cfg, 0)

			if err != nil {
				fmt.Println(err)
			}
		}
		bench.ReportMetric(float64(test.NominalGasCost), "nominalGas/op")
	})
}

func BenchmarkBytecodePrecompile(b *testing.B) {

	tests, err := loadJson("bytecodePrecompiles")
	if err != nil {
		b.Fatal(err)
	}
	for _, test := range tests {
		benchmarkPrecompiled(test, b)
	}
}
