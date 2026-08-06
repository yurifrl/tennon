package auth

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/yurifrl/tennon/domain"
	"github.com/yurifrl/tennon/store"
)

func TestPolicyAuthorizer_Resolution(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))

	// Create in-memory store
	st, err := store.NewFromConfig(store.Config{Backend: "memory"})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	// Create a test connector with tags
	connector := &domain.Connector{
		Kind: domain.KindTennonConnector,
		Meta: domain.ObjectMeta{Name: "test-connector", Namespace: "tennon"},
		Spec: domain.ConnectorSpec{
			Tags: []string{"sensitive"},
		},
	}
	connRes, _ := connector.ToResource()
	if _, err := st.Apply(ctx, connRes); err != nil {
		t.Fatal(err)
	}

	// Create policies
	policies := []*domain.Policy{
		{
			Meta: domain.ObjectMeta{Name: "allow-alice", Namespace: "tennon"},
			Spec: domain.PolicySpec{
				Rules: []domain.Rule{
					{
						Subjects: []domain.SubjectMatch{
							{Kind: "User", Match: "alice@example.com"},
						},
						Resource: domain.ResourceMatch{Connector: "test-connector"},
						Effect:   domain.EffectAllow,
					},
				},
			},
		},
		{
			Meta: domain.ObjectMeta{Name: "ask-sensitive", Namespace: "tennon"},
			Spec: domain.PolicySpec{
				Rules: []domain.Rule{
					{
						Resource: domain.ResourceMatch{Tag: "sensitive"},
						Effect:   domain.EffectAsk,
					},
				},
			},
		},
		{
			Meta: domain.ObjectMeta{Name: "block-bob", Namespace: "tennon"},
			Spec: domain.PolicySpec{
				Rules: []domain.Rule{
					{
						Subjects: []domain.SubjectMatch{
							{Kind: "User", Match: "bob@example.com"},
						},
						Resource: domain.ResourceMatch{Tool: "delete_*"},
						Effect:   domain.EffectBlock,
					},
				},
			},
		},
	}

	for _, p := range policies {
		res, _ := p.ToResource()
		if _, err := st.Apply(ctx, res); err != nil {
			t.Fatal(err)
		}
	}

	// Create authorizer
	authz, err := NewPolicyAuthorizer(ctx, st, logger)
	if err != nil {
		t.Fatal(err)
	}
	defer authz.Close()

	// Wait a bit for policies to load
	time.Sleep(50 * time.Millisecond)

	tests := []struct {
		name    string
		subject domain.Subject
		call    domain.ToolCall
		want    domain.Effect
	}{
		{
			name:    "alice allowed",
			subject: domain.Subject{Kind: "User", ID: "alice", Email: "alice@example.com"},
			call:    domain.ToolCall{Connector: "test-connector", Tool: "read_data"},
			// alice has allow, but tag "sensitive" forces ask => ask wins
			want: domain.EffectAsk,
		},
		{
			name:    "bob blocked on delete",
			subject: domain.Subject{Kind: "User", ID: "bob", Email: "bob@example.com"},
			call:    domain.ToolCall{Connector: "test-connector", Tool: "delete_all"},
			// bob has explicit block on delete_* => block wins over ask from tag
			want: domain.EffectBlock,
		},
		{
			name:    "unknown user default deny",
			subject: domain.Subject{Kind: "User", ID: "charlie", Email: "charlie@example.com"},
			call:    domain.ToolCall{Connector: "other-connector", Tool: "read_data"},
			want:    domain.EffectBlock, // default-deny: no matching allow
		},
		{
			name:    "sensitive tag requires ask",
			subject: domain.Subject{Kind: "User", ID: "dave", Email: "dave@example.com"},
			call:    domain.ToolCall{Connector: "test-connector", Tool: "write_data"},
			// no user-specific rules, but tag is sensitive => ask
			want: domain.EffectAsk,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := authz.Decide(ctx, tt.subject, tt.call)
			if got.Effect != tt.want {
				t.Errorf("Decide() = %v, want %v (reason: %s)", got.Effect, tt.want, got.Reason)
			}
		})
	}
}

func TestPolicyAuthorizer_GroupMatch(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))

	st, err := store.NewFromConfig(store.Config{Backend: "memory"})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	// Policy that grants access to okta:sre group
	policy := &domain.Policy{
		Meta: domain.ObjectMeta{Name: "allow-sre", Namespace: "tennon"},
		Spec: domain.PolicySpec{
			Rules: []domain.Rule{
				{
					Subjects: []domain.SubjectMatch{
						{Kind: "Group", Provider: "okta", Match: "sre"},
					},
					Resource: domain.ResourceMatch{Connector: "prod-db"},
					Effect:   domain.EffectAllow,
				},
			},
		},
	}
	res, _ := policy.ToResource()
	if _, err := st.Apply(ctx, res); err != nil {
		t.Fatal(err)
	}

	authz, err := NewPolicyAuthorizer(ctx, st, logger)
	if err != nil {
		t.Fatal(err)
	}
	defer authz.Close()

	time.Sleep(50 * time.Millisecond)

	// User with okta:sre group membership
	subject := domain.Subject{
		Kind:   "User",
		ID:     "eve",
		Email:  "eve@example.com",
		Groups: []string{"okta:sre", "okta:eng"},
	}

	call := domain.ToolCall{Connector: "prod-db", Tool: "query"}
	got := authz.Decide(ctx, subject, call)
	if got.Effect != domain.EffectAllow {
		t.Errorf("group match failed: got %v, want allow", got.Effect)
	}

	// User without the group
	subject2 := domain.Subject{
		Kind:   "User",
		ID:     "frank",
		Email:  "frank@example.com",
		Groups: []string{"okta:eng"},
	}
	got2 := authz.Decide(ctx, subject2, call)
	if got2.Effect != domain.EffectBlock {
		t.Errorf("non-member should be denied: got %v, want block", got2.Effect)
	}
}

func TestPolicyAuthorizer_LiveUpdate(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))

	st, err := store.NewFromConfig(store.Config{Backend: "memory"})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	authz, err := NewPolicyAuthorizer(ctx, st, logger)
	if err != nil {
		t.Fatal(err)
	}
	defer authz.Close()

	time.Sleep(50 * time.Millisecond)

	subject := domain.Subject{Kind: "User", ID: "grace", Email: "grace@example.com"}
	call := domain.ToolCall{Connector: "api", Tool: "write"}

	// Initially no policy => block (default-deny)
	got1 := authz.Decide(ctx, subject, call)
	if got1.Effect != domain.EffectBlock {
		t.Errorf("before policy: got %v, want block", got1.Effect)
	}

	// Add a policy that allows grace
	policy := &domain.Policy{
		Meta: domain.ObjectMeta{Name: "allow-grace", Namespace: "tennon"},
		Spec: domain.PolicySpec{
			Rules: []domain.Rule{
				{
					Subjects: []domain.SubjectMatch{
						{Kind: "User", Match: "grace@example.com"},
					},
					Resource: domain.ResourceMatch{Connector: "api"},
					Effect:   domain.EffectAllow,
				},
			},
		},
	}
	res, _ := policy.ToResource()
	if _, err := st.Apply(ctx, res); err != nil {
		t.Fatal(err)
	}

	// Wait for watch to pick up the change
	time.Sleep(100 * time.Millisecond)

	// Now grace should be allowed
	got2 := authz.Decide(ctx, subject, call)
	if got2.Effect != domain.EffectAllow {
		t.Errorf("after policy: got %v, want allow", got2.Effect)
	}
}
