package app

import (
	"context"
	"fmt"
	"strings"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func (s *NodeSettingsService) LoadNetworkSettings(ctx context.Context, target NodeSettingsTarget) (NodeNetworkSettings, error) {
	cfg, err := s.loadConfig(ctx, target, generated.AdminMessage_NETWORK_CONFIG, "get_config.network")
	if err != nil {
		return NodeNetworkSettings{}, err
	}
	network := cfg.GetNetwork()
	if network == nil {
		return NodeNetworkSettings{}, fmt.Errorf("network config payload is empty")
	}

	settings := NodeNetworkSettings{
		NodeID:           strings.TrimSpace(target.NodeID),
		WifiEnabled:      network.GetWifiEnabled(),
		WifiSSID:         network.GetWifiSsid(),
		WifiPSK:          network.GetWifiPsk(),
		NTPServer:        network.GetNtpServer(),
		EthernetEnabled:  network.GetEthEnabled(),
		AddressMode:      int32(network.GetAddressMode()),
		RsyslogServer:    network.GetRsyslogServer(),
		EnabledProtocols: network.GetEnabledProtocols(),
		IPv6Enabled:      network.GetIpv6Enabled(),
	}
	if ipv4 := network.GetIpv4Config(); ipv4 != nil {
		settings.IPv4Address = ipv4.GetIp()
		settings.IPv4Gateway = ipv4.GetGateway()
		settings.IPv4Subnet = ipv4.GetSubnet()
		settings.IPv4DNS = ipv4.GetDns()
	}

	return settings, nil
}

func (s *NodeSettingsService) SaveNetworkSettings(ctx context.Context, target NodeSettingsTarget, settings NodeNetworkSettings) error {
	return s.saveConfig(ctx, target, "set_config.network", &generated.Config{
		PayloadVariant: &generated.Config_Network{
			Network: &generated.Config_NetworkConfig{
				WifiEnabled:      settings.WifiEnabled,
				WifiSsid:         settings.WifiSSID,
				WifiPsk:          settings.WifiPSK,
				NtpServer:        strings.TrimSpace(settings.NTPServer),
				EthEnabled:       settings.EthernetEnabled,
				AddressMode:      generated.Config_NetworkConfig_AddressMode(settings.AddressMode),
				RsyslogServer:    strings.TrimSpace(settings.RsyslogServer),
				EnabledProtocols: settings.EnabledProtocols,
				Ipv6Enabled:      settings.IPv6Enabled,
				Ipv4Config: &generated.Config_NetworkConfig_IpV4Config{
					Ip:      settings.IPv4Address,
					Gateway: settings.IPv4Gateway,
					Subnet:  settings.IPv4Subnet,
					Dns:     settings.IPv4DNS,
				},
			},
		},
	})
}

func (s *NodeSettingsService) LoadSerialSettings(ctx context.Context, target NodeSettingsTarget) (NodeSerialSettings, error) {
	cfg, err := s.loadModuleConfig(ctx, target, generated.AdminMessage_SERIAL_CONFIG, "get_module_config.serial")
	if err != nil {
		return NodeSerialSettings{}, err
	}
	serial := cfg.GetSerial()
	if serial == nil {
		return NodeSerialSettings{}, fmt.Errorf("serial module config payload is empty")
	}

	return NodeSerialSettings{
		NodeID:                    strings.TrimSpace(target.NodeID),
		Enabled:                   serial.GetEnabled(),
		EchoEnabled:               serial.GetEcho(),
		RXGPIO:                    serial.GetRxd(),
		TXGPIO:                    serial.GetTxd(),
		Baud:                      int32(serial.GetBaud()),
		Timeout:                   serial.GetTimeout(),
		Mode:                      int32(serial.GetMode()),
		OverrideConsoleSerialPort: serial.GetOverrideConsoleSerialPort(),
	}, nil
}

func (s *NodeSettingsService) SaveSerialSettings(ctx context.Context, target NodeSettingsTarget, settings NodeSerialSettings) error {
	return s.saveModuleConfig(ctx, target, "set_module_config.serial", &generated.ModuleConfig{
		PayloadVariant: &generated.ModuleConfig_Serial{
			Serial: &generated.ModuleConfig_SerialConfig{
				Enabled:                   settings.Enabled,
				Echo:                      settings.EchoEnabled,
				Rxd:                       settings.RXGPIO,
				Txd:                       settings.TXGPIO,
				Baud:                      generated.ModuleConfig_SerialConfig_Serial_Baud(settings.Baud),
				Timeout:                   settings.Timeout,
				Mode:                      generated.ModuleConfig_SerialConfig_Serial_Mode(settings.Mode),
				OverrideConsoleSerialPort: settings.OverrideConsoleSerialPort,
			},
		},
	})
}

func (s *NodeSettingsService) LoadExternalNotificationSettings(ctx context.Context, target NodeSettingsTarget) (NodeExternalNotificationSettings, error) {
	cfg, err := s.loadModuleConfig(ctx, target, generated.AdminMessage_EXTNOTIF_CONFIG, "get_module_config.external_notification")
	if err != nil {
		return NodeExternalNotificationSettings{}, err
	}
	notification := cfg.GetExternalNotification()
	if notification == nil {
		return NodeExternalNotificationSettings{}, fmt.Errorf("external notification module config payload is empty")
	}
	ringtone, err := s.LoadRingtone(ctx, target)
	if err != nil {
		return NodeExternalNotificationSettings{}, err
	}

	return NodeExternalNotificationSettings{
		NodeID:             strings.TrimSpace(target.NodeID),
		Enabled:            notification.GetEnabled(),
		OutputMS:           notification.GetOutputMs(),
		OutputGPIO:         notification.GetOutput(),
		OutputVibraGPIO:    notification.GetOutputVibra(),
		OutputBuzzerGPIO:   notification.GetOutputBuzzer(),
		OutputActiveHigh:   notification.GetActive(),
		AlertMessageLED:    notification.GetAlertMessage(),
		AlertMessageVibra:  notification.GetAlertMessageVibra(),
		AlertMessageBuzzer: notification.GetAlertMessageBuzzer(),
		AlertBellLED:       notification.GetAlertBell(),
		AlertBellVibra:     notification.GetAlertBellVibra(),
		AlertBellBuzzer:    notification.GetAlertBellBuzzer(),
		UsePWMBuzzer:       notification.GetUsePwm(),
		NagTimeoutSecs:     notification.GetNagTimeout(),
		Ringtone:           ringtone,
		UseI2SAsBuzzer:     notification.GetUseI2SAsBuzzer(),
	}, nil
}

func (s *NodeSettingsService) SaveExternalNotificationSettings(ctx context.Context, target NodeSettingsTarget, settings NodeExternalNotificationSettings) error {
	return s.runEditSettingsWrite(ctx, target, "set_module_config.external_notification", func(saveCtx context.Context, nodeNum uint32) error {
		if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "set_module_config.external_notification", &generated.AdminMessage{
			PayloadVariant: &generated.AdminMessage_SetModuleConfig{
				SetModuleConfig: &generated.ModuleConfig{
					PayloadVariant: &generated.ModuleConfig_ExternalNotification{
						ExternalNotification: &generated.ModuleConfig_ExternalNotificationConfig{
							Enabled:            settings.Enabled,
							OutputMs:           settings.OutputMS,
							Output:             settings.OutputGPIO,
							OutputVibra:        settings.OutputVibraGPIO,
							OutputBuzzer:       settings.OutputBuzzerGPIO,
							Active:             settings.OutputActiveHigh,
							AlertMessage:       settings.AlertMessageLED,
							AlertMessageVibra:  settings.AlertMessageVibra,
							AlertMessageBuzzer: settings.AlertMessageBuzzer,
							AlertBell:          settings.AlertBellLED,
							AlertBellVibra:     settings.AlertBellVibra,
							AlertBellBuzzer:    settings.AlertBellBuzzer,
							UsePwm:             settings.UsePWMBuzzer,
							NagTimeout:         settings.NagTimeoutSecs,
							UseI2SAsBuzzer:     settings.UseI2SAsBuzzer,
						},
					},
				},
			},
		}); err != nil {
			return fmt.Errorf("set external notification module config: %w", err)
		}

		if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "set_ringtone_message", &generated.AdminMessage{
			PayloadVariant: &generated.AdminMessage_SetRingtoneMessage{SetRingtoneMessage: settings.Ringtone},
		}); err != nil {
			return fmt.Errorf("set ringtone message: %w", err)
		}

		return nil
	})
}

