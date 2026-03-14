package ui

import (
	"fmt"

	"fyne.io/fyne/v2"

	meshapp "github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/domain"
)

func handleNodeShareContactAction(window fyne.Window, dep RuntimeDependencies, node domain.Node) {
	if window == nil {
		window = currentRuntimeWindow(dep)
	}
	if window == nil {
		return
	}

	rawURL, err := meshapp.BuildSharedContactURL(node)
	if err != nil {
		showErrorModal(dep, fmt.Errorf("build shared contact URL: %w", err))

		return
	}

	showQRCodeShareModal(window, qrShareModalPayload{
		Title: "Share contact",
		URL:   rawURL,
	})
}
