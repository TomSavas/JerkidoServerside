package main

import (
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

func Log(err error, tag string, msg string) {
    if err != nil {
        LogInfo(tag, msg + " Error: " + err.Error())
    }
}

func LogWithTrace(err error, tag string, msg string) {
    if err != nil {
        LogInfo(tag, msg + " Error: " + err.Error())
    }
}

func Fatal(err error, msg string) {
    if err != nil {
        panic(errors.Wrap(err, msg))
    }
}
