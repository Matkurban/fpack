#!/usr/bin/env bash
# Regenerates the localization code (standard package:intl workflow):
#   lib/l10n/app_localizations.dart --extract_to_arb--> lib/l10n/app_en.arb (template)
#   lib/l10n/app_*.arb --generate_from_arb--> lib/l10n/gen/messages_*.dart
# intl_translation's code generator runs as a global tool, not a dependency:
# its analyzer version differs from the one jaspr_builder needs.
set -euo pipefail
cd "$(dirname "$0")/.."
version=0.22.0
dart pub global list | grep -q "^intl_translation $version" || dart pub global activate intl_translation "$version" >/dev/null
dart pub global run intl_translation:extract_to_arb --output-dir=lib/l10n --output-file=app_en.arb --locale=en lib/l10n/app_localizations.dart
sed -i.bak '/"@@last_modified"/d' lib/l10n/app_en.arb && rm lib/l10n/app_en.arb.bak  # keep the output reproducible
rm -rf lib/l10n/gen && mkdir -p lib/l10n/gen
dart pub global run intl_translation:generate_from_arb --output-dir=lib/l10n/gen --no-use-deferred-loading \
  lib/l10n/app_localizations.dart lib/l10n/app_*.arb
dart format lib/l10n >/dev/null
