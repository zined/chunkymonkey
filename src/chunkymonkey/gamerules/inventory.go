package gamerules

import (
	"os"

	"chunkymonkey/proto"
	. "chunkymonkey/types"
	"nbt"
)

type IInventorySubscriber interface {
	// SlotUpdate is called when a slot inside the inventory changes its
	// contents.
	SlotUpdate(slot *Slot, slotId SlotId)

	// ProgressUpdate is called when a progress bar inside the inventory changes.
	ProgressUpdate(prgBarId PrgBarId, value PrgBarValue)
}

// IInventory is the general interface provided by inventory implementations.
type IInventory interface {
	NumSlots() SlotId
	Click(slotId SlotId, cursor *Slot, rightClick bool, shiftClick bool, txId TxId, expectedSlot *Slot) (txState TxState)
	SetSubscriber(subscriber IInventorySubscriber)
	MakeProtoSlots() []proto.WindowSlot
	WriteProtoSlots(slots []proto.WindowSlot)
	TakeAllItems() (items []Slot)
	ReadNbtSlot(tag nbt.ITag, slotId SlotId) (err os.Error)
}

type Inventory struct {
	slots      []Slot
	subscriber IInventorySubscriber
}

// Init initializes the inventory. onUnsubscribed is called in a new goroutine
// when the number of subscribers to the inventory reaches zero (but is not
// called initially).
func (inv *Inventory) Init(size int) {
	inv.slots = make([]Slot, size)
}

func (inv *Inventory) NumSlots() SlotId {
	return SlotId(len(inv.slots))
}

func (inv *Inventory) SetSubscriber(subscriber IInventorySubscriber) {
	inv.subscriber = subscriber
}

// Click takes the default actions upon a click event from a player.
func (inv *Inventory) Click(slotId SlotId, cursor *Slot, rightClick bool, shiftClick bool, txId TxId, expectedSlot *Slot) TxState {
	if slotId < 0 || int(slotId) > len(inv.slots) {
		return TxStateRejected
	}

	clickedSlot := &inv.slots[slotId]

	if !expectedSlot.Equals(clickedSlot) {
		return TxStateRejected
	}

	// Apply the change.
	if cursor.Count == 0 {
		if rightClick {
			clickedSlot.Split(cursor)
		} else {
			clickedSlot.Swap(cursor)
		}
	} else {
		var changed bool

		if rightClick {
			changed = clickedSlot.AddOne(cursor)
		} else {
			changed = clickedSlot.Add(cursor)
		}

		if !changed {
			clickedSlot.Swap(cursor)
		}
	}

	// We send slot updates in case we have custom max counts that differ from
	// the client's idea.
	inv.slotUpdate(clickedSlot, slotId)

	return TxStateAccepted
}

// TakeOnlyClick only allows items to be taken from the slot, and it only
// allows the *whole* stack to be taken, otherwise no items are taken at all.
func (inv *Inventory) TakeOnlyClick(slotId SlotId, cursor *Slot, rightClick, shiftClick bool, txId TxId, expectedSlot *Slot) TxState {
	if slotId < 0 || int(slotId) > len(inv.slots) {
		return TxStateRejected
	}

	clickedSlot := &inv.slots[slotId]

	if !expectedSlot.Equals(clickedSlot) {
		return TxStateRejected
	}

	// Apply the change.
	cursor.AddWhole(clickedSlot)

	// We send slot updates in case we have custom max counts that differ from
	// the client's idea.
	inv.slotUpdate(clickedSlot, slotId)

	return TxStateAccepted
}

func (inv *Inventory) Slot(slotId SlotId) Slot {
	return inv.slots[slotId]
}

func (inv *Inventory) TakeOneItem(slotId SlotId, into *Slot) {
	slot := &inv.slots[slotId]
	if into.AddOne(slot) {
		inv.slotUpdate(slot, slotId)
	}
}

// PutItem attempts to put the given item into the inventory.
func (inv *Inventory) PutItem(item *Slot) {
	// TODO optimize this algorithm, maybe by maintaining a map of non-full
	// slots containing an item of various item type IDs. Additionally, it
	// should prefer to put stackable items into stacks of the same type,
	// rather than in empty slots.
	for slotIndex := range inv.slots {
		if item.Count <= 0 {
			break
		}
		slot := &inv.slots[slotIndex]
		if slot.Add(item) {
			inv.slotUpdate(slot, SlotId(slotIndex))
		}
	}
}

// CanTakeItem returns true if it can take at least one item from the passed
// Slot.
func (inv *Inventory) CanTakeItem(item *Slot) bool {
	if item.Count <= 0 {
		return false
	}

	itemCopy := *item

	for slotIndex := range inv.slots {
		slotCopy := inv.slots[slotIndex]

		if slotCopy.Add(&itemCopy) {
			return true
		}
	}

	return false
}

func (inv *Inventory) MakeProtoSlots() []proto.WindowSlot {
	slots := make([]proto.WindowSlot, len(inv.slots))
	inv.WriteProtoSlots(slots)
	return slots
}

// WriteProtoSlots stores into the slots parameter the proto version of the
// item data in the inventory.
// Precondition: len(slots) == len(inv.slots)
func (inv *Inventory) WriteProtoSlots(slots []proto.WindowSlot) {
	for i := range inv.slots {
		src := &inv.slots[i]
		slots[i] = proto.WindowSlot{
			ItemTypeId: src.ItemTypeId,
			Count:      src.Count,
			Data:       src.Data,
		}
	}
}

// TakeAllItems empties the inventory, and returns all items that were inside
// it inside a slice of Slots.
func (inv *Inventory) TakeAllItems() (items []Slot) {
	items = make([]Slot, 0, len(inv.slots)-1)

	for i := range inv.slots {
		curSlot := &inv.slots[i]
		if curSlot.Count > 0 {
			var taken Slot
			taken.Swap(curSlot)
			items = append(items, taken)
			inv.slotUpdate(curSlot, SlotId(i))
		}
	}

	return
}

// Send message about the slot change to the relevant places.
func (inv *Inventory) slotUpdate(slot *Slot, slotId SlotId) {
	if inv.subscriber != nil {
		inv.subscriber.SlotUpdate(slot, slotId)
	}
}

func (inv *Inventory) ReadNbtSlot(tag nbt.ITag, slotId SlotId) (err os.Error) {
	if slotId < 0 || int(slotId) >= len(inv.slots) {
		return os.NewError("Bad slot ID")
	}
	return inv.slots[slotId].ReadNbt(tag)
}