package updater

import (
	"os"
	"testing"
)

func TestIsHomebrewCaskroomPath(t *testing.T) {
	tests := []struct {
		name   string
		statFn func(string) (os.FileInfo, error)
		want   bool
	}{
		{
			name: "apple silicon caskroom exists",
			statFn: func(path string) (os.FileInfo, error) {
				if path == "/opt/homebrew/Caskroom/litelens" {
					// Simulate directory exists
					return nil, nil
				}
				return nil, os.ErrNotExist
			},
			want: true,
		},
		{
			name: "intel caskroom exists",
			statFn: func(path string) (os.FileInfo, error) {
				if path == "/usr/local/Caskroom/litelens" {
					// Simulate directory exists
					return nil, nil
				}
				return nil, os.ErrNotExist
			},
			want: true,
		},
		{
			name: "both caskroom directories exist",
			statFn: func(path string) (os.FileInfo, error) {
				// Simulate both exist (shouldn't happen normally, but test the OR logic)
				return nil, nil
			},
			want: true,
		},
		{
			name: "no homebrew installed (manual install)",
			statFn: func(path string) (os.FileInfo, error) {
				// Simulate neither directory exists
				return nil, os.ErrNotExist
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isHomebrewCaskroomPathChecked(tt.statFn)
			if got != tt.want {
				t.Errorf("isHomebrewCaskroomPathChecked() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsWingetManagedPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "winget packages dir",
			path: `C:\Users\alice\AppData\Local\Microsoft\WinGet\Packages\litelensapp.LiteLens_abc123\litelens.exe`,
			want: true,
		},
		{
			name: "winget links shim",
			path: `C:\Users\alice\AppData\Local\Microsoft\WinGet\Links\litelens.exe`,
			want: true,
		},
		{
			name: "case-insensitive match",
			path: `C:\Users\alice\AppData\Local\microsoft\winget\packages\litelensapp.LiteLens_abc123\litelens.exe`,
			want: true,
		},
		{
			name: "manual download",
			path: `C:\Users\alice\Downloads\litelens-windows-amd64.exe`,
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isWingetManagedPath(tt.path); got != tt.want {
				t.Errorf("isWingetManagedPath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
