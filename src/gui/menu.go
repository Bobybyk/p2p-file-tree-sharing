package gui

import (
	"fyne.io/fyne/v2"
	"os"
	"protocoles-internet-2023/config"
)

func MakeMenu() *fyne.MainMenu {
	quitItem := fyne.NewMenuItem("Quit", func() {
		os.Exit(0)
	})
	fileCategory := fyne.NewMenu("File", quitItem)

	debugCheckbox := fyne.NewMenuItem("Debug", nil)
	debugCheckbox.Action = func() {
		config.SetDebug(!debugCheckbox.Checked)
		debugCheckbox.Checked = !debugCheckbox.Checked
	}
	debugCheckbox.Checked = true

	extendedDebug := fyne.NewMenuItem("Extended debug", nil)
	extendedDebug.Action = func() {
		config.SetDebugSpam(!extendedDebug.Checked)
		extendedDebug.Checked = !extendedDebug.Checked
	}
	debugCheckbox.Checked = true

	loggingCategory := fyne.NewMenu("Logging", debugCheckbox, extendedDebug)

	return fyne.NewMainMenu(fileCategory, loggingCategory)
}
