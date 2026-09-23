package solana

import (
	"fmt"

	"github.com/MevYu/solana-go/encoding"
)

// TransactionConfig is the resource configuration embedded in a v1 message.
// Nil fields are absent from the wire. PriorityFee is a total in lamports.
type TransactionConfig struct {
	PriorityFee                 *uint64 `json:"priorityFee"`
	ComputeUnitLimit            *uint32 `json:"computeUnitLimit"`
	LoadedAccountsDataSizeLimit *uint32 `json:"loadedAccountsDataSizeLimit"`
	HeapSize                    *uint32 `json:"heapSize"`
}

const (
	v1PriorityFeeMask      uint32 = 0x03
	v1ComputeUnitLimitMask uint32 = 0x04
	v1LoadedDataSizeMask   uint32 = 0x08
	v1HeapSizeMask         uint32 = 0x10
	v1KnownConfigMask      uint32 = 0x1f
)

func (c *TransactionConfig) mask() uint32 {
	if c == nil {
		return 0
	}
	var mask uint32
	if c.PriorityFee != nil {
		mask |= v1PriorityFeeMask
	}
	if c.ComputeUnitLimit != nil {
		mask |= v1ComputeUnitLimitMask
	}
	if c.LoadedAccountsDataSizeLimit != nil {
		mask |= v1LoadedDataSizeMask
	}
	if c.HeapSize != nil {
		mask |= v1HeapSizeMask
	}
	return mask
}

func (m *Message) serializedSizeV1() int {
	size := 1 + 3 + 4 + HashSize + 1 + 1 + len(m.AccountKeys)*PublicKeySize
	if c := m.TransactionConfig; c != nil {
		if c.PriorityFee != nil {
			size += 8
		}
		if c.ComputeUnitLimit != nil {
			size += 4
		}
		if c.LoadedAccountsDataSizeLimit != nil {
			size += 4
		}
		if c.HeapSize != nil {
			size += 4
		}
	}
	for i := range m.Instructions {
		size += 4 + len(m.Instructions[i].Accounts) + len(m.Instructions[i].Data)
	}
	return size
}

func (m *Message) marshalV1Into(e *encoding.Encoder) error {
	if len(m.AccountKeys) > 255 || len(m.Instructions) > 255 {
		return fmt.Errorf("solana: message: v1 account or instruction count exceeds 255")
	}
	for i := range m.Instructions {
		ix := &m.Instructions[i]
		if len(ix.Accounts) > 255 || len(ix.Data) > 65535 {
			return fmt.Errorf("solana: message: v1 instruction %d exceeds wire length", i)
		}
	}
	e.WriteUint8(versionPrefixMask | byte(MessageVersion1))
	e.WriteUint8(m.Header.NumRequiredSignatures)
	e.WriteUint8(m.Header.NumReadonlySignedAccounts)
	e.WriteUint8(m.Header.NumReadonlyUnsignedAccounts)
	e.WriteUint32(m.TransactionConfig.mask())
	e.WriteBytes(m.RecentBlockhash[:])
	e.WriteUint8(uint8(len(m.Instructions)))
	e.WriteUint8(uint8(len(m.AccountKeys)))
	for i := range m.AccountKeys {
		e.WriteBytes(m.AccountKeys[i][:])
	}
	if c := m.TransactionConfig; c != nil {
		if c.PriorityFee != nil {
			e.WriteUint64(*c.PriorityFee)
		}
		if c.ComputeUnitLimit != nil {
			e.WriteUint32(*c.ComputeUnitLimit)
		}
		if c.LoadedAccountsDataSizeLimit != nil {
			e.WriteUint32(*c.LoadedAccountsDataSizeLimit)
		}
		if c.HeapSize != nil {
			e.WriteUint32(*c.HeapSize)
		}
	}
	for i := range m.Instructions {
		ix := &m.Instructions[i]
		e.WriteUint8(ix.ProgramIDIndex)
		e.WriteUint8(uint8(len(ix.Accounts)))
		e.WriteUint16(uint16(len(ix.Data)))
	}
	for i := range m.Instructions {
		e.WriteBytes(m.Instructions[i].Accounts)
		e.WriteBytes(m.Instructions[i].Data)
	}
	return nil
}

