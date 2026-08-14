package geometry

import (
	"reflect"
	"testing"
)

func TestNewRectNormalizesDragDirection(t *testing.T) {
	got := NewRect(Point{X: 700, Y: 500}, Point{X: 100, Y: 200})
	want := Rect{Left: 100, Top: 200, Right: 700, Bottom: 500}
	if got != want {
		t.Fatalf("NewRect() = %#v, want %#v", got, want)
	}
}

func TestMaskFramesLeavesExactlyOneHole(t *testing.T) {
	screen := Rect{Left: 0, Top: 0, Right: 1920, Bottom: 1080}
	allowed := Rect{Left: 300, Top: 200, Right: 1500, Bottom: 900}
	want := []Rect{
		{Left: 0, Top: 0, Right: 1920, Bottom: 200},
		{Left: 0, Top: 900, Right: 1920, Bottom: 1080},
		{Left: 0, Top: 200, Right: 300, Bottom: 900},
		{Left: 1500, Top: 200, Right: 1920, Bottom: 900},
	}
	if got := MaskFrames(screen, allowed); !reflect.DeepEqual(got, want) {
		t.Fatalf("MaskFrames() = %#v, want %#v", got, want)
	}
}

func TestMaskFramesSupportsNegativeMonitorCoordinates(t *testing.T) {
	screen := Rect{Left: -1920, Top: -120, Right: 0, Bottom: 960}
	allowed := Rect{Left: -1700, Top: 80, Right: -400, Bottom: 800}
	masks := MaskFrames(screen, allowed)

	if len(masks) != 4 {
		t.Fatalf("len(MaskFrames()) = %d, want 4", len(masks))
	}
	for _, mask := range masks {
		if !screen.ContainsRect(mask) {
			t.Fatalf("mask %#v is outside screen %#v", mask, screen)
		}
		if !mask.Intersect(allowed).Empty() {
			t.Fatalf("mask %#v overlaps allowed rect %#v", mask, allowed)
		}
	}
}

func TestMaskFramesClipsPartiallyOutOfBoundsHole(t *testing.T) {
	screen := Rect{Left: 0, Top: 0, Right: 1000, Bottom: 800}
	allowed := Rect{Left: -50, Top: 100, Right: 600, Bottom: 900}
	want := []Rect{
		{Left: 0, Top: 0, Right: 1000, Bottom: 100},
		{Left: 600, Top: 100, Right: 1000, Bottom: 800},
	}
	if got := MaskFrames(screen, allowed); !reflect.DeepEqual(got, want) {
		t.Fatalf("MaskFrames() = %#v, want %#v", got, want)
	}
}

func TestMaskFramesMasksWholeNonSelectedMonitor(t *testing.T) {
	screen := Rect{Left: 1920, Top: 0, Right: 3840, Bottom: 1080}
	allowed := Rect{Left: 200, Top: 200, Right: 1200, Bottom: 900}
	want := []Rect{screen}
	if got := MaskFrames(screen, allowed); !reflect.DeepEqual(got, want) {
		t.Fatalf("MaskFrames() = %#v, want %#v", got, want)
	}
}

func TestClampPointIncludesExclusiveBottomRightForDrag(t *testing.T) {
	bounds := Rect{Left: -100, Top: 20, Right: 900, Bottom: 620}
	got := ClampPoint(Point{X: 1200, Y: -50}, bounds)
	want := Point{X: 900, Y: 20}
	if got != want {
		t.Fatalf("ClampPoint() = %#v, want %#v", got, want)
	}
}

func TestMaskFramesAreDisjointAndPreserveScreenArea(t *testing.T) {
	tests := []struct {
		name    string
		screen  Rect
		allowed Rect
	}{
		{
			name:    "center hole",
			screen:  Rect{Left: 0, Top: 0, Right: 2560, Bottom: 1440},
			allowed: Rect{Left: 400, Top: 300, Right: 2200, Bottom: 1200},
		},
		{
			name:    "hole touching two edges",
			screen:  Rect{Left: -1600, Top: -900, Right: 0, Bottom: 0},
			allowed: Rect{Left: -1600, Top: -900, Right: -500, Bottom: -100},
		},
		{
			name:    "hole outside screen",
			screen:  Rect{Left: 0, Top: 0, Right: 1920, Bottom: 1080},
			allowed: Rect{Left: 2200, Top: 200, Right: 3000, Bottom: 900},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			masks := MaskFrames(test.screen, test.allowed)
			hole := test.screen.Intersect(test.allowed)
			coveredArea := int64(hole.Width()) * int64(hole.Height())

			for index, mask := range masks {
				if !test.screen.ContainsRect(mask) {
					t.Fatalf("mask %#v is outside screen %#v", mask, test.screen)
				}
				if !mask.Intersect(hole).Empty() {
					t.Fatalf("mask %#v overlaps hole %#v", mask, hole)
				}
				coveredArea += int64(mask.Width()) * int64(mask.Height())
				for _, other := range masks[index+1:] {
					if !mask.Intersect(other).Empty() {
						t.Fatalf("masks %#v and %#v overlap", mask, other)
					}
				}
			}

			screenArea := int64(test.screen.Width()) * int64(test.screen.Height())
			if coveredArea != screenArea {
				t.Fatalf("covered area = %d, screen area = %d", coveredArea, screenArea)
			}
		})
	}
}

func TestMaskFramesReturnsNothingForEmptyScreen(t *testing.T) {
	if got := MaskFrames(Rect{}, Rect{}); len(got) != 0 {
		t.Fatalf("MaskFrames() = %#v, want no frames", got)
	}
}
