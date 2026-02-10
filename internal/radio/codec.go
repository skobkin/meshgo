package radio

import (
	"github.com/skobkin/meshgo/internal/connectors"
	"github.com/skobkin/meshgo/internal/domain"
)

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

type EncodedText struct {
	Payload         []byte
	DeviceMessageID string
	WantAck         bool
}

type Codec interface {
	EncodeWantConfig() ([]byte, error)
	EncodeHeartbeat() ([]byte, error)
	EncodeText(chatKey, text string) (EncodedText, error)
	DecodeFromRadio(payload []byte) (DecodedFrame, error)
}
