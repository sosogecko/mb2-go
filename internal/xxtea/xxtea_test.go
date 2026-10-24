package xxtea

import "testing"

func TestEncryptStringNotEmpty(t *testing.T) {
	got := EncryptString(`[{"id":1}]`, "BWAjPhtokenFhShxnc5")
	if got == "" {
		t.Fatal("encrypt returned empty")
	}
}

func TestEncryptEmpty(t *testing.T) {
	if EncryptString("", "key") != "" {
		t.Fatal("empty input should return empty")
	}
}
