package ui

import "fyne.io/fyne/v2"

type basicAppWrapper struct {
	fyne.App
}

type appRunQuitSpy struct {
	fyne.App
	runCalls  int
	quitCalls int
}

func (a *appRunQuitSpy) Run() {
	a.runCalls++
}

func (a *appRunQuitSpy) Quit() {
	a.quitCalls++
}

type appRunWindowSpy struct {
	fyne.App
	runCalls      int
	createdWindow *windowSpy
}

func (a *appRunWindowSpy) Run() {
	a.runCalls++
}

func (a *appRunWindowSpy) NewWindow(title string) fyne.Window {
	window := &windowSpy{Window: a.App.NewWindow(title)}
	a.createdWindow = window

	return window
}

type trayAppSpy struct {
	fyne.App
	trayMenu *fyne.Menu
	trayIcon fyne.Resource
}

func (a *trayAppSpy) SetSystemTrayMenu(menu *fyne.Menu) {
	a.trayMenu = menu
}

func (a *trayAppSpy) SetSystemTrayIcon(icon fyne.Resource) {
	a.trayIcon = icon
}

func (a *trayAppSpy) SetSystemTrayWindow(fyne.Window) {}

type windowSpy struct {
	fyne.Window
	showCalls      int
	hideCalls      int
	focusCalls     int
	closeIntercept func()
}

func (w *windowSpy) Show() {
	w.showCalls++
	if w.Window != nil {
		w.Window.Show()
	}
}

func (w *windowSpy) Hide() {
	w.hideCalls++
	if w.Window != nil {
		w.Window.Hide()
	}
}

func (w *windowSpy) RequestFocus() {
	w.focusCalls++
	if w.Window != nil {
		w.Window.RequestFocus()
	}
}

func (w *windowSpy) SetCloseIntercept(fn func()) {
	w.closeIntercept = fn
	if w.Window != nil {
		w.Window.SetCloseIntercept(fn)
	}
}

type lifecycleSpy struct {
	onEnteredForeground func()
	onExitedForeground  func()
	onStarted           func()
	onStopped           func()
}

func (l *lifecycleSpy) SetOnEnteredForeground(fn func()) {
	l.onEnteredForeground = fn
}

func (l *lifecycleSpy) SetOnExitedForeground(fn func()) {
	l.onExitedForeground = fn
}

func (l *lifecycleSpy) SetOnStarted(fn func()) {
	l.onStarted = fn
}

func (l *lifecycleSpy) SetOnStopped(fn func()) {
	l.onStopped = fn
}

type lifecycleAppSpy struct {
	fyne.App
	lifecycle fyne.Lifecycle
}

func (a *lifecycleAppSpy) Lifecycle() fyne.Lifecycle {
	if a.lifecycle != nil {
		return a.lifecycle
	}

	return a.App.Lifecycle()
}
