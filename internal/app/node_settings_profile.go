package app

import (
	"context"
	"fmt"
	"strings"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
	"google.golang.org/protobuf/proto"
)

func (s *NodeSettingsService) LoadRingtone(ctx context.Context, target NodeSettingsTarget) (string, error) {
	return s.loadAdminString(
		ctx,
		target,
		"get_ringtone",
		&generated.AdminMessage{
			PayloadVariant: &generated.AdminMessage_GetRingtoneRequest{GetRingtoneRequest: true},
		},
		func(message *generated.AdminMessage) string { return message.GetGetRingtoneResponse() },
	)
}

func (s *NodeSettingsService) SaveRingtone(ctx context.Context, target NodeSettingsTarget, ringtone string) error {
	return s.saveAdminString(ctx, target, "set_ringtone_message", func(value string) *generated.AdminMessage {
		return &generated.AdminMessage{
			PayloadVariant: &generated.AdminMessage_SetRingtoneMessage{SetRingtoneMessage: value},
		}
	}, ringtone)
}

func (s *NodeSettingsService) LoadCannedMessages(ctx context.Context, target NodeSettingsTarget) (string, error) {
	return s.loadAdminString(
		ctx,
		target,
		"get_canned_message_module_messages",
		&generated.AdminMessage{
			PayloadVariant: &generated.AdminMessage_GetCannedMessageModuleMessagesRequest{
				GetCannedMessageModuleMessagesRequest: true,
			},
		},
		func(message *generated.AdminMessage) string {
			return message.GetGetCannedMessageModuleMessagesResponse()
		},
	)
}

func (s *NodeSettingsService) SaveCannedMessages(ctx context.Context, target NodeSettingsTarget, messages string) error {
	return s.saveAdminString(ctx, target, "set_canned_message_module_messages", func(value string) *generated.AdminMessage {
		return &generated.AdminMessage{
			PayloadVariant: &generated.AdminMessage_SetCannedMessageModuleMessages{
				SetCannedMessageModuleMessages: value,
			},
		}
	}, messages)
}

