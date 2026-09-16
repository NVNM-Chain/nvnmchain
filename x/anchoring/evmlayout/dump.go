package evmlayout

import (
	"encoding/binary"
	"io"

	"github.com/ethereum/go-ethereum/common"
)

// TEMPOSB v1, what Tempo's `init-from-binary-dump` reads: blocks of (slot, value) pairs for one
// account, each behind a 40-byte header of magic, version, flags, address and pair count.
const (
	dumpMagic   = "TEMPOSB\x00"
	dumpVersion = 1
	dumpHeader  = 40
	dumpPair    = 2 * common.HashLength
)

// Address is the contract's account on Tempo, the precompile's old address. Only a dump's header
// carries it; no slot depends on it.
var Address = common.HexToAddress("0x0000000000000000000000000000000000000a00")

// DumpBlockPairs is how many pairs a block holds, which bounds what a Dump buffers to 64 MiB.
const DumpBlockPairs = 1 << 20

// Dump writes slots as TEMPOSB v1 under one account, which Tempo's genesis alloc must already
// hold with the contract's code: the loader refuses any other, and the dump carries only storage.
type Dump struct {
	out     io.Writer
	address common.Address
	pending []byte
}

// NewDump writes to out, under address.
func NewDump(out io.Writer, address common.Address) *Dump {
	return &Dump{out: out, address: address}
}

// Put is a Sink: `evmlayout.Migrate(ctx, k, dump.Put)`.
func (d *Dump) Put(w Write) error {
	d.pending = append(d.pending, w.Slot[:]...)
	d.pending = append(d.pending, w.Value[:]...)
	if len(d.pending) == DumpBlockPairs*dumpPair {
		return d.flush()
	}
	return nil
}

// Close writes the last block. It leaves the writer open.
func (d *Dump) Close() error {
	return d.flush()
}

func (d *Dump) flush() error {
	if len(d.pending) == 0 {
		return nil
	}
	var header [dumpHeader]byte
	copy(header[:8], dumpMagic)
	binary.BigEndian.PutUint16(header[8:10], dumpVersion)
	// header[10:12] is flags, zero in v1.
	copy(header[12:32], d.address[:])
	binary.BigEndian.PutUint64(header[32:40], uint64(len(d.pending)/dumpPair))
	if _, err := d.out.Write(header[:]); err != nil {
		return err
	}
	if _, err := d.out.Write(d.pending); err != nil {
		return err
	}
	d.pending = d.pending[:0]
	return nil
}
