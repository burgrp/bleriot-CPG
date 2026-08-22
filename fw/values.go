package main

func normalizeBool(value int32) int32 {
	if value == 0 {
		return 0
	}
	return 1
}

func rgbBytes(value int32) [3]byte {
	return [3]byte{
		byte(value >> 8),
		byte(value >> 16),
		byte(value),
	}
}
