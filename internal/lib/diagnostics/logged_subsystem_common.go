package diagnostics

import (
	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
)

// Provides common logging facilities for any service subsystem.
type LoggedSubsystemCommon struct {
	Logger    snx_lib_logging.Logger
	System    string
	Subsystem string
}

// Logging helper function (not to be used outside this file).
func _makeQualifiedLogArgs(
	system string,
	subsystem string,
	keyVals ...any,
) []any {

	if system != "" {

		if subsystem != "" {

			args := make([]any, 0, 4+len(keyVals))

			args = append(args,
				"subsystem", subsystem,
				"system", system,
			)

			args = append(args,
				keyVals...,
			)

			return args
		} else {

			args := make([]any, 0, 2+len(keyVals))

			args = append(args,
				"system", system,
			)

			args = append(args,
				keyVals...,
			)

			return args
		}
	} else {

		if subsystem != "" {

			args := make([]any, 0, 2+len(keyVals))

			args = append(args,
				"subsystem", subsystem,
			)

			args = append(args,
				keyVals...,
			)

			return args
		} else {

			return keyVals
		}
	}
}

// Takes a message and a set of key/value pairs and logs with level DEBUG,
// including the instance's system and/or subsystem if specified.
func (lsc *LoggedSubsystemCommon) QualifiedLogDebug(
	message string,
	keyVals ...any,
) {
	args := _makeQualifiedLogArgs(lsc.System, lsc.Subsystem, keyVals...)

	lsc.Logger.Debug(message, args...)
}

// Takes a message and a set of key/value pairs and logs with level ERR,
// including the instance's system and/or subsystem if specified.
func (lsc *LoggedSubsystemCommon) QualifiedLogError(
	message string,
	keyVals ...any,
) {
	args := _makeQualifiedLogArgs(lsc.System, lsc.Subsystem, keyVals...)

	lsc.Logger.Error(message, args...)
}

// Takes a message and a set of key/value pairs and logs with level INFO,
// including the instance's system and/or subsystem if specified.
func (lsc *LoggedSubsystemCommon) QualifiedLogInfo(
	message string,
	keyVals ...any,
) {
	args := _makeQualifiedLogArgs(lsc.System, lsc.Subsystem, keyVals...)

	lsc.Logger.Info(message, args...)
}

// Takes a message and a set of key/value pairs and logs with level NOTICE,
// including the instance's system and/or subsystem if specified.
//
// Note:
// The current implementation emits on INFO, until such time as the logging
// library is replaced.
func (lsc *LoggedSubsystemCommon) QualifiedLogNotice(
	message string,
	keyVals ...any,
) {
	args := _makeQualifiedLogArgs(lsc.System, lsc.Subsystem, keyVals...)

	lsc.Logger.Info(message, args...)
}

// Takes a message and a set of key/value pairs and logs with level WARN,
// including the instance's system and/or subsystem if specified.
func (lsc *LoggedSubsystemCommon) QualifiedLogWarn(
	message string,
	keyVals ...any,
) {
	args := _makeQualifiedLogArgs(lsc.System, lsc.Subsystem, keyVals...)

	lsc.Logger.Warn(message, args...)
}
