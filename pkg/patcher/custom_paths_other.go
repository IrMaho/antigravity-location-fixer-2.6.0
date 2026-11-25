//go:build !windows

package patcher

func findRegistryAndDriveInstallDirs() []string {
	return nil
}
