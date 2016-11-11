package platform

func setupClipboard() bool {
	return false
}

// Stub
func getClipboardData() []byte {
	return nil
}

func putClipboardData(data []byte) error {
	return nil
}

func watchClipboard(out chan []byte) {

}
