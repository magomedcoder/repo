package handler

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestReadReceiveCommandsFlushZero(t *testing.T) {
	old := strings.Repeat("0", 40)
	newHash := strings.Repeat("a", 40)
	line := old + " " + newHash + " refs/heads/main\x00report-status"
	pkt := pktLine(line) + "0000" + "PACK"

	cmds, err := readReceiveCommands(bytes.NewReader([]byte(pkt)))
	if err != nil {
		t.Fatal(err)
	}

	if len(cmds) != 1 {
		t.Fatalf("cmds %d", len(cmds))
	}

	if cmds[0][0] != old || cmds[0][1] != newHash || cmds[0][2] != "refs/heads/main" {
		t.Fatalf("cmd %+v", cmds[0])
	}
}

func TestReadReceiveCommandsEmptyFlush(t *testing.T) {
	cmds, err := readReceiveCommands(bytes.NewReader([]byte("0000PACK")))
	if err != nil {
		t.Fatal(err)
	}

	if len(cmds) != 0 {
		t.Fatalf("cmds %d", len(cmds))
	}
}

func pktLine(s string) string {
	return fmt.Sprintf("%04x", len(s)+4) + s
}
