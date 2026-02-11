package radio

import (
	"github.com/skobkin/meshgo/internal/connectors"
	"github.com/skobkin/meshgo/internal/domain"
)

// DecodedFrame is a parsed inbound radio frame with optional event payloads.
type DecodedFrame struct {
	Raw              []byte
	NodeUpdate       *domain.NodeUpdate
	Channels         *domain.ChannelList
	TextMessage      *domain.ChatMessage
	MessageStatus    *domain.MessageStatusUpdate
	ConfigSnapshot   *connectors.ConfigSnapshot
	ConfigCompleteID uint32
	WantConfigReady  bool
}

// EncodedText contains an outbound text frame and its tracking metadata.
type EncodedText struct {
	Payload         []byte
	DeviceMessageID string
	WantAck         bool
}

// Codec translates between transport frames and domain events.
type Codec interface {
	EncodeWantConfig() ([]byte, error)
	EncodeHeartbeat() ([]byte, error)
	EncodeText(chatKey, text string) (EncodedText, error)
	DecodeFromRadio(payload []byte) (DecodedFrame, error)
}
