package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/zalando/go-keyring"

	"hark/internal/config"
	"hark/internal/ipc"
	"hark/internal/secrets"
)

func saveProvider(t *testing.T, app *appState, id, baseURL string, models ...string) {
	t.Helper()
	if _, err := app.providersSave(context.Background(), requestWithParams(t, "providers_save", providerSaveRequest{
		ID: id, Label: id, BaseURL: baseURL, Models: models,
	})); err != nil {
		t.Fatalf("providersSave returned error: %v", err)
	}
}

func TestProvidersSaveRegistersProviderAndModels(t *testing.T) {
	app := newTestApp(&fakeHistory{})

	saveProvider(t, app, "local", "http://localhost:8000/v1", "llama-3", "llama-2")

	cfg := app.snapshotConfig()
	if !hasProvider(cfg, "local", "http://localhost:8000/v1") {
		t.Fatalf("merged providers = %#v, want local", cfg.Providers)
	}
	if !hasModel(cfg, "llama-3", "local") || !hasModel(cfg, "llama-2", "local") {
		t.Fatalf("merged models = %#v, want llama-3 and llama-2 -> local", cfg.Provider.Models)
	}
	if _, ok := app.snapshotProviders()["local"]; !ok {
		t.Fatal("providers map does not contain local")
	}
}

func TestProvidersSaveDefaultsLabelToID(t *testing.T) {
	app := newTestApp(&fakeHistory{})
	if _, err := app.providersSave(context.Background(), requestWithParams(t, "providers_save", providerSaveRequest{
		ID: "corp-gw", BaseURL: "https://gw.example.com/v1",
	})); err != nil {
		t.Fatalf("providersSave returned error: %v", err)
	}

	cfg := app.snapshotConfig()
	for _, spec := range cfg.Providers {
		if spec.ID == "corp-gw" && spec.Label != "corp-gw" {
			t.Fatalf("label = %q, want default %q", spec.Label, "corp-gw")
		}
	}
}

func TestProvidersSaveUpdatesBaseURL(t *testing.T) {
	app := newTestApp(&fakeHistory{})

	saveProvider(t, app, "local", "http://localhost:8000/v1", "llama-3")
	saveProvider(t, app, "local", "http://localhost:9000/v1", "llama-3")

	cfg := app.snapshotConfig()
	if !hasProvider(cfg, "local", "http://localhost:9000/v1") {
		t.Fatalf("merged providers = %#v, want updated base URL", cfg.Providers)
	}
	if !hasModel(cfg, "llama-3", "local") {
		t.Fatal("model was dropped during an in-place edit")
	}
}

func TestProvidersSaveRemovesDroppedModels(t *testing.T) {
	app := newTestApp(&fakeHistory{})

	saveProvider(t, app, "local", "http://localhost:8000/v1", "llama-3", "llama-2")
	saveProvider(t, app, "local", "http://localhost:8000/v1", "llama-3")

	cfg := app.snapshotConfig()
	if !hasModel(cfg, "llama-3", "local") {
		t.Fatal("kept model is missing")
	}
	if hasModel(cfg, "llama-2", "local") {
		t.Fatal("dropped model still present")
	}
}

func TestProvidersSaveRejectsBuiltinID(t *testing.T) {
	app := newTestApp(&fakeHistory{})
	_, err := app.providersSave(context.Background(), requestWithParams(t, "providers_save", providerSaveRequest{
		ID: "openai", BaseURL: "http://localhost/v1",
	}))
	if err == nil {
		t.Fatal("expected error for a built-in provider id")
	}
}

func TestProvidersSaveRejectsInvalidBaseURL(t *testing.T) {
	app := newTestApp(&fakeHistory{})
	_, err := app.providersSave(context.Background(), requestWithParams(t, "providers_save", providerSaveRequest{
		ID: "local", BaseURL: "not-a-url",
	}))
	if err == nil {
		t.Fatal("expected error for an invalid base URL")
	}
}

func TestProvidersSaveRejectsConfigProvider(t *testing.T) {
	app := newTestApp(&fakeHistory{})
	app.baseCfg.Providers = append(app.baseCfg.Providers, config.ProviderSpec{ID: "cfg-provider", BaseURL: "http://cfg/v1"})
	app.cfg = app.baseCfg

	_, err := app.providersSave(context.Background(), requestWithParams(t, "providers_save", providerSaveRequest{
		ID: "cfg-provider", BaseURL: "http://elsewhere/v1",
	}))
	if err == nil {
		t.Fatal("expected error editing a config.lua provider")
	}
}

