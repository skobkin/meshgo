package radio

import (
	"github.com/skobkin/meshgo/internal/connectors"
	"github.com/skobkin/meshgo/internal/domain"
	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

// DecodedFrame is a parsed inbound radio frame with optional event payloads.
type DecodedFrame struct {
	Raw              []byte
	NodeUpdate       *domain.NodeUpdate
	Channels         *domain.ChannelList
	TextMessage      *domain.ChatMessage
	MessageStatus    *domain.MessageStatusUpdate
	ConfigSnapshot   *connectors.ConfigSnapshot
	AdminMessage     *AdminMessageEvent
	ConfigCompleteID uint32
	WantConfigReady  bool
}

// EncodedText contains an outbound text frame and its tracking metadata.
type EncodedText struct {
	Payload         []byte
	DeviceMessageID string
	WantAck         bool
}

// EncodedAdmin contains an outbound admin frame and its tracking metadata.
type EncodedAdmin struct {
	Payload         []byte
	DeviceMessageID string
}

// AdminMessageEvent is a decoded admin payload received from the mesh.
type AdminMessageEvent struct {
	From      uint32
	To        uint32
	PacketID  uint32
	RequestID uint32
	ReplyID   uint32
	Message   *generated.AdminMessage
}

// Codec translates between transport frames and domain events.
type Codec interface {
	EncodeWantConfig() ([]byte, error)
	EncodeHeartbeat() ([]byte, error)
	EncodeText(chatKey, text string) (EncodedText, error)
	EncodeAdmin(to uint32, channel uint32, wantResponse bool, payload *generated.AdminMessage) (EncodedAdmin, error)
	DecodeFromRadio(payload []byte) (DecodedFrame, error)
}
