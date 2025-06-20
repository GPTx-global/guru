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
		GetCmdQueryAttributes(),
		GetCmdQueryExchanges(),
		GetCmdQueryIsAdmin(),
		GetCmdQueryNextExchangeId(),
		GetCmdQueryRatemeter(),
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

func GetCmdQueryAttributes() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "attributes [id] [key]",
		Short: "Query the exchange attrbite",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryAttributesRequest{Id: args[0], Key: args[1]}
			res, err := queryClient.Attributes(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func GetCmdQueryExchanges() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exchanges",
		Short: "Query the list of all exchanges",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryExchangesRequest{}

			res, err := queryClient.Exchanges(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func GetCmdQueryIsAdmin() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "is_admin [address]",
		Short: "Query if the address is an admin",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryIsAdminRequest{Address: args[0]}

			res, err := queryClient.IsAdmin(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func GetCmdQueryNextExchangeId() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "next_exchange_id",
		Short: "Query the exchange id for registering new exchange",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryNextExchangeIdRequest{}
			res, err := queryClient.NextExchangeId(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func GetCmdQueryRatemeter() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ratemeter",
		Short: "Query the ratemeter",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryRatemeterRequest{}
			res, err := queryClient.Ratemeter(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
