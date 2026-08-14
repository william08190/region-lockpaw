package geometry

// Point and Rect use Win32's physical-pixel coordinate convention. A Rect's
// right and bottom edges are exclusive.
type Point struct {
	X int32
	Y int32
}

type Rect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

func NewRect(a, b Point) Rect {
	return Rect{
		Left:   min(a.X, b.X),
		Top:    min(a.Y, b.Y),
		Right:  max(a.X, b.X),
		Bottom: max(a.Y, b.Y),
	}
}

func (r Rect) Width() int32 {
	return max(0, r.Right-r.Left)
}

func (r Rect) Height() int32 {
	return max(0, r.Bottom-r.Top)
}

func (r Rect) Empty() bool {
	return r.Width() == 0 || r.Height() == 0
}

func (r Rect) ContainsPoint(point Point) bool {
	return point.X >= r.Left && point.X < r.Right && point.Y >= r.Top && point.Y < r.Bottom
}

func (r Rect) ContainsRect(other Rect) bool {
	return !other.Empty() && other.Left >= r.Left && other.Top >= r.Top && other.Right <= r.Right && other.Bottom <= r.Bottom
}

func (r Rect) Intersect(other Rect) Rect {
	result := Rect{
		Left:   max(r.Left, other.Left),
		Top:    max(r.Top, other.Top),
		Right:  min(r.Right, other.Right),
		Bottom: min(r.Bottom, other.Bottom),
	}
	if result.Empty() {
		return Rect{}
	}
	return result
}

func ClampPoint(point Point, bounds Rect) Point {
	return Point{
		X: min(max(point.X, bounds.Left), bounds.Right),
		Y: min(max(point.Y, bounds.Top), bounds.Bottom),
	}
}

// MaskFrames returns up to four non-overlapping rectangles covering screen
// except for allowed. If allowed does not overlap screen, the whole screen is
// returned as one mask.
func MaskFrames(screen, allowed Rect) []Rect {
	if screen.Empty() {
		return nil
	}
	hole := screen.Intersect(allowed)
	if hole.Empty() {
		return []Rect{screen}
	}

	candidates := []Rect{
		{Left: screen.Left, Top: screen.Top, Right: screen.Right, Bottom: hole.Top},
		{Left: screen.Left, Top: hole.Bottom, Right: screen.Right, Bottom: screen.Bottom},
		{Left: screen.Left, Top: hole.Top, Right: hole.Left, Bottom: hole.Bottom},
		{Left: hole.Right, Top: hole.Top, Right: screen.Right, Bottom: hole.Bottom},
	}

	masks := make([]Rect, 0, len(candidates))
	for _, candidate := range candidates {
		if !candidate.Empty() {
			masks = append(masks, candidate)
		}
	}
	return masks
}
