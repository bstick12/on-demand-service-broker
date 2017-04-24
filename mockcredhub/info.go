// Copyright (C) 2016-Present Pivotal Software, Inc. All rights reserved.
// This program and the accompanying materials are made available under the terms of the under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package mockcredhub

import "github.com/pivotal-cf/on-demand-service-broker/mockhttp"

type getInfoMock struct {
	*mockhttp.MockHttp
}

func GetInfo() *getInfoMock {
	return &getInfoMock{
		mockhttp.NewMockedHttpRequest("GET", "/info"),
	}
}

func (m *getInfoMock) RespondsWithCredhubUaaUrl(url string) *mockhttp.MockHttp {
	return m.RespondsWith(`
    {
		  "auth-server": {
		    "url": "` + url + `"
		  },
		  "app": {
		    "name": "CredHub for PCF",
		    "version": "xxx"
		  }
		}`,
	)
}
