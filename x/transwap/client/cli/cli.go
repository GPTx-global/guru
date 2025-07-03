package cli

import (
	// "github.com/cosmos/cosmos-sdk/client"
	"github.com/spf13/cobra"
)

// GetQueryCmd returns the query commands for IBC connections
func GetQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        "ibc-exchange",
		Short:                      "IBC fungible token exchange query subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
	}

	queryCmd.AddCommand(
		GetCmdQueryDenomTrace(),
		GetCmdQueryDenomTraces(),
		GetCmdParams(),
		GetCmdQueryEscrowAddress(),
		GetCmdQueryDenomHash(),
	)

	return queryCmd
}
