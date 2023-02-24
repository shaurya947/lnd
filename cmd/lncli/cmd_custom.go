package main

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lnwire"
	"github.com/urfave/cli"
)

var sendCustomCommand = cli.Command{
	Name: "sendcustom",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name: "peer",
		},
		cli.Uint64Flag{
			Name: "type",
		},
		cli.StringFlag{
			Name: "data",
		},
	},
	Action: actionDecorator(sendCustom),
}

func sendCustom(ctx *cli.Context) error {
	ctxc := getContext()
	client, cleanUp := getClient(ctx)
	defer cleanUp()

	peer, err := hex.DecodeString(ctx.String("peer"))
	if err != nil {
		return err
	}

	msgType := ctx.Uint64("type")

	data, err := hex.DecodeString(ctx.String("data"))
	if err != nil {
		return err
	}

	_, err = client.SendCustomMessage(
		ctxc,
		&lnrpc.SendCustomMessageRequest{
			Peer: peer,
			Type: uint32(msgType),
			Data: data,
		},
	)

	return err
}

var sendChannelErrorCommand = cli.Command{
	Name: "sendchannelerror",
	Description: "Send an error to the remote peer on a specific channel " +
		"to initiate a remote force close",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name: "peer",
		},
		cli.StringFlag{
			Name: "chan_point",
		},
	},
	Action: actionDecorator(sendChannelError),
}

func sendChannelError(ctx *cli.Context) error {
	ctxc := getContext()
	client, cleanUp := getClient(ctx)
	defer cleanUp()

	peer, err := hex.DecodeString(ctx.String("peer"))
	if err != nil {
		return err
	}

	outPoint, err := parseOutPoint(ctx.String("chan_point"))
	if err != nil {
		return err
	}
	channelID := lnwire.NewChanIDFromOutPoint(outPoint)

	// Channel ID (32 byte) + u16 for the data length (which will be 0).
	data := make([]byte, 34)
	copy(data[:32], channelID[:])

	_, err = client.SendCustomMessage(ctxc, &lnrpc.SendCustomMessageRequest{
		Peer: peer,
		Type: uint32(lnwire.MsgError),
		Data: data,
	})

	return err
}

var subscribeCustomCommand = cli.Command{
	Name:   "subscribecustom",
	Action: actionDecorator(subscribeCustom),
}

func subscribeCustom(ctx *cli.Context) error {
	ctxc := getContext()
	client, cleanUp := getClient(ctx)
	defer cleanUp()

	stream, err := client.SubscribeCustomMessages(
		ctxc,
		&lnrpc.SubscribeCustomMessagesRequest{},
	)
	if err != nil {
		return err
	}

	for {
		msg, err := stream.Recv()
		if err != nil {
			return err
		}

		fmt.Printf("Received from peer %x: type=%d, data=%x\n",
			msg.Peer, msg.Type, msg.Data)
	}
}

func parseOutPoint(s string) (*wire.OutPoint, error) {
	split := strings.Split(s, ":")
	if len(split) != 2 || len(split[0]) == 0 || len(split[1]) == 0 {
		return nil, errBadChanPoint
	}

	index, err := strconv.ParseInt(split[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("unable to decode output index: %v", err)
	}

	txid, err := chainhash.NewHashFromStr(split[0])
	if err != nil {
		return nil, fmt.Errorf("unable to parse hex string: %v", err)
	}

	return &wire.OutPoint{
		Hash:  *txid,
		Index: uint32(index),
	}, nil
}
