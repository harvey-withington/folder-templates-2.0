package placement

import "testing"

func TestCentre(t *testing.T) {
	tests := []struct {
		name          string
		work          Rect
		width, height int
		want          Rect
	}{
		{"primary", Rect{0, 0, 1920, 1040}, 1040, 700, Rect{440, 170, 1480, 870}},
		{"monitor to the left", Rect{-2560, 0, 0, 1400}, 1000, 600, Rect{-1780, 400, -780, 1000}},
		{"monitor above, offset", Rect{300, -1080, 2220, -40}, 800, 600, Rect{860, -860, 1660, -260}},
		{"larger than the work area", Rect{0, 0, 1366, 728}, 2000, 900, Rect{0, 0, 1366, 728}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Centre(tt.work, tt.width, tt.height); got != tt.want {
				t.Errorf("Centre(%v, %d, %d) = %v, want %v", tt.work, tt.width, tt.height, got, tt.want)
			}
		})
	}
}

func TestScale(t *testing.T) {
	tests := []struct{ dips, dpi, want int }{
		{1040, 96, 1040},
		{1040, 144, 1560},
		{700, 120, 875},
		{1001, 144, 1502}, // 1501.5 rounds up
	}
	for _, tt := range tests {
		if got := scale(tt.dips, tt.dpi); got != tt.want {
			t.Errorf("scale(%d, %d) = %d, want %d", tt.dips, tt.dpi, got, tt.want)
		}
	}
}
