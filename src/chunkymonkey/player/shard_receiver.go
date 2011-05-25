package player

import (
	"chunkymonkey/slot"
	. "chunkymonkey/types"
)

// playerShardReceiver receives events from chunk shards and acts upon them. It
// implements shardserver_external.IPlayerConnection.
type playerShardReceiver struct {
	player *Player
}

func (psr *playerShardReceiver) Init(player *Player) {
	psr.player = player
}

func (psr *playerShardReceiver) GetEntityId() EntityId {
	return psr.player.EntityId
}

func (psr *playerShardReceiver) TransmitPacket(packet []byte) {
	psr.player.TransmitPacket(packet)
}

func (psr *playerShardReceiver) InventorySubscribed(shardInvId int32, invTypeId InvTypeId) {
	// TODO
}

func (psr *playerShardReceiver) InventoryUpdate(shardInvId int32, slotIds []SlotId, slots []slot.Slot) {
	// TODO
}

func (psr *playerShardReceiver) RequestPlaceHeldItem(target BlockXyz) {
	psr.player.Enqueue(func(_ *Player) {
		psr.player.requestPlaceHeldItem(&target)
	})
}

func (psr *playerShardReceiver) RequestOfferItem(fromChunk ChunkXz, entityId EntityId, item slot.Slot) {
	psr.player.Enqueue(func(_ *Player) {
		psr.player.requestOfferItem(&fromChunk, entityId, &item)
	})
}

func (psr *playerShardReceiver) RequestGiveItem(atPosition AbsXyz, item slot.Slot) {
	psr.player.Enqueue(func(_ *Player) {
		psr.player.requestGiveItem(&atPosition, &item)
	})
}