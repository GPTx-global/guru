package cli

import (
	"fmt"
	"io/ioutil"

	"github.com/GPTx-global/guru/x/cex/types"
	"github.com/cosmos/cosmos-sdk/client"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/spf13/cobra"
)

func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("%s transactions subcommands", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(NewSwapTxCmd())
	cmd.AddCommand(NewRegisterAdminTxCmd())
	cmd.AddCommand(NewRemoveAdminTxCmd())
	cmd.AddCommand(NewRegisterExchangeTxCmd())
	cmd.AddCommand(NewUpdateExchangeTxCmd())
	cmd.AddCommand(NewChangeModeratorTxCmd())
	return cmd
}

func NewSwapTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "swap [exchange_id] [from_denom] [to_denom] [amount] --flags",
		Short: "Swap coins from one denom to another",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			id, ok := math.NewIntFromString(args[0])
			if !ok {
				return errorsmod.Wrapf(types.ErrInvalidExchangeId, "")
			}

			amount, err := sdk.ParseCoinNormalized(args[3])
			if err != nil {
				return err
			}

			msg := types.NewMsgSwap(clientCtx.GetFromAddress(), id, args[1], args[2], amount)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewRegisterAdminTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register_admin [admin_address] [exchane_id] --from moderator_address",
		Short: "Register a new admin or reset admin for exchange (exchange_id is optional)",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			newAdminAddr, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return err
			}

			exchangeId := math.NewInt(0)
			ok := true
			if len(args) > 1 {
				exchangeId, ok = math.NewIntFromString(args[1])
				if !ok {
					return errorsmod.Wrapf(types.ErrInvalidExchangeId, "")
				}
			}

			msg := types.NewMsgRegisterAdmin(clientCtx.GetFromAddress(), newAdminAddr, exchangeId)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewRemoveAdminTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove_admin [admin_address] --from [moderator_address]",
		Short: "remove the admin_address from admin list and all registered exchanges",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			newAdminAddr, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return err
			}

			msg := types.NewMsgRemoveAdmin(clientCtx.GetFromAddress(), newAdminAddr)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewRegisterExchangeTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register_exchange [path_to_json] --from [admin_address]",
		Short: "Register a new exchange",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			cdc := codec.NewProtoCodec(clientCtx.InterfaceRegistry)

			var exchange types.Exchange
			// if err := cdc.UnmarshalInterfaceJSON([]byte(args[0]), &exchange); err != nil {

			// 	// check for file path if JSON input is not provided
			// 	contents, err := ioutil.ReadFile(args[0])
			// 	if err != nil {
			// 		return errorsmod.Wrapf(types.ErrInvalidJsonFile, "%s", err)
			// 	}

			// 	if err := cdc.UnmarshalInterfaceJSON(contents, &exchange); err != nil {
			// 		return errorsmod.Wrapf(types.ErrInvalidJsonFile, "%s", err)
			// 	}
			// }

			if err := cdc.UnmarshalJSON([]byte(args[0]), &exchange); err != nil {
				// If that fails, treat it as a filepath
				contents, err := ioutil.ReadFile(args[0])
				if err != nil {
					return errorsmod.Wrapf(types.ErrInvalidJsonFile, "%s", err)
				}

				if err := cdc.UnmarshalJSON(contents, &exchange); err != nil {
					return errorsmod.Wrapf(types.ErrInvalidJsonFile, "%s", err)
				}
			}

			msg := types.NewMsgRegisterExchange(clientCtx.GetFromAddress(), &exchange)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewUpdateExchangeTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update_exchange [id] [key] [value] --from [admin_address]",
		Short: "Update the exchange attribute",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			id, ok := math.NewIntFromString(args[0])
			if !ok {
				return errorsmod.Wrapf(types.ErrInvalidExchangeId, "")
			}

			msg := types.NewMsgUpdateExchange(clientCtx.GetFromAddress(), id, args[1], args[2])

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewChangeModeratorTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "change_moderator [new_moderator_address] --from [moderator_address]",
		Short: "Change the moderator address for the cex module",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			newModAddr, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return err
			}

			msg := types.NewMsgChangeModerator(clientCtx.GetFromAddress(), newModAddr)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
