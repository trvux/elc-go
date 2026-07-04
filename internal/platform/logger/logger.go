package logger

import "go.uber.org/zap"

func New(env string) (*zap.Logger, error) {
	var base *zap.Logger
	var err error

	if env == "production" {
		base, err = zap.NewProduction()
	} else {
		base, err = zap.NewDevelopment()
	}
	if err != nil {
		return nil, err
	}

	return base.With(zap.String("service", "elc-go")), nil
}
