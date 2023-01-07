package TextColors

func AllColors() *map[string]string {
	colors := make(map[string]string)
	colors["reset"] = "\033[0m"
	colors["Bold"] = "\033[1m"
	colors["Dim"] = "\033[2m"
	colors["Underlined"] = "\033[4m"
	colors["Blink"] = "\033[5m"
	colors["Reverse"] = "\033[7m"
	colors["Hidden"] = "\033[8m"

	colors["ResetBold"] = "\033[21m"
	colors["ResetDim"] = "\033[22m"
	colors["ResetUnderlined"] = "\033[24m"
	colors["ResetBlink"] = "\033[25m"
	colors["ResetReverse"] = "\033[27m"
	colors["ResetHidden"] = "\033[28m"

	colors["Default"] = "\033[39m"
	colors["Black"] = "\033[30m"
	colors["Red"] = "\033[31m"
	colors["Green"] = "\033[32m"
	colors["Yellow"] = "\033[33m"
	colors["Blue"] = "\033[34m"
	colors["Magenta"] = "\033[35m"
	colors["Cyan"] = "\033[36m"
	colors["LightGray"] = "\033[37m"
	colors["DarkGray"] = "\033[90m"
	colors["LightRed"] = "\033[91m"
	colors["LightGreen"] = "\033[92m"
	colors["LightYellow"] = "\033[93m"
	colors["LightBlue"] = "\033[94m"
	colors["LightMagenta"] = "\033[95m"
	colors["LightCyan"] = "\033[96m"
	colors["White"] = "\033[97m"

	return &colors
}
