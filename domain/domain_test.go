package domain

import "testing"

func TestToolsSpec_ExposeAndUpstreamName(t *testing.T) {
	s := ToolsSpec{ExposePrefix: "grafana_"}
	if got := s.ExposedName("get_dashboard"); got != "grafana_get_dashboard" {
		t.Fatalf("ExposedName = %q", got)
	}
	if got := s.ExposedName("grafana_get_dashboard"); got != "grafana_get_dashboard" {
		t.Fatalf("ExposedName double-prefixed: %q", got)
	}
	if got := s.UpstreamName("grafana_get_dashboard"); got != "get_dashboard" {
		t.Fatalf("UpstreamName = %q", got)
	}
	if got := s.UpstreamName("unprefixed"); got != "unprefixed" {
		t.Fatalf("UpstreamName passthrough = %q", got)
	}
}

func TestToolsSpec_Permits(t *testing.T) {
	cases := []struct {
		name string
		spec ToolsSpec
		tool string
		want bool
	}{
		{"empty allows all", ToolsSpec{}, "anything", true},
		{"allow glob match", ToolsSpec{Allow: []string{"get_*"}}, "get_x", true},
		{"allow glob miss", ToolsSpec{Allow: []string{"get_*"}}, "set_x", false},
		{"deny wins over allow", ToolsSpec{Allow: []string{"*"}, Deny: []string{"*delete*"}}, "hard_delete_all", false},
		{"deny only", ToolsSpec{Deny: []string{"drop_*"}}, "drop_table", false},
		{"deny only miss", ToolsSpec{Deny: []string{"drop_*"}}, "read_table", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.spec.Permits(tc.tool); got != tc.want {
				t.Fatalf("Permits(%q) = %v, want %v", tc.tool, got, tc.want)
			}
		})
	}
}

func TestConnectorRoundTrip(t *testing.T) {
	c := &Connector{
		Kind: KindTennonConnector,
		Meta: ObjectMeta{Name: "grafana", Namespace: "tennon"},
		Spec: ConnectorSpec{
			Runtime:   RuntimeSpec{Remote: &RemoteRuntime{URL: "https://x/mcp"}},
			Transport: "streamable-http",
			Tools:     ToolsSpec{ExposePrefix: "grafana_"},
			Headers:   map[string]string{"Authorization": "Bearer {{secret:g.token}}"},
		},
		Status: ConnectorStatus{Phase: PhaseReady, DiscoveredTools: 3, URL: "/mcp/grafana"},
	}
	res, err := c.ToResource()
	if err != nil {
		t.Fatal(err)
	}
	got, err := ConnectorFromResource(res)
	if err != nil {
		t.Fatal(err)
	}
	if got.Spec.RuntimeKind() != "remote" || got.Spec.Runtime.Remote.URL != "https://x/mcp" {
		t.Fatalf("runtime lost: %+v", got.Spec.Runtime)
	}
	if got.Status.DiscoveredTools != 3 || got.Status.URL != "/mcp/grafana" {
		t.Fatalf("status lost: %+v", got.Status)
	}
	if got.Spec.Headers["Authorization"] != "Bearer {{secret:g.token}}" {
		t.Fatalf("headers lost: %+v", got.Spec.Headers)
	}
}