func (s *NodeSettingsService) LoadStoreForwardSettings(ctx context.Context, target NodeSettingsTarget) (NodeStoreForwardSettings, error) {
	cfg, err := s.loadModuleConfig(ctx, target, generated.AdminMessage_STOREFORWARD_CONFIG, "get_module_config.store_forward")
	if err != nil {
		return NodeStoreForwardSettings{}, err
	}
	storeForward := cfg.GetStoreForward()
	if storeForward == nil {
		return NodeStoreForwardSettings{}, fmt.Errorf("store forward module config payload is empty")
	}

	return NodeStoreForwardSettings{
		NodeID:              strings.TrimSpace(target.NodeID),
		Enabled:             storeForward.GetEnabled(),
		Heartbeat:           storeForward.GetHeartbeat(),
		Records:             storeForward.GetRecords(),
		HistoryReturnMax:    storeForward.GetHistoryReturnMax(),
		HistoryReturnWindow: storeForward.GetHistoryReturnWindow(),
		IsServer:            storeForward.GetIsServer(),
	}, nil
}

func (s *NodeSettingsService) SaveStoreForwardSettings(ctx context.Context, target NodeSettingsTarget, settings NodeStoreForwardSettings) error {
	return s.saveModuleConfig(ctx, target, "set_module_config.store_forward", &generated.ModuleConfig{
		PayloadVariant: &generated.ModuleConfig_StoreForward{
			StoreForward: &generated.ModuleConfig_StoreForwardConfig{
				Enabled:             settings.Enabled,
				Heartbeat:           settings.Heartbeat,
				Records:             settings.Records,
				HistoryReturnMax:    settings.HistoryReturnMax,
				HistoryReturnWindow: settings.HistoryReturnWindow,
				IsServer:            settings.IsServer,
			},
		},
	})
}

