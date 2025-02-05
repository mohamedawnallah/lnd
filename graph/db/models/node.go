package models

import (
	"fmt"
	"image/color"
	"net"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"github.com/lightningnetwork/lnd/lnwire"
)

// LightningNode represents an individual vertex/node within the channel graph.
// A node is connected to other nodes by one or more channel edges emanating
// from it. As the graph is directed, a node will also have an incoming edge
// attached to it for each outgoing edge.
type LightningNode struct {
	// PubKeyBytes is the raw bytes of the public key of the target node.
	PubKeyBytes [33]byte
	pubKey      *btcec.PublicKey

	// HaveNodeAnnouncement indicates whether we received a node
	// announcement for this particular node. If true, the remaining fields
	// will be set, if false only the PubKey is known for this node.
	HaveNodeAnnouncement bool

	// LastUpdate is the last time the vertex information for this node has
	// been updated.
	LastUpdate time.Time

	// Address is the TCP address this node is reachable over.
	Addresses []net.Addr

	// DNSHostnameAddress is the DNS hostname address for this node.
	DNSHostnameAddress *lnwire.DNSHostnameAddress

	// Color is the selected color for the node.
	Color color.RGBA

	// Alias is a nick-name for the node. The alias can be used to confirm
	// a node's identity or to serve as a short ID for an address book.
	Alias string

	// AuthSigBytes is the raw signature under the advertised public key
	// which serves to authenticate the attributes announced by this node.
	AuthSigBytes []byte

	// Features is the list of protocol features supported by this node.
	Features *lnwire.FeatureVector

	// ExtraOpaqueData is the set of data that was appended to this
	// message, some of which we may not actually know how to iterate or
	// parse. By holding onto this data, we ensure that we're able to
	// properly validate the set of signatures that cover these new fields,
	// and ensure we're able to make upgrades to the network in a forwards
	// compatible manner.
	ExtraOpaqueData []byte

	// TODO(roasbeef): discovery will need storage to keep it's last IP
	// address and re-announce if interface changes?

	// TODO(roasbeef): add update method and fetch?
}

// PubKey is the node's long-term identity public key. This key will be used to
// authenticated any advertisements/updates sent by the node.
//
// NOTE: By having this method to access an attribute, we ensure we only need
// to fully deserialize the pubkey if absolutely necessary.
func (l *LightningNode) PubKey() (*btcec.PublicKey, error) {
	if l.pubKey != nil {
		return l.pubKey, nil
	}

	key, err := btcec.ParsePubKey(l.PubKeyBytes[:])
	if err != nil {
		return nil, err
	}
	l.pubKey = key

	return key, nil
}

// AuthSig is a signature under the advertised public key which serves to
// authenticate the attributes announced by this node.
//
// NOTE: By having this method to access an attribute, we ensure we only need
// to fully deserialize the signature if absolutely necessary.
func (l *LightningNode) AuthSig() (*ecdsa.Signature, error) {
	return ecdsa.ParseSignature(l.AuthSigBytes)
}

// AddPubKey is a setter-link method that can be used to swap out the public
// key for a node.
func (l *LightningNode) AddPubKey(key *btcec.PublicKey) {
	l.pubKey = key
	copy(l.PubKeyBytes[:], key.SerializeCompressed())
}

// NodeAnnouncement retrieves the latest node announcement of the node.
func (l *LightningNode) NodeAnnouncement(signed bool) (*lnwire.NodeAnnouncement,
	error) {

	if !l.HaveNodeAnnouncement {
		return nil, fmt.Errorf("node does not have node announcement")
	}

	alias, err := lnwire.NewNodeAlias(l.Alias)
	if err != nil {
		return nil, err
	}

	nodeAnn := &lnwire.NodeAnnouncement{
		Features:           l.Features.RawFeatureVector,
		NodeID:             l.PubKeyBytes,
		RGBColor:           l.Color,
		Alias:              alias,
		Addresses:          l.Addresses,
		DNSHostnameAddress: l.DNSHostnameAddress,
		Timestamp:          uint32(l.LastUpdate.Unix()),
		ExtraOpaqueData:    l.ExtraOpaqueData,
	}

	if !signed {
		return nodeAnn, nil
	}

	sig, err := lnwire.NewSigFromECDSARawSignature(l.AuthSigBytes)
	if err != nil {
		return nil, err
	}

	nodeAnn.Signature = sig

	return nodeAnn, nil
}
