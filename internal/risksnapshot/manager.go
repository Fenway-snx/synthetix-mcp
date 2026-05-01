package risksnapshot

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
	"github.com/shopspring/decimal"
)

const (
	defaultSnapshotMaxAge = 3 * time.Minute
	// Upstream caps open-orders/positions at 100 per request. Keep
	// page size at the enforced maximum to minimize round-trips.
	pageSize int = 100
)

// Transport-neutral order item.
type HydrationOrder struct {
	ClientOrderID     string
	OrderType         string
	Price             string
	Quantity          string
	ReduceOnly        bool
	RemainingQuantity string
	Side              string
	Symbol            string
	VenueOrderID      string
}

// Transport-neutral position item.
type HydrationPosition struct {
	Quantity string
	Side     string
	Symbol   string
}

type HydrationClient interface {
	GetOpenOrders(ctx context.Context, subAccountID int64, limit, offset int) ([]HydrationOrder, error)
	GetPositions(ctx context.Context, subAccountID int64, limit, offset int) ([]HydrationPosition, error)
}

type Order struct {
	ClientOrderID     string
	OrderType         string
	Price             decimal.Decimal
	Quantity          decimal.Decimal
	ReduceOnly        bool
	RemainingQuantity decimal.Decimal
	Side              string
	Symbol            string
	VenueOrderID      string
}

type OrderRef struct {
	ClientOrderID string
	VenueOrderID  string
}

type Snapshot struct {
	openOrdersByClient map[string]Order
	openOrdersByVenue  map[string]Order
	positionsBySymbol  map[string]decimal.Decimal
	refreshedAtMs      int64
	subAccountID       int64
}

type Manager struct {
	client         HydrationClient
	maxSnapshotAge time.Duration
	mu             sync.Mutex
	entries        map[int64]*entry
	sessionEntries map[string]int64
	observers      []TransitionObserver
}

// PositionTransition is emitted whenever the manager ingests a
// fresh snapshot and detects a meaningful change in a position
// quantity. nonzero → zero is the canonical "trade closed" signal
// that triggers an MCP notification; zero → nonzero (a new
// position) and a sign flip (long → short via cross-trade) are
// also surfaced so observers can decide which to act on.
type PositionTransition struct {
	SubAccountID int64
	SessionID    string
	Symbol       string
	Prior        decimal.Decimal
	Current      decimal.Decimal
	Kind         TransitionKind
	ObservedAt   time.Time
}

// TransitionKind enumerates the shapes of position quantity change
// the manager can detect from snapshot diffs alone (no fill data).
type TransitionKind string

const (
	TransitionOpened    TransitionKind = "OPENED"
	TransitionClosed    TransitionKind = "CLOSED"
	TransitionFlipped   TransitionKind = "FLIPPED"
	TransitionAdjusted  TransitionKind = "ADJUSTED"
)

// TransitionObserver is invoked synchronously while the manager
// holds no internal locks. Observers must not call back into the
// Manager on the same goroutine; spawn a goroutine if any further
// Manager calls are needed.
type TransitionObserver func(PositionTransition)

type entry struct {
	hydrating bool
	snapshot  *Snapshot
	stale     bool
	waitCh    chan struct{}
}

func NewManager(client HydrationClient) *Manager {
	return &Manager{
		client:         client,
		maxSnapshotAge: defaultSnapshotMaxAge,
		entries:        make(map[int64]*entry),
		sessionEntries: make(map[string]int64),
	}
}

func (m *Manager) SetMaxSnapshotAge(maxAge time.Duration) {
	if m == nil || maxAge <= 0 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.maxSnapshotAge = maxAge
}

// SubscribeTransitions registers a TransitionObserver that the
// manager will call after each fresh hydration whenever the new
// snapshot's positions differ from the prior snapshot's. The hook
// is the foundation for trade-closed MCP notifications: the
// notifications layer subscribes here and pushes a card via the
// streaming notifier when Kind == TransitionClosed.
//
// Observers are invoked synchronously after the manager releases
// its lock. The order of invocation matches registration order.
func (m *Manager) SubscribeTransitions(observer TransitionObserver) {
	if m == nil || observer == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.observers = append(m.observers, observer)
}

