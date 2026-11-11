package markdown

func addSyntaxWarning(warnings []Warning, code, message string, rng SourceRange) []Warning {
	if len(warnings) >= warningLimit {
		return warnings
	}
	return append(warnings, Warning{Code: code, Message: message, Range: rng})
}
