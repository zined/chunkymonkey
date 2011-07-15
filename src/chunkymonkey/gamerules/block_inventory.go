package gamerules

import (
	. "chunkymonkey/types"
)


// blockInventory is the data stored in Chunk.SetBlockExtra by some block
// aspects that contain inventories. It also implements IInventorySubscriber to
// relay events to player(s) subscribed to the inventories.
type blockInventory struct {
	instance           BlockInstance
	inv                IInventory
	subscribers        map[EntityId]IShardPlayerClient
	ejectOnUnsubscribe bool
	invTypeId          InvTypeId
}

// newBlockInventory creates a new blockInventory.
func newBlockInventory(instance *BlockInstance, inv IInventory, ejectOnUnsubscribe bool, invTypeId InvTypeId) *blockInventory {
	blkInv := &blockInventory{
		instance:           *instance,
		inv:                inv,
		subscribers:        make(map[EntityId]IShardPlayerClient),
		ejectOnUnsubscribe: ejectOnUnsubscribe,
		invTypeId:          invTypeId,
	}

	blkInv.inv.SetSubscriber(blkInv)

	return blkInv
}

func (blkInv *blockInventory) Click(player IShardPlayerClient, click *Click) {
	txState := blkInv.inv.Click(click)

	player.ReqInventoryCursorUpdate(blkInv.instance.BlockLoc, click.Cursor)

	// Inform client of operation status.
	player.ReqInventoryTxState(blkInv.instance.BlockLoc, click.TxId, txState == TxStateAccepted)
}

func (blkInv *blockInventory) SlotUpdate(slot *Slot, slotId SlotId) {
	for _, subscriber := range blkInv.subscribers {
		subscriber.ReqInventorySlotUpdate(blkInv.instance.BlockLoc, *slot, slotId)
	}
}

func (blkInv *blockInventory) ProgressUpdate(prgBarId PrgBarId, value PrgBarValue) {
	for _, subscriber := range blkInv.subscribers {
		subscriber.ReqInventoryProgressUpdate(blkInv.instance.BlockLoc, prgBarId, value)
	}
}

func (blkInv *blockInventory) AddSubscriber(player IShardPlayerClient) {
	entityId := player.GetEntityId()
	blkInv.subscribers[entityId] = player

	// Register self for automatic removal when IShardPlayerClient unsubscribes
	// from the chunk.
	blkInv.instance.Chunk.AddOnUnsubscribe(entityId, blkInv)

	slots := blkInv.inv.MakeProtoSlots()

	player.ReqInventorySubscribed(blkInv.instance.BlockLoc, blkInv.invTypeId, slots)
}

func (blkInv *blockInventory) RemoveSubscriber(entityId EntityId) {
	blkInv.subscribers[entityId] = nil, false
	blkInv.instance.Chunk.RemoveOnUnsubscribe(entityId, blkInv)
	if blkInv.ejectOnUnsubscribe && len(blkInv.subscribers) == 0 {
		blkInv.EjectItems()
	}
}

func (blkInv *blockInventory) Destroyed() {
	for _, subscriber := range blkInv.subscribers {
		subscriber.ReqInventoryUnsubscribed(blkInv.instance.BlockLoc)
		blkInv.instance.Chunk.RemoveOnUnsubscribe(subscriber.GetEntityId(), blkInv)
	}
	blkInv.subscribers = nil
}

// Unsubscribed implements IUnsubscribed. It removes a player's
// subscription to the inventory when they unsubscribe from the chunk.
func (blkInv *blockInventory) Unsubscribed(entityId EntityId) {
	blkInv.subscribers[entityId] = nil, false
}

// EjectItems removes all items from the inventory and drops them at the
// location of the block.
func (blkInv *blockInventory) EjectItems() {
	items := blkInv.inv.TakeAllItems()

	for _, slot := range items {
		spawnItemInBlock(&blkInv.instance, slot.ItemTypeId, slot.Count, slot.Data)
	}
}
