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
	ConfigSnapshot   *connectors.ConfigSnapshot
	ConfigCompleteID uint32
	WantConfigReady  bool
}

type Codec interface {
	EncodeWantConfig() ([]byte, error)
	EncodeHeartbeat() ([]byte, error)
	EncodeText(chatKey, text string) ([]byte, error)
	DecodeFromRadio(payload []byte) (DecodedFrame, error)
}