func TestProvidersSaveRejectsModelOwnedElsewhere(t *testing.T) {
	app := newTestApp(&fakeHistory{})
	saveProvider(t, app, "one", "http://localhost:8000/v1", "shared-model")

	_, err := app.providersSave(context.Background(), requestWithParams(t, "providers_save", providerSaveRequest{
		ID: "two", BaseURL: "http://localhost:9000/v1", Models: []string{"shared-model"},
	}))
	if err == nil {
		t.Fatal("expected error for a model owned by another provider")
	}
}

func TestProvidersRemoveDeletesProviderAndModels(t *testing.T) {
	app := newTestApp(&fakeHistory{})
	saveProvider(t, app, "local", "http://localhost:8000/v1", "llama-3")

	if _, err := app.providersRemove(context.Background(), requestWithParams(t, "providers_remove", providerRemoveRequest{ID: "local"})); err != nil {
		t.Fatalf("providersRemove returned error: %v", err)
	}

	cfg := app.snapshotConfig()
	if hasProvider(cfg, "local", "") || hasModel(cfg, "llama-3", "local") {
		t.Fatalf("provider/model not removed: %#v %#v", cfg.Providers, cfg.Provider.Models)
	}
	if _, ok := app.snapshotProviders()["local"]; ok {
		t.Fatal("providers map still contains local")
	}
}

func TestProvidersRemoveRejectsUnmanagedProvider(t *testing.T) {
	app := newTestApp(&fakeHistory{})
	_, err := app.providersRemove(context.Background(), requestWithParams(t, "providers_remove", providerRemoveRequest{ID: "openai"}))
	if err == nil {
		t.Fatal("expected error removing a provider not managed from the panel")
	}
}

func listProviders(t *testing.T, app *appState) []providerListEntry {
	t.Helper()
	result, err := app.providersList(context.Background(), ipc.Request{Method: "providers_list"})
	if err != nil {
		t.Fatalf("providersList returned error: %v", err)
	}
	entries, ok := result.([]providerListEntry)
	if !ok {
		t.Fatalf("providersList returned %T, want []providerListEntry", result)
	}
	return entries
}

func findProviderEntry(t *testing.T, entries []providerListEntry, id string) providerListEntry {
	t.Helper()
	for _, entry := range entries {
		if entry.ID == id {
			return entry
		}
	}
	t.Fatalf("provider %q not in list %#v", id, entries)
	return providerListEntry{}
}

func addConfigProvider(app *appState, id, label, baseURL, modelID string) {
	app.baseCfg.Providers = append(app.baseCfg.Providers, config.ProviderSpec{ID: id, Label: label, BaseURL: baseURL})
	app.baseCfg.Provider.Models = append(app.baseCfg.Provider.Models, config.ModelConfig{
		ID: modelID, Label: modelID + " label", Provider: id, ReasoningEfforts: []string{"auto"},
	})
	app.cfg = app.baseCfg
}

func TestProvidersListIncludesConfigProvidersAsUnmanaged(t *testing.T) {
	keyring.MockInit()
	app := newTestApp(&fakeHistory{})
	addConfigProvider(app, "lmstudio", "LM Studio", "http://127.0.0.1:1234/v1", "qwen")
	saveProvider(t, app, "local", "http://localhost:8000/v1", "llama-3")

	entries := listProviders(t, app)
	if len(entries) != 2 {
		t.Fatalf("entries = %#v, want lmstudio and local", entries)
	}

	cfgEntry := findProviderEntry(t, entries, "lmstudio")
	if cfgEntry.Managed {
		t.Error("config.lua provider must be reported as unmanaged")
	}
	if cfgEntry.Label != "LM Studio" || cfgEntry.BaseURL != "http://127.0.0.1:1234/v1" {
		t.Errorf("config entry = %#v", cfgEntry)
	}
	if len(cfgEntry.Models) != 1 || cfgEntry.Models[0].ID != "qwen" || cfgEntry.Models[0].Label != "qwen label" {
		t.Errorf("config entry models = %#v, want qwen from config.lua", cfgEntry.Models)
	}

	stored := findProviderEntry(t, entries, "local")
	if !stored.Managed {
		t.Error("panel-created provider must be reported as managed")
	}
	if len(stored.Models) != 1 || stored.Models[0].ID != "llama-3" {
		t.Errorf("stored entry models = %#v", stored.Models)
	}
}

func TestProvidersListSortsByID(t *testing.T) {
	keyring.MockInit()
	app := newTestApp(&fakeHistory{})
	saveProvider(t, app, "zeta", "http://z/v1")
	addConfigProvider(app, "alpha", "Alpha", "http://a/v1", "m")
	saveProvider(t, app, "mid", "http://m/v1")

	var ids []string
	for _, entry := range listProviders(t, app) {
		ids = append(ids, entry.ID)
	}
	if len(ids) != 3 || ids[0] != "alpha" || ids[1] != "mid" || ids[2] != "zeta" {
		t.Fatalf("ids = %v, want [alpha mid zeta]", ids)
	}
}

