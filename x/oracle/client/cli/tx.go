package cli

import (
	"fmt"
	"strconv"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	gurutypes "github.com/GPTx-global/guru/types"
	"github.com/GPTx-global/guru/x/oracle/types"
)

// GetTxCmd returns the transaction commands for this module
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("%s transactions subcommands", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		NewRegisterOracleRequestDocCmd(),
		NewUpdateOracleRequestDocCmd(),
		NewSubmitOracleDataCmd(),
	)

	return cmd
}

// NewRegisterOracleRequestDocCmd implements the register oracle request document command
func NewRegisterOracleRequestDocCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-request [request-doc] [fee]",
		Short: "Register a new oracle request document",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			var requestDoc types.RequestOracleDoc

			// requestDoc := args[0]
			// fee, err := types.ParseCoin(args[1])
			fee := gurutypes.NewGuruCoin(math.NewInt(630000))
			if err != nil {
				return err
			}

			msg := types.NewMsgRegisterOracleRequestDoc(
				requestDoc,
				fee,
				clientCtx.GetFromAddress().String(),
				"", // signature will be added by the client
			)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewUpdateOracleRequestDocCmd implements the update oracle request document command
func NewUpdateOracleRequestDocCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-request [request-id] [request-doc] [reason]",
		Short: "Update an existing oracle request document",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			requestId := args[0]
			requestIdUint64, err := strconv.ParseUint(requestId, 10, 64)
			if err != nil {
				return fmt.Errorf("failed to parse request ID: %w", err)
			}

			var requestDoc types.RequestOracleDoc
			if err := clientCtx.Codec.UnmarshalJSON([]byte(args[1]), &requestDoc); err != nil {
				return fmt.Errorf("failed to parse request document: %w", err)
			}
			reason := args[2]

			msg := types.NewMsgUpdateOracleRequestDoc(
				requestIdUint64,
				requestDoc,
				clientCtx.GetFromAddress().String(),
				"", // signature will be added by the client
				reason,
			)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewSubmitOracleDataCmd implements the submit oracle data command
func NewSubmitOracleDataCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-data [request-id] [raw-data]",
		Short: "Submit oracle data for a request",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			requestId := args[0]
			rawData := args[1]

			msg := types.NewMsgSubmitOracleData(
				requestId,
				rawData,
				clientCtx.GetFromAddress().String(),
				"", // signature will be added by the client
			)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