func (s *NodeSettingsService) ExportProfile(ctx context.Context, target NodeSettingsTarget) (*generated.DeviceProfile, error) {
	user, err := s.LoadUserSettings(ctx, target)
	if err != nil {
		return nil, err
	}
	lora, err := s.LoadLoRaSettings(ctx, target)
	if err != nil {
		return nil, err
	}
	channels, err := s.LoadChannelSettings(ctx, target)
	if err != nil {
		return nil, err
	}
	deviceCfg, err := s.loadConfig(ctx, target, generated.AdminMessage_DEVICE_CONFIG, "get_config.device")
	if err != nil {
		return nil, err
	}
	positionCfg, err := s.loadConfig(ctx, target, generated.AdminMessage_POSITION_CONFIG, "get_config.position")
	if err != nil {
		return nil, err
	}
	powerCfg, err := s.loadConfig(ctx, target, generated.AdminMessage_POWER_CONFIG, "get_config.power")
	if err != nil {
		return nil, err
	}
	networkCfg, err := s.loadConfig(ctx, target, generated.AdminMessage_NETWORK_CONFIG, "get_config.network")
	if err != nil {
		return nil, err
	}
	displayCfg, err := s.loadConfig(ctx, target, generated.AdminMessage_DISPLAY_CONFIG, "get_config.display")
	if err != nil {
		return nil, err
	}
	loraCfg, err := s.loadConfig(ctx, target, generated.AdminMessage_LORA_CONFIG, "get_config.lora")
	if err != nil {
		return nil, err
	}
	bluetoothCfg, err := s.loadConfig(ctx, target, generated.AdminMessage_BLUETOOTH_CONFIG, "get_config.bluetooth")
	if err != nil {
		return nil, err
	}
	securityCfg, err := s.loadConfig(ctx, target, generated.AdminMessage_SECURITY_CONFIG, "get_config.security")
	if err != nil {
		return nil, err
	}

	mqttCfg, err := s.loadModuleConfig(ctx, target, generated.AdminMessage_MQTT_CONFIG, "get_module_config.mqtt")
	if err != nil {
		return nil, err
	}
	serialCfg, err := s.loadModuleConfig(ctx, target, generated.AdminMessage_SERIAL_CONFIG, "get_module_config.serial")
	if err != nil {
		return nil, err
	}
	externalNotificationCfg, err := s.loadModuleConfig(ctx, target, generated.AdminMessage_EXTNOTIF_CONFIG, "get_module_config.external_notification")
	if err != nil {
		return nil, err
	}
	storeForwardCfg, err := s.loadModuleConfig(ctx, target, generated.AdminMessage_STOREFORWARD_CONFIG, "get_module_config.store_forward")
	if err != nil {
		return nil, err
	}
	rangeTestCfg, err := s.loadModuleConfig(ctx, target, generated.AdminMessage_RANGETEST_CONFIG, "get_module_config.range_test")
	if err != nil {
		return nil, err
	}
	telemetryCfg, err := s.loadModuleConfig(ctx, target, generated.AdminMessage_TELEMETRY_CONFIG, "get_module_config.telemetry")
	if err != nil {
		return nil, err
	}
	cannedCfg, err := s.loadModuleConfig(ctx, target, generated.AdminMessage_CANNEDMSG_CONFIG, "get_module_config.canned_message")
	if err != nil {
		return nil, err
	}
	audioCfg, err := s.loadModuleConfig(ctx, target, generated.AdminMessage_AUDIO_CONFIG, "get_module_config.audio")
	if err != nil {
		return nil, err
	}
	remoteHardwareCfg, err := s.loadModuleConfig(ctx, target, generated.AdminMessage_REMOTEHARDWARE_CONFIG, "get_module_config.remote_hardware")
	if err != nil {
		return nil, err
	}
	neighborInfoCfg, err := s.loadModuleConfig(ctx, target, generated.AdminMessage_NEIGHBORINFO_CONFIG, "get_module_config.neighbor_info")
	if err != nil {
		return nil, err
	}
	ambientLightingCfg, err := s.loadModuleConfig(ctx, target, generated.AdminMessage_AMBIENTLIGHTING_CONFIG, "get_module_config.ambient_lighting")
	if err != nil {
		return nil, err
	}
	detectionSensorCfg, err := s.loadModuleConfig(ctx, target, generated.AdminMessage_DETECTIONSENSOR_CONFIG, "get_module_config.detection_sensor")
	if err != nil {
		return nil, err
	}
	paxcounterCfg, err := s.loadModuleConfig(ctx, target, generated.AdminMessage_PAXCOUNTER_CONFIG, "get_module_config.paxcounter")
	if err != nil {
		return nil, err
	}
	statusMessageCfg, err := s.loadModuleConfig(ctx, target, generated.AdminMessage_STATUSMESSAGE_CONFIG, "get_module_config.status_message")
	if err != nil {
		return nil, err
	}
	ringtone, err := s.LoadRingtone(ctx, target)
	if err != nil {
		return nil, err
	}
	cannedMessages, err := s.LoadCannedMessages(ctx, target)
	if err != nil {
		return nil, err
	}

	var channelURL string
	if len(channels.Channels) > 0 {
		channelURL, _ = BuildChannelShareURL(channels.Channels, lora, false)
	}

	profile := &generated.DeviceProfile{
		LongName:  strPtr(strings.TrimSpace(user.LongName)),
		ShortName: strPtr(strings.TrimSpace(user.ShortName)),
		Config: &generated.LocalConfig{
			Device:    cloneProtoMessage(deviceCfg.GetDevice()),
			Position:  cloneProtoMessage(positionCfg.GetPosition()),
			Power:     cloneProtoMessage(powerCfg.GetPower()),
			Network:   cloneProtoMessage(networkCfg.GetNetwork()),
			Display:   cloneProtoMessage(displayCfg.GetDisplay()),
			Lora:      cloneProtoMessage(loraCfg.GetLora()),
			Bluetooth: cloneProtoMessage(bluetoothCfg.GetBluetooth()),
			Security:  cloneProtoMessage(securityCfg.GetSecurity()),
		},
		ModuleConfig: &generated.LocalModuleConfig{
			Mqtt:                 cloneProtoMessage(mqttCfg.GetMqtt()),
			Serial:               cloneProtoMessage(serialCfg.GetSerial()),
			ExternalNotification: cloneProtoMessage(externalNotificationCfg.GetExternalNotification()),
			StoreForward:         cloneProtoMessage(storeForwardCfg.GetStoreForward()),
			RangeTest:            cloneProtoMessage(rangeTestCfg.GetRangeTest()),
			Telemetry:            cloneProtoMessage(telemetryCfg.GetTelemetry()),
			CannedMessage:        cloneProtoMessage(cannedCfg.GetCannedMessage()),
			Audio:                cloneProtoMessage(audioCfg.GetAudio()),
			RemoteHardware:       cloneProtoMessage(remoteHardwareCfg.GetRemoteHardware()),
			NeighborInfo:         cloneProtoMessage(neighborInfoCfg.GetNeighborInfo()),
			AmbientLighting:      cloneProtoMessage(ambientLightingCfg.GetAmbientLighting()),
			DetectionSensor:      cloneProtoMessage(detectionSensorCfg.GetDetectionSensor()),
			Paxcounter:           cloneProtoMessage(paxcounterCfg.GetPaxcounter()),
			Statusmessage:        cloneProtoMessage(statusMessageCfg.GetStatusmessage()),
		},
		Ringtone:       strPtr(ringtone),
		CannedMessages: strPtr(cannedMessages),
	}
	if channelURL != "" {
		profile.ChannelUrl = strPtr(channelURL)
	}

	return profile, nil
}

