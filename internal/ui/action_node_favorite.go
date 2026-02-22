package ui

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"

	"github.com/skobkin/meshgo/internal/domain"
)

var nodeFavoriteShowErrorDialog = dialog.ShowError

func handleNodeFavoriteAction(window fyne.Window, dep RuntimeDependencies, node domain.Node, favorite bool) {
	if window == nil {
		return
	}
	if dep.Actions.NodeFavorite == nil {
		nodeFavoriteShowErrorDialog(fmt.Errorf("favorite action is unavailable: radio service is not configured"), window)

		return
	}
	go func(nodeID string, wantFavorite bool) {
		if err := dep.Actions.NodeFavorite.SetFavorite(context.Background(), nodeID, wantFavorite); err != nil {
			fyne.Do(func() {
				nodeFavoriteShowErrorDialog(err, window)
			})
		}
	}(node.NodeID, favorite)
}