// Injects the hydration client after construction; wiring needs to
// happen after the broker boots.
func (m *Manager) SetHydrationClient(client HydrationClient) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.client = client
}

func (m *Manager) EnsureHydrated(ctx context.Context, sessionID string, subAccountID int64) (*Snapshot, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session ID is required")
	}
	if subAccountID <= 0 {
		return nil, fmt.Errorf("subaccount ID is required")
	}
	if m == nil || m.client == nil {
		return nil, fmt.Errorf("risk snapshot manager is unavailable")
	}

	for {
		m.mu.Lock()
		m.bindSessionLocked(sessionID, subAccountID)
		state := m.ensureEntryLocked(subAccountID)
		if state.snapshot != nil && !state.stale && !m.snapshotExpiredLocked(state.snapshot) {
			snapshot := state.snapshot.clone()
			m.mu.Unlock()
			return snapshot, nil
		}
		if state.hydrating {
			waitCh := state.waitCh
			m.mu.Unlock()
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-waitCh:
			}
			continue
		}

		state.hydrating = true
		state.waitCh = make(chan struct{})
		waitCh := state.waitCh
		m.mu.Unlock()

		snapshot, err := hydrateSnapshot(ctx, m.client, subAccountID)

		m.mu.Lock()
		state = m.ensureEntryLocked(subAccountID)
		var transitions []PositionTransition
		var observersCopy []TransitionObserver
		if err == nil {
			transitions = diffPositionTransitionsLocked(state.snapshot, snapshot, sessionID, subAccountID)
			state.snapshot = snapshot
			state.stale = false
			if len(transitions) > 0 && len(m.observers) > 0 {
				observersCopy = append(observersCopy, m.observers...)
			}
		}
		state.hydrating = false
		close(waitCh)
		state.waitCh = nil
		m.mu.Unlock()

		for _, t := range transitions {
			for _, observer := range observersCopy {
				observer(t)
			}
		}

		if err != nil {
			return nil, err
		}
		return snapshot.clone(), nil
	}
}

func (m *Manager) Invalidate(sessionID string) {
	if m == nil || sessionID == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	subAccountID, ok := m.sessionEntries[sessionID]
	delete(m.sessionEntries, sessionID)
	if ok {
		m.deleteEntryIfUnusedLocked(subAccountID)
	}
}

func (m *Manager) SessionClosed(sessionID string) {
	m.Invalidate(sessionID)
}

func (m *Manager) RemoveOrders(sessionID string, refs []OrderRef) {
	if m == nil || sessionID == "" || len(refs) == 0 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	state := m.entryForSessionLocked(sessionID)
	if state == nil || state.snapshot == nil {
		return
	}
	for _, ref := range refs {
		state.snapshot.removeOrder(ref)
	}
}

func (m *Manager) RemoveOrdersBySymbol(sessionID string, symbol string) {
	if m == nil || sessionID == "" || strings.TrimSpace(symbol) == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	state := m.entryForSessionLocked(sessionID)
	if state == nil || state.snapshot == nil {
		return
	}
	state.snapshot.removeOrdersBySymbol(symbol)
}

func (m *Manager) UpsertOrder(sessionID string, order Order) {
	if m == nil || sessionID == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	state := m.entryForSessionLocked(sessionID)
	if state == nil || state.snapshot == nil {
		return
	}
	state.snapshot.upsertOrder(order)
}

func (s *Snapshot) AllOpenOrders() []Order {
	if s == nil {
		return []Order{}
	}
	out := make([]Order, 0, len(s.openOrdersByVenue))
	for _, order := range s.openOrdersByVenue {
		out = append(out, order)
	}
	return out
}

