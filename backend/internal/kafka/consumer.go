package kafka

import (
	"RSOI_PROJECT/internal/models"
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/IBM/sarama"
)

type ConsumerGroupHandler struct {
	ready  chan bool
	events *[]models.Event
	mu     sync.Mutex
}

func NewConsumerGroup(brokers []string, groupID, topic string) (sarama.ConsumerGroup, error) {
	config := sarama.NewConfig()
	config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	config.Version = sarama.V3_5_0_0

	consumerGroup, err := sarama.NewConsumerGroup(brokers, groupID, config)
	if err != nil {
		return nil, err
	}
	return consumerGroup, nil
}

func Consume(consumerGroup sarama.ConsumerGroup, topic string, eventsStore *[]models.Event) {
	handler := &ConsumerGroupHandler{
		ready:  make(chan bool),
		events: eventsStore,
	}

	ctx := context.Background()
	go func() {
		for {
			if err := consumerGroup.Consume(ctx, []string{topic}, handler); err != nil {
				log.Printf("Consumer error: %v", err)
			}
			if ctx.Err() != nil {
				return
			}
			handler.ready = make(chan bool)
		}
	}()
	<-handler.ready
	log.Println("Consumer started and ready")
}

func (h *ConsumerGroupHandler) Setup(_ sarama.ConsumerGroupSession) error {
	close(h.ready)
	return nil
}

func (h *ConsumerGroupHandler) Cleanup(_ sarama.ConsumerGroupSession) error {
	return nil
}

func (h *ConsumerGroupHandler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		var event models.Event
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("Failed to unmarshal event: %v", err)
			continue
		}
		h.mu.Lock()
		*h.events = append(*h.events, event)
		h.mu.Unlock()
		log.Printf("Received event: %+v", event)
		sess.MarkMessage(msg, "")
	}
	return nil
}
