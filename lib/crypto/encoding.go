package crypto

import (
	"encoding/base64"
	"encoding/hex"
)

const (
	// KeySize is the size of the encryption key in bytes
	KeySize = 32
	// IVSize is the size of the IV in bytes
	IVSize = 12
	// LegacyIVSize is the size of the IV in bytes for the legacy encryption scheme
	LegacyIVSize = 16
	// AADSize is the size in bytes of the Additional Authenticated Data for the
	// GCM encryption
	AADSize = 16
)

// Hex encode bytes
func (c *SCrypto) Hex(src []byte, maxLen int) []byte {
	dst := make([]byte, hex.EncodedLen(len(src)))
	hex.Encode(dst, src)
	if len(dst) > maxLen {
		// avoid extraneous padding
		dst = dst[:maxLen]
	}
	return dst
}

// Unhex bytes
func (c *SCrypto) Unhex(src []byte, maxLen int) []byte {
	dst := make([]byte, hex.DecodedLen(len(src)))
	hex.Decode(dst, src)
	if len(dst) > maxLen {
		// avoid extraneous padding
		dst = dst[:maxLen]
	}
	return dst
}

// Base64Encode bytes
func (c *SCrypto) Base64Encode(src []byte, maxLen int) []byte {
	dst := make([]byte, base64.StdEncoding.EncodedLen(len(src)))
	base64.StdEncoding.Encode(dst, src)
	if len(dst) > maxLen {
		// avoid extraneous padding
		dst = dst[:maxLen]
	}
	return dst
}

// Base64Decode bytes
func (c *SCrypto) Base64Decode(src []byte, maxLen int) []byte {
	dst := make([]byte, base64.StdEncoding.DecodedLen(len(src)))
	base64.StdEncoding.Decode(dst, src)
	if len(dst) > maxLen {
		// avoid extraneous padding
		dst = dst[:maxLen]
	}
	return dst
}
