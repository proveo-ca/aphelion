//go:build linux

package decision

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync/atomic"
)

func (b *Broker) Load(ctx context.Context) error {
	if b == nil {
		return fmt.Errorf("decision broker is nil")
	}

	b.mu.Lock()
	if b.loaded {
		b.mu.Unlock()
		return nil
	}
	store := b.durable
	b.mu.Unlock()
	if store == nil {
		b.mu.Lock()
		b.loaded = true
		b.mu.Unlock()
		return nil
	}

	persisted, err := store.LoadPending(ctx)
	if err != nil {
		return fmt.Errorf("load pending decisions: %w", err)
	}

	loadedPending := make(map[string]*pendingDecision, len(persisted))
	maxSeq := uint64(0)
	for _, row := range persisted {
		id := strings.TrimSpace(row.Pending.ID)
		if id == "" {
			continue
		}
		req := normalizeRequest(row.Pending.Request)
		if len(req.Choices) == 0 || !containsChoice(req.Choices, req.DefaultChoice) {
			continue
		}
		ownerKey := strings.TrimSpace(row.OwnerKey)
		if ownerKey == "" {
			ownerKey = decisionOwnerKey(req)
		}
		seq := row.Seq
		if seq == 0 {
			if parsed, ok := parseBase36(id); ok {
				seq = parsed
			}
		}
		pending := &pendingDecision{
			request: PendingDecision{
				ID:                id,
				Request:           req,
				Delivery:          row.Delivery,
				LoadedFromDurable: true,
			},
			delivery:     row.Delivery,
			resultCh:     make(chan string, 1),
			ownerKey:     ownerKey,
			exclusiveKey: decisionExclusiveKey(req, ownerKey),
			seq:          seq,
		}
		loadedPending[id] = pending
		if seq > maxSeq {
			maxSeq = seq
		}
	}

	loadedByOwner := make(map[string]string, len(loadedPending))
	for id, pending := range loadedPending {
		if pending.exclusiveKey == "" {
			continue
		}
		existingID, ok := loadedByOwner[pending.exclusiveKey]
		if !ok {
			loadedByOwner[pending.exclusiveKey] = id
			continue
		}
		existing := loadedPending[existingID]
		if existing == nil || existing.seq < pending.seq {
			loadedByOwner[pending.exclusiveKey] = id
		}
	}

	staleIDs := make([]string, 0)
	for id, pending := range loadedPending {
		if pending.exclusiveKey == "" {
			continue
		}
		if loadedByOwner[pending.exclusiveKey] != id {
			delete(loadedPending, id)
			staleIDs = append(staleIDs, id)
		}
	}

	b.mu.Lock()
	if b.loaded {
		b.mu.Unlock()
		return nil
	}
	b.pending = loadedPending
	b.byOwner = loadedByOwner
	if maxSeq > 0 {
		atomic.StoreUint64(&b.nextID, maxSeq)
	}
	b.loaded = true
	b.mu.Unlock()

	for _, id := range staleIDs {
		_ = store.DeletePending(ctx, id)
	}
	return nil
}

func (b *Broker) ensureLoaded(ctx context.Context) error {
	if b == nil {
		return fmt.Errorf("decision broker is nil")
	}
	b.mu.Lock()
	loaded := b.loaded
	b.mu.Unlock()
	if loaded {
		return nil
	}
	return b.Load(ctx)
}

func (b *Broker) DetachByOwner(ctx context.Context, ownerKey string) (int, error) {
	if b == nil {
		return 0, fmt.Errorf("decision broker is nil")
	}
	if err := b.ensureLoaded(ctx); err != nil {
		return 0, err
	}
	ownerKey = strings.TrimSpace(ownerKey)
	if ownerKey == "" {
		return 0, nil
	}

	detached := make([]*pendingDecision, 0, 1)
	b.mu.Lock()
	for id, pending := range b.pending {
		if pending == nil || pending.ownerKey != ownerKey {
			continue
		}
		delete(b.pending, id)
		if pending.exclusiveKey != "" {
			if ownerID, exists := b.byOwner[pending.exclusiveKey]; exists && ownerID == id {
				delete(b.byOwner, pending.exclusiveKey)
			}
		}
		detached = append(detached, pending)
	}
	store := b.durable
	b.mu.Unlock()

	for _, pending := range detached {
		b.emitEvent(ctx, pending, EventTypeDetached, strings.TrimSpace(pending.request.DefaultChoice), false, "owner_detach")
		resolveDefaultChoice(pending)
	}
	if store == nil {
		return len(detached), nil
	}
	removed, err := store.DetachByOwner(ctx, ownerKey)
	if err != nil {
		return len(detached), err
	}
	if removed > len(detached) {
		return removed, nil
	}
	return len(detached), nil
}

