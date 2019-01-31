package main

import (
    "log"
    "github.com/pkg/errors"
)

func Handle(err error, msg string, handler func(error)) {
    if err != nil {
        handler(errors.WithMessage(err, msg))
    }
}

func HandleWithTrace(err error, msg string, handler func(error)) {
    if err != nil {
        handler(errors.Wrap(err, msg))
    }
}

func Log(err error, msg string) {
    if err != nil {
        log.Printf(errors.WithMessage(err, msg))
    }
}

func LogWithTrace(err error, msg string) {
    if err != nil {
        log.Printf("%+v", errors.Wrap(err, msg))
    }
}

func Fatal(err error, msg string) {
    if err != nil {
        panic(errors.Wrap(err, msg))
    }
}
