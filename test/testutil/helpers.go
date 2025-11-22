package testutil

// BoolPtr returns a pointer to the given bool value.
// This is useful for testing optional boolean fields.
func BoolPtr(b bool) *bool {
	return &b
}

// StringPtr returns a pointer to the given string value.
// This is useful for testing optional string fields.
func StringPtr(s string) *string {
	return &s
}

// IntPtr returns a pointer to the given int value.
// This is useful for testing optional int fields.
func IntPtr(i int) *int {
	return &i
}
