package ui

import (
	"fmt"
	"sync"

	meshapp "github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/bus"
	"github.com/skobkin/meshgo/internal/connectors"
)

func startUIEventListeners(
	messageBus bus.MessageBus,
	onConnStatus func(connectors.ConnectionStatus),
	onNodeInfo func(),
) func() {
	if messageBus == nil {
		appLogger.Debug("skipping UI event listeners: message bus is nil")

		return func() {}
	}

	connSub := messageBus.Subscribe(connectors.TopicConnStatus)
	nodeSub := messageBus.Subscribe(connectors.TopicNodeInfo)
	appLogger.Debug(
		"subscribed to UI bus topics",
		"topics", []string{connectors.TopicConnStatus, connectors.TopicNodeInfo},
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
				status, ok := raw.(connectors.ConnectionStatus)
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

	go func() {
		for {
			select {
			case <-done:
				return
			case _, ok := <-nodeSub:
				if !ok {
					appLogger.Debug("node info subscription closed")

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

	return func() {
		stopOnce.Do(func() {
			appLogger.Debug("stopping UI event listeners")
			close(done)
			messageBus.Unsubscribe(connSub, connectors.TopicConnStatus)
			messageBus.Unsubscribe(nodeSub, connectors.TopicNodeInfo)
		})
	}
}

func startUpdateSnapshotListener(
	snapshots <-chan meshapp.UpdateSnapshot,
	onSnapshot func(meshapp.UpdateSnapshot),
) func() {
	if snapshots == nil {
		appLogger.Debug("skipping update snapshot listener: channel is nil")

		return func() {}
	}

	done := make(chan struct{})
	var stopOnce sync.Once

	go func() {
		for {
			select {
			case <-done:
				return
			case snapshot, ok := <-snapshots:
				if !ok {
					appLogger.Debug("update snapshot channel closed")

					return
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
		})
	}
}
