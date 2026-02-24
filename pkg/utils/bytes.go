package utils

// ConvertToFixedLength34 converts a byte slice to a fixed length array of 34 bytes
func ConvertToFixedLength34(data []byte) [34]byte {
	var result [34]byte
	copy(result[:], data)
	return result
}
