package main

import (
    "net/http"
)

type Observer = User

func GetObserverFromRequest(request *http.Request) (Observer, error) {
	return GetUserFromRequest(request)
}
