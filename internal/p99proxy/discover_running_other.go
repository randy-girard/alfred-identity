//go:build !darwin

package p99proxy

func discoverRunningDataDirs() []string {
	return nil
}
