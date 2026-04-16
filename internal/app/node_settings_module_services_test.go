package app

import (
	"context"
	"strconv"
	"testing"

	"github.com/skobkin/meshgo/internal/bus"
	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func TestNodeSettingsServiceLoadNewModuleSettings(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "Serial",
			run: func(t *testing.T) {
				var messageBus bus.MessageBus
				sender := stubAdminSender{
					send: func(to uint32, _ uint32, wantResponse bool, payload *generated.AdminMessage) (string, error) {
						if payload.GetGetModuleConfigRequest() != generated.AdminMessage_SERIAL_CONFIG || !wantResponse {
							t.Fatalf("unexpected serial load request")
						}
						publishAdminReply(messageBus, to, 1, &generated.AdminMessage{
							PayloadVariant: &generated.AdminMessage_GetModuleConfigResponse{
								GetModuleConfigResponse: &generated.ModuleConfig{
									PayloadVariant: &generated.ModuleConfig_Serial{
										Serial: &generated.ModuleConfig_SerialConfig{
											Enabled: true, Echo: true, Rxd: 7, Txd: 8,
											Baud: generated.ModuleConfig_SerialConfig_BAUD_9600,
											Mode: generated.ModuleConfig_SerialConfig_TEXTMSG,
										},
									},
								},
							},
						})

						return "1", nil
					},
				}
				service, busRef := newTestNodeSettingsService(t, sender, false)
				messageBus = busRef
				ctx, cancel := contextWithTimeout(t)
				defer cancel()
				settings, err := service.LoadSerialSettings(ctx, mustHexNodeTarget())
				if err != nil {
					t.Fatalf("load serial settings: %v", err)
				}
				if settings.Baud != int32(generated.ModuleConfig_SerialConfig_BAUD_9600) || settings.Mode != int32(generated.ModuleConfig_SerialConfig_TEXTMSG) {
					t.Fatalf("unexpected serial settings: %+v", settings)
				}
			},
		},
		{
			name: "ExternalNotification",
			run: func(t *testing.T) {
				var messageBus bus.MessageBus
				call := 0
				sender := stubAdminSender{
					send: func(to uint32, _ uint32, wantResponse bool, payload *generated.AdminMessage) (string, error) {
						call++
						switch call {
						case 1:
							if payload.GetGetModuleConfigRequest() != generated.AdminMessage_EXTNOTIF_CONFIG || !wantResponse {
								t.Fatalf("unexpected external notification config request")
							}
							publishAdminReply(messageBus, to, 1, &generated.AdminMessage{
								PayloadVariant: &generated.AdminMessage_GetModuleConfigResponse{
									GetModuleConfigResponse: &generated.ModuleConfig{
										PayloadVariant: &generated.ModuleConfig_ExternalNotification{
											ExternalNotification: &generated.ModuleConfig_ExternalNotificationConfig{
												Enabled: true, OutputMs: 10, Output: 1, NagTimeout: 60, UsePwm: true,
											},
										},
									},
								},
							})
						case 2:
							if !payload.GetGetRingtoneRequest() || !wantResponse {
								t.Fatalf("unexpected ringtone request")
							}
							publishAdminReply(messageBus, to, 2, &generated.AdminMessage{
								PayloadVariant: &generated.AdminMessage_GetRingtoneResponse{GetRingtoneResponse: "beep"},
							})
						default:
							t.Fatalf("unexpected send call %d", call)
						}

						return strconv.Itoa(call), nil
					},
				}
				service, busRef := newTestNodeSettingsService(t, sender, false)
				messageBus = busRef
				ctx, cancel := contextWithTimeout(t)
				defer cancel()
				settings, err := service.LoadExternalNotificationSettings(ctx, mustHexNodeTarget())
				if err != nil {
					t.Fatalf("load external notification settings: %v", err)
				}
				if settings.OutputMS != 10 || settings.Ringtone != "beep" || !settings.UsePWMBuzzer {
					t.Fatalf("unexpected external notification settings: %+v", settings)
				}
			},
		},
		{
			name: "StoreForward",
			run: func(t *testing.T) {
				var messageBus bus.MessageBus
				sender := stubAdminSender{
					send: func(to uint32, _ uint32, wantResponse bool, payload *generated.AdminMessage) (string, error) {
						if payload.GetGetModuleConfigRequest() != generated.AdminMessage_STOREFORWARD_CONFIG || !wantResponse {
							t.Fatalf("unexpected store forward request")
						}
						publishAdminReply(messageBus, to, 1, &generated.AdminMessage{
							PayloadVariant: &generated.AdminMessage_GetModuleConfigResponse{
								GetModuleConfigResponse: &generated.ModuleConfig{
									PayloadVariant: &generated.ModuleConfig_StoreForward{
										StoreForward: &generated.ModuleConfig_StoreForwardConfig{
											Enabled: true, Heartbeat: true, Records: 50, IsServer: true,
										},
									},
								},
							},
						})

						return "1", nil
					},
				}
				service, busRef := newTestNodeSettingsService(t, sender, false)
				messageBus = busRef
				ctx, cancel := contextWithTimeout(t)
				defer cancel()
				settings, err := service.LoadStoreForwardSettings(ctx, mustHexNodeTarget())
				if err != nil {
					t.Fatalf("load store forward settings: %v", err)
				}
				if !settings.Enabled || !settings.IsServer || settings.Records != 50 {
					t.Fatalf("unexpected store forward settings: %+v", settings)
				}
			},
		},
		{
			name: "Telemetry",
			run: func(t *testing.T) {
				var messageBus bus.MessageBus
				sender := stubAdminSender{
					send: func(to uint32, _ uint32, wantResponse bool, payload *generated.AdminMessage) (string, error) {
						if payload.GetGetModuleConfigRequest() != generated.AdminMessage_TELEMETRY_CONFIG || !wantResponse {
							t.Fatalf("unexpected telemetry request")
						}
						publishAdminReply(messageBus, to, 1, &generated.AdminMessage{
							PayloadVariant: &generated.AdminMessage_GetModuleConfigResponse{
								GetModuleConfigResponse: &generated.ModuleConfig{
									PayloadVariant: &generated.ModuleConfig_Telemetry{
										Telemetry: &generated.ModuleConfig_TelemetryConfig{
											DeviceUpdateInterval: 600, AirQualityEnabled: true, HealthScreenEnabled: true,
										},
									},
								},
							},
						})

						return "1", nil
					},
				}
				service, busRef := newTestNodeSettingsService(t, sender, false)
				messageBus = busRef
				ctx, cancel := contextWithTimeout(t)
				defer cancel()
				settings, err := service.LoadTelemetrySettings(ctx, mustHexNodeTarget())
				if err != nil {
					t.Fatalf("load telemetry settings: %v", err)
				}
				if settings.DeviceUpdateInterval != 600 || !settings.AirQualityEnabled || !settings.HealthScreenEnabled {
					t.Fatalf("unexpected telemetry settings: %+v", settings)
				}
			},
		},
		{
			name: "CannedMessage",
			run: func(t *testing.T) {
				var messageBus bus.MessageBus
				call := 0
				sender := stubAdminSender{
					send: func(to uint32, _ uint32, wantResponse bool, payload *generated.AdminMessage) (string, error) {
						call++
						switch call {
						case 1:
							if payload.GetGetModuleConfigRequest() != generated.AdminMessage_CANNEDMSG_CONFIG || !wantResponse {
								t.Fatalf("unexpected canned message request")
							}
							publishAdminReply(messageBus, to, 1, &generated.AdminMessage{
								PayloadVariant: &generated.AdminMessage_GetModuleConfigResponse{
									GetModuleConfigResponse: &generated.ModuleConfig{
										PayloadVariant: &generated.ModuleConfig_CannedMessage{
											CannedMessage: &generated.ModuleConfig_CannedMessageConfig{
												Enabled: true, Rotary1Enabled: true,
												InputbrokerEventCw: generated.ModuleConfig_CannedMessageConfig_LEFT,
											},
										},
									},
								},
							})
						case 2:
							if !payload.GetGetCannedMessageModuleMessagesRequest() || !wantResponse {
								t.Fatalf("unexpected canned messages text request")
							}
							publishAdminReply(messageBus, to, 2, &generated.AdminMessage{
								PayloadVariant: &generated.AdminMessage_GetCannedMessageModuleMessagesResponse{
									GetCannedMessageModuleMessagesResponse: "one\ntwo",
								},
							})
						default:
							t.Fatalf("unexpected send call %d", call)
						}

						return strconv.Itoa(call), nil
					},
				}
				service, busRef := newTestNodeSettingsService(t, sender, false)
				messageBus = busRef
				ctx, cancel := contextWithTimeout(t)
				defer cancel()
				settings, err := service.LoadCannedMessageSettings(ctx, mustHexNodeTarget())
				if err != nil {
					t.Fatalf("load canned message settings: %v", err)
				}
				if !settings.Enabled || settings.Messages != "one\ntwo" || settings.InputBrokerEventCW != int32(generated.ModuleConfig_CannedMessageConfig_LEFT) {
					t.Fatalf("unexpected canned message settings: %+v", settings)
				}
			},
		},
		{
			name: "Audio",
			run: func(t *testing.T) {
				var messageBus bus.MessageBus
				sender := stubAdminSender{
					send: func(to uint32, _ uint32, wantResponse bool, payload *generated.AdminMessage) (string, error) {
						if payload.GetGetModuleConfigRequest() != generated.AdminMessage_AUDIO_CONFIG || !wantResponse {
							t.Fatalf("unexpected audio request")
						}
						publishAdminReply(messageBus, to, 1, &generated.AdminMessage{
							PayloadVariant: &generated.AdminMessage_GetModuleConfigResponse{
								GetModuleConfigResponse: &generated.ModuleConfig{
									PayloadVariant: &generated.ModuleConfig_Audio{
										Audio: &generated.ModuleConfig_AudioConfig{
											Codec2Enabled: true, PttPin: 12, Bitrate: generated.ModuleConfig_AudioConfig_CODEC2_3200,
										},
									},
								},
							},
						})

						return "1", nil
					},
				}
				service, busRef := newTestNodeSettingsService(t, sender, false)
				messageBus = busRef
				ctx, cancel := contextWithTimeout(t)
				defer cancel()
				settings, err := service.LoadAudioSettings(ctx, mustHexNodeTarget())
				if err != nil {
					t.Fatalf("load audio settings: %v", err)
				}
				if !settings.Codec2Enabled || settings.PTTPin != 12 || settings.Bitrate != int32(generated.ModuleConfig_AudioConfig_CODEC2_3200) {
					t.Fatalf("unexpected audio settings: %+v", settings)
				}
			},
		},
		{
			name: "RemoteHardware",
			run: func(t *testing.T) {
				var messageBus bus.MessageBus
				sender := stubAdminSender{
					send: func(to uint32, _ uint32, wantResponse bool, payload *generated.AdminMessage) (string, error) {
						if payload.GetGetModuleConfigRequest() != generated.AdminMessage_REMOTEHARDWARE_CONFIG || !wantResponse {
							t.Fatalf("unexpected remote hardware request")
						}
						publishAdminReply(messageBus, to, 1, &generated.AdminMessage{
							PayloadVariant: &generated.AdminMessage_GetModuleConfigResponse{
								GetModuleConfigResponse: &generated.ModuleConfig{
									PayloadVariant: &generated.ModuleConfig_RemoteHardware{
										RemoteHardware: &generated.ModuleConfig_RemoteHardwareConfig{
											Enabled:       true,
											AvailablePins: []*generated.RemoteHardwarePin{{GpioPin: 3}, {GpioPin: 4}},
										},
									},
								},
							},
						})

						return "1", nil
					},
				}
				service, busRef := newTestNodeSettingsService(t, sender, false)
				messageBus = busRef
				ctx, cancel := contextWithTimeout(t)
				defer cancel()
				settings, err := service.LoadRemoteHardwareSettings(ctx, mustHexNodeTarget())
				if err != nil {
					t.Fatalf("load remote hardware settings: %v", err)
				}
				if !settings.Enabled || len(settings.AvailablePins) != 2 || settings.AvailablePins[1] != 4 {
					t.Fatalf("unexpected remote hardware settings: %+v", settings)
				}
			},
		},
		{
			name: "NeighborInfo",
			run: func(t *testing.T) {
				var messageBus bus.MessageBus
				sender := stubAdminSender{
					send: func(to uint32, _ uint32, wantResponse bool, payload *generated.AdminMessage) (string, error) {
						if payload.GetGetModuleConfigRequest() != generated.AdminMessage_NEIGHBORINFO_CONFIG || !wantResponse {
							t.Fatalf("unexpected neighbor info request")
						}
						publishAdminReply(messageBus, to, 1, &generated.AdminMessage{
							PayloadVariant: &generated.AdminMessage_GetModuleConfigResponse{
								GetModuleConfigResponse: &generated.ModuleConfig{
									PayloadVariant: &generated.ModuleConfig_NeighborInfo{
										NeighborInfo: &generated.ModuleConfig_NeighborInfoConfig{
											Enabled: true, UpdateInterval: 3600, TransmitOverLora: true,
										},
									},
								},
							},
						})

						return "1", nil
					},
				}
				service, busRef := newTestNodeSettingsService(t, sender, false)
				messageBus = busRef
				ctx, cancel := contextWithTimeout(t)
				defer cancel()
				settings, err := service.LoadNeighborInfoSettings(ctx, mustHexNodeTarget())
				if err != nil {
					t.Fatalf("load neighbor info settings: %v", err)
				}
				if !settings.Enabled || settings.UpdateIntervalSecs != 3600 || !settings.TransmitOverLoRa {
					t.Fatalf("unexpected neighbor info settings: %+v", settings)
				}
			},
		},
		{
			name: "AmbientLighting",
			run: func(t *testing.T) {
				var messageBus bus.MessageBus
				sender := stubAdminSender{
					send: func(to uint32, _ uint32, wantResponse bool, payload *generated.AdminMessage) (string, error) {
						if payload.GetGetModuleConfigRequest() != generated.AdminMessage_AMBIENTLIGHTING_CONFIG || !wantResponse {
							t.Fatalf("unexpected ambient lighting request")
						}
						publishAdminReply(messageBus, to, 1, &generated.AdminMessage{
							PayloadVariant: &generated.AdminMessage_GetModuleConfigResponse{
								GetModuleConfigResponse: &generated.ModuleConfig{
									PayloadVariant: &generated.ModuleConfig_AmbientLighting{
										AmbientLighting: &generated.ModuleConfig_AmbientLightingConfig{
											LedState: true, Current: 10, Red: 1, Green: 2, Blue: 3,
										},
									},
								},
							},
						})

						return "1", nil
					},
				}
				service, busRef := newTestNodeSettingsService(t, sender, false)
				messageBus = busRef
				ctx, cancel := contextWithTimeout(t)
				defer cancel()
				settings, err := service.LoadAmbientLightingSettings(ctx, mustHexNodeTarget())
				if err != nil {
					t.Fatalf("load ambient lighting settings: %v", err)
				}
				if !settings.LEDState || settings.Current != 10 || settings.Blue != 3 {
					t.Fatalf("unexpected ambient lighting settings: %+v", settings)
				}
			},
		},
		{
			name: "DetectionSensor",
			run: func(t *testing.T) {
				var messageBus bus.MessageBus
				sender := stubAdminSender{
					send: func(to uint32, _ uint32, wantResponse bool, payload *generated.AdminMessage) (string, error) {
						if payload.GetGetModuleConfigRequest() != generated.AdminMessage_DETECTIONSENSOR_CONFIG || !wantResponse {
							t.Fatalf("unexpected detection sensor request")
						}
						publishAdminReply(messageBus, to, 1, &generated.AdminMessage{
							PayloadVariant: &generated.AdminMessage_GetModuleConfigResponse{
								GetModuleConfigResponse: &generated.ModuleConfig{
									PayloadVariant: &generated.ModuleConfig_DetectionSensor{
										DetectionSensor: &generated.ModuleConfig_DetectionSensorConfig{
											Enabled: true, MinimumBroadcastSecs: 30, DetectionTriggerType: generated.ModuleConfig_DetectionSensorConfig_FALLING_EDGE,
										},
									},
								},
							},
						})

						return "1", nil
					},
				}
				service, busRef := newTestNodeSettingsService(t, sender, false)
				messageBus = busRef
				ctx, cancel := contextWithTimeout(t)
				defer cancel()
				settings, err := service.LoadDetectionSensorSettings(ctx, mustHexNodeTarget())
				if err != nil {
					t.Fatalf("load detection sensor settings: %v", err)
				}
				if !settings.Enabled || settings.MinimumBroadcastSecs != 30 || settings.DetectionTriggerType != int32(generated.ModuleConfig_DetectionSensorConfig_FALLING_EDGE) {
					t.Fatalf("unexpected detection sensor settings: %+v", settings)
				}
			},
		},
		{
			name: "Paxcounter",
			run: func(t *testing.T) {
				var messageBus bus.MessageBus
				sender := stubAdminSender{
					send: func(to uint32, _ uint32, wantResponse bool, payload *generated.AdminMessage) (string, error) {
						if payload.GetGetModuleConfigRequest() != generated.AdminMessage_PAXCOUNTER_CONFIG || !wantResponse {
							t.Fatalf("unexpected paxcounter request")
						}
						publishAdminReply(messageBus, to, 1, &generated.AdminMessage{
							PayloadVariant: &generated.AdminMessage_GetModuleConfigResponse{
								GetModuleConfigResponse: &generated.ModuleConfig{
									PayloadVariant: &generated.ModuleConfig_Paxcounter{
										Paxcounter: &generated.ModuleConfig_PaxcounterConfig{
											Enabled: true, PaxcounterUpdateInterval: 900, WifiThreshold: -70, BleThreshold: -80,
										},
									},
								},
							},
						})

						return "1", nil
					},
				}
				service, busRef := newTestNodeSettingsService(t, sender, false)
				messageBus = busRef
				ctx, cancel := contextWithTimeout(t)
				defer cancel()
				settings, err := service.LoadPaxcounterSettings(ctx, mustHexNodeTarget())
				if err != nil {
					t.Fatalf("load paxcounter settings: %v", err)
				}
				if !settings.Enabled || settings.UpdateIntervalSecs != 900 || settings.BLEThreshold != -80 {
					t.Fatalf("unexpected paxcounter settings: %+v", settings)
				}
			},
		},
		{
			name: "StatusMessage",
			run: func(t *testing.T) {
				var messageBus bus.MessageBus
				sender := stubAdminSender{
					send: func(to uint32, _ uint32, wantResponse bool, payload *generated.AdminMessage) (string, error) {
						if payload.GetGetModuleConfigRequest() != generated.AdminMessage_STATUSMESSAGE_CONFIG || !wantResponse {
							t.Fatalf("unexpected status message request")
						}
						publishAdminReply(messageBus, to, 1, &generated.AdminMessage{
							PayloadVariant: &generated.AdminMessage_GetModuleConfigResponse{
								GetModuleConfigResponse: &generated.ModuleConfig{
									PayloadVariant: &generated.ModuleConfig_Statusmessage{
										Statusmessage: &generated.ModuleConfig_StatusMessageConfig{NodeStatus: "Ready"},
									},
								},
							},
						})

						return "1", nil
					},
				}
				service, busRef := newTestNodeSettingsService(t, sender, false)
				messageBus = busRef
				ctx, cancel := contextWithTimeout(t)
				defer cancel()
				settings, err := service.LoadStatusMessageSettings(ctx, mustHexNodeTarget())
				if err != nil {
					t.Fatalf("load status message settings: %v", err)
				}
				if settings.NodeStatus != "Ready" {
					t.Fatalf("unexpected status message settings: %+v", settings)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.run)
	}
}

func TestNodeSettingsServiceSaveNewModuleSettings_ImmediateStatusEvents(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "Serial",
			run: func(t *testing.T) {
				runSimpleModuleSaveTest(t, func(service *NodeSettingsService, ctx context.Context) error {
					return service.SaveSerialSettings(ctx, mustLocalNodeTarget(), NodeSerialSettings{
						NodeID: "!00000001", Enabled: true, RXGPIO: 7, TXGPIO: 8,
						Baud: int32(generated.ModuleConfig_SerialConfig_BAUD_9600),
						Mode: int32(generated.ModuleConfig_SerialConfig_TEXTMSG),
					})
				}, func(payload *generated.AdminMessage) {
					serial := payload.GetSetModuleConfig().GetSerial()
					if serial == nil || serial.GetBaud() != generated.ModuleConfig_SerialConfig_BAUD_9600 || serial.GetMode() != generated.ModuleConfig_SerialConfig_TEXTMSG {
						t.Fatalf("unexpected serial payload: %+v", serial)
					}
				})
			},
		},
		{
			name: "ExternalNotification",
			run: func(t *testing.T) {
				runEditSequenceTest(t, 4, func(service *NodeSettingsService, ctx context.Context) error {
					return service.SaveExternalNotificationSettings(ctx, mustLocalNodeTarget(), NodeExternalNotificationSettings{
						NodeID: "!00000001", Enabled: true, OutputMS: 10, Ringtone: "beep",
					})
				}, func(call int, payload *generated.AdminMessage) {
					switch call {
					case 1:
						if payload.GetSetModuleConfig().GetExternalNotification().GetOutputMs() != 10 {
							t.Fatalf("unexpected external notification payload")
						}
					case 2:
						if payload.GetSetRingtoneMessage() != "beep" {
							t.Fatalf("unexpected ringtone payload: %q", payload.GetSetRingtoneMessage())
						}
					}
				})
			},
		},
		{
			name: "StoreForward",
			run: func(t *testing.T) {
				runSimpleModuleSaveTest(t, func(service *NodeSettingsService, ctx context.Context) error {
					return service.SaveStoreForwardSettings(ctx, mustLocalNodeTarget(), NodeStoreForwardSettings{
						NodeID: "!00000001", Enabled: true, Records: 50, IsServer: true,
					})
				}, func(payload *generated.AdminMessage) {
					storeForward := payload.GetSetModuleConfig().GetStoreForward()
					if storeForward == nil || !storeForward.GetIsServer() || storeForward.GetRecords() != 50 {
						t.Fatalf("unexpected store forward payload: %+v", storeForward)
					}
				})
			},
		},
		{
			name: "Telemetry",
			run: func(t *testing.T) {
				runSimpleModuleSaveTest(t, func(service *NodeSettingsService, ctx context.Context) error {
					return service.SaveTelemetrySettings(ctx, mustLocalNodeTarget(), NodeTelemetrySettings{
						NodeID: "!00000001", DeviceUpdateInterval: 600, AirQualityEnabled: true,
					})
				}, func(payload *generated.AdminMessage) {
					telemetry := payload.GetSetModuleConfig().GetTelemetry()
					if telemetry == nil || telemetry.GetDeviceUpdateInterval() != 600 || !telemetry.GetAirQualityEnabled() {
						t.Fatalf("unexpected telemetry payload: %+v", telemetry)
					}
				})
			},
		},
		{
			name: "CannedMessage",
			run: func(t *testing.T) {
				runEditSequenceTest(t, 4, func(service *NodeSettingsService, ctx context.Context) error {
					return service.SaveCannedMessageSettings(ctx, mustLocalNodeTarget(), NodeCannedMessageSettings{
						NodeID: "!00000001", Enabled: true, Messages: "one\ntwo",
						InputBrokerEventCW: int32(generated.ModuleConfig_CannedMessageConfig_LEFT),
					})
				}, func(call int, payload *generated.AdminMessage) {
					switch call {
					case 1:
						if payload.GetSetModuleConfig().GetCannedMessage().GetInputbrokerEventCw() != generated.ModuleConfig_CannedMessageConfig_LEFT {
							t.Fatalf("unexpected canned message module payload")
						}
					case 2:
						if payload.GetSetCannedMessageModuleMessages() != "one\ntwo" {
							t.Fatalf("unexpected canned messages text payload")
						}
					}
				})
			},
		},
		{
			name: "Audio",
			run: func(t *testing.T) {
				runSimpleModuleSaveTest(t, func(service *NodeSettingsService, ctx context.Context) error {
					return service.SaveAudioSettings(ctx, mustLocalNodeTarget(), NodeAudioSettings{
						NodeID: "!00000001", Codec2Enabled: true, Bitrate: int32(generated.ModuleConfig_AudioConfig_CODEC2_3200),
					})
				}, func(payload *generated.AdminMessage) {
					audio := payload.GetSetModuleConfig().GetAudio()
					if audio == nil || audio.GetBitrate() != generated.ModuleConfig_AudioConfig_CODEC2_3200 {
						t.Fatalf("unexpected audio payload: %+v", audio)
					}
				})
			},
		},
		{
			name: "RemoteHardware",
			run: func(t *testing.T) {
				runSimpleModuleSaveTest(t, func(service *NodeSettingsService, ctx context.Context) error {
					return service.SaveRemoteHardwareSettings(ctx, mustLocalNodeTarget(), NodeRemoteHardwareSettings{
						NodeID: "!00000001", Enabled: true, AvailablePins: []uint32{3, 4},
					})
				}, func(payload *generated.AdminMessage) {
					hardware := payload.GetSetModuleConfig().GetRemoteHardware()
					if hardware == nil || len(hardware.GetAvailablePins()) != 2 || hardware.GetAvailablePins()[1].GetGpioPin() != 4 {
						t.Fatalf("unexpected remote hardware payload: %+v", hardware)
					}
				})
			},
		},
		{
			name: "NeighborInfo",
			run: func(t *testing.T) {
				runSimpleModuleSaveTest(t, func(service *NodeSettingsService, ctx context.Context) error {
					return service.SaveNeighborInfoSettings(ctx, mustLocalNodeTarget(), NodeNeighborInfoSettings{
						NodeID: "!00000001", Enabled: true, UpdateIntervalSecs: 3600, TransmitOverLoRa: true,
					})
				}, func(payload *generated.AdminMessage) {
					neighbor := payload.GetSetModuleConfig().GetNeighborInfo()
					if neighbor == nil || neighbor.GetUpdateInterval() != 3600 || !neighbor.GetTransmitOverLora() {
						t.Fatalf("unexpected neighbor info payload: %+v", neighbor)
					}
				})
			},
		},
		{
			name: "AmbientLighting",
			run: func(t *testing.T) {
				runSimpleModuleSaveTest(t, func(service *NodeSettingsService, ctx context.Context) error {
					return service.SaveAmbientLightingSettings(ctx, mustLocalNodeTarget(), NodeAmbientLightingSettings{
						NodeID: "!00000001", LEDState: true, Current: 10, Blue: 3,
					})
				}, func(payload *generated.AdminMessage) {
					lighting := payload.GetSetModuleConfig().GetAmbientLighting()
					if lighting == nil || !lighting.GetLedState() || lighting.GetBlue() != 3 {
						t.Fatalf("unexpected ambient lighting payload: %+v", lighting)
					}
				})
			},
		},
		{
			name: "DetectionSensor",
			run: func(t *testing.T) {
				runSimpleModuleSaveTest(t, func(service *NodeSettingsService, ctx context.Context) error {
					return service.SaveDetectionSensorSettings(ctx, mustLocalNodeTarget(), NodeDetectionSensorSettings{
						NodeID: "!00000001", Enabled: true, DetectionTriggerType: int32(generated.ModuleConfig_DetectionSensorConfig_FALLING_EDGE),
					})
				}, func(payload *generated.AdminMessage) {
					detection := payload.GetSetModuleConfig().GetDetectionSensor()
					if detection == nil || detection.GetDetectionTriggerType() != generated.ModuleConfig_DetectionSensorConfig_FALLING_EDGE {
						t.Fatalf("unexpected detection sensor payload: %+v", detection)
					}
				})
			},
		},
		{
			name: "Paxcounter",
			run: func(t *testing.T) {
				runSimpleModuleSaveTest(t, func(service *NodeSettingsService, ctx context.Context) error {
					return service.SavePaxcounterSettings(ctx, mustLocalNodeTarget(), NodePaxcounterSettings{
						NodeID: "!00000001", Enabled: true, UpdateIntervalSecs: 900, BLEThreshold: -80,
					})
				}, func(payload *generated.AdminMessage) {
					pax := payload.GetSetModuleConfig().GetPaxcounter()
					if pax == nil || pax.GetPaxcounterUpdateInterval() != 900 || pax.GetBleThreshold() != -80 {
						t.Fatalf("unexpected paxcounter payload: %+v", pax)
					}
				})
			},
		},
		{
			name: "StatusMessage",
			run: func(t *testing.T) {
				runSimpleModuleSaveTest(t, func(service *NodeSettingsService, ctx context.Context) error {
					return service.SaveStatusMessageSettings(ctx, mustLocalNodeTarget(), NodeStatusMessageSettings{
						NodeID: "!00000001", NodeStatus: "Ready",
					})
				}, func(payload *generated.AdminMessage) {
					status := payload.GetSetModuleConfig().GetStatusmessage()
					if status == nil || status.GetNodeStatus() != "Ready" {
						t.Fatalf("unexpected status message payload: %+v", status)
					}
				})
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.run)
	}
}