func (s *Snapshot) LookupOrder(venueOrderID string, clientOrderID string) (*Order, bool) {
	if s == nil {
		return nil, false
	}
	if clientOrderID = strings.TrimSpace(clientOrderID); clientOrderID != "" {
		order, ok := s.openOrdersByClient[clientOrderID]
		if !ok {
			return nil, false
		}
		copy := order
		return &copy, true
	}
	if venueOrderID = strings.TrimSpace(venueOrderID); venueOrderID != "" {
		order, ok := s.openOrdersByVenue[venueOrderID]
		if !ok {
			return nil, false
		}
		copy := order
		return &copy, true
	}
	return nil, false
}

func (s *Snapshot) PendingExposure(symbol string) (decimal.Decimal, decimal.Decimal) {
	if s == nil {
		return decimal.Zero, decimal.Zero
	}
	buyPending := decimal.Zero
	sellPending := decimal.Zero
	for _, order := range s.openOrdersByVenue {
		if !strings.EqualFold(order.Symbol, symbol) || order.ReduceOnly {
			continue
		}
		if strings.EqualFold(order.Side, "SELL") {
			sellPending = sellPending.Add(order.RemainingQuantity)
			continue
		}
		buyPending = buyPending.Add(order.RemainingQuantity)
	}
	return buyPending, sellPending
}

func (s *Snapshot) SignedPosition(symbol string) decimal.Decimal {
	if s == nil {
		return decimal.Zero
	}
	return s.positionsBySymbol[normalizeSymbol(symbol)]
}

func (s *Snapshot) clone() *Snapshot {
	if s == nil {
		return nil
	}
	out := &Snapshot{
		openOrdersByClient: make(map[string]Order, len(s.openOrdersByClient)),
		openOrdersByVenue:  make(map[string]Order, len(s.openOrdersByVenue)),
		positionsBySymbol:  make(map[string]decimal.Decimal, len(s.positionsBySymbol)),
		refreshedAtMs:      s.refreshedAtMs,
		subAccountID:       s.subAccountID,
	}
	for key, value := range s.openOrdersByClient {
		out.openOrdersByClient[key] = value
	}
	for key, value := range s.openOrdersByVenue {
		out.openOrdersByVenue[key] = value
	}
	for key, value := range s.positionsBySymbol {
		out.positionsBySymbol[key] = value
	}
	return out
}

func (s *Snapshot) removeOrder(ref OrderRef) {
	if s == nil {
		return
	}
	if ref.ClientOrderID = strings.TrimSpace(ref.ClientOrderID); ref.ClientOrderID != "" {
		order, ok := s.openOrdersByClient[ref.ClientOrderID]
		if ok {
			delete(s.openOrdersByClient, ref.ClientOrderID)
			if order.VenueOrderID != "" {
				delete(s.openOrdersByVenue, order.VenueOrderID)
			}
		}
	}
	if ref.VenueOrderID = strings.TrimSpace(ref.VenueOrderID); ref.VenueOrderID != "" {
		order, ok := s.openOrdersByVenue[ref.VenueOrderID]
		if ok {
			delete(s.openOrdersByVenue, ref.VenueOrderID)
			if order.ClientOrderID != "" {
				delete(s.openOrdersByClient, order.ClientOrderID)
			}
		}
	}
}

func (s *Snapshot) removeOrdersBySymbol(symbol string) {
	if s == nil {
		return
	}
	symbol = normalizeSymbol(symbol)
	for venueOrderID, order := range s.openOrdersByVenue {
		if normalizeSymbol(order.Symbol) != symbol {
			continue
		}
		delete(s.openOrdersByVenue, venueOrderID)
		if order.ClientOrderID != "" {
			delete(s.openOrdersByClient, order.ClientOrderID)
		}
	}
}

