package ui

import (
	"fmt"
	"sync"

	meshapp "github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/bus"
	"github.com/skobkin/meshgo/internal/radio/busmsg"
)

func startUIEventListeners(
	messageBus bus.MessageBus,
	onConnStatus func(busmsg.ConnectionStatus),
	onNodeInfo func(),
) func() {
	if messageBus == nil {
		appLogger.Debug("skipping UI event listeners: message bus is nil")

		return func() {}
	}

	connSub := messageBus.Subscribe(bus.TopicConnStatus)
	nodeCoreSub := messageBus.Subscribe(bus.TopicNodeCore)
	nodePositionSub := messageBus.Subscribe(bus.TopicNodePosition)
	nodeTelemetrySub := messageBus.Subscribe(bus.TopicNodeTelemetry)
	appLogger.Debug(
		"subscribed to UI bus topics",
		"topics", []string{bus.TopicConnStatus, bus.TopicNodeCore, bus.TopicNodePosition, bus.TopicNodeTelemetry},
	)
	done := make(chan struct{})
	var stopOnce sync.Once

	go func() {
		for {
			select {
			case <-done:
				return
			case raw, ok := <-connSub:
				if !ok {
					appLogger.Debug("connection status subscription closed")

					return
				}
				status, ok := raw.(busmsg.ConnectionStatus)
				if !ok {
					appLogger.Debug("ignoring unexpected connection status payload", "payload_type", fmt.Sprintf("%T", raw))

					continue
				}
				select {
				case <-done:
					return
				default:
				}
				if onConnStatus != nil {
					onConnStatus(status)
				}
			}
		}
	}()

	startNodeListener := func(sub bus.Subscription, topic string) {
		go func() {
			for {
				select {
				case <-done:
					return
				case _, ok := <-sub:
					if !ok {
						appLogger.Debug("node subscription closed", "topic", topic)

						return
					}
					select {
					case <-done:
						return
					default:
					}
					if onNodeInfo != nil {
						onNodeInfo()
					}
				}
			}
		}()
	}

	startNodeListener(nodeCoreSub, bus.TopicNodeCore)
	startNodeListener(nodePositionSub, bus.TopicNodePosition)
	startNodeListener(nodeTelemetrySub, bus.TopicNodeTelemetry)

	return func() {
		stopOnce.Do(func() {
			appLogger.Debug("stopping UI event listeners")
			close(done)
			messageBus.Unsubscribe(connSub, bus.TopicConnStatus)
			messageBus.Unsubscribe(nodeCoreSub, bus.TopicNodeCore)
			messageBus.Unsubscribe(nodePositionSub, bus.TopicNodePosition)
			messageBus.Unsubscribe(nodeTelemetrySub, bus.TopicNodeTelemetry)
		})
	}
}

func startUpdateSnapshotListener(
	messageBus bus.MessageBus,
	onSnapshot func(meshapp.UpdateSnapshot),
) func() {
	if messageBus == nil {
		appLogger.Debug("skipping update snapshot listener: message bus is nil")

		return func() {}
	}

	snapshotSub := messageBus.Subscribe(bus.TopicUpdateSnapshot)
	done := make(chan struct{})
	var stopOnce sync.Once

	go func() {
		for {
			select {
			case <-done:
				return
			case raw, ok := <-snapshotSub:
				if !ok {
					appLogger.Debug("update snapshot subscription closed")

					return
				}
				snapshot, ok := raw.(meshapp.UpdateSnapshot)
				if !ok {
					appLogger.Debug("ignoring unexpected update snapshot payload", "payload_type", fmt.Sprintf("%T", raw))

					continue
				}
				select {
				case <-done:
					return
				default:
				}
				if onSnapshot != nil {
					onSnapshot(snapshot)
				}
			}
		}
	}()

	return func() {
		stopOnce.Do(func() {
			appLogger.Debug("stopping update snapshot listener")
			close(done)
			messageBus.Unsubscribe(snapshotSub, bus.TopicUpdateSnapshot)
		})
	}
}
