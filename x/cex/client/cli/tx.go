package cli

import (
	"fmt"

	"github.com/GPTx-global/guru/x/cex/types"
	"github.com/cosmos/cosmos-sdk/client"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
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
	cmd.AddCommand(NewRegisterReserveAccountTxCmd())
	cmd.AddCommand(NewRegisterAdminTxCmd())
	cmd.AddCommand(NewUpdateRateTxCmd())
	cmd.AddCommand(NewChangeModeratorTxCmd())
	return cmd
}

func NewSwapTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "swap FROM_DENOM FROM_CHANNEL TO_DENOM TO_CHANNEL AMOUNT --flags",
		Short: "Swap coins from one denom to another",
		Args:  cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) error {

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			amount, err := sdk.ParseCoinNormalized(args[4])
			if err != nil {
				return err
			}

			msg := types.NewMsgSwap(clientCtx.GetFromAddress(), args[0], args[1], args[2], args[3], amount)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewRegisterReserveAccountTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register_reserve NEW_RESERVE_ACCOUNT --from MODERATOR_ADDRESS",
		Short: "Register new reserve account address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			newReserveAcc, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return err
			}

			msg := types.NewMsgRegisterReserveAccount(clientCtx.GetFromAddress(), newReserveAcc)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewRegisterAdminTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register_admin NEW_ADMIN PAIR_DENOM --from MODERATOR_ADDRESS",
		Short: "Register new admin for the existing coin pair",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			newAdminAddr, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return err
			}

			msg := types.NewMsgRegisterAdmin(clientCtx.GetFromAddress(), newAdminAddr, args[1])

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewUpdateRateTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update_rate PAIR_DENOM NEW_RATE --from ADMIN_ADDRESS",
		Short: "Update the currency rate for the existing coin pair",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgUpdateRate(clientCtx.GetFromAddress(), args[0], sdk.MustNewDecFromStr(args[1]))

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewChangeModeratorTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "change_moderator NEW_MODERATOR_ADDRESS --from MODERATOR_ADDRESS",
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
