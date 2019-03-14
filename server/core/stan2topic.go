package core

import (
	"fmt"
	"sync"
	"time"

	"github.com/ibm-messaging/mq-golang/ibmmq"
	"github.com/nats-io/go-nats-streaming"
	"github.com/nats-io/nats-mq/server/conf"
)

// Stan2TopicConnector connects a STAN channel to an MQ Topic
type Stan2TopicConnector struct {
	sync.Mutex

	config conf.ConnectorConfig
	bridge Bridge

	qMgr  *ibmmq.MQQueueManager
	topic *ibmmq.MQObject

	sub stan.Subscription

	stats ConnectorStats
}

// NewStan2TopicConnector create a new Stan to MQ connector
func NewStan2TopicConnector(bridge Bridge, config conf.ConnectorConfig) Connector {
	return &Stan2TopicConnector{
		config: config,
		bridge: bridge,
		stats:  NewConnectorStats(),
	}
}

func (mq *Stan2TopicConnector) String() string {
	return fmt.Sprintf("STAN:%s to Topic:%s", mq.config.Channel, mq.config.Topic)
}

// Stats returns a copy of the current stats for this connector
func (mq *Stan2TopicConnector) Stats() ConnectorStats {
	mq.Lock()
	defer mq.Unlock()
	return mq.stats
}

// Config returns the configuraiton for this connector
func (mq *Stan2TopicConnector) Config() conf.ConnectorConfig {
	return mq.config
}

// Start the connector
func (mq *Stan2TopicConnector) Start() error {
	mq.Lock()
	defer mq.Unlock()

	mq.stats.Name = mq.String()

	if mq.bridge.Stan() == nil {
		return fmt.Errorf("%s connector requires nats streaming to be available", mq.String())
	}

	mqconfig := mq.config.MQ
	topicName := mq.config.Topic

	mq.bridge.Logger().Tracef("starting connection %s", mq.String())

	qMgr, err := ConnectToQueueManager(mqconfig)
	if err != nil {
		return err
	}

	mq.bridge.Logger().Tracef("connected to queue manager %s at %s as %s for %s", mqconfig.QueueManager, mqconfig.ConnectionName, mqconfig.ChannelName, mq.String())

	mq.qMgr = qMgr

	// Create the Object Descriptor that allows us to give the queue name
	mqod := ibmmq.NewMQOD()
	openOptions := ibmmq.MQOO_OUTPUT
	mqod.ObjectType = ibmmq.MQOT_TOPIC
	mqod.ObjectString = topicName

	qObject, err := mq.qMgr.Open(mqod, openOptions)

	if err != nil {
		return err
	}

	mq.topic = &qObject
	mq.bridge.Logger().Tracef("opened %s", topicName)

	sub, err := mq.bridge.Stan().Subscribe(mq.config.Channel, mq.messageHandler)

	if err != nil {
		return err
	}

	mq.sub = sub

	mq.stats.AddConnect()
	mq.bridge.Logger().Tracef("reading %s", mq.config.Channel)
	mq.bridge.Logger().Noticef("started connection %s", mq.String())

	return nil
}

func (mq *Stan2TopicConnector) messageHandler(m *stan.Msg) {
	mq.Lock()
	defer mq.Unlock()
	start := time.Now()

	qmgrFlag := mq.qMgr

	if mq.config.ExcludeHeaders {
		qmgrFlag = nil
	}

	mq.stats.AddMessageIn(int64(len(m.Data)))
	mqmd, handle, buffer, err := mq.bridge.NATSToMQMessage(m.Data, "", qmgrFlag)

	pmo := ibmmq.NewMQPMO()
	pmo.Options = ibmmq.MQPMO_NO_SYNCPOINT
	pmo.OriginalMsgHandle = handle

	// Now put the message to the queue
	err = mq.topic.Put(mqmd, pmo, buffer)

	if err != nil {
		mq.bridge.Logger().Noticef("MQ publish failure, %s, %s", mq.String(), err.Error())
	} else {
		mq.stats.AddMessageOut(int64(len(buffer)))
		mq.stats.AddRequestTime(time.Now().Sub(start))
	}
}

// Shutdown the connector
func (mq *Stan2TopicConnector) Shutdown() error {
	mq.Lock()
	defer mq.Unlock()
	mq.stats.AddDisconnect()

	mq.bridge.Logger().Noticef("shutting down connection %s", mq.String())

	var err error

	topic := mq.topic
	mq.topic = nil

	if topic != nil {
		err = topic.Close(0)
	}

	if mq.qMgr != nil {
		_ = mq.qMgr.Disc()
		mq.qMgr = nil
		mq.bridge.Logger().Tracef("disconnected from queue manager for %s", mq.String())
	}

	if mq.sub != nil {
		mq.sub.Unsubscribe()
		mq.sub = nil
	}

	return err // ignore the disconnect error
}