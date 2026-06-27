package notify

func (d *Dispatcher) BroadcastNetwork(payload any) {
	if d == nil || d.hub == nil {
		return
	}
	d.hub.Broadcast(ChannelNetwork, "network.ticker.updated", payload)
}

func (d *Dispatcher) BroadcastSlots(payload any) {
	if d == nil || d.hub == nil {
		return
	}
	d.hub.Broadcast(ChannelSlots, "slots.feed.updated", payload)
}

func (d *Dispatcher) BroadcastLeaders(payload any) {
	if d == nil || d.hub == nil {
		return
	}
	d.hub.Broadcast(ChannelLeaders, "leaders.schedule.updated", payload)
}

func (d *Dispatcher) BroadcastPipeline(payload any) {
	if d == nil || d.hub == nil {
		return
	}
	d.hub.Broadcast(ChannelPipeline, "lifecycle.pipeline.updated", payload)
}

func (d *Dispatcher) BroadcastLanding(payload any) {
	if d == nil || d.hub == nil {
		return
	}
	d.hub.Broadcast(ChannelLanding, "landing.probability.updated", payload)
}

func (d *Dispatcher) BroadcastRecovery(payload any) {
	if d == nil || d.hub == nil {
		return
	}
	d.hub.Broadcast(ChannelRecovery, "recovery.actions.updated", payload)
}

func (d *Dispatcher) BroadcastCharts(payload any) {
	if d == nil || d.hub == nil {
		return
	}
	d.hub.Broadcast(ChannelCharts, "charts.series.updated", payload)
}
