package ui

import (
	"fmt"
	"image"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	qrcode "github.com/skip2/go-qrcode"
)

const (
	shareQRCodeImageSize = 320
	shareModalWidth      = 620
	shareModalHeight     = 560
)

type qrShareModalPayload struct {
	Title string
	URL   string
}

func showInfoModal(dep RuntimeDependencies, title, message string) {
	title = strings.TrimSpace(title)
	message = strings.TrimSpace(message)
	if title == "" || message == "" {
		return
	}

	window := currentRuntimeWindow(dep)
	if dep.UIHooks.ShowInfoDialog != nil {
		dep.UIHooks.ShowInfoDialog(title, message, window)

		return
	}
	if window == nil {
		return
	}

	dialog.ShowInformation(title, message, window)
}

func showBusyDialog(window fyne.Window, title, message string) dialog.Dialog {
	if window == nil {
		return nil
	}

	progress := widget.NewProgressBarInfinite()
	progress.Start()
	content := container.NewVBox(
		widget.NewLabel(strings.TrimSpace(message)),
		progress,
	)
	d := dialog.NewCustomWithoutButtons(strings.TrimSpace(title), content, window)
	d.Resize(fyne.NewSize(420, 140))
	d.Show()

	return d
}

func showQRCodeShareModal(window fyne.Window, payload qrShareModalPayload) {
	if window == nil {
		return
	}

	urlText := strings.TrimSpace(payload.URL)
	if urlText == "" {
		return
	}

	var qrErr error
	qrImage, err := generateQRCodeImage(urlText)
	if err != nil {
		qrErr = err
	}

	urlEntry := widget.NewMultiLineEntry()
	urlEntry.SetText(urlText)
	urlEntry.Wrapping = fyne.TextWrapBreak
	urlEntry.Disable()
	urlEntry.SetMinRowsVisible(4)

	qrBox := buildQRCodeContent(qrImage, qrErr)
	copyStatus := widget.NewLabel("")
	closeButton := widget.NewButton("Close", nil)

	content := container.NewBorder(
		nil,
		container.NewHBox(
			copyStatus,
			layout.NewSpacer(),
			widget.NewButton("Copy URL", func() {
				if err := copyTextToClipboard(urlText); err != nil {
					copyStatus.SetText("Copy failed: " + err.Error())

					return
				}
				copyStatus.SetText("URL copied to clipboard.")
			}),
			closeButton,
		),
		nil,
		nil,
		container.NewVBox(
			qrBox,
			widget.NewLabel("Shareable URL"),
			urlEntry,
		),
	)

	modal := dialog.NewCustomWithoutButtons(strings.TrimSpace(payload.Title), content, window)
	closeButton.OnTapped = modal.Hide
	modal.Resize(fyne.NewSize(shareModalWidth, shareModalHeight))
	modal.Show()
}

func buildQRCodeContent(qrImage image.Image, qrErr error) fyne.CanvasObject {
	if qrImage != nil {
		img := canvas.NewImageFromImage(qrImage)
		img.FillMode = canvas.ImageFillContain
		img.SetMinSize(fyne.NewSize(shareQRCodeImageSize, shareQRCodeImageSize))

		return container.NewCenter(img)
	}

	message := "QR code is unavailable."
	if qrErr != nil {
		message = fmt.Sprintf("QR code generation failed: %v", qrErr)
	}

	label := widget.NewLabel(message)
	label.Wrapping = fyne.TextWrapWord

	return container.NewCenter(container.NewVBox(widget.NewLabel("QR code"), label))
}

func generateQRCodeImage(content string) (image.Image, error) {
	code, err := qrcode.New(strings.TrimSpace(content), qrcode.Medium)
	if err != nil {
		return nil, err
	}

	return code.Image(960), nil
}
