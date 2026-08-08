package domain

import "encoding/json"

// OAuthApp is a reusable OAuth client registration used by multiple
// connectors. One app serves every connector hitting the same OAuth endpoint.
type OAuthApp struct {
	Meta   ObjectMeta     `json:"metadata"`
	Spec   OAuthAppSpec   `json:"spec"`
	Status OAuthAppStatus `json:"status,omitempty"`
}

// OAuthAppSpec defines the OAuth configuration.
type OAuthAppSpec struct {
	AuthorizationURL             string   `json:"authorizationUrl"`
	TokenURL                     string   `json:"tokenUrl"`
	ClientID                     string   `json:"clientId"`
	ClientSecretRef              SecretRef `json:"clientSecretRef"`
	Scopes                       []string `json:"scopes,omitempty"`
	SupportsDynamicRegistration  bool     `json:"supportsDynamicRegistration,omitempty"`
}

// SecretRef points to a key in a Secret.
type SecretRef struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

// OAuthAppStatus tracks which connectors use this app.
type OAuthAppStatus struct {
	Connectors []string `json:"connectors,omitempty"`
}

// ToResource converts an OAuthApp to a store Resource.
func (a *OAuthApp) ToResource() (*Resource, error) {
	spec, err := json.Marshal(a.Spec)
	if err != nil {
		return nil, err
	}
	status, err := json.Marshal(a.Status)
	if err != nil {
		return nil, err
	}
	return &Resource{
		Kind:   KindOAuthApp,
		Meta:   a.Meta,
		Spec:   spec,
		Status: status,
	}, nil
}

// OAuthAppFromResource converts a store Resource to an OAuthApp.
func OAuthAppFromResource(r *Resource) (*OAuthApp, error) {
	app := &OAuthApp{Meta: r.Meta}
	if len(r.Spec) > 0 {
		if err := json.Unmarshal(r.Spec, &app.Spec); err != nil {
			return nil, err
		}
	}
	if len(r.Status) > 0 {
		if err := json.Unmarshal(r.Status, &app.Status); err != nil {
			return nil, err
		}
	}
	return app, nil
}