func (s *NodeSettingsService) ImportProfile(ctx context.Context, target NodeSettingsTarget, profile *generated.DeviceProfile) error {
	if profile == nil {
		return fmt.Errorf("device profile is empty")
	}

	return s.runEditSettingsWrite(ctx, target, "install_profile", func(saveCtx context.Context, nodeNum uint32) error {
		if profile.LongName != nil || profile.ShortName != nil {
			currentUser, err := s.LoadUserSettings(saveCtx, target)
			if err != nil {
				return fmt.Errorf("load user settings for profile import: %w", err)
			}
			if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "set_owner", &generated.AdminMessage{
				PayloadVariant: &generated.AdminMessage_SetOwner{
					SetOwner: &generated.User{
						Id:             strings.TrimSpace(target.NodeID),
						LongName:       firstNonEmpty(profile.GetLongName(), currentUser.LongName),
						ShortName:      firstNonEmpty(profile.GetShortName(), currentUser.ShortName),
						IsLicensed:     currentUser.HamLicensed,
						IsUnmessagable: boolPtr(currentUser.IsUnmessageable),
					},
				},
			}); err != nil {
				return fmt.Errorf("set owner from profile: %w", err)
			}
		}

		if cfg := profile.GetConfig(); cfg != nil {
			if cfg.GetDevice() != nil {
				if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "set_config.device", &generated.AdminMessage{
					PayloadVariant: &generated.AdminMessage_SetConfig{SetConfig: &generated.Config{
						PayloadVariant: &generated.Config_Device{Device: cloneProtoMessage(cfg.GetDevice())},
					}},
				}); err != nil {
					return fmt.Errorf("set device config from profile: %w", err)
				}
			}
			if cfg.GetPosition() != nil {
				if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "set_config.position", &generated.AdminMessage{
					PayloadVariant: &generated.AdminMessage_SetConfig{SetConfig: &generated.Config{
						PayloadVariant: &generated.Config_Position{Position: cloneProtoMessage(cfg.GetPosition())},
					}},
				}); err != nil {
					return fmt.Errorf("set position config from profile: %w", err)
				}
			}
			if cfg.GetPower() != nil {
				if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "set_config.power", &generated.AdminMessage{
					PayloadVariant: &generated.AdminMessage_SetConfig{SetConfig: &generated.Config{
						PayloadVariant: &generated.Config_Power{Power: cloneProtoMessage(cfg.GetPower())},
					}},
				}); err != nil {
					return fmt.Errorf("set power config from profile: %w", err)
				}
			}
			if cfg.GetNetwork() != nil {
				if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "set_config.network", &generated.AdminMessage{
					PayloadVariant: &generated.AdminMessage_SetConfig{SetConfig: &generated.Config{
						PayloadVariant: &generated.Config_Network{Network: cloneProtoMessage(cfg.GetNetwork())},
					}},
				}); err != nil {
					return fmt.Errorf("set network config from profile: %w", err)
				}
			}
			if cfg.GetDisplay() != nil {
				if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "set_config.display", &generated.AdminMessage{
					PayloadVariant: &generated.AdminMessage_SetConfig{SetConfig: &generated.Config{
						PayloadVariant: &generated.Config_Display{Display: cloneProtoMessage(cfg.GetDisplay())},
					}},
				}); err != nil {
					return fmt.Errorf("set display config from profile: %w", err)
				}
			}
			if cfg.GetLora() != nil {
				if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "set_config.lora", &generated.AdminMessage{
					PayloadVariant: &generated.AdminMessage_SetConfig{SetConfig: &generated.Config{
						PayloadVariant: &generated.Config_Lora{Lora: cloneProtoMessage(cfg.GetLora())},
					}},
				}); err != nil {
					return fmt.Errorf("set lora config from profile: %w", err)
				}
			}
			if cfg.GetBluetooth() != nil {
				if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "set_config.bluetooth", &generated.AdminMessage{
					PayloadVariant: &generated.AdminMessage_SetConfig{SetConfig: &generated.Config{
						PayloadVariant: &generated.Config_Bluetooth{Bluetooth: cloneProtoMessage(cfg.GetBluetooth())},
					}},
				}); err != nil {
					return fmt.Errorf("set bluetooth config from profile: %w", err)
				}
			}
			if cfg.GetSecurity() != nil {
				if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "set_config.security", &generated.AdminMessage{
					PayloadVariant: &generated.AdminMessage_SetConfig{SetConfig: &generated.Config{
						PayloadVariant: &generated.Config_Security{Security: cloneProtoMessage(cfg.GetSecurity())},
					}},
				}); err != nil {
					return fmt.Errorf("set security config from profile: %w", err)
				}
			}
		}

		if fixed := profile.GetFixedPosition(); fixed != nil {
			if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "set_fixed_position", &generated.AdminMessage{
				PayloadVariant: &generated.AdminMessage_SetFixedPosition{
					SetFixedPosition: cloneProtoMessage(fixed),
				},
			}); err != nil {
				return fmt.Errorf("set fixed position from profile: %w", err)
			}
		}

		if moduleCfg := profile.GetModuleConfig(); moduleCfg != nil {
			setModule := func(action string, config *generated.ModuleConfig) error {
				return s.sendAdminAndWaitStatus(saveCtx, nodeNum, action, &generated.AdminMessage{
					PayloadVariant: &generated.AdminMessage_SetModuleConfig{SetModuleConfig: config},
				})
			}
			if moduleCfg.GetMqtt() != nil {
				if err := setModule("set_module_config.mqtt", &generated.ModuleConfig{PayloadVariant: &generated.ModuleConfig_Mqtt{Mqtt: cloneProtoMessage(moduleCfg.GetMqtt())}}); err != nil {
					return fmt.Errorf("set mqtt config from profile: %w", err)
				}
			}
			if moduleCfg.GetSerial() != nil {
				if err := setModule("set_module_config.serial", &generated.ModuleConfig{PayloadVariant: &generated.ModuleConfig_Serial{Serial: cloneProtoMessage(moduleCfg.GetSerial())}}); err != nil {
					return fmt.Errorf("set serial config from profile: %w", err)
				}
			}
			if moduleCfg.GetExternalNotification() != nil {
				if err := setModule("set_module_config.external_notification", &generated.ModuleConfig{PayloadVariant: &generated.ModuleConfig_ExternalNotification{ExternalNotification: cloneProtoMessage(moduleCfg.GetExternalNotification())}}); err != nil {
					return fmt.Errorf("set external notification config from profile: %w", err)
				}
			}
			if moduleCfg.GetStoreForward() != nil {
				if err := setModule("set_module_config.store_forward", &generated.ModuleConfig{PayloadVariant: &generated.ModuleConfig_StoreForward{StoreForward: cloneProtoMessage(moduleCfg.GetStoreForward())}}); err != nil {
					return fmt.Errorf("set store forward config from profile: %w", err)
				}
			}
			if moduleCfg.GetRangeTest() != nil {
				if err := setModule("set_module_config.range_test", &generated.ModuleConfig{PayloadVariant: &generated.ModuleConfig_RangeTest{RangeTest: cloneProtoMessage(moduleCfg.GetRangeTest())}}); err != nil {
					return fmt.Errorf("set range test config from profile: %w", err)
				}
			}
			if moduleCfg.GetTelemetry() != nil {
				if err := setModule("set_module_config.telemetry", &generated.ModuleConfig{PayloadVariant: &generated.ModuleConfig_Telemetry{Telemetry: cloneProtoMessage(moduleCfg.GetTelemetry())}}); err != nil {
					return fmt.Errorf("set telemetry config from profile: %w", err)
				}
			}
			if moduleCfg.GetCannedMessage() != nil {
				if err := setModule("set_module_config.canned_message", &generated.ModuleConfig{PayloadVariant: &generated.ModuleConfig_CannedMessage{CannedMessage: cloneProtoMessage(moduleCfg.GetCannedMessage())}}); err != nil {
					return fmt.Errorf("set canned message config from profile: %w", err)
				}
			}
			if moduleCfg.GetAudio() != nil {
				if err := setModule("set_module_config.audio", &generated.ModuleConfig{PayloadVariant: &generated.ModuleConfig_Audio{Audio: cloneProtoMessage(moduleCfg.GetAudio())}}); err != nil {
					return fmt.Errorf("set audio config from profile: %w", err)
				}
			}
			if moduleCfg.GetRemoteHardware() != nil {
				if err := setModule("set_module_config.remote_hardware", &generated.ModuleConfig{PayloadVariant: &generated.ModuleConfig_RemoteHardware{RemoteHardware: cloneProtoMessage(moduleCfg.GetRemoteHardware())}}); err != nil {
					return fmt.Errorf("set remote hardware config from profile: %w", err)
				}
			}
			if moduleCfg.GetNeighborInfo() != nil {
				if err := setModule("set_module_config.neighbor_info", &generated.ModuleConfig{PayloadVariant: &generated.ModuleConfig_NeighborInfo{NeighborInfo: cloneProtoMessage(moduleCfg.GetNeighborInfo())}}); err != nil {
					return fmt.Errorf("set neighbor info config from profile: %w", err)
				}
			}
			if moduleCfg.GetAmbientLighting() != nil {
				if err := setModule("set_module_config.ambient_lighting", &generated.ModuleConfig{PayloadVariant: &generated.ModuleConfig_AmbientLighting{AmbientLighting: cloneProtoMessage(moduleCfg.GetAmbientLighting())}}); err != nil {
					return fmt.Errorf("set ambient lighting config from profile: %w", err)
				}
			}
			if moduleCfg.GetDetectionSensor() != nil {
				if err := setModule("set_module_config.detection_sensor", &generated.ModuleConfig{PayloadVariant: &generated.ModuleConfig_DetectionSensor{DetectionSensor: cloneProtoMessage(moduleCfg.GetDetectionSensor())}}); err != nil {
					return fmt.Errorf("set detection sensor config from profile: %w", err)
				}
			}
			if moduleCfg.GetPaxcounter() != nil {
				if err := setModule("set_module_config.paxcounter", &generated.ModuleConfig{PayloadVariant: &generated.ModuleConfig_Paxcounter{Paxcounter: cloneProtoMessage(moduleCfg.GetPaxcounter())}}); err != nil {
					return fmt.Errorf("set paxcounter config from profile: %w", err)
				}
			}
			if moduleCfg.GetStatusmessage() != nil {
				if err := setModule("set_module_config.status_message", &generated.ModuleConfig{PayloadVariant: &generated.ModuleConfig_Statusmessage{Statusmessage: cloneProtoMessage(moduleCfg.GetStatusmessage())}}); err != nil {
					return fmt.Errorf("set status message config from profile: %w", err)
				}
			}
		}

		if profile.Ringtone != nil {
			if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "set_ringtone_message", &generated.AdminMessage{
				PayloadVariant: &generated.AdminMessage_SetRingtoneMessage{SetRingtoneMessage: profile.GetRingtone()},
			}); err != nil {
				return fmt.Errorf("set ringtone from profile: %w", err)
			}
		}
		if profile.CannedMessages != nil {
			if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "set_canned_message_module_messages", &generated.AdminMessage{
				PayloadVariant: &generated.AdminMessage_SetCannedMessageModuleMessages{
					SetCannedMessageModuleMessages: profile.GetCannedMessages(),
				},
			}); err != nil {
				return fmt.Errorf("set canned messages from profile: %w", err)
			}
		}

		return nil
	})
}

func DecodeDeviceProfile(raw []byte) (*generated.DeviceProfile, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("device profile is empty")
	}
	var profile generated.DeviceProfile
	if err := proto.Unmarshal(raw, &profile); err != nil {
		return nil, fmt.Errorf("decode device profile: %w", err)
	}

	return &profile, nil
}

func EncodeDeviceProfile(profile *generated.DeviceProfile) ([]byte, error) {
	if profile == nil {
		return nil, fmt.Errorf("device profile is empty")
	}

	raw, err := proto.Marshal(profile)
	if err != nil {
		return nil, fmt.Errorf("encode device profile: %w", err)
	}

	return raw, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}

	return ""
}

func strPtr(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	return &value
}
