package util

func MapStringStringRemoveKeys(m map[string]string, keys []string) {
	for _, key := range keys {
		delete(m, key)
	}
}
