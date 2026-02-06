package paa

// PaxType defines the pixel format in a PAA file.
type PaxType uint32

// PAA pixel format identifiers (2-byte file magic values).
const (
	PaxDXT1   PaxType = 6  // 0x01FF DXT1
	PaxDXT2   PaxType = 7  // 0x02FF DXT2
	PaxDXT3   PaxType = 8  // 0x03FF DXT3
	PaxDXT4   PaxType = 9  // 0x04FF DXT4
	PaxDXT5   PaxType = 10 // 0x05FF DXT5
	PaxARGB4  PaxType = 4  // 0x4444 ARGB4
	PaxARGBA5 PaxType = 3  // 0x1555 ARGBA5
	PaxARGB8  PaxType = 5  // 0x8888 ARGB8
	PaxGRAYA  PaxType = 1  // 0x8080 GRAYA
)

// Bytes returns the 2-byte file magic for this format.
func (p PaxType) Bytes() []byte {
	switch p {
	case PaxDXT1:
		return []byte{1, 255}
	case PaxDXT2:
		return []byte{2, 255}
	case PaxDXT3:
		return []byte{3, 255}
	case PaxDXT4:
		return []byte{4, 255}
	case PaxDXT5:
		return []byte{5, 255}
	case PaxARGB4:
		return []byte{68, 68}
	case PaxARGBA5:
		return []byte{85, 21}
	case PaxARGB8:
		return []byte{136, 136}
	case PaxGRAYA:
		return []byte{128, 128}
	default:
		return []byte{0, 0}
	}
}

// PaxTypeFromBytes parses the 2-byte magic into a PaxType.
func PaxTypeFromBytes(b []byte) (PaxType, bool) {
	if len(b) < 2 {
		return 0, false
	}

	v := uint16(b[0]) | uint16(b[1])<<8
	switch v {
	case 0xFF01:
		return PaxDXT1, true
	case 0xFF02:
		return PaxDXT2, true
	case 0xFF03:
		return PaxDXT3, true
	case 0xFF04:
		return PaxDXT4, true
	case 0xFF05:
		return PaxDXT5, true
	case 0x4444:
		return PaxARGB4, true
	case 0x1555:
		return PaxARGBA5, true
	case 0x8888:
		return PaxARGB8, true
	case 0x8080:
		return PaxGRAYA, true
	}

	return 0, false
}

// isDXT returns true if the PaxType is a DXT texture.
func isDXT(t PaxType) bool {
	return t == PaxDXT1 || t == PaxDXT2 || t == PaxDXT3 || t == PaxDXT4 || t == PaxDXT5
}
