package e2e

import (
	authcmd "github.com/GPTx-global/guru/tests/e2e/cmd/auth"
	authzcmd "github.com/GPTx-global/guru/tests/e2e/cmd/authz"
	bankcmd "github.com/GPTx-global/guru/tests/e2e/cmd/bank"
	capabilitycmd "github.com/GPTx-global/guru/tests/e2e/cmd/capability"
	crisiscmd "github.com/GPTx-global/guru/tests/e2e/cmd/crisis"
	distrcmd "github.com/GPTx-global/guru/tests/e2e/cmd/distribution"
	epochscmd "github.com/GPTx-global/guru/tests/e2e/cmd/epochs"
	erc20cmd "github.com/GPTx-global/guru/tests/e2e/cmd/erc20"
	evidencecmd "github.com/GPTx-global/guru/tests/e2e/cmd/evidence"
	evmcmd "github.com/GPTx-global/guru/tests/e2e/cmd/evm"
	feegrantcmd "github.com/GPTx-global/guru/tests/e2e/cmd/feegrant"
	feemarketcmd "github.com/GPTx-global/guru/tests/e2e/cmd/feemarket"
	genutilcmd "github.com/GPTx-global/guru/tests/e2e/cmd/genutil"
	govcmd "github.com/GPTx-global/guru/tests/e2e/cmd/gov"
	ibccmd "github.com/GPTx-global/guru/tests/e2e/cmd/ibc"
	icacmd "github.com/GPTx-global/guru/tests/e2e/cmd/ica"
	inflationcmd "github.com/GPTx-global/guru/tests/e2e/cmd/inflation"
	paramscmd "github.com/GPTx-global/guru/tests/e2e/cmd/params"
	slashingcmd "github.com/GPTx-global/guru/tests/e2e/cmd/slashing"
	stakingcmd "github.com/GPTx-global/guru/tests/e2e/cmd/staking"
	transfercmd "github.com/GPTx-global/guru/tests/e2e/cmd/transfer"
	upgradecmd "github.com/GPTx-global/guru/tests/e2e/cmd/upgrade"
	vestingcmd "github.com/GPTx-global/guru/tests/e2e/cmd/vesting"
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
