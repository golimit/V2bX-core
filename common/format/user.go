package format

// UserTag builds the limiter/counter key for a user on a node.
// Avoids fmt.Sprintf on the connection hot path.
func UserTag(tag string, uuid string) string {
	return tag + "|" + uuid
}