func (s *Snapshot) upsertOrder(order Order) {
	if s == nil {
		return
	}
	order.Symbol = normalizeSymbol(order.Symbol)
	if order.ClientOrderID != "" {
		if existing, ok := s.openOrdersByClient[order.ClientOrderID]; ok && order.VenueOrderID == "" {
			order.VenueOrderID = existing.VenueOrderID
		}
	}
	if order.VenueOrderID != "" {
		if existing, ok := s.openOrdersByVenue[order.VenueOrderID]; ok && order.ClientOrderID == "" {
			order.ClientOrderID = existing.ClientOrderID
		}
	}
	if order.RemainingQuantity.LessThanOrEqual(decimal.Zero) {
		s.removeOrder(OrderRef{ClientOrderID: order.ClientOrderID, VenueOrderID: order.VenueOrderID})
		return
	}
	if order.ClientOrderID != "" {
		s.openOrdersByClient[order.ClientOrderID] = order
	}
	if order.VenueOrderID != "" {
		s.openOrdersByVenue[order.VenueOrderID] = order
	}
}

func hydrateSnapshot(ctx context.Context, client HydrationClient, subAccountID int64) (*Snapshot, error) {
	positions, err := loadAllPositions(ctx, client, subAccountID)
	if err != nil {
		return nil, err
	}
	openOrders, err := loadAllOpenOrders(ctx, client, subAccountID)
	if err != nil {
		return nil, err
	}

	snapshot := &Snapshot{
		openOrdersByClient: make(map[string]Order),
		openOrdersByVenue:  make(map[string]Order),
		positionsBySymbol:  make(map[string]decimal.Decimal),
		refreshedAtMs:      snx_lib_utils_time.Now().UnixMilli(),
		subAccountID:       subAccountID,
	}
	for symbol, quantity := range positions {
		snapshot.positionsBySymbol[symbol] = quantity
	}
	for _, order := range openOrders {
		snapshot.upsertOrder(order)
	}
	return snapshot, nil
}

func loadAllOpenOrders(ctx context.Context, client HydrationClient, subAccountID int64) ([]Order, error) {
	offset := 0
	out := make([]Order, 0)
	for {
		items, err := client.GetOpenOrders(ctx, subAccountID, pageSize, offset)
		if err != nil {
			return nil, fmt.Errorf("load open orders: %w", err)
		}
		for _, item := range items {
			price, err := decimal.NewFromString(zeroIfEmpty(item.Price))
			if err != nil {
				return nil, fmt.Errorf("parse open order price: %w", err)
			}
			quantity, err := decimal.NewFromString(zeroIfEmpty(item.Quantity))
			if err != nil {
				return nil, fmt.Errorf("parse open order total quantity: %w", err)
			}
			remainingQuantity, err := decimal.NewFromString(zeroIfEmpty(firstNonEmpty(item.RemainingQuantity, item.Quantity)))
			if err != nil {
				return nil, fmt.Errorf("parse open order quantity: %w", err)
			}
			out = append(out, Order{
				ClientOrderID:     item.ClientOrderID,
				OrderType:         strings.ToUpper(strings.TrimSpace(item.OrderType)),
				Price:             price,
				Quantity:          quantity,
				ReduceOnly:        item.ReduceOnly,
				RemainingQuantity: remainingQuantity,
				Side:              strings.ToUpper(strings.TrimSpace(item.Side)),
				Symbol:            normalizeSymbol(item.Symbol),
				VenueOrderID:      item.VenueOrderID,
			})
		}
		if len(items) < pageSize {
			return out, nil
		}
		offset += pageSize
	}
}

