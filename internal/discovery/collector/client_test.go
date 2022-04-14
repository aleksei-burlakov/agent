package collector

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"
	_ "github.com/trento-project/agent/test"
	"github.com/trento-project/agent/test/helpers"
)

const (
	DummyMachineID = "dummy-machine-id"
	DummyAgentID   = "779cdd70-e9e2-58ca-b18a-bf3eb3f71244"
)

type CollectorClientTestSuite struct {
	suite.Suite
}

func TestCollectorClientTestSuite(t *testing.T) {
	suite.Run(t, new(CollectorClientTestSuite))
}

func (suite *CollectorClientTestSuite) SetupSuite() {
	fileSystem = afero.NewMemMapFs()

	afero.WriteFile(fileSystem, machineIdPath, []byte(DummyMachineID), 0644)
}

func (suite *CollectorClientTestSuite) TestCollectorClient_NewClientWithoutTLS() {
	_, err := NewCollectorClient(&Config{
		ServerUrl: "http://localhost",
		ApiKey:    "some-api-key",
	})

	suite.NoError(err)
}

func (suite *CollectorClientTestSuite) TestCollectorClient_PublishingSuccess() {
	collectorClient, err := NewCollectorClient(&Config{
		ServerUrl: "https://localhost",
		ApiKey:    "some-api-key",
	})

	suite.NoError(err)

	discoveredDataPayload := struct {
		FieldA string
	}{
		FieldA: "some discovered field",
	}

	discoveryType := "the_discovery_type"

	collectorClient.httpClient.Transport = helpers.RoundTripFunc(func(req *http.Request) *http.Response {
		requestBody, _ := json.Marshal(map[string]interface{}{
			"agent_id":       DummyAgentID,
			"discovery_type": discoveryType,
			"payload":        discoveredDataPayload,
		})

		bodyBytes, _ := ioutil.ReadAll(req.Body)

		suite.EqualValues(requestBody, bodyBytes)

		suite.Equal(req.URL.String(), "https://localhost/api/collect")
		return &http.Response{
			StatusCode: 202,
		}
	})

	err = collectorClient.Publish(discoveryType, discoveredDataPayload)

	suite.NoError(err)
}

func (suite *CollectorClientTestSuite) TestCollectorClient_PublishingFailure() {
	collectorClient, err := NewCollectorClient(&Config{
		ServerUrl: "http://localhost",
		ApiKey:    "some-api-key",
	})

	suite.NoError(err)

	collectorClient.httpClient.Transport = helpers.RoundTripFunc(func(req *http.Request) *http.Response {
		suite.Equal(req.URL.String(), "http://localhost/api/collect")
		return &http.Response{
			StatusCode: 500,
		}
	})

	err = collectorClient.Publish("some_discovery_type", struct{}{})

	suite.Error(err)
}

func (suite *CollectorClientTestSuite) TestCollectorClient_Heartbeat() {
	collectorClient, err := NewCollectorClient(&Config{
		ServerUrl: "https://localhost",
		ApiKey:    "some-api-key",
	})

	suite.NoError(err)

	collectorClient.httpClient.Transport = helpers.RoundTripFunc(func(req *http.Request) *http.Response {
		suite.Equal(req.URL.String(), fmt.Sprintf("https://localhost/api/hosts/%s/heartbeat", DummyAgentID))
		return &http.Response{
			StatusCode: 204,
		}
	})
	err = collectorClient.Heartbeat()

	suite.NoError(err)
}
