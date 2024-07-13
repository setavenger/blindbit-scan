package config

import (
	"fmt"
	"testing"
)

func TestConfigLoadFromFile(t *testing.T) {
	if err := LoadConfigs("../../blindbit.example.toml"); err != nil {
		t.Fatalf("Err: %s", err)
	}

	fmt.Printf("ScanSecretKey: %x\n", ScanSecretKey)
	fmt.Printf("SpendPubKey:   %x\n", SpendPubKey)

	fmt.Printf("LabelCount:    %d\n", LabelCount)
	fmt.Printf("BirthHeight:   %d\n", BirthHeight)
}
