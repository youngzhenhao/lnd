package peer

import "github.com/btcsuite/btcd/wire"

// channelView is a view into the current active/global channel state machine
// for a given link.
type channelView interface {
	// OweCommitment returns a boolean value reflecting whether we need to
	// send out a commitment signature because there are outstanding local
	// updates and/or updates in the local commit tx that aren't reflected
	// in the remote commit tx yet.
	OweCommitment() bool

	// MarkCoopBroadcasted persistently marks that the channel close
	// transaction has been broadcast.
	MarkCoopBroadcasted(*wire.MsgTx, bool) error
}

// linkController is capable of controlling the flow out incoming/outgoing
// HTLCs to/from the link.
type linkController interface {
	// DisableAdds instructs the channel link to disable process new adds
	// in the specified direction. An error is returned if the link is
	// already disabled in that direction.
	DisableAdds(outgoing bool) error

	// TODO(roasbeef): also use this to understand if link there or not for
	// balance call?
}

// chanObserver implements the chancloser.ChanObserver interface for the
// existing LightningChannel struct/instance.
type chanObserver struct {
	chanView channelView
	link     linkController
}

// newChanObserver creates a new instance of a chanObserver from an active
// channelView.
func newChanObserver(chanView channelView,
	link linkController) *chanObserver {

	return &chanObserver{
		chanView: chanView,
		link:     link,
	}
}

// NoDanglingUpdates returns true if there are no dangling updates in the
// channel. In other words, there are no active update messages that haven't
// already been covered by a commit sig.
func (l *chanObserver) NoDanglingUpdates() bool {
	return !l.chanView.OweCommitment()
}

// DisableIncomingAdds instructs the channel link to disable process new
// incoming add messages.
func (l *chanObserver) DisableIncomingAdds() error {
	// If there's no link, then we don't need to disable any adds.
	if l.link == nil {
		return nil
	}

	return l.link.DisableAdds(false)
}

// DisableOutgoingAdds instructs the channel link to disable process new
// outgoing add messages.
func (l *chanObserver) DisableOutgoingAdds() error {
	// If there's no link, then we don't need to disable any adds.
	if l.link == nil {
		return nil
	}

	return l.link.DisableAdds(true)
}

// MarkCoopBroadcasted persistently marks that the channel close transaction
// has been broadcast.
func (l *chanObserver) MarkCoopBroadcasted(tx *wire.MsgTx, local bool) error {
	return l.chanView.MarkCoopBroadcasted(tx, local)
}
