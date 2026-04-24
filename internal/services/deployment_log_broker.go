package services

import (
	"sync"

	"bloope/internal/models"
)

type DeploymentLogBroker struct {
	mu          sync.RWMutex
	subscribers map[string]map[chan *models.DeploymentLog]struct{}
}

func NewDeploymentLogBroker() *DeploymentLogBroker {
	return &DeploymentLogBroker{
		subscribers: make(map[string]map[chan *models.DeploymentLog]struct{}),
	}
}

func (b *DeploymentLogBroker) Subscribe(deploymentID string) (<-chan *models.DeploymentLog, func()) {
	ch := make(chan *models.DeploymentLog, 128)
	var once sync.Once

	b.mu.Lock()
	if _, ok := b.subscribers[deploymentID]; !ok {
		b.subscribers[deploymentID] = make(map[chan *models.DeploymentLog]struct{})
	}
	b.subscribers[deploymentID][ch] = struct{}{}
	b.mu.Unlock()

	unsubscribe := func() {
		once.Do(func() {
			b.mu.Lock()
			defer b.mu.Unlock()

			if deploymentSubscribers, ok := b.subscribers[deploymentID]; ok {
				delete(deploymentSubscribers, ch)
				if len(deploymentSubscribers) == 0 {
					delete(b.subscribers, deploymentID)
				}
			}

			close(ch)
		})
	}

	return ch, unsubscribe
}

func (b *DeploymentLogBroker) Broadcast(logEntry *models.DeploymentLog) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch := range b.subscribers[logEntry.DeploymentID] {
		select {
		case ch <- logEntry:
		default:
		}
	}
}
