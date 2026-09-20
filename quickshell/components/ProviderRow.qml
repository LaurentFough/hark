import QtQuick

Item {
    id: control

    property string providerId: ""
    property string label: ""
    property string baseUrl: ""
    property var models: []
    property bool managed: true
    property bool keyConfigured: false
    // Empty when the daemon does not report key status.
    property string keySource: ""
    property bool busy: false
    property var theme: null
    property string fontFamily: ""

    readonly property bool keyInKeyring: keyConfigured && keySource === "secret-service"
    readonly property string keyStatusText: {
        if (keySource === "")
            return "";
        if (keySource === "unknown")
            return "API key status unavailable";
        if (keySource === "environment")
            return "API key from environment variable";
        return keyConfigured ? "API key saved in keyring" : "No API key";
    }
    readonly property string detailText: (managed ? "" : "config.lua · ") + keyStatusText

    signal editRequested()
    signal removeRequested()
    signal keySetRequested()
    signal keyClearRequested()

    function c(name, fallback) {
        return theme && theme[name] ? theme[name] : fallback;
    }

    function cornerRadius(maximum) {
        const configured = theme && theme.corner_radius !== undefined ? Number(theme.corner_radius) : maximum;
        return Math.min(Math.max(0, configured), maximum);
    }

    function fontSize(role, fallback) {
        const configured = theme ? Number(theme["font_" + role]) : NaN;
        return isFinite(configured) && configured > 0 ? configured : fallback;
    }

    implicitHeight: contentColumn.implicitHeight

    Column {
        id: contentColumn

        width: parent.width
        spacing: 3

        Item {
            width: parent.width
            height: 26

            Text {
                anchors.left: parent.left
                anchors.right: buttonRow.left
                anchors.rightMargin: 8
                anchors.verticalCenter: parent.verticalCenter
                text: control.label
                color: control.c("text_strong", "#f2f4f8")
                font.family: control.fontFamily
                font.pixelSize: control.fontSize("subtitle", 13)
                font.weight: Font.DemiBold
                elide: Text.ElideRight
            }

            Row {
                id: buttonRow

                anchors.right: parent.right
                anchors.verticalCenter: parent.verticalCenter
                spacing: 6

                PaletteButton {
                    visible: !control.managed
                    text: control.keyInKeyring ? "Change key" : "Set key"
                    tooltipText: "Store an API key in the keyring"
                    theme: control.theme
                    enabled: !control.busy
                    onClicked: control.keySetRequested()
                }

                PaletteButton {
                    visible: !control.managed && control.keyInKeyring
                    text: "Clear key"
                    tooltipText: "Remove the API key from the keyring"
                    danger: true
                    theme: control.theme
                    enabled: !control.busy
                    onClicked: control.keyClearRequested()
                }

                PaletteButton {
                    visible: control.managed
                    text: "Edit"
                    tooltipText: "Edit this provider"
                    theme: control.theme
                    enabled: !control.busy
                    onClicked: control.editRequested()
                }

                PaletteButton {
                    visible: control.managed
                    text: "Remove"
                    tooltipText: "Remove this provider and its models"
                    danger: true
                    theme: control.theme
                    enabled: !control.busy
                    onClicked: control.removeRequested()
                }
            }
        }

        Text {
            width: parent.width
            text: control.baseUrl
            color: control.c("text_muted", "#8a93a3")
            font.family: control.fontFamily
            font.pixelSize: control.fontSize("body_small", 11)
            elide: Text.ElideMiddle
        }

        Text {
            width: parent.width
            visible: control.detailText.length > 0
            text: control.detailText
            color: control.c("text_muted", "#8a93a3")
            font.family: control.fontFamily
            font.pixelSize: control.fontSize("body_small", 11)
            elide: Text.ElideRight
        }

        Flow {
            width: parent.width
            spacing: 4
            visible: control.models.length > 0

            Repeater {
                model: control.models

                delegate: Rectangle {
                    height: 20
                    width: modelChipText.implicitWidth + 16
                    radius: control.cornerRadius(5)
                    color: control.c("surface_elevated", "#1b1f28")
                    border.width: 1
                    border.color: control.c("panel_border", "#2b303b")

                    Text {
                        id: modelChipText

                        anchors.centerIn: parent
                        text: String(modelData ?? "")
                        color: control.c("text", "#d3d8e2")
                        font.family: control.fontFamily
                        font.pixelSize: control.fontSize("body_small", 11)
                    }
                }
            }
        }
    }
}
