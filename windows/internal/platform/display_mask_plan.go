package platform

import "github.com/william08190/region-lockpaw/windows/internal/geometry"

func containsAllowedRegion(displays []geometry.Rect, allowed geometry.Rect) bool {
	for _, display := range displays {
		if display.ContainsRect(allowed) {
			return true
		}
	}
	return false
}

// displayChangeMaskRegion returns the hole that can safely remain usable after
// a display-layout change. An empty result means every active display must be
// masked until the original region is available again.
func displayChangeMaskRegion(displays []geometry.Rect, allowed geometry.Rect) (geometry.Rect, bool) {
	if !containsAllowedRegion(displays, allowed) {
		return geometry.Rect{}, false
	}
	for _, display := range displays {
		if len(geometry.MaskFrames(display, allowed)) > 0 {
			return allowed, true
		}
	}
	return geometry.Rect{}, false
}
