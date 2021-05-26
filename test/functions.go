/*
Copyright © 2021 Google

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package test

import (
	"fmt"
	"strings"
)

// errorDiff compares the incoming string against the error provided.
// Checks whether substring is present in got.Error() string.
func ErrorDiff(want string, got error) string {
	if got == nil && want == "" {
		return ""
	}
	if got == nil && want != "" {
		return fmt.Sprintf("\t- %s", want)
	}
	if got != nil && want == "" {
		return fmt.Sprintf("\t+ %s", got.Error())
	}
	if got != nil && !strings.Contains(got.Error(), want) {
		return fmt.Sprintf("\t+ %s\n\t- %s", got.Error(), want)
	}
	return ""
}