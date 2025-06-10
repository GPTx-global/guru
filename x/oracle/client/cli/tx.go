package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/spf13/cobra"

	"github.com/GPTx-global/guru/x/oracle/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
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
		NewUpdateModeratorAddressCmd(),
		NewUpdatePredefinedOracleCmd(),
		NewDeletePredefinedOracleCmd(),
	)

	return cmd
}

// NewRegisterOracleRequestDocCmd implements the register oracle request document command
func NewRegisterOracleRequestDocCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-request [path/to/request-doc.json]",
		Short: "Register a new oracle request document",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			requestDoc, err := parseRequestDocJson(args[0])
			if err != nil {
				return err
			}

			msg := types.NewMsgRegisterOracleRequestDoc(
				clientCtx.GetFromAddress().String(),
				*requestDoc,
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
		Use:   "update-request [path/to/request-doc.json] [reason]",
		Short: "Update an existing oracle request document",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			requestDoc, err := parseRequestDocJson(args[0])
			if err != nil {
				return err
			}

			reason := args[1]

			msg := types.NewMsgUpdateOracleRequestDoc(
				clientCtx.GetFromAddress().String(),
				*requestDoc,
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
		Use:   "submit-data [request-id] [nonce] [raw-data]",
		Short: "Submit oracle data for a request",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			requestId := args[0]
			nonce := args[1]
			rawData := args[2]

			requestIdUint64, err := strconv.ParseUint(requestId, 10, 64)
			if err != nil {
				return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "request id is not a valid uint64")
			}

			nonceUint64, err := strconv.ParseUint(nonce, 10, 64)
			if err != nil {
				return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "nonce is not a valid uint64")
			}

			msg := types.NewMsgSubmitOracleData(
				requestIdUint64,
				nonceUint64,
				rawData,
				clientCtx.GetFromAddress().String(),
				"NOT USED", // signature will be added by the client
				clientCtx.GetFromAddress().String(),
			)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewUpdateModeratorAddressCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-moderator-address [moderator-address]",
		Short: "Update the moderator address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgUpdateModeratorAddress(
				clientCtx.GetFromAddress().String(),
				args[0],
			)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewUpdatePredefinedOracleCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "update-predefined-oracle request-id name predefined-oracle-type",
		Short: "Update a predefined oracle",
		// Long:  strings.TrimSpace(fmt.Sprintf(`Update a predefined oracle

		// `, version.AppName)),

		Example: strings.TrimSpace(fmt.Sprintf(`$ %s tx oracle update-predefined-oracle 1 "Min Gas Price" 1
$ %s tx oracle update-predefined-oracle 1 "Currency KRW" 2
$ %s tx oracle update-predefined-oracle 1 "Currency USD" 3
$ %s tx oracle update-predefined-oracle 1 "Currency EUR" 4
$ %s tx oracle update-predefined-oracle 1 "Currency JPY" 5
		`, version.AppName, version.AppName, version.AppName, version.AppName, version.AppName)),
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			requestId := args[0]
			name := args[1]
			predefinedOracleType := args[2]

			requestIdUint64, err := strconv.ParseUint(requestId, 10, 64)
			if err != nil {
				return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "request id is not a valid uint64")
			}

			predefinedOracleTypeUint32, err := strconv.ParseUint(predefinedOracleType, 10, 32)
			if err != nil {
				return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "predefined oracle type is not a valid uint32")
			}

			msg := types.NewMsgUpdatePredefinedOracle(
				clientCtx.GetFromAddress().String(),
				types.PredefinedOracle{
					RequestId: requestIdUint64,
					Name:      name,
					Type:      types.PredefinedOracleType(predefinedOracleTypeUint32),
				},
			)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewDeletePredefinedOracleCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "delete-predefined-oracle [predefined-oracle-type]",
		Short: "Delete a predefined oracle",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgDeletePredefinedOracle(
				clientCtx.GetFromAddress().String(),
				types.PredefinedOracleType(types.PredefinedOracleType_value[args[0]]),
			)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func parseRequestDocJson(path string) (*types.OracleRequestDoc, error) {
	var doc types.OracleRequestDoc

	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(contents, &doc)
	if err != nil {
		return nil, err
	}

	return &doc, nil
}
