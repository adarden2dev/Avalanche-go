// (c) 2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package platformvm

import (
	"fmt"
	"time"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/engine/common"
	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/avalanchego/utils/logging"
	"github.com/ava-labs/avalanchego/vms/platformvm/message"
)

type network struct {
	log logging.Logger
	// gossip related attributes
	gossipActivationTime time.Time
	appSender            common.AppSender
	mempool              *blockBuilder
	vm                   *VM

	requestID uint32
	// requestMaps allow checking that solicited content matched the requested one
	// They are populated upon sending an AppRequest and cleaned up upon AppResponse (also failed ones)
	requestsContent map[ /*requestID*/ uint32]ids.ID

	requestHandler  message.Handler
	responseHandler message.Handler
	gossipHandler   message.Handler
}

func newNetwork(activationTime time.Time, appSender common.AppSender, vm *VM) *network {
	n := &network{
		log:                  vm.ctx.Log,
		gossipActivationTime: activationTime,
		appSender:            appSender,
		mempool:              &vm.blockBuilder,
		vm:                   vm,
		requestsContent:      make(map[uint32]ids.ID),
	}
	n.requestHandler = &RequestHandler{
		NoopHandler: message.NoopHandler{Log: n.log},
		net:         n,
	}
	n.responseHandler = &ResponseHandler{
		NoopHandler: message.NoopHandler{Log: n.log},
		net:         n,
	}
	n.gossipHandler = &GossipHandler{
		NoopHandler: message.NoopHandler{Log: n.log},
		net:         n,
	}
	return n
}

func (n *network) AppRequestFailed(nodeID ids.ShortID, requestID uint32) error {
	n.log.Debug(
		"AppRequestFailed called with %s and requestID %d",
		nodeID.PrefixedString(constants.NodeIDPrefix),
		requestID,
	)

	if time.Now().Before(n.gossipActivationTime) {
		n.log.Warn("AppRequestFailed called before activation time")
		return nil
	}

	delete(n.requestsContent, requestID)
	return nil
}

func (n *network) AppRequest(nodeID ids.ShortID, requestID uint32, msgBytes []byte) error {
	return n.handle(
		n.requestHandler,
		"Request",
		nodeID,
		requestID,
		msgBytes,
	)
}

func (n *network) AppResponse(nodeID ids.ShortID, requestID uint32, msgBytes []byte) error {
	return n.handle(
		n.responseHandler,
		"Response",
		nodeID,
		requestID,
		msgBytes,
	)
}

func (n *network) AppGossip(nodeID ids.ShortID, msgBytes []byte) error {
	return n.handle(
		n.gossipHandler,
		"Gossip",
		nodeID,
		0,
		msgBytes,
	)
}

func (n *network) GossipTx(tx *Tx) error {
	txID := tx.ID()
	n.log.Debug("gossiping tx %s", txID)

	msg := message.TxNotify{
		TxID: txID,
	}
	msgBytes, err := message.Build(&msg)
	if err != nil {
		return fmt.Errorf("GossipTx: failed to build TxNotify message with: %w", err)
	}
	return n.appSender.SendAppGossip(msgBytes)
}

func (n *network) handle(
	handler message.Handler,
	handlerName string,
	nodeID ids.ShortID,
	requestID uint32,
	msgBytes []byte,
) error {
	n.log.Debug(
		"App%s message handler called from %s with requestID %d and %d bytes",
		handlerName,
		nodeID.PrefixedString(constants.NodeIDPrefix),
		requestID,
		len(msgBytes),
	)

	if time.Now().Before(n.gossipActivationTime) {
		n.log.Debug("App%s message called before activation time", handlerName)
		return nil
	}

	msg, err := message.Parse(msgBytes)
	if err != nil {
		n.log.Debug(
			"dropping App%s message due to failing to parse message",
			handlerName,
		)
		return nil
	}

	return msg.Handle(handler, nodeID, requestID)
}

type RequestHandler struct {
	message.NoopHandler

	net *network
}

