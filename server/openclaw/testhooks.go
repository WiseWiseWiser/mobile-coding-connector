package openclaw

// SetTestDataDir redirects openclaw config/state reads and writes for tests.
func SetTestDataDir(dir string) {
	_testDataDir = dir
}