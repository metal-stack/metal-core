package ipmi2_0

const (
	PersistentUEFI = uint8(0xe0) // 224 or -96
	OnlyNextBootUEFI = uint8(0xa0) // 160 or -32

	HD   = uint8(0x08) // 8
	PXE  = uint8(0x04) // 4
	BIOS = uint8(0x18) // 24
)
