package cli

import (
	"github.com/GPTx-global/guru/x/cex/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
)

// GetQueryCmd returns the cli query commands for the xmsquare module.
func GetQueryCmd() *cobra.Command {
	cexQueryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the xmsquare module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cexQueryCmd.AddCommand(
		GetCmdQueryModeratorAddress(),
		GetCmdQueryReserveAccount(),
		GetCmdQueryReserve(),
		GetCmdQueryRate(),
		GetCmdQueryAdmin(),
	)

	return cexQueryCmd
}

func GetCmdQueryModeratorAddress() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "moderator_address",
		Short: "Query the current moderator address",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryModeratorAddressRequest{}
			res, err := queryClient.ModeratorAddress(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func GetCmdQueryReserveAccount() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reserve_account",
		Short: "Query the current reserve account address",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryReserveAccountRequest{}
			res, err := queryClient.ReserveAccount(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func GetCmdQueryReserve() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reserve [PAIR_DENOM]",
		Short: "Query the current reserve for one coin pair or all",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryReserveRequest{}
			if len(args) == 1 {
				req.Denom = args[0]
			}
			res, err := queryClient.Reserve(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func GetCmdQueryRate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rate [PAIR_DENOM]",
		Short: "Query the rate by the pair denom",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryRateRequest{PairDenom: args[0]}
			res, err := queryClient.Rate(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func GetCmdQueryAdmin() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "admin [PAIR_DENOM]",
		Short: "Query the admin address by the pair denom",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryAdminRequest{PairDenom: args[0]}
			res, err := queryClient.Admin(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
