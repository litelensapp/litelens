// Package version holds the sentinel value shared between the app's version
// string (see the root version.go, injected via ldflags for releases) and any
// package that needs to special-case unversioned development builds.
package version

// Dev is the default app version reported by development builds. Release
// builds override it via ldflags with a real semver tag.
const Dev = "dev"
