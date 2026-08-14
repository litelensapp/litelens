package updater

import "testing"

func TestIsHomebrewCaskroomPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "apple silicon caskroom",
			path: "/opt/homebrew/Caskroom/litelens/1.2.3/litelens.app/Contents/MacOS/litelens",
			want: true,
		},
		{
			name: "intel caskroom",
			path: "/usr/local/Caskroom/litelens/1.2.3/litelens.app/Contents/MacOS/litelens",
			want: true,
		},
		{
			name: "manual install.sh copy",
			path: "/Applications/LiteLens.app/Contents/MacOS/litelens",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsHomebrewCaskroomPath(tt.path); got != tt.want {
				t.Errorf("IsHomebrewCaskroomPath(%q) = %v, want %v", tt.path, got, tt.want)
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
