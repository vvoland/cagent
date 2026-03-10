package spinner

import (
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/require"

	"github.com/docker/docker-agent/pkg/tui/animation"
)

func TestSpinnerCopyDoesNotLeakAnimationSubscription(t *testing.T) {
	s1 := New(ModeSpinnerOnly, lipgloss.NewStyle())
	cmd := s1.Init()
	require.NotNil(t, cmd)
	require.True(t, animation.HasActive())

	// Copy the spinner value and stop via the copy; should still stop the shared subscription.
	s2 := s1
	s2.Stop()
	require.False(t, animation.HasActive())
}

func TestFrameWrapsAround(t *testing.T) {
	n := len(spinnerFrames)
	require.Equal(t, spinnerFrames[0], Frame(0))
	require.Equal(t, spinnerFrames[1], Frame(1))
	require.Equal(t, spinnerFrames[0], Frame(n), "Frame should wrap around")
}

func TestSelectFrames(t *testing.T) {
	require.Equal(t, brailleFrames, selectFrames(false))
	require.Equal(t, asciiFrames, selectFrames(true))
}

func TestInMultiplexer(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want bool
	}{
		{"plain terminal", map[string]string{"TERM": "xterm-256color"}, false},
		{"TMUX set", map[string]string{"TMUX": "/tmp/tmux-1000/default,123,0"}, true},
		{"STY set", map[string]string{"STY": "12345.pts-0"}, true},
		{"TERM=tmux-256color", map[string]string{"TERM": "tmux-256color"}, true},
		{"TERM=screen-256color", map[string]string{"TERM": "screen-256color"}, true},
		{"TERM=screen", map[string]string{"TERM": "screen"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all multiplexer env vars
			t.Setenv("TMUX", "")
			t.Setenv("STY", "")
			t.Setenv("TERM", "")
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			require.Equal(t, tt.want, inMultiplexer())
		})
	}
}

func BenchmarkSpinner_ModeSpinnerOnly(b *testing.B) {
	s := New(ModeSpinnerOnly, lipgloss.NewStyle())
	for b.Loop() {
		_ = s.View()
	}
}

func BenchmarkSpinner_ModeBoth(b *testing.B) {
	s := New(ModeBoth, lipgloss.NewStyle())
	for b.Loop() {
		_ = s.View()
	}
}