func (b *Broker) DetachDecision(ctx context.Context, id string, reason string) (PendingDecision, bool, error) {
	if b == nil {
		return PendingDecision{}, false, fmt.Errorf("decision broker is nil")
	}
	if err := b.ensureLoaded(ctx); err != nil {
		return PendingDecision{}, false, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return PendingDecision{}, false, nil
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "detached"
	}

	b.mu.Lock()
	pending := b.pending[id]
	if pending == nil {
		b.mu.Unlock()
		return PendingDecision{}, false, nil
	}
	delete(b.pending, id)
	if pending.exclusiveKey != "" {
		if ownerID, exists := b.byOwner[pending.exclusiveKey]; exists && ownerID == id {
			delete(b.byOwner, pending.exclusiveKey)
		}
	}
	store := b.durable
	b.mu.Unlock()

	b.emitEvent(ctx, pending, EventTypeDetached, strings.TrimSpace(pending.request.DefaultChoice), false, reason)
	resolveDefaultChoice(pending)
	if store != nil {
		if err := store.DeletePending(ctx, id); err != nil {
			return pending.request, true, err
		}
	}
	return pending.request, true, nil
}

func (b *Broker) PendingByOwnerKind(ownerKey string, kind Kind) (PendingDecision, bool) {
	if b == nil {
		return PendingDecision{}, false
	}
	if err := b.ensureLoaded(context.Background()); err != nil {
		return PendingDecision{}, false
	}
	ownerKey = strings.TrimSpace(ownerKey)
	kind = Kind(strings.TrimSpace(string(kind)))
	if ownerKey == "" || kind == "" {
		return PendingDecision{}, false
	}
	exclusiveKey := decisionExclusiveKey(Request{Kind: kind}, ownerKey)
	b.mu.Lock()
	defer b.mu.Unlock()
	id := strings.TrimSpace(b.byOwner[exclusiveKey])
	if id == "" {
		return PendingDecision{}, false
	}
	pending := b.pending[id]
	if pending == nil {
		return PendingDecision{}, false
	}
	return pending.request, true
}

func (b *Broker) PendingDecisions(ctx context.Context) ([]PendingDecision, error) {
	if b == nil {
		return nil, fmt.Errorf("decision broker is nil")
	}
	if err := b.ensureLoaded(ctx); err != nil {
		return nil, err
	}
	b.mu.Lock()
	out := make([]PendingDecision, 0, len(b.pending))
	for _, pending := range b.pending {
		if pending == nil {
			continue
		}
		out = append(out, pending.request)
	}
	b.mu.Unlock()
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out, nil
}

type durableChatSenderDetacher interface {
	DetachByChatSender(ctx context.Context, chatID int64, senderID int64) (int, error)
}

func (b *Broker) DetachByChatSender(ctx context.Context, chatID int64, senderID int64) (int, error) {
	if b == nil {
		return 0, fmt.Errorf("decision broker is nil")
	}
	if err := b.ensureLoaded(ctx); err != nil {
		return 0, err
	}
	if chatID == 0 || senderID == 0 {
		return 0, nil
	}

	detached := make([]*pendingDecision, 0, 1)
	b.mu.Lock()
	for id, pending := range b.pending {
		if pending == nil || pending.request.ChatID != chatID || pending.request.SenderID != senderID {
			continue
		}
		delete(b.pending, id)
		if pending.exclusiveKey != "" {
			if ownerID, exists := b.byOwner[pending.exclusiveKey]; exists && ownerID == id {
				delete(b.byOwner, pending.exclusiveKey)
			}
		}
		detached = append(detached, pending)
	}
	store := b.durable
	b.mu.Unlock()

	for _, pending := range detached {
		b.emitEvent(ctx, pending, EventTypeDetached, strings.TrimSpace(pending.request.DefaultChoice), false, "chat_sender_detach")
		resolveDefaultChoice(pending)
	}
	if store == nil {
		return len(detached), nil
	}
	if detacher, ok := store.(durableChatSenderDetacher); ok {
		removed, err := detacher.DetachByChatSender(ctx, chatID, senderID)
		if err != nil {
			return len(detached), err
		}
		if removed > len(detached) {
			return removed, nil
		}
	}
	return len(detached), nil
}