func (s *NodeSettingsService) LoadTelemetrySettings(ctx context.Context, target NodeSettingsTarget) (NodeTelemetrySettings, error) {
	cfg, err := s.loadModuleConfig(ctx, target, generated.AdminMessage_TELEMETRY_CONFIG, "get_module_config.telemetry")
	if err != nil {
		return NodeTelemetrySettings{}, err
	}
	telemetry := cfg.GetTelemetry()
	if telemetry == nil {
		return NodeTelemetrySettings{}, fmt.Errorf("telemetry module config payload is empty")
	}

	return NodeTelemetrySettings{
		NodeID:                        strings.TrimSpace(target.NodeID),
		DeviceUpdateInterval:          telemetry.GetDeviceUpdateInterval(),
		EnvironmentUpdateInterval:     telemetry.GetEnvironmentUpdateInterval(),
		EnvironmentMeasurementEnabled: telemetry.GetEnvironmentMeasurementEnabled(),
		EnvironmentScreenEnabled:      telemetry.GetEnvironmentScreenEnabled(),
		EnvironmentDisplayFahrenheit:  telemetry.GetEnvironmentDisplayFahrenheit(),
		AirQualityEnabled:             telemetry.GetAirQualityEnabled(),
		AirQualityInterval:            telemetry.GetAirQualityInterval(),
		PowerMeasurementEnabled:       telemetry.GetPowerMeasurementEnabled(),
		PowerUpdateInterval:           telemetry.GetPowerUpdateInterval(),
		PowerScreenEnabled:            telemetry.GetPowerScreenEnabled(),
		HealthMeasurementEnabled:      telemetry.GetHealthMeasurementEnabled(),
		HealthUpdateInterval:          telemetry.GetHealthUpdateInterval(),
		HealthScreenEnabled:           telemetry.GetHealthScreenEnabled(),
		DeviceTelemetryEnabled:        telemetry.GetDeviceTelemetryEnabled(),
		AirQualityScreenEnabled:       telemetry.GetAirQualityScreenEnabled(),
	}, nil
}

