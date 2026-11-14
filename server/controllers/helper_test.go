package controllers

import (
	"testing"

	"go.mau.fi/whatsmeow/types"
)

func TestNumberToJidAcceptsInternationalNumbers(t *testing.T) {
	validNumbers := []string{
		"5561999211277", // Brazil (13 digits)
		"13233923870",   // United States, +1 (11 digits)
		"34662418782",   // Spain, +34 (11 digits)
	}

	for _, number := range validNumbers {
		if _, err := numberToJid(number); err != nil {
			t.Errorf("numberToJid(%q) returned unexpected error: %v", number, err)
		}
	}
}

func TestNumberToJidRejectsTooShortNumbers(t *testing.T) {
	tooShort := []string{"123", "5561"}

	for _, number := range tooShort {
		if _, err := numberToJid(number); err == nil {
			t.Errorf("numberToJid(%q) expected an error, got nil", number)
		}
	}
}

func TestRevokeChatJidAcceptsStatusBroadcast(t *testing.T) {
	for _, input := range []string{"status@broadcast", " STATUS@broadcast "} {
		jid, err := revokeChatJid(input)
		if err != nil {
			t.Fatalf("revokeChatJid(%q) returned unexpected error: %v", input, err)
		}
		if *jid != types.StatusBroadcastJID {
			t.Errorf("revokeChatJid(%q) = %s, want %s", input, jid, types.StatusBroadcastJID)
		}
	}
}

func TestRevokeChatJidKeepsNumbersAndGroups(t *testing.T) {
	jid, err := revokeChatJid("5511999999999")
	if err != nil || jid.Server != types.DefaultUserServer || jid.User != "5511999999999" {
		t.Errorf("revokeChatJid(number) = %v, %v", jid, err)
	}
	jid, err = revokeChatJid("120363000000000000@g.us")
	if err != nil || jid.Server != types.GroupServer {
		t.Errorf("revokeChatJid(group) = %v, %v", jid, err)
	}
	if _, err := revokeChatJid("other@broadcast"); err == nil {
		t.Errorf("revokeChatJid(other@broadcast) expected an error from numberToJid, got nil")
	}
}

func TestGroupStatusRevokeError(t *testing.T) {
	group := types.NewJID("120363000000000001", types.GroupServer)
	if msg := groupStatusRevokeError(group, true); msg != "" {
		t.Errorf("own group status: %q, want valid", msg)
	}
	if msg := groupStatusRevokeError(group, false); msg == "" {
		t.Error("another member's group status was accepted")
	}
	for _, jid := range []types.JID{types.StatusBroadcastJID, types.NewJID("5511999999999", types.DefaultUserServer)} {
		if msg := groupStatusRevokeError(jid, true); msg == "" {
			t.Errorf("groupStatus with remoteJid %s was accepted", jid)
		}
	}
}
