package core

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats-mq/server/conf"
	"github.com/stretchr/testify/require"
)

func TestMonitoringPages(t *testing.T) {
	start := time.Now()
	subject := "test"
	queue := "DEV.QUEUE.1"

	connect := []conf.ConnectorConfig{
		conf.ConnectorConfig{
			Type:           "NATS2Queue",
			Subject:        subject,
			Queue:          queue,
			ExcludeHeaders: true,
		},
	}

	tbs, err := StartTestEnvironment(connect)
	require.NoError(t, err)
	defer tbs.Close()

	client := http.Client{}
	response, err := client.Get(tbs.Bridge.GetMonitoringRootURL())
	require.NoError(t, err)
	defer response.Body.Close()
	contents, err := ioutil.ReadAll(response.Body)
	require.NoError(t, err)
	html := string(contents)
	require.True(t, strings.Contains(html, "/varz"))
	require.True(t, strings.Contains(html, "/healthz"))

	response, err = client.Get(tbs.Bridge.GetMonitoringRootURL() + "healthz")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)

	response, err = client.Get(tbs.Bridge.GetMonitoringRootURL() + "varz")
	require.NoError(t, err)
	defer response.Body.Close()
	contents, err = ioutil.ReadAll(response.Body)
	require.NoError(t, err)

	bridgeStats := BridgeStats{}
	err = json.Unmarshal(contents, &bridgeStats)
	require.NoError(t, err)

	now := time.Now()
	require.True(t, bridgeStats.StartTime >= start.Unix())
	require.True(t, bridgeStats.StartTime <= now.Unix())
	require.True(t, bridgeStats.ServerTime >= start.Unix())
	require.True(t, bridgeStats.ServerTime <= now.Unix())

	require.Equal(t, bridgeStats.HTTPRequests["/"], int64(1))
	require.Equal(t, bridgeStats.HTTPRequests["/varz"], int64(1))
	require.Equal(t, bridgeStats.HTTPRequests["/healthz"], int64(1))

	require.Equal(t, 1, len(bridgeStats.Connections))
	require.True(t, bridgeStats.Connections[0].Connected)
	require.Equal(t, int64(1), bridgeStats.Connections[0].Connects)
	require.Equal(t, int64(0), bridgeStats.Connections[0].MessagesIn)
	require.Equal(t, int64(0), bridgeStats.Connections[0].MessagesOut)
	require.Equal(t, int64(0), bridgeStats.Connections[0].BytesIn)
	require.Equal(t, int64(0), bridgeStats.Connections[0].BytesOut)
}