package platform

import (
	"testing"

	"github.com/william08190/region-lockpaw/windows/internal/geometry"
)

func TestDisplayChangeMaskRegionPreservesAvailableRegion(t *testing.T) {
	displays := []geometry.Rect{{Left: 0, Top: 0, Right: 1920, Bottom: 1080}}
	allowed := geometry.Rect{Left: 200, Top: 100, Right: 1400, Bottom: 900}

	got, preserved := displayChangeMaskRegion(displays, allowed)
	if !preserved || got != allowed {
		t.Fatalf("displayChangeMaskRegion() = (%#v, %t), want (%#v, true)", got, preserved, allowed)
	}
}

func TestDisplayChangeMaskRegionMasksAllWhenSelectedDisplayIsUnavailable(t *testing.T) {
	displays := []geometry.Rect{{Left: 0, Top: 0, Right: 1920, Bottom: 1080}}
	allowed := geometry.Rect{Left: 2200, Top: 100, Right: 3400, Bottom: 900}

	got, preserved := displayChangeMaskRegion(displays, allowed)
	if preserved || !got.Empty() {
		t.Fatalf("displayChangeMaskRegion() = (%#v, %t), want full-mask fallback", got, preserved)
	}
}

func TestDisplayChangeMaskRegionMasksAllWhenRegionWouldLeaveNoMask(t *testing.T) {
	display := geometry.Rect{Left: 0, Top: 0, Right: 1920, Bottom: 1080}

	got, preserved := displayChangeMaskRegion([]geometry.Rect{display}, display)
	if preserved || !got.Empty() {
		t.Fatalf("displayChangeMaskRegion() = (%#v, %t), want full-mask fallback", got, preserved)
	}
}

func TestDisplayChangeMaskRegionPreservesFullDisplayWhenOthersRemainMasked(t *testing.T) {
	allowed := geometry.Rect{Left: 0, Top: 0, Right: 1920, Bottom: 1080}
	displays := []geometry.Rect{
		allowed,
		{Left: 1920, Top: 0, Right: 3840, Bottom: 1080},
	}

	got, preserved := displayChangeMaskRegion(displays, allowed)
	if !preserved || got != allowed {
		t.Fatalf("displayChangeMaskRegion() = (%#v, %t), want (%#v, true)", got, preserved, allowed)
	}
}
