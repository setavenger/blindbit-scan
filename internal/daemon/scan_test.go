package daemon

import (
	"testing"

	"github.com/setavenger/blindbit-scan/internal/config"
)

func TestSyncBlockIntensiveBlocks(t *testing.T) {
	t.Logf("Starting %s", "TestSyncBlockIntensiveBlocks")
	config.SetupConfigs("./test_dir")
	config.LabelCount = 0
	d, err := SetupDaemon("./non-existent-file-path-to-create-new-wallet")
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("labels: %d", len(d.Wallet.Labels))
	t.Logf("step - %s", "001")
	utxos, err := d.syncBlock(200769)
	// utxos, err := d.syncBlock(197546)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("found %d utxos", len(utxos))
	t.Logf("step - %s", "end")
}

func TestSyncToTipIntensiveBlocks(t *testing.T) {
	t.Log("Syncing to tip")
	var targetBlock uint64 = 197546
	config.SetupConfigs("./test_dir")
	d, err := SetupDaemon("./non-existent-file-path-to-create-new-wallet")
	if err != nil {
		t.Fatal(err)
	}
	// manipulate wallet such that last scan height was right before the target block
	t.Log("Syncing to tip")
	d.Wallet.LastScanHeight = targetBlock - 1
	d.Wallet.BirthHeight = targetBlock - 1

	t.Log("Syncing to tip")
	err = d.SyncToTip(targetBlock)
	if err != nil {
		t.Fatal(err)
	}
}
