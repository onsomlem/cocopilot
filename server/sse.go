package server

func broadcastUpdate(eventType string) {
	sseMutex.RLock()
	defer sseMutex.RUnlock()
	for _, client := range sseClients {
		if client.eventType != "" && client.eventType != eventType {
			continue
		}
		select {
		case client.ch <- eventType:
		default:
			// Client buffer full, skip
		}
	}
}

func publishV2Event(event Event) {
	v2EventMu.RLock()
	defer v2EventMu.RUnlock()

	for _, subscriber := range v2EventSubscribers {
		if subscriber.projectID != "" && subscriber.projectID != event.ProjectID {
			continue
		}
		if subscriber.kind != "" && subscriber.kind != event.Kind {
			continue
		}
		select {
		case subscriber.ch <- event:
		default:
			// Client buffer full, skip
		}
	}
}