func loadAllPositions(ctx context.Context, client HydrationClient, subAccountID int64) (map[string]decimal.Decimal, error) {
	offset := 0
	out := make(map[string]decimal.Decimal)
	for {
		items, err := client.GetPositions(ctx, subAccountID, pageSize, offset)
		if err != nil {
			return nil, fmt.Errorf("load positions: %w", err)
		}
		for _, item := range items {
			quantity, err := decimal.NewFromString(zeroIfEmpty(item.Quantity))
			if err != nil {
				return nil, fmt.Errorf("parse position quantity: %w", err)
			}
			if strings.EqualFold(item.Side, "short") {
				quantity = quantity.Neg()
			}
			out[normalizeSymbol(item.Symbol)] = quantity
		}
		if len(items) < pageSize {
			return out, nil
		}
		offset += pageSize
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func (m *Manager) bindSessionLocked(sessionID string, subAccountID int64) {
	previousSubAccountID, ok := m.sessionEntries[sessionID]
	m.sessionEntries[sessionID] = subAccountID
	if ok && previousSubAccountID != subAccountID {
		m.deleteEntryIfUnusedLocked(previousSubAccountID)
	}
}

func (m *Manager) deleteEntryIfUnusedLocked(subAccountID int64) {
	for _, mappedSubAccountID := range m.sessionEntries {
		if mappedSubAccountID == subAccountID {
			return
		}
	}
	delete(m.entries, subAccountID)
}

func (m *Manager) ensureEntryLocked(subAccountID int64) *entry {
	state := m.entries[subAccountID]
	if state == nil {
		state = &entry{}
		m.entries[subAccountID] = state
	}
	return state
}

func (m *Manager) entryForSessionLocked(sessionID string) *entry {
	subAccountID, ok := m.sessionEntries[sessionID]
	if !ok {
		return nil
	}
	return m.entries[subAccountID]
}

func (m *Manager) snapshotExpiredLocked(snapshot *Snapshot) bool {
	if snapshot == nil || m.maxSnapshotAge <= 0 {
		return false
	}
	return snx_lib_utils_time.Since(time.UnixMilli(snapshot.refreshedAtMs)) >= m.maxSnapshotAge
}

// diffPositionTransitionsLocked compares the position quantities on
// two snapshots and emits a PositionTransition for each symbol whose
// quantity meaningfully changed. "Meaningfully" excludes pure
// magnitude-up changes on the same side (those are open-position
// adds, not state-machine transitions worth a notification); we
// still emit Kind=TransitionAdjusted for trims so observers can
// decide. The caller must hold m.mu.
func diffPositionTransitionsLocked(prior, current *Snapshot, sessionID string, subAccountID int64) []PositionTransition {
	if current == nil {
		return nil
	}
	priorMap := map[string]decimal.Decimal{}
	if prior != nil {
		for sym, qty := range prior.positionsBySymbol {
			priorMap[sym] = qty
		}
	}

	now := time.UnixMilli(current.refreshedAtMs)
	if now.IsZero() {
		now = snx_lib_utils_time.Now()
	}

	var out []PositionTransition
	seen := map[string]struct{}{}
	for sym, currentQty := range current.positionsBySymbol {
		seen[sym] = struct{}{}
		priorQty := priorMap[sym]
		if priorQty.Equal(currentQty) {
			continue
		}
		out = append(out, PositionTransition{
			SubAccountID: subAccountID,
			SessionID:    sessionID,
			Symbol:       sym,
			Prior:        priorQty,
			Current:      currentQty,
			Kind:         classifyTransition(priorQty, currentQty),
			ObservedAt:   now,
		})
	}
	for sym, priorQty := range priorMap {
		if _, ok := seen[sym]; ok {
			continue
		}
		if priorQty.IsZero() {
			continue
		}
		out = append(out, PositionTransition{
			SubAccountID: subAccountID,
			SessionID:    sessionID,
			Symbol:       sym,
			Prior:        priorQty,
			Current:      decimal.Zero,
			Kind:         TransitionClosed,
			ObservedAt:   now,
		})
	}
	return out
}

func classifyTransition(prior, current decimal.Decimal) TransitionKind {
	switch {
	case prior.IsZero() && !current.IsZero():
		return TransitionOpened
	case !prior.IsZero() && current.IsZero():
		return TransitionClosed
	case prior.Sign() != current.Sign() && !prior.IsZero() && !current.IsZero():
		return TransitionFlipped
	default:
		return TransitionAdjusted
	}
}

func normalizeSymbol(symbol string) string {
	return strings.ToUpper(strings.TrimSpace(symbol))
}

func zeroIfEmpty(value string) string {
	if strings.TrimSpace(value) == "" {
		return "0"
	}
	return value
}