func (s *NodeSettingsService) SaveTelemetrySettings(ctx context.Context, target NodeSettingsTarget, settings NodeTelemetrySettings) error {
	return s.saveModuleConfig(ctx, target, "set_module_config.telemetry", &generated.ModuleConfig{
		PayloadVariant: &generated.ModuleConfig_Telemetry{
			Telemetry: &generated.ModuleConfig_TelemetryConfig{
				DeviceUpdateInterval:          settings.DeviceUpdateInterval,
				EnvironmentUpdateInterval:     settings.EnvironmentUpdateInterval,
				EnvironmentMeasurementEnabled: settings.EnvironmentMeasurementEnabled,
				EnvironmentScreenEnabled:      settings.EnvironmentScreenEnabled,
				EnvironmentDisplayFahrenheit:  settings.EnvironmentDisplayFahrenheit,
				AirQualityEnabled:             settings.AirQualityEnabled,
				AirQualityInterval:            settings.AirQualityInterval,
				PowerMeasurementEnabled:       settings.PowerMeasurementEnabled,
				PowerUpdateInterval:           settings.PowerUpdateInterval,
				PowerScreenEnabled:            settings.PowerScreenEnabled,
				HealthMeasurementEnabled:      settings.HealthMeasurementEnabled,
				HealthUpdateInterval:          settings.HealthUpdateInterval,
				HealthScreenEnabled:           settings.HealthScreenEnabled,
				DeviceTelemetryEnabled:        settings.DeviceTelemetryEnabled,
				AirQualityScreenEnabled:       settings.AirQualityScreenEnabled,
			},
		},
	})
}

func (s *NodeSettingsService) LoadCannedMessageSettings(ctx context.Context, target NodeSettingsTarget) (NodeCannedMessageSettings, error) {
	cfg, err := s.loadModuleConfig(ctx, target, generated.AdminMessage_CANNEDMSG_CONFIG, "get_module_config.canned_message")
	if err != nil {
		return NodeCannedMessageSettings{}, err
	}
	canned := cfg.GetCannedMessage()
	if canned == nil {
		return NodeCannedMessageSettings{}, fmt.Errorf("canned message module config payload is empty")
	}
	messages, err := s.LoadCannedMessages(ctx, target)
	if err != nil {
		return NodeCannedMessageSettings{}, err
	}

	return NodeCannedMessageSettings{
		NodeID:                strings.TrimSpace(target.NodeID),
		Rotary1Enabled:        canned.GetRotary1Enabled(),
		InputBrokerPinA:       canned.GetInputbrokerPinA(),
		InputBrokerPinB:       canned.GetInputbrokerPinB(),
		InputBrokerPinPress:   canned.GetInputbrokerPinPress(),
		InputBrokerEventCW:    int32(canned.GetInputbrokerEventCw()),
		InputBrokerEventCCW:   int32(canned.GetInputbrokerEventCcw()),
		InputBrokerEventPress: int32(canned.GetInputbrokerEventPress()),
		UpDown1Enabled:        canned.GetUpdown1Enabled(),
		//nolint:staticcheck // Android parity still requires these deprecated fields until upstream removes them.
		Enabled: canned.Enabled,
		//nolint:staticcheck // Android parity still requires these deprecated fields until upstream removes them.
		AllowInputSource: canned.AllowInputSource,
		SendBell:         canned.GetSendBell(),
		Messages:         messages,
	}, nil
}