func (h *RequestHandler) HandleTxNotify(nodeID ids.ShortID, requestID uint32, msg *message.TxNotify) error {
	h.net.log.Debug(
		"AppRequest called with TxNotify from %s with requestID %d and txID %s",
		nodeID.PrefixedString(constants.NodeIDPrefix),
		requestID,
		msg.TxID,
	)

	tx := h.net.mempool.Get(msg.TxID)
	if tx == nil {
		h.net.log.Trace(
			"dropping AppRequest from %s with requestID %d for unknown tx %s",
			nodeID.PrefixedString(constants.NodeIDPrefix),
			requestID,
			msg.TxID,
		)
		return nil
	}

	reply := &message.Tx{
		Tx: tx.Bytes(),
	}
	replyBytes, err := message.Build(reply)
	if err != nil {
		h.net.log.Warn(
			"failed to build response Tx message with tx %s with: %s",
			msg.TxID,
			err,
		)
		return nil
	}

	if err := h.net.appSender.SendAppResponse(nodeID, requestID, replyBytes); err != nil {
		return fmt.Errorf("failed to send AppResponse with: %s", err)
	}
	return nil
}

type ResponseHandler struct {
	message.NoopHandler

	net *network
}

func (h *ResponseHandler) HandleTx(nodeID ids.ShortID, requestID uint32, msg *message.Tx) error {
	h.net.log.Debug(
		"AppResponse called with Tx from %s with requestID %d and %d bytes",
		nodeID.PrefixedString(constants.NodeIDPrefix),
		requestID,
		len(msg.Tx),
	)

	// check that the received transaction matches the requested transaction
	expectedTxID, ok := h.net.requestsContent[requestID]
	if !ok {
		h.net.log.Verbo("AppResponse provided unrequested tx")
		return nil
	}
	delete(h.net.requestsContent, requestID)

	tx := &Tx{}
	_, err := Codec.Unmarshal(msg.Tx, tx)
	if err != nil {
		h.net.log.Verbo("AppResponse provided invalid tx: %s", err)
		return nil
	}
	unsignedBytes, err := Codec.Marshal(CodecVersion, &tx.UnsignedTx)
	if err != nil {
		h.net.log.Warn("AppResponse failed to marshal unsigned tx: %s", err)
		return nil
	}
	tx.Initialize(unsignedBytes, msg.Tx)

	txID := tx.ID()
	if txID != expectedTxID {
		h.net.log.Verbo(
			"AppResponse provided txID %s when it was expecting",
			txID,
			expectedTxID,
		)
		return nil
	}

	if h.net.mempool.WasDropped(txID) {
		// If the tx is being dropped - just ignore it
		return nil
	}

	// add to mempool
	err = h.net.mempool.AddUnverifiedTx(tx)
	if err != nil {
		h.net.log.Debug(
			"AppResponse failed AddUnverifiedTx from %s with: %s",
			nodeID.PrefixedString(constants.NodeIDPrefix),
			err,
		)
		return nil
	}
	return h.net.GossipTx(tx)
}

type GossipHandler struct {
	message.NoopHandler

	net *network
}

func (h *GossipHandler) HandleTxNotify(nodeID ids.ShortID, requestID uint32, msg *message.TxNotify) error {
	h.net.log.Debug(
		"AppGossip called with TxNotify from %s with requestID %d and txID %s",
		nodeID.PrefixedString(constants.NodeIDPrefix),
		requestID,
		msg.TxID,
	)

	switch {
	case h.net.mempool.Has(msg.TxID):
		return nil
	case h.net.mempool.WasDropped(msg.TxID):
		return nil
	}

	nodes := ids.ShortSet{
		nodeID: struct{}{},
	}
	h.net.requestID++
	if err := h.net.appSender.SendAppRequest(nodes, h.net.requestID, msg.Bytes()); err != nil {
		return fmt.Errorf("AppGossip: failed sending AppRequest with: %s", err)
	}

	// record txID to validate response
	h.net.requestsContent[h.net.requestID] = msg.TxID
	return nil
}
