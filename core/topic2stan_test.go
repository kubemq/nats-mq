package core

import (
	"bytes"
	"github.com/ibm-messaging/mq-golang/ibmmq"
	"github.com/nats-io/nats-mq/message"
	"testing"
	"time"

	stan "github.com/nats-io/go-nats-streaming"
	"github.com/stretchr/testify/require"
)

func TestSimpleSendOnTopicReceiveOnStan(t *testing.T) {
	channel := "test"
	topic := "dev/"
	msg := "hello world"

	connect := []ConnectionConfig{
		ConnectionConfig{
			Type:           "Topic2Stan",
			Channel:        channel,
			Topic:          topic,
			ExcludeHeaders: true,
		},
	}

	tbs, err := StartTestEnvironment(connect)
	require.NoError(t, err)
	defer tbs.Close()

	received := ""
	done := make(chan bool)

	sub, err := tbs.SC.Subscribe(channel, func(msg *stan.Msg) {
		received = string(msg.Data)
		done <- true
	})
	defer sub.Unsubscribe()

	err = tbs.putMessageOnTopic(topic, ibmmq.NewMQMD(), []byte(msg))
	require.NoError(t, err)

	timer := time.NewTimer(3 * time.Second)
	go func() {
		<-timer.C
		done <- true
	}()

	<-done
	require.Equal(t, msg, received)
}

func TestSendOnTopicReceiveOnStanMQMD(t *testing.T) {
	start := time.Now().UTC()
	channel := "test"
	topic := "dev/"
	msg := "hello world"
	id := bytes.Repeat([]byte{1}, int(ibmmq.MQ_MSG_ID_LENGTH))
	corr := bytes.Repeat([]byte{1}, int(ibmmq.MQ_CORREL_ID_LENGTH))

	connect := []ConnectionConfig{
		ConnectionConfig{
			Type:           "Topic2Stan",
			Channel:        channel,
			Topic:          topic,
			ExcludeHeaders: false,
		},
	}

	tbs, err := StartTestEnvironment(connect)
	require.NoError(t, err)
	defer tbs.Close()

	var received []byte
	done := make(chan bool)

	sub, err := tbs.SC.Subscribe(channel, func(msg *stan.Msg) {
		received = msg.Data
		done <- true
	})
	defer sub.Unsubscribe()

	mqmd := ibmmq.NewMQMD()
	mqmd.CorrelId = corr
	mqmd.MsgId = id
	err = tbs.putMessageOnTopic(topic, ibmmq.NewMQMD(), []byte(msg))
	require.NoError(t, err)

	// don't wait forever
	timer := time.NewTimer(3 * time.Second)
	go func() {
		<-timer.C
		done <- true
	}()

	<-done

	require.True(t, len(received) > 0)

	bridgeMessage, err := message.DecodeBridgeMessage(received)
	require.NoError(t, err)

	require.Equal(t, msg, string(bridgeMessage.Body))
	require.Equal(t, start.Format("20060102"), bridgeMessage.Header.PutDate)
	require.True(t, start.Format("15040500") < bridgeMessage.Header.PutTime)

	// TODO looks like topics generate these, perhaps it is a setting
	//require.ElementsMatch(t, id, bridgeMessage.Header.MsgID)
	//require.ElementsMatch(t, corr, bridgeMessage.Header.CorrelID)
}
