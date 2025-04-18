// The MIT License
//
// Copyright (c) 2020 Temporal Technologies Inc.  All rights reserved.
//
// Copyright (c) 2020 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cassandra

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/server/temporal/environment"
)

type (
	HandlerTestSuite struct {
		*require.Assertions // override suite.Suite.Assertions with require.Assertions; this means that s.NotNil(nil) will stop the test, not merely log an error
		suite.Suite
	}
)

func TestHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}

func (s *HandlerTestSuite) SetupTest() {
	s.Assertions = require.New(s.T()) // Have to define our overridden assertions in the test setup. If we did it earlier, s.T() will return nil
}

func (s *HandlerTestSuite) TestValidateCQLClientConfig() {
	config := new(CQLClientConfig)
	s.NotNil(validateCQLClientConfig(config))

	config.Hosts = environment.GetCassandraAddress()
	s.NotNil(validateCQLClientConfig(config))

	config.Keyspace = "foobar"
	s.Nil(validateCQLClientConfig(config))
}

func (s *HandlerTestSuite) TestParsingOfOptionsMap() {
	parsedMap := parseOptionsMap("key1=value1 ,key2= value2,key3=value3")

	s.Assert().Equal("value1", parsedMap["key1"])
	s.Assert().Equal("value2", parsedMap["key2"])
	s.Assert().Equal("value3", parsedMap["key3"])
	s.Assert().Equal("", parsedMap["key4"])

	parsedMap2 := parseOptionsMap("key1=,=value2")

	s.Assert().Equal(0, len(parsedMap2))
}

func (s *HandlerTestSuite) TestDropKeyspaceError() {
	// fake exit function to avoid exiting the application
	back := osExit
	defer func() { osExit = back }()
	osExit = func(i int) {
		s.Equal(1, i)
	}
	args := []string{"./tool", "drop-keyspace", "-f", "--keyspace", ""}
	app := buildCLIOptions()
	err := app.Run(args)
	s.Nil(err)
}

func (s *HandlerTestSuite) TestCreateKeyspaceError() {
	// fake exit function to avoid exiting the application
	back := osExit
	defer func() { osExit = back }()
	osExit = func(i int) {
		s.Equal(1, i)
	}
	args := []string{"./tool", "create-keyspace", "--keyspace", ""}
	app := buildCLIOptions()
	err := app.Run(args)
	s.Nil(err)
}
