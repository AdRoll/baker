package baker

func terminalWidth() uint {
	// On windows we assume, for now, a 120 char wide terminal.
	// FIXME: find a better way to obtain terminal size on windows platforms.
	return 120
}