func (s *NodeSettingsService) SaveCannedMessageSettings(ctx context.Context, target NodeSettingsTarget, settings NodeCannedMessageSettings) error {
	return s.runEditSettingsWrite(ctx, target, "set_module_config.canned_message", func(saveCtx context.Context, nodeNum uint32) error {
		if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "set_module_config.canned_message", &generated.AdminMessage{
			PayloadVariant: &generated.AdminMessage_SetModuleConfig{
				SetModuleConfig: &generated.ModuleConfig{
					PayloadVariant: &generated.ModuleConfig_CannedMessage{
						CannedMessage: &generated.ModuleConfig_CannedMessageConfig{
							Rotary1Enabled:        settings.Rotary1Enabled,
							InputbrokerPinA:       settings.InputBrokerPinA,
							InputbrokerPinB:       settings.InputBrokerPinB,
							InputbrokerPinPress:   settings.InputBrokerPinPress,
							InputbrokerEventCw:    generated.ModuleConfig_CannedMessageConfig_InputEventChar(settings.InputBrokerEventCW),
							InputbrokerEventCcw:   generated.ModuleConfig_CannedMessageConfig_InputEventChar(settings.InputBrokerEventCCW),
							InputbrokerEventPress: generated.ModuleConfig_CannedMessageConfig_InputEventChar(settings.InputBrokerEventPress),
							Updown1Enabled:        settings.UpDown1Enabled,
							Enabled:               settings.Enabled,
							AllowInputSource:      settings.AllowInputSource,
							SendBell:              settings.SendBell,
						},
					},
				},
			},
		}); err != nil {
			return fmt.Errorf("set canned message module config: %w", err)
		}

		if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "set_canned_message_module_messages", &generated.AdminMessage{
			PayloadVariant: &generated.AdminMessage_SetCannedMessageModuleMessages{
				SetCannedMessageModuleMessages: settings.Messages,
			},
		}); err != nil {
			return fmt.Errorf("set canned message messages: %w", err)
		}

		return nil
	})
}

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

func (s *NodeSettingsService) LoadRemoteHardwareSettings(ctx context.Context, target NodeSettingsTarget) (NodeRemoteHardwareSettings, error) {
	cfg, err := s.loadModuleConfig(ctx, target, generated.AdminMessage_REMOTEHARDWARE_CONFIG, "get_module_config.remote_hardware")
	if err != nil {
		return NodeRemoteHardwareSettings{}, err
	}
	remoteHardware := cfg.GetRemoteHardware()
	if remoteHardware == nil {
		return NodeRemoteHardwareSettings{}, fmt.Errorf("remote hardware module config payload is empty")
	}
	settings := NodeRemoteHardwareSettings{
		NodeID:                  strings.TrimSpace(target.NodeID),
		Enabled:                 remoteHardware.GetEnabled(),
		AllowUndefinedPinAccess: remoteHardware.GetAllowUndefinedPinAccess(),
	}
	for _, pin := range remoteHardware.GetAvailablePins() {
		if pin == nil {
			continue
		}
		settings.AvailablePins = append(settings.AvailablePins, pin.GetGpioPin())
	}

	return settings, nil
}

func (s *NodeSettingsService) SaveRemoteHardwareSettings(ctx context.Context, target NodeSettingsTarget, settings NodeRemoteHardwareSettings) error {
	availablePins := make([]*generated.RemoteHardwarePin, 0, len(settings.AvailablePins))
	for _, pin := range settings.AvailablePins {
		availablePins = append(availablePins, &generated.RemoteHardwarePin{GpioPin: pin})
	}

	return s.saveModuleConfig(ctx, target, "set_module_config.remote_hardware", &generated.ModuleConfig{
		PayloadVariant: &generated.ModuleConfig_RemoteHardware{
			RemoteHardware: &generated.ModuleConfig_RemoteHardwareConfig{
				Enabled:                 settings.Enabled,
				AllowUndefinedPinAccess: settings.AllowUndefinedPinAccess,
				AvailablePins:           availablePins,
			},
		},
	})
}

