/* Copyright 2022 SolarWinds Worldwide, LLC. All rights reserved.
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at:
*
*	http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and limitations
* under the License.
*/

package logger

import (
	"log"
	"os"
)

type Logger interface {
	Info(v ...interface {})
	Error(v ...interface {})
	Fatal(v ...interface {})
}

type logger struct {
	infoLogger log.Logger
	errorLogger log.Logger
}

func (l logger) Info(v ...interface {}) {
	l.infoLogger.Println(v...)
}

func (l logger) Error(v ...interface {}) {
	l.infoLogger.Println(v...)
}

func (l logger) Fatal(v ...interface {}) {
	l.Error(v...)
	os.Exit(1)
}

func NewLogger(prefix string) (Logger) {
	return &logger {
		infoLogger: *log.New(log.Writer(), prefix + " INFO ", log.Lmsgprefix),
		errorLogger: *log.New(log.Writer(), prefix + " ERROR ", log.Lmsgprefix),
	}
}
