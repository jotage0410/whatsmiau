package whatsmiau

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/puzpuzpuz/xsync/v4"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"

	"github.com/verbeux-ai/whatsmiau/models"
)

var (
	testStatusViewer = types.NewJID("111111111111111", types.HiddenUserServer)
	testStatusAt     = time.Unix(1794666450, 0)
)

// statusEventsHarness runs events through the instance handlers and collects what would be posted
// to the webhook, as JSON.
func statusEventsHarness(t *testing.T, events ...string) (*Whatsmiau, *models.Instance, func() []map[string]any) {
	t.Helper()
	s := &Whatsmiau{clients: xsync.NewMap[string, *whatsmeow.Client](), emitter: make(chan emitter, 10)}
	s.clients.Store("inst", &whatsmeow.Client{})
	enabled := true
	instance := &models.Instance{ID: "inst", Webhook: models.InstanceWebhook{Enabled: &enabled, Url: "http://webhook.test", Events: events}}
	collect := func() []map[string]any {
		var out []map[string]any
		for {
			select {
			case e := <-s.emitter:
				raw, err := json.Marshal(e.data)
				if err != nil {
					t.Fatalf("marshal webhook: %v", err)
				}
				var body map[string]any
				if err := json.Unmarshal(raw, &body); err != nil {
					t.Fatalf("unmarshal webhook: %v", err)
				}
				out = append(out, body)
			default:
				return out
			}
		}
	}
	return s, instance, collect
}

// incoming builds the event whatsmeow dispatches for a decrypted message.
func incoming(chat types.JID, id string, msg *waE2E.Message) *events.Message {
	evt := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{Chat: chat, Sender: testStatusViewer, IsGroup: chat.Server != types.DefaultUserServer},
			ID:            id,
			Timestamp:     testStatusAt,
		},
		RawMessage: msg,
	}
	return evt.UnwrapRaw()
}

func path(body map[string]any, keys ...string) any {
	var cur any = body
	for _, k := range keys {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = m[k]
	}
	return cur
}

// A group stanza can bring the sender key in a pairwise part, which whatsmeow emits as a message of
// its own next to the real content. It used to reach the webhook as messageType "unknown".
func TestKeyDistributionOnlyIsNotEmitted(t *testing.T) {
	s, instance, collect := statusEventsHarness(t, "MESSAGES_UPSERT")
	group := types.NewJID("120363000000000001", types.GroupServer)
	skdm := &waE2E.SenderKeyDistributionMessage{GroupID: proto.String(group.String()), AxolotlSenderKeyDistributionMessage: []byte{1, 2, 3}}

	s.handleMessageEvent("inst", instance, incoming(group, "K1", &waE2E.Message{SenderKeyDistributionMessage: skdm, MessageContextInfo: &waE2E.MessageContextInfo{}}), enabledWebhookEvents(instance.Webhook))
	if got := collect(); len(got) != 0 {
		t.Fatalf("a sender key distribution reached the webhook: %v", got)
	}

	// With content next to it, the content is emitted as before.
	s.handleMessageEvent("inst", instance, incoming(group, "K2", &waE2E.Message{SenderKeyDistributionMessage: skdm, Conversation: proto.String("oi")}), enabledWebhookEvents(instance.Webhook))
	got := collect()
	if len(got) != 1 || path(got[0], "data", "messageType") != "conversation" {
		t.Fatalf("webhooks = %v, want one conversation", got)
	}
}
