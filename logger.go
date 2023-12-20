package manywest

import (
	"log/slog"
	"os"

	"github.com/taylormonacelli/littlecow"
)

func getLogger(logLevelString, logFormat string) (*slog.Logger, error) {
	logLevel, err := littlecow.LevelFromString(logLevelString)
	if err != nil {
		return nil, err
	}

	opts := littlecow.NewHandlerOptions(logLevel, littlecow.RemoveTimestampAndTruncateSource)

	var handler slog.Handler
	handler = slog.NewTextHandler(os.Stderr, opts)
	if logFormat == "json" {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	}

	return slog.New(handler), nil
}
