# Changelog

All notable user-visible changes to Hark are documented here. The project
follows [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.1.9] - 2026-09-20

### Added

- Configurable OpenAI-compatible providers (from `xmmanuellx/hark`): a
  `providers` table in `config.lua`, a provider manager in Settings with
  add, edit, and remove and any number of model ids per provider, and
  `harkctl provider` commands.
- Settings lists providers defined in `config.lua` alongside panel-managed
  ones, shows each provider's API key status, and can set or clear a key for
  `config.lua` providers. `harkctl provider list` gains `managed`,
  `key_configured`, and `key_source`.
- A Settings toggle to hide the Hark icon in the Omarchy bar.
- `scripts/install-omarchy.sh`, which builds the runtime and links this
  checkout as the Omarchy plugin, so a fork can be installed from this single
  repository instead of the generated `hark-plugin` repository. The release
  workflow no longer tries to publish to `konradk/hark-plugin` from forks.

### Changed

- Custom providers on a loopback `base_url` no longer require an API key.
- `harkd.service` is now wanted by `graphical-session.target` instead of
  `default.target`. Rerunning `scripts/install.sh` moves an existing enabled
  service to the new target.
- `harkctl` prints each error on a single line, so the overlay shows the whole
  message instead of only the last line of multi-line tool output.

### Fixed

- Copy, paste, screenshots, and window focus restore failed when the daemon
  started before the compositor exported its session, which left
  `WAYLAND_DISPLAY` and `HYPRLAND_INSTANCE_SIGNATURE` unset. Subprocesses now
  recover both from `XDG_RUNTIME_DIR`.
- Copying an answer could hang until something else replaced the clipboard,
  because `wl-copy` keeps a background process holding standard error open.
- Fall back to the plugin's own directory when the host omits `__sourceDir`
  from the manifest, which newer Omarchy shells do for third-party plugins.
- Removed unused code that made the CI static analysis step fail.

## [0.1.7] - 2026-08-13

- Added native xAI support for Grok 4.6 and Grok 4.5, including streaming,
  image input, web search, citations, and API key management. Grok 4.6 is also
  available through OpenRouter.
- Moved supported reasoning efforts into each model definition, so the UI and
  backend expose only the values accepted by that model and provider.
- Improved settings navigation: opening settings leaves the conversation view,
  hides the prompt composer, and provides a discreet close action.

## [0.1.6] - 2026-08-12

- Improved model picker readability with content-aware popup sizing, cleaner
  selected labels, and tooltips for truncated names.

## [0.1.5] - 2026-08-12

- Smoothed OpenRouter response streaming and refined web-search status, stop,
  and screenshot capture behavior.

## [0.1.4] - 2026-08-11

- Added the preview image required for the Omarchy marketplace listing and
  included it in packaged releases.

## [0.1.3] - 2026-08-11

- Added author and license metadata required by the Omarchy marketplace.

## [0.1.2] - 2026-08-09

- Limited the model picker to providers with a configured API key and safely
  handled previously selected unavailable models.

## [0.1.1] - 2026-08-09

- Kept the API key setup hint visible when chat history exists or the user is
  typing, preventing prompts from failing without explanation.

## [0.1.0] - 2026-08-09

- Initial public release with OpenAI and OpenRouter chat, conversation history,
  screenshot attachments, clipboard integration, and both Omarchy plugin and
  standalone Hyprland installation modes.

[Unreleased]: https://github.com/konradk/hark/compare/v0.1.7...HEAD
[0.1.7]: https://github.com/konradk/hark/compare/v0.1.6...v0.1.7
[0.1.6]: https://github.com/konradk/hark/compare/v0.1.5...v0.1.6
[0.1.5]: https://github.com/konradk/hark/compare/v0.1.4...v0.1.5
[0.1.4]: https://github.com/konradk/hark/compare/v0.1.3...v0.1.4
[0.1.3]: https://github.com/konradk/hark/compare/v0.1.2...v0.1.3
[0.1.2]: https://github.com/konradk/hark/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/konradk/hark/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/konradk/hark/releases/tag/v0.1.0
