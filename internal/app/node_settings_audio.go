package app

import (
	"context"
	"fmt"
	"strings"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func (s *NodeSettingsService) LoadAudioSettings(ctx context.Context, target NodeSettingsTarget) (NodeAudioSettings, error) {
	cfg, err := s.loadModuleConfig(ctx, target, generated.AdminMessage_AUDIO_CONFIG, "get_module_config.audio")
	if err != nil {
		return NodeAudioSettings{}, err
	}
	audio := cfg.GetAudio()
	if audio == nil {
		return NodeAudioSettings{}, fmt.Errorf("audio module config payload is empty")
	}

	return NodeAudioSettings{
		NodeID:        strings.TrimSpace(target.NodeID),
		Codec2Enabled: audio.GetCodec2Enabled(),
		PTTPin:        audio.GetPttPin(),
		Bitrate:       int32(audio.GetBitrate()),
		I2SWordSelect: audio.GetI2SWs(),
		I2SDataIn:     audio.GetI2SSd(),
		I2SDataOut:    audio.GetI2SDin(),
		I2SClock:      audio.GetI2SSck(),
	}, nil
}

func (s *NodeSettingsService) SaveAudioSettings(ctx context.Context, target NodeSettingsTarget, settings NodeAudioSettings) error {
	return s.saveModuleConfig(ctx, target, "set_module_config.audio", &generated.ModuleConfig{
		PayloadVariant: &generated.ModuleConfig_Audio{
			Audio: &generated.ModuleConfig_AudioConfig{
				Codec2Enabled: settings.Codec2Enabled,
				PttPin:        settings.PTTPin,
				Bitrate:       generated.ModuleConfig_AudioConfig_Audio_Baud(settings.Bitrate),
				I2SWs:         settings.I2SWordSelect,
				I2SSd:         settings.I2SDataIn,
				I2SDin:        settings.I2SDataOut,
				I2SSck:        settings.I2SClock,
			},
		},
	})
}
