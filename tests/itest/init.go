package itest

import (
	authcmd "github.com/GPTx-global/guru/tests/itest/testcases/auth"
	authzcmd "github.com/GPTx-global/guru/tests/itest/testcases/authz"
	bankcmd "github.com/GPTx-global/guru/tests/itest/testcases/bank"
	capabilitycmd "github.com/GPTx-global/guru/tests/itest/testcases/capability"
	crisiscmd "github.com/GPTx-global/guru/tests/itest/testcases/crisis"
	distrcmd "github.com/GPTx-global/guru/tests/itest/testcases/distribution"
	epochscmd "github.com/GPTx-global/guru/tests/itest/testcases/epochs"
	erc20cmd "github.com/GPTx-global/guru/tests/itest/testcases/erc20"
	evidencecmd "github.com/GPTx-global/guru/tests/itest/testcases/evidence"
	evmcmd "github.com/GPTx-global/guru/tests/itest/testcases/evm"
	feegrantcmd "github.com/GPTx-global/guru/tests/itest/testcases/feegrant"
	feemarketcmd "github.com/GPTx-global/guru/tests/itest/testcases/feemarket"
	genutilcmd "github.com/GPTx-global/guru/tests/itest/testcases/genutil"
	govcmd "github.com/GPTx-global/guru/tests/itest/testcases/gov"
	ibccmd "github.com/GPTx-global/guru/tests/itest/testcases/ibc"
	icacmd "github.com/GPTx-global/guru/tests/itest/testcases/ica"
	inflationcmd "github.com/GPTx-global/guru/tests/itest/testcases/inflation"
	paramscmd "github.com/GPTx-global/guru/tests/itest/testcases/params"
	slashingcmd "github.com/GPTx-global/guru/tests/itest/testcases/slashing"
	stakingcmd "github.com/GPTx-global/guru/tests/itest/testcases/staking"
	transfercmd "github.com/GPTx-global/guru/tests/itest/testcases/transfer"
	upgradecmd "github.com/GPTx-global/guru/tests/itest/testcases/upgrade"
	vestingcmd "github.com/GPTx-global/guru/tests/itest/testcases/vesting"
)

func init() {
	authcmd.Init()
	authzcmd.Init()
	bankcmd.Init()
	capabilitycmd.Init()
	crisiscmd.Init()
	distrcmd.Init()
	epochscmd.Init()
	erc20cmd.Init()
	evidencecmd.Init()
	evmcmd.Init()
	feegrantcmd.Init()
	feemarketcmd.Init()
	genutilcmd.Init()
	govcmd.Init()
	ibccmd.Init()
	icacmd.Init()
	inflationcmd.Init()
	paramscmd.Init()
	slashingcmd.Init()
	stakingcmd.Init()
	transfercmd.Init()
	upgradecmd.Init()
	vestingcmd.Init()
}
