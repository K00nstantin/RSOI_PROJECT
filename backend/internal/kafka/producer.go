package kafka

import (
	"encoding/json"

	"github.com/IBM/sarama"
)

type Producer struct {
	producer sarama.SyncProducer
	topic    string
}

func NewProducer(brokers []string, topic string) (*Producer, error) {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5
	config.Producer.Return.Successes = true
	// Устанавливаем версию протокола (совместима с Kafka 3.5+)
	config.Version = sarama.V3_5_0_0 // или V2_8_0_0

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, err
	}
	return &Producer{
		producer: producer,
		topic:    topic,
	}, nil
}

func (p *Producer) SendEvent(event interface{}) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	msg := &sarama.ProducerMessage{
		Topic: p.topic,
		Value: sarama.StringEncoder(data),
	}
	_, _, err = p.producer.SendMessage(msg)
	return err
}

func (p *Producer) Close() error {
	return p.producer.Close()
}