func (s *NodeSettingsService) LoadNeighborInfoSettings(ctx context.Context, target NodeSettingsTarget) (NodeNeighborInfoSettings, error) {
	cfg, err := s.loadModuleConfig(ctx, target, generated.AdminMessage_NEIGHBORINFO_CONFIG, "get_module_config.neighbor_info")
	if err != nil {
		return NodeNeighborInfoSettings{}, err
	}
	neighborInfo := cfg.GetNeighborInfo()
	if neighborInfo == nil {
		return NodeNeighborInfoSettings{}, fmt.Errorf("neighbor info module config payload is empty")
	}

	return NodeNeighborInfoSettings{
		NodeID:             strings.TrimSpace(target.NodeID),
		Enabled:            neighborInfo.GetEnabled(),
		UpdateIntervalSecs: neighborInfo.GetUpdateInterval(),
		TransmitOverLoRa:   neighborInfo.GetTransmitOverLora(),
	}, nil
}

func (s *NodeSettingsService) SaveNeighborInfoSettings(ctx context.Context, target NodeSettingsTarget, settings NodeNeighborInfoSettings) error {
	return s.saveModuleConfig(ctx, target, "set_module_config.neighbor_info", &generated.ModuleConfig{
		PayloadVariant: &generated.ModuleConfig_NeighborInfo{
			NeighborInfo: &generated.ModuleConfig_NeighborInfoConfig{
				Enabled:          settings.Enabled,
				UpdateInterval:   settings.UpdateIntervalSecs,
				TransmitOverLora: settings.TransmitOverLoRa,
			},
		},
	})
}

func (s *NodeSettingsService) LoadAmbientLightingSettings(ctx context.Context, target NodeSettingsTarget) (NodeAmbientLightingSettings, error) {
	cfg, err := s.loadModuleConfig(ctx, target, generated.AdminMessage_AMBIENTLIGHTING_CONFIG, "get_module_config.ambient_lighting")
	if err != nil {
		return NodeAmbientLightingSettings{}, err
	}
	ambientLighting := cfg.GetAmbientLighting()
	if ambientLighting == nil {
		return NodeAmbientLightingSettings{}, fmt.Errorf("ambient lighting module config payload is empty")
	}

	return NodeAmbientLightingSettings{
		NodeID:   strings.TrimSpace(target.NodeID),
		LEDState: ambientLighting.GetLedState(),
		Current:  ambientLighting.GetCurrent(),
		Red:      ambientLighting.GetRed(),
		Green:    ambientLighting.GetGreen(),
		Blue:     ambientLighting.GetBlue(),
	}, nil
}

func (s *NodeSettingsService) SaveAmbientLightingSettings(ctx context.Context, target NodeSettingsTarget, settings NodeAmbientLightingSettings) error {
	return s.saveModuleConfig(ctx, target, "set_module_config.ambient_lighting", &generated.ModuleConfig{
		PayloadVariant: &generated.ModuleConfig_AmbientLighting{
			AmbientLighting: &generated.ModuleConfig_AmbientLightingConfig{
				LedState: settings.LEDState,
				Current:  settings.Current,
				Red:      settings.Red,
				Green:    settings.Green,
				Blue:     settings.Blue,
			},
		},
	})
}

func (s *NodeSettingsService) LoadDetectionSensorSettings(ctx context.Context, target NodeSettingsTarget) (NodeDetectionSensorSettings, error) {
	cfg, err := s.loadModuleConfig(ctx, target, generated.AdminMessage_DETECTIONSENSOR_CONFIG, "get_module_config.detection_sensor")
	if err != nil {
		return NodeDetectionSensorSettings{}, err
	}
	detectionSensor := cfg.GetDetectionSensor()
	if detectionSensor == nil {
		return NodeDetectionSensorSettings{}, fmt.Errorf("detection sensor module config payload is empty")
	}

	return NodeDetectionSensorSettings{
		NodeID:               strings.TrimSpace(target.NodeID),
		Enabled:              detectionSensor.GetEnabled(),
		MinimumBroadcastSecs: detectionSensor.GetMinimumBroadcastSecs(),
		StateBroadcastSecs:   detectionSensor.GetStateBroadcastSecs(),
		SendBell:             detectionSensor.GetSendBell(),
		Name:                 detectionSensor.GetName(),
		MonitorPin:           detectionSensor.GetMonitorPin(),
		DetectionTriggerType: int32(detectionSensor.GetDetectionTriggerType()),
		UsePullup:            detectionSensor.GetUsePullup(),
	}, nil
}