func TestProvidersListDefaultsEmptyLabelToID(t *testing.T) {
	keyring.MockInit()
	app := newTestApp(&fakeHistory{})
	addConfigProvider(app, "corp-gw", "", "https://gw.example.com/v1", "m")

	if got := findProviderEntry(t, listProviders(t, app), "corp-gw").Label; got != "corp-gw" {
		t.Fatalf("label = %q, want the id when the label is empty", got)
	}
}

func TestProvidersListReportsKeyStatus(t *testing.T) {
	keyring.MockInit()
	app := newTestApp(&fakeHistory{})
	addConfigProvider(app, "keyed", "Keyed", "http://k/v1", "m1")
	addConfigProvider(app, "bare", "Bare", "http://b/v1", "m2")
	if err := secrets.SetProviderAPIKey("keyed", "secret-value"); err != nil {
		t.Fatalf("store key: %v", err)
	}

	entries := listProviders(t, app)

	keyed := findProviderEntry(t, entries, "keyed")
	if !keyed.KeyConfigured || keyed.KeySource != "secret-service" {
		t.Errorf("keyed = configured %v source %q, want true secret-service", keyed.KeyConfigured, keyed.KeySource)
	}
	bare := findProviderEntry(t, entries, "bare")
	if bare.KeyConfigured || bare.KeySource != "none" {
		t.Errorf("bare = configured %v source %q, want false none", bare.KeyConfigured, bare.KeySource)
	}
}

func TestProvidersListReportsEnvironmentKey(t *testing.T) {
	keyring.MockInit()
	t.Setenv("ENVED_API_KEY", "from-env")
	app := newTestApp(&fakeHistory{})
	addConfigProvider(app, "enved", "Enved", "http://e/v1", "m")

	entry := findProviderEntry(t, listProviders(t, app), "enved")
	if !entry.KeyConfigured || entry.KeySource != "environment" {
		t.Fatalf("entry = configured %v source %q, want true environment", entry.KeyConfigured, entry.KeySource)
	}
}

func TestProvidersListNeverExposesKeyValue(t *testing.T) {
	keyring.MockInit()
	app := newTestApp(&fakeHistory{})
	addConfigProvider(app, "keyed", "Keyed", "http://k/v1", "m")
	if err := secrets.SetProviderAPIKey("keyed", "super-secret-token"); err != nil {
		t.Fatalf("store key: %v", err)
	}

	encoded, err := json.Marshal(listProviders(t, app))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if bytes.Contains(encoded, []byte("super-secret-token")) {
		t.Fatalf("provider list leaked the key value: %s", encoded)
	}
}

func TestProvidersListReportsUnknownWhenKeyringFails(t *testing.T) {
	keyring.MockInitWithError(errors.New("secret service unavailable"))
	app := newTestApp(&fakeHistory{})
	addConfigProvider(app, "lmstudio", "LM Studio", "http://127.0.0.1:1234/v1", "m")

	entry := findProviderEntry(t, listProviders(t, app), "lmstudio")
	if entry.KeyConfigured || entry.KeySource != keySourceUnknown {
		t.Fatalf("entry = configured %v source %q, want false %q", entry.KeyConfigured, entry.KeySource, keySourceUnknown)
	}
}

func TestProvidersListConfigProviderShadowsStoredProvider(t *testing.T) {
	keyring.MockInit()
	app := newTestApp(&fakeHistory{})
	saveProvider(t, app, "dup", "http://stored/v1", "stored-model")
	addConfigProvider(app, "dup", "From config", "http://config/v1", "config-model")

	entries := listProviders(t, app)
	if len(entries) != 1 {
		t.Fatalf("entries = %#v, want a single shadowed entry", entries)
	}
	if entries[0].Managed || entries[0].BaseURL != "http://config/v1" {
		t.Fatalf("entry = %#v, want the unmanaged config.lua definition", entries[0])
	}
}

func hasProvider(cfg config.Config, id, baseURL string) bool {
	for _, spec := range cfg.Providers {
		if spec.ID == id && (baseURL == "" || spec.BaseURL == baseURL) {
			return true
		}
	}
	return false
}

func hasModel(cfg config.Config, id, provider string) bool {
	for _, model := range cfg.Provider.Models {
		if model.ID == id && model.Provider == provider {
			return true
		}
	}
	return false
}
