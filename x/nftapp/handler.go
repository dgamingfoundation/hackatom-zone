package nftapp

import (
	"fmt"
	"github.com/cosmos/cosmos-sdk/x/ibc"
	"github.com/cosmos/cosmos-sdk/x/ibc/02-client/tendermint"
	"sync"

	"github.com/dgamingfoundation/hackatom-zone/x/nftapp/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	uuid "github.com/satori/go.uuid"
)

// NewHandler returns a handler for "nft" type messages.
func NewHandler(keeper Keeper, ibcKeeper *ibc.Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		switch msg := msg.(type) {
		case types.MsgCreateNFT:
			return handleMsgCreateNFT(ctx, keeper, msg)
		case types.MsgTransferTokenToHub:
			return handleMsgTransferTokenToHub(ctx, keeper, ibcKeeper, msg)
		default:
			errMsg := fmt.Sprintf("Unrecognized nftapp Msg type: %v", msg.Type())
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

// Handle a message to create nft
func handleMsgCreateNFT(ctx sdk.Context, keeper Keeper, msg types.MsgCreateNFT) sdk.Result {
	id := uuid.NewV4()
	nft := types.NewBaseNFT(id.String(), msg.Owner, msg.TokenURI)
	keeper.CreateNFT(ctx, nft)
	return sdk.Result{}
}

var once sync.Once

// Handle a message to create nft
func handleMsgTransferTokenToHub(
	ctx sdk.Context,
	keeper Keeper,
	ibcKeeper *ibc.Keeper,
	msg types.MsgTransferTokenToHub,
) sdk.Result {
	token, err := keeper.GetNFT(ctx, msg.TokenID)
	if err != nil {
		return sdk.Result{Code: sdk.CodeUnknownRequest, Log: err.Error()}
	}

	var (
		result sdk.Result
	)
	once.Do(func() {
		_, err := ibcKeeper.Client().Create(ctx, types.ClientID, tendermint.ConsensusState{
			ChainID: "NFTChain",
		})
		if err != nil {
			result = sdk.Result{Code: sdk.CodeUnknownRequest, Log: err.Error()}
			return
		}

		//err = ibcKeeper.OpenConnection(ctx, types.ConnectionID, types.CounterpartyID, types.ClientID, types.CounterpartyClientID)
		//if err != nil {
		//	result = sdk.Result{Code: sdk.CodeUnknownRequest, Log: err.Error()}
		//	return
		//}

		//err = ibcKeeper.OpenChannel(ctx, types.ZoneModule, types.ConnectionID, types.ChannelID, types.CounterpartyID, types.HubModule)
		//if err != nil {
		//	result = sdk.Result{Code: sdk.CodeUnknownRequest, Log: err.Error()}
		//	return
		//}

		_, err = ibcKeeper.Connection().Query(ctx, types.ConnectionID)
		if err != nil {
			result = sdk.Result{Code: sdk.CodeUnknownRequest, Log: err.Error()}
			return
		}

		obj, err := ibcKeeper.Channel().Query(ctx, types.ConnectionID, types.ChannelID)
		if err != nil {
			result = sdk.Result{Code: sdk.CodeUnknownRequest, Log: err.Error()}
			return
		}
		fmt.Println("Seqsend", obj.SeqSend(ctx))

	})

	if !result.IsOK() {
		return result
	}

	packet := types.NewSellTokenPacket(token, msg.Price)
	if err := ibcKeeper.Channel().Send(ctx, types.ConnectionID, types.ChannelID, packet); err != nil {
		fmt.Println(">>>", err)
		return sdk.Result{Code: sdk.CodeUnknownRequest, Log: err.Error()}
	}

	obj, err := ibcKeeper.Channel().Query(ctx, types.ConnectionID, types.ChannelID)
	if err != nil {
		result = sdk.Result{Code: sdk.CodeUnknownRequest, Log: err.Error()}
		return result
	}
	fmt.Println("Seqsend", obj.SeqSend(ctx))

	if err := keeper.DeleteNFT(ctx, msg.Owner, msg.TokenID); err != nil {
		return sdk.ErrUnauthorized(fmt.Sprintf("failed to delete token: %v", err.Error())).Result()
	}

	return sdk.Result{}
}