func (s *NodeSettingsService) SaveDetectionSensorSettings(ctx context.Context, target NodeSettingsTarget, settings NodeDetectionSensorSettings) error {
	return s.saveModuleConfig(ctx, target, "set_module_config.detection_sensor", &generated.ModuleConfig{
		PayloadVariant: &generated.ModuleConfig_DetectionSensor{
			DetectionSensor: &generated.ModuleConfig_DetectionSensorConfig{
				Enabled:              settings.Enabled,
				MinimumBroadcastSecs: settings.MinimumBroadcastSecs,
				StateBroadcastSecs:   settings.StateBroadcastSecs,
				SendBell:             settings.SendBell,
				Name:                 settings.Name,
				MonitorPin:           settings.MonitorPin,
				DetectionTriggerType: generated.ModuleConfig_DetectionSensorConfig_TriggerType(settings.DetectionTriggerType),
				UsePullup:            settings.UsePullup,
			},
		},
	})
}

func (s *NodeSettingsService) LoadPaxcounterSettings(ctx context.Context, target NodeSettingsTarget) (NodePaxcounterSettings, error) {
	cfg, err := s.loadModuleConfig(ctx, target, generated.AdminMessage_PAXCOUNTER_CONFIG, "get_module_config.paxcounter")
	if err != nil {
		return NodePaxcounterSettings{}, err
	}
	paxcounter := cfg.GetPaxcounter()
	if paxcounter == nil {
		return NodePaxcounterSettings{}, fmt.Errorf("paxcounter module config payload is empty")
	}

	return NodePaxcounterSettings{
		NodeID:             strings.TrimSpace(target.NodeID),
		Enabled:            paxcounter.GetEnabled(),
		UpdateIntervalSecs: paxcounter.GetPaxcounterUpdateInterval(),
		WifiThreshold:      paxcounter.GetWifiThreshold(),
		BLEThreshold:       paxcounter.GetBleThreshold(),
	}, nil
}

func (s *NodeSettingsService) SavePaxcounterSettings(ctx context.Context, target NodeSettingsTarget, settings NodePaxcounterSettings) error {
	return s.saveModuleConfig(ctx, target, "set_module_config.paxcounter", &generated.ModuleConfig{
		PayloadVariant: &generated.ModuleConfig_Paxcounter{
			Paxcounter: &generated.ModuleConfig_PaxcounterConfig{
				Enabled:                  settings.Enabled,
				PaxcounterUpdateInterval: settings.UpdateIntervalSecs,
				WifiThreshold:            settings.WifiThreshold,
				BleThreshold:             settings.BLEThreshold,
			},
		},
	})
}

func (s *NodeSettingsService) LoadStatusMessageSettings(ctx context.Context, target NodeSettingsTarget) (NodeStatusMessageSettings, error) {
	cfg, err := s.loadModuleConfig(ctx, target, generated.AdminMessage_STATUSMESSAGE_CONFIG, "get_module_config.status_message")
	if err != nil {
		return NodeStatusMessageSettings{}, err
	}
	statusMessage := cfg.GetStatusmessage()
	if statusMessage == nil {
		return NodeStatusMessageSettings{}, fmt.Errorf("status message module config payload is empty")
	}

	return NodeStatusMessageSettings{
		NodeID:     strings.TrimSpace(target.NodeID),
		NodeStatus: statusMessage.GetNodeStatus(),
	}, nil
}

func (s *NodeSettingsService) SaveStatusMessageSettings(ctx context.Context, target NodeSettingsTarget, settings NodeStatusMessageSettings) error {
	return s.saveModuleConfig(ctx, target, "set_module_config.status_message", &generated.ModuleConfig{
		PayloadVariant: &generated.ModuleConfig_Statusmessage{
			Statusmessage: &generated.ModuleConfig_StatusMessageConfig{
				NodeStatus: settings.NodeStatus,
			},
		},
	})
}