func runSimpleModuleSaveTest(
	t *testing.T,
	save func(*NodeSettingsService, context.Context) error,
	check func(*generated.AdminMessage),
) {
	t.Helper()

	runEditSequenceTest(t, 3, save, func(call int, payload *generated.AdminMessage) {
		if call == 1 {
			check(payload)
		}
	})
}

func runEditSequenceTest(
	t *testing.T,
	expectedCalls int,
	save func(*NodeSettingsService, context.Context) error,
	check func(call int, payload *generated.AdminMessage),
) {
	t.Helper()

	var messageBus bus.MessageBus
	call := 0
	packetIDs := make([]uint32, expectedCalls)
	for i := range packetIDs {
		packetIDs[i] = uint32(100 + i)
	}
	sender := stubAdminSender{
		send: func(_ uint32, _ uint32, wantResponse bool, payload *generated.AdminMessage) (string, error) {
			if wantResponse {
				t.Fatalf("expected wantResponse=false for save flow")
			}
			switch call {
			case 0:
				if !payload.GetBeginEditSettings() {
					t.Fatalf("expected begin edit settings payload")
				}
			case expectedCalls - 1:
				if !payload.GetCommitEditSettings() {
					t.Fatalf("expected commit edit settings payload")
				}
			default:
				check(call, payload)
			}
			publishSentStatus(messageBus, packetIDs[call])
			call++

			return stringFromUint32(packetIDs[call-1]), nil
		},
	}
	service, busRef := newTestNodeSettingsService(t, sender, true)
	messageBus = busRef

	ctx, cancel := contextWithTimeout(t)
	defer cancel()

	if err := save(service, ctx); err != nil {
		t.Fatalf("save settings: %v", err)
	}
	if call != expectedCalls {
		t.Fatalf("unexpected send calls count: got %d want %d", call, expectedCalls)
	}
}
