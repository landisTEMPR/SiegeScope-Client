package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// SiegeScopeTheme is a custom theme matching the SiegeScope website
type SiegeScopeTheme struct{}

var _ fyne.Theme = (*SiegeScopeTheme)(nil)

// Color definitions matching the website //255, 119, 51
var (
	orangePrimary = color.NRGBA{R: 255, G: 119, B: 51, A: 225}
	redAccent     = color.NRGBA{R: 220, G: 38, B: 38, A: 255}

	// Background colors
	darkBackground = color.NRGBA{R: 17, G: 17, B: 27, A: 255}
	darkSurface    = color.NRGBA{R: 26, G: 26, B: 46, A: 255}
	darkInput      = color.NRGBA{R: 35, G: 35, B: 55, A: 255}
	darkHover      = color.NRGBA{R: 45, G: 45, B: 65, A: 255}

	// Text colors
	whiteText       = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	grayText        = color.NRGBA{R: 156, G: 163, B: 175, A: 255}
	placeholderText = color.NRGBA{R: 107, G: 114, B: 128, A: 255}

	// Status colors
	greenSuccess = color.NRGBA{R: 34, G: 197, B: 94, A: 255}
	yellowWarn   = color.NRGBA{R: 234, G: 179, B: 8, A: 255}
	redError     = color.NRGBA{R: 239, G: 68, B: 68, A: 255}

	// Other
	separator = color.NRGBA{R: 55, G: 55, B: 75, A: 255}
	shadow    = color.NRGBA{R: 0, G: 0, B: 0, A: 150}
)

func (t *SiegeScopeTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {

	switch name {
	case theme.ColorNamePrimary:
		return orangePrimary
	case theme.ColorNameFocus:
		return orangePrimary
	case theme.ColorNameSelection:
		return color.NRGBA{R: 0, G: 255, B: 0, A: 100}
	case theme.ColorNameBackground:
		return darkBackground
	case theme.ColorNameOverlayBackground:
		return darkSurface
	case theme.ColorNameMenuBackground:
		return darkSurface
	case theme.ColorNameInputBackground:
		return darkInput
	case theme.ColorNameButton:
		return orangePrimary
	case theme.ColorNameDisabledButton:
		return color.NRGBA{R: 70, G: 70, B: 90, A: 255}
	case theme.ColorNameHover:
		return darkHover
	case theme.ColorNamePressed:
		return redAccent
	case theme.ColorNameForeground:
		return whiteText
	case theme.ColorNameDisabled:
		return grayText
	case theme.ColorNamePlaceHolder:
		return placeholderText
	case theme.ColorNameSuccess:
		return greenSuccess
	case theme.ColorNameWarning:
		return yellowWarn
	case theme.ColorNameError:
		return redError
	case theme.ColorNameSeparator:
		return separator
	case theme.ColorNameShadow:
		return shadow
	case theme.ColorNameScrollBar:
		return color.NRGBA{R: 80, G: 80, B: 100, A: 255}
	case theme.ColorNameInputBorder:
		return color.NRGBA{R: 70, G: 70, B: 90, A: 255}
	case theme.ColorNameHeaderBackground:
		return darkSurface
	default:
		return theme.DarkTheme().Color(name, variant)
	}
}

func (t *SiegeScopeTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DarkTheme().Font(style)
}

func (t *SiegeScopeTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DarkTheme().Icon(name)
}

func (t *SiegeScopeTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNamePadding:
		return 6
	case theme.SizeNameInlineIcon:
		return 20
	case theme.SizeNameScrollBar:
		return 10
	case theme.SizeNameScrollBarSmall:
		return 4
	case theme.SizeNameText:
		return 14
	case theme.SizeNameHeadingText:
		return 20
	case theme.SizeNameSubHeadingText:
		return 16
	case theme.SizeNameCaptionText:
		return 12
	case theme.SizeNameInputBorder:
		return 1
	case theme.SizeNameInputRadius:
		return 8
	case theme.SizeNameSelectionRadius:
		return 4
	default:
		return theme.DarkTheme().Size(name)
	}
}
