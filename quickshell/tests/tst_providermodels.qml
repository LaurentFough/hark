import QtQuick
import QtTest
import "../components"

TestCase {
    id: testCase

    name: "ProviderModels"

    ListModel {
        id: providersModel
    }

    SettingsPanel {
        id: panel
    }

    ProviderRow {
        id: row
    }

    SignalSpy {
        id: keySaveSpy

        target: panel
        signalName: "providerKeySaveRequested"
    }

    function init() {
        panel.resetProviderForm();
        row.managed = true;
        row.keyConfigured = false;
        row.keySource = "";
    }

    function test_listModelStoresJsonString() {
        providersModel.append({
            "id": "deepseek",
            "label": "Deepseek",
            "baseUrl": "https://api.deepseek.com",
            "modelsJson": JSON.stringify(["a", "b"])
        });

        const entry = providersModel.get(0);
        const models = JSON.parse(entry.modelsJson);
        compare(models.length, 2);
        compare(String(models[0]), "a");
        compare(String(models[1]), "b");
    }

    function test_beginEditProviderCollectsModelIds() {
        panel.beginEditProvider("deepseek", "Deepseek", "https://api.deepseek.com", ["a", "b"]);
        compare(panel.providerFormModels.length, 2);
        compare(panel.providerFormModels[0], "a");
        compare(panel.providerFormModels[1], "b");
    }

    function test_rowHidesKeyStatusWhenDaemonDoesNotReportIt() {
        compare(row.keyStatusText, "");
        compare(row.detailText, "");
    }

    function test_rowKeyStatusText_data() {
        return [
            {
                "tag": "keyring",
                "configured": true,
                "source": "secret-service",
                "text": "API key saved in keyring"
            },
            {
                "tag": "environment",
                "configured": true,
                "source": "environment",
                "text": "API key from environment variable"
            },
            {
                "tag": "none",
                "configured": false,
                "source": "none",
                "text": "No API key"
            },
            {
                "tag": "unknown",
                "configured": false,
                "source": "unknown",
                "text": "API key status unavailable"
            }
        ];
    }

    function test_rowKeyStatusText(data) {
        row.keyConfigured = data.configured;
        row.keySource = data.source;
        compare(row.keyStatusText, data.text);
    }

    function test_rowMarksConfigProviders() {
        row.managed = false;
        row.keySource = "none";
        compare(row.detailText, "config.lua · No API key");
        row.managed = true;
        compare(row.detailText, "No API key");
    }

    function test_rowOnlyOffersClearForKeyringKeys() {
        row.keyConfigured = true;
        row.keySource = "secret-service";
        verify(row.keyInKeyring);
        row.keySource = "environment";
        verify(!row.keyInKeyring);
        row.keyConfigured = false;
        row.keySource = "none";
        verify(!row.keyInKeyring);
    }

    function test_beginSetProviderKeyTargetsOneProvider() {
        panel.beginSetProviderKey("lmstudio");
        compare(panel.keyEditingProviderID, "lmstudio");
        compare(panel.providerKeyInput, "");
        panel.beginSetProviderKey("other");
        compare(panel.keyEditingProviderID, "other");
    }

    function test_openingOtherFormsClosesKeyForm() {
        panel.beginSetProviderKey("lmstudio");
        panel.beginAddProvider();
        compare(panel.keyEditingProviderID, "");
        verify(panel.providerFormVisible);

        panel.beginSetProviderKey("lmstudio");
        verify(!panel.providerFormVisible);

        panel.beginEditProvider("deepseek", "Deepseek", "https://api.deepseek.com", []);
        compare(panel.keyEditingProviderID, "");
    }

    function test_resetProviderFormClearsTypedKey() {
        panel.beginSetProviderKey("lmstudio");
        panel.providerKeyInput = "sk-should-not-linger";
        panel.resetProviderForm();
        compare(panel.keyEditingProviderID, "");
        compare(panel.providerKeyInput, "");
    }

    function test_keySaveSignalCarriesIdAndKey() {
        keySaveSpy.clear();
        panel.providerKeySaveRequested("lmstudio", "sk-test");
        compare(keySaveSpy.count, 1);
        compare(keySaveSpy.signalArguments[0][0], "lmstudio");
        compare(keySaveSpy.signalArguments[0][1], "sk-test");
    }
}