func decodeMessageV1(d *encoding.Decoder) (*Message, error) {
	if _, err := d.ReadUint8(); err != nil {
		return nil, fmt.Errorf("solana: message: v1 version: %w", err)
	}
	m := &Message{Version: MessageVersion1, TransactionConfig: &TransactionConfig{}}
	fields := []*uint8{
		&m.Header.NumRequiredSignatures,
		&m.Header.NumReadonlySignedAccounts,
		&m.Header.NumReadonlyUnsignedAccounts,
	}
	for _, field := range fields {
		v, err := d.ReadUint8()
		if err != nil {
			return nil, fmt.Errorf("solana: message: v1 header: %w", err)
		}
		*field = v
	}
	mask, err := d.ReadUint32()
	if err != nil {
		return nil, fmt.Errorf("solana: message: v1 config mask: %w", err)
	}
	if mask&^v1KnownConfigMask != 0 || mask&v1PriorityFeeMask == 1 || mask&v1PriorityFeeMask == 2 {
		return nil, fmt.Errorf("solana: message: invalid v1 config mask %#x", mask)
	}
	blockhash, err := d.ReadBytes(HashSize)
	if err != nil {
		return nil, fmt.Errorf("solana: message: v1 lifetime: %w", err)
	}
	copy(m.RecentBlockhash[:], blockhash)
	ixCount, err := d.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("solana: message: v1 instruction count: %w", err)
	}
	keyCount, err := d.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("solana: message: v1 account count: %w", err)
	}
	m.AccountKeys = make([]PublicKey, keyCount)
	for i := range m.AccountKeys {
		b, err := d.ReadBytes(PublicKeySize)
		if err != nil {
			return nil, fmt.Errorf("solana: message: v1 account %d: %w", i, err)
		}
		copy(m.AccountKeys[i][:], b)
	}
	c := m.TransactionConfig
	if mask&v1PriorityFeeMask != 0 {
		v, err := d.ReadUint64()
		if err != nil {
			return nil, fmt.Errorf("solana: message: v1 priority fee: %w", err)
		}
		c.PriorityFee = &v
	}
	if mask&v1ComputeUnitLimitMask != 0 {
		v, err := d.ReadUint32()
		if err != nil {
			return nil, fmt.Errorf("solana: message: v1 compute unit limit: %w", err)
		}
		c.ComputeUnitLimit = &v
	}
	if mask&v1LoadedDataSizeMask != 0 {
		v, err := d.ReadUint32()
		if err != nil {
			return nil, fmt.Errorf("solana: message: v1 loaded account data size: %w", err)
		}
		c.LoadedAccountsDataSizeLimit = &v
	}
	if mask&v1HeapSizeMask != 0 {
		v, err := d.ReadUint32()
		if err != nil {
			return nil, fmt.Errorf("solana: message: v1 heap size: %w", err)
		}
		c.HeapSize = &v
	}
	m.Instructions = make([]CompiledInstruction, ixCount)
	accountCounts := make([]uint8, ixCount)
	dataCounts := make([]uint16, ixCount)
	for i := range m.Instructions {
		ix := &m.Instructions[i]
		if ix.ProgramIDIndex, err = d.ReadUint8(); err != nil {
			return nil, fmt.Errorf("solana: message: v1 instruction %d program index: %w", i, err)
		}
		if accountCounts[i], err = d.ReadUint8(); err != nil {
			return nil, fmt.Errorf("solana: message: v1 instruction %d account count: %w", i, err)
		}
		if dataCounts[i], err = d.ReadUint16(); err != nil {
			return nil, fmt.Errorf("solana: message: v1 instruction %d data count: %w", i, err)
		}
	}
	for i := range m.Instructions {
		accounts, err := d.ReadBytes(int(accountCounts[i]))
		if err != nil {
			return nil, fmt.Errorf("solana: message: v1 instruction %d accounts: %w", i, err)
		}
		data, err := d.ReadBytes(int(dataCounts[i]))
		if err != nil {
			return nil, fmt.Errorf("solana: message: v1 instruction %d data: %w", i, err)
		}
		m.Instructions[i].Accounts = append(Uint8Slice(nil), accounts...)
		m.Instructions[i].Data = append(Base58Data(nil), data...)
	}
	return m, nil
}
