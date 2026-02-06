// Package img registers the PAA image format with the standard image package.
// Import it with a blank import to enable image.Decode and image.DecodeConfig for PAA:
//
//	import _ "github.com/woozymasta/paa/img"
package img

import (
	"image"

	"github.com/woozymasta/paa"
)

func init() {
	image.RegisterFormat("paa_dxt1", "\x01\xff", paa.Decode, paa.DecodeConfig)
	image.RegisterFormat("paa_dxt2", "\x02\xff", paa.Decode, paa.DecodeConfig)
	image.RegisterFormat("paa_dxt3", "\x03\xff", paa.Decode, paa.DecodeConfig)
	image.RegisterFormat("paa_dxt4", "\x04\xff", paa.Decode, paa.DecodeConfig)
	image.RegisterFormat("paa_dxt5", "\x05\xff", paa.Decode, paa.DecodeConfig)
	image.RegisterFormat("paa_argb4", "\x44\x44", paa.Decode, paa.DecodeConfig)
	image.RegisterFormat("paa_argb1555", "\x55\x15", paa.Decode, paa.DecodeConfig)
	image.RegisterFormat("paa_argb8", "\x88\x88", paa.Decode, paa.DecodeConfig)
	image.RegisterFormat("paa_graya", "\x80\x80", paa.Decode, paa.DecodeConfig)
}