func (b *Broker) DetachAll(ctx context.Context) (int, error) {
	if b == nil {
		return 0, fmt.Errorf("decision broker is nil")
	}
	if err := b.ensureLoaded(ctx); err != nil {
		return 0, err
	}

	detached := make([]*pendingDecision, 0, len(b.pending))
	b.mu.Lock()
	for id, pending := range b.pending {
		if pending == nil {
			continue
		}
		delete(b.pending, id)
		detached = append(detached, pending)
	}
	b.byOwner = make(map[string]string)
	store := b.durable
	b.mu.Unlock()

	for _, pending := range detached {
		b.emitEvent(ctx, pending, EventTypeDetached, strings.TrimSpace(pending.request.DefaultChoice), false, "detach_all")
		resolveDefaultChoice(pending)
	}
	if store == nil {
		return len(detached), nil
	}
	removed, err := store.DetachAll(ctx)
	if err != nil {
		return len(detached), err
	}
	if removed > len(detached) {
		return removed, nil
	}
	return len(detached), nil
}

func (b *Broker) upsertPending(ctx context.Context, pending *pendingDecision) error {
	if b == nil || pending == nil {
		return nil
	}
	b.mu.Lock()
	store := b.durable
	b.mu.Unlock()
	if store == nil {
		return nil
	}
	return store.UpsertPending(ctx, DurableDecision{
		Pending:  pending.request,
		Seq:      pending.seq,
		OwnerKey: pending.ownerKey,
		Delivery: pending.delivery,
	})
}

func (b *Broker) clearSupersededPendingForAutoResolve(ctx context.Context, pending *pendingDecision) error {
	if b == nil || pending == nil || strings.TrimSpace(pending.exclusiveKey) == "" {
		return nil
	}
	b.mu.Lock()
	existingID := strings.TrimSpace(b.byOwner[pending.exclusiveKey])
	if existingID == "" {
		b.mu.Unlock()
		return nil
	}
	existing := b.pending[existingID]
	if existing == nil {
		delete(b.byOwner, pending.exclusiveKey)
		b.mu.Unlock()
		return b.clearWithContext(ctx, existingID)
	}
	delete(b.pending, existingID)
	delete(b.byOwner, pending.exclusiveKey)
	b.mu.Unlock()

	b.emitEvent(ctx, existing, EventTypeDetached, strings.TrimSpace(existing.request.DefaultChoice), false, "superseded_by_auto_resolve")
	resolveDefaultChoice(existing)
	return b.clearWithContext(ctx, existingID)
}

func (b *Broker) clearWithContext(ctx context.Context, id string) error {
	if b == nil {
		return nil
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	b.mu.Lock()
	pending, ok := b.pending[id]
	if ok {
		delete(b.pending, id)
		if pending.exclusiveKey != "" {
			if ownerID, exists := b.byOwner[pending.exclusiveKey]; exists && ownerID == id {
				delete(b.byOwner, pending.exclusiveKey)
			}
		}
	}
	store := b.durable
	b.mu.Unlock()
	if store == nil {
		return nil
	}
	return store.DeletePending(ctx, id)
}

func (b *Broker) activateOwner(ctx context.Context, id string) bool {
	var staleID string
	var stalePending *pendingDecision
	var supersededID string
	var supersededPending *pendingDecision

	b.mu.Lock()
	pending, ok := b.pending[id]
	if !ok {
		b.mu.Unlock()
		return false
	}
	ownerKey := pending.exclusiveKey
	if ownerKey == "" {
		b.mu.Unlock()
		return false
	}

	existingID, ok := b.byOwner[ownerKey]
	if !ok || existingID == id {
		b.byOwner[ownerKey] = id
		b.mu.Unlock()
		return false
	}
	existing := b.pending[existingID]
	if existing == nil {
		b.byOwner[ownerKey] = id
		b.mu.Unlock()
		return false
	}
	if existing.seq > pending.seq {
		delete(b.pending, id)
		staleID = id
		stalePending = pending
		b.mu.Unlock()
		b.emitEvent(ctx, stalePending, EventTypeDetached, strings.TrimSpace(stalePending.request.DefaultChoice), false, "stale_superseded")
		select {
		case stalePending.resultCh <- stalePending.request.DefaultChoice:
		default:
		}
		_ = b.clearWithContext(ctx, staleID)
		return true
	}
	delete(b.pending, existingID)
	b.byOwner[ownerKey] = id
	supersededID = existingID
	supersededPending = existing
	b.mu.Unlock()
	b.emitEvent(ctx, supersededPending, EventTypeDetached, strings.TrimSpace(supersededPending.request.DefaultChoice), false, "superseded_by_newer")
	select {
	case supersededPending.resultCh <- supersededPending.request.DefaultChoice:
	default:
	}
	_ = b.clearWithContext(ctx, supersededID)
	return false
}
