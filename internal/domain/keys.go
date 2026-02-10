package domain

import "fmt"

func ChatKeyForChannel(index int) string {
	return fmt.Sprintf("channel:%d", index)
}

func ChatKeyForDM(nodeID string) string {
	return "dm:" + nodeID
}
