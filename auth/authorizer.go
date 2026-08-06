package auth

import (
	"context"
	"log/slog"
	"path"
	"strings"
	"sync"

	"github.com/yurifrl/tennon/domain"
)

// PolicyAuthorizer implements domain.Authorizer by evaluating Policy rules
// gathered from the store. It applies MOST-RESTRICTIVE-WINS resolution:
// block > ask > allow, with default-deny when no rules match.
//
// A full Cedar backend can slot behind this same port later; this Go
// implementation provides equivalent semantics for Phase 3.
type PolicyAuthorizer struct {
	store  domain.Store
	logger *slog.Logger

	mu       sync.RWMutex
	policies []*domain.Policy // cached compiled policies
	stop     func()           // watch cancellation
}

// NewPolicyAuthorizer creates an authorizer that watches Policy objects from
// the store and caches them in-memory for fast evaluation. It starts the watch
// immediately.
func NewPolicyAuthorizer(ctx context.Context, store domain.Store, logger *slog.Logger) (*PolicyAuthorizer, error) {
	a := &PolicyAuthorizer{
		store:  store,
		logger: logger,
	}

	// Load initial policies
	if err := a.reload(ctx); err != nil {
		return nil, err
	}

	// Start watch for live updates
	events, stopFn, err := store.Watch(ctx, domain.KindPolicy)
	if err != nil {
		return nil, err
	}
	a.stop = stopFn

	go a.watchLoop(ctx, events)

	return a, nil
}

// Decide evaluates all matching Policy rules for (subject, toolcall) and
// returns the most-restrictive effect. Any block > any ask > any allow.
// Default-deny: no matching allow => block.
func (a *PolicyAuthorizer) Decide(ctx context.Context, sub domain.Subject, call domain.ToolCall) domain.Decision {
	a.mu.RLock()
	policies := a.policies
	a.mu.RUnlock()

	var (
		foundBlock bool
		foundAsk   bool
		foundAllow bool
	)

	// Gather all matching rules
	for _, policy := range policies {
		for _, rule := range policy.Spec.Rules {
			if a.matches(ctx, sub, call, rule) {
				switch rule.Effect {
				case domain.EffectBlock:
					foundBlock = true
				case domain.EffectAsk:
					foundAsk = true
				case domain.EffectAllow:
					foundAllow = true
				}
			}
		}
	}

	// Most-restrictive-wins: block > ask > allow
	if foundBlock {
		return domain.Decision{Effect: domain.EffectBlock, Reason: "blocked by policy"}
	}
	if foundAsk {
		return domain.Decision{Effect: domain.EffectAsk, Reason: "approval required by policy"}
	}
	if foundAllow {
		return domain.Decision{Effect: domain.EffectAllow, Reason: "allowed by policy"}
	}

	// Default-deny: no matching rule => block
	return domain.Decision{Effect: domain.EffectBlock, Reason: "no matching policy (default-deny)"}
}

// matches reports whether a rule applies to this (subject, toolcall).
func (a *PolicyAuthorizer) matches(ctx context.Context, sub domain.Subject, call domain.ToolCall, rule domain.Rule) bool {
	// Check subject match
	if len(rule.Subjects) > 0 {
		if !a.matchesSubject(sub, rule.Subjects) {
			return false
		}
	}

	// Check resource match
	return a.matchesResource(ctx, call, rule.Resource)
}

// matchesSubject reports whether the subject matches any of the SubjectMatch entries.
func (a *PolicyAuthorizer) matchesSubject(sub domain.Subject, subjects []domain.SubjectMatch) bool {
	for _, sm := range subjects {
		switch sm.Kind {
		case "User":
			// Match by ID or email
			if sub.Kind == "User" && (sub.ID == sm.Match || sub.Email == sm.Match) {
				return true
			}
		case "Group":
			// Match by provider:group membership in subject.Groups
			groupID := sm.Provider + ":" + sm.Match
			for _, g := range sub.Groups {
				if g == groupID {
					return true
				}
			}
		case "EmailDomain":
			// Match by email suffix
			if sub.Email != "" && strings.HasSuffix(sub.Email, "@"+sm.Match) {
				return true
			}
		case "Token":
			if sub.Kind == "Token" && sub.ID == sm.Match {
				return true
			}
		case "ServiceAccount":
			if sub.Kind == "ServiceAccount" && sub.ID == sm.Match {
				return true
			}
		}
	}
	return false
}

// matchesResource reports whether the tool call matches the resource selector.
func (a *PolicyAuthorizer) matchesResource(ctx context.Context, call domain.ToolCall, rm domain.ResourceMatch) bool {
	// Match by connector name
	if rm.Connector != "" && rm.Connector == call.Connector {
		return true
	}

	// Match by tool glob
	if rm.Tool != "" {
		ok, err := path.Match(rm.Tool, call.Tool)
		if err == nil && ok {
			return true
		}
	}

	// Match by tag: load the connector and check its tags
	if rm.Tag != "" {
		connector, err := a.loadConnector(ctx, call.Connector)
		if err == nil {
			for _, tag := range connector.Spec.Tags {
				if tag == rm.Tag {
					return true
				}
			}
		}
	}

	return false
}

// loadConnector fetches a connector from the store for tag matching.
func (a *PolicyAuthorizer) loadConnector(ctx context.Context, name string) (*domain.Connector, error) {
	// Assume "tennon" namespace for now; a full implementation would track
	// namespace per connector or accept it in the call.
	res, err := a.store.Get(ctx, domain.KindTennonConnector, "tennon", name)
	if err != nil {
		// Try other connector kinds
		if res, err = a.store.Get(ctx, domain.KindOpenApiConnector, "tennon", name); err != nil {
			if res, err = a.store.Get(ctx, domain.KindGraphqlConnector, "tennon", name); err != nil {
				return nil, err
			}
		}
	}
	return domain.ConnectorFromResource(res)
}

// reload loads all Policy objects from the store.
func (a *PolicyAuthorizer) reload(ctx context.Context) error {
	resources, err := a.store.List(ctx, domain.KindPolicy, "")
	if err != nil {
		return err
	}

	policies := make([]*domain.Policy, 0, len(resources))
	for _, res := range resources {
		policy, err := domain.PolicyFromResource(res)
		if err != nil {
			a.logger.Warn("failed to parse policy", "name", res.Meta.Name, "err", err)
			continue
		}
		policies = append(policies, policy)
	}

	a.mu.Lock()
	a.policies = policies
	a.mu.Unlock()

	a.logger.Info("policies loaded", "count", len(policies))
	return nil
}

// watchLoop listens for policy changes and reloads on each event.
func (a *PolicyAuthorizer) watchLoop(ctx context.Context, events <-chan domain.WatchEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-events:
			if !ok {
				a.logger.Warn("policy watch closed")
				return
			}
			a.logger.Info("policy watch event", "type", event.Type, "name", event.Resource.Meta.Name)
			// Reload all policies on any change (simple approach for Phase 3)
			if err := a.reload(ctx); err != nil {
				a.logger.Error("failed to reload policies", "err", err)
			}
		}
	}
}

// Close stops the policy watch.
func (a *PolicyAuthorizer) Close() {
	if a.stop != nil {
		a.stop()
	}
}
