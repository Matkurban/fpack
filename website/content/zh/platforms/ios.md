---
title: "iOS"
description: "IPA 导出、签名与 flavor。"
lang: zh-CN
---

```bash
fpack build ipa                                   # Flutter 默认（App Store Connect）
fpack build ipa --export-method ad-hoc
fpack build ipa --export-options-plist ios/ExportOptions.plist
fpack build ipa --no-codesign                     # 未签名 IPA，用于后续重签名
```

`doctor` 会检查 Xcode、CocoaPods（存在 Podfile 时）、工程中的 `DEVELOPMENT_TEAM`、签名证书，并在失败时把 Xcode 的关键错误（证书、描述文件、团队、Pod 等）翻译成具体的修复建议。

## 导出选项

| 设置 | fpack.yaml | 参数 / 环境变量 |
| --- | --- | --- |
| 导出方式 | `ios.export_method` | `--export-method`、`FPACK_IOS_EXPORT_METHOD` |
| 自己的 ExportOptions.plist | `ios.export_options_plist` | `--export-options-plist`、`FPACK_IOS_EXPORT_OPTIONS_PLIST` |
| 未签名 IPA | `ios.codesign: false` | `--no-codesign`、`FPACK_IOS_CODESIGN` |
| 团队 / 签名方式 / 描述文件 | `ios.team_id`、`ios.signing_style`、`ios.provisioning_profiles` | – |

设置 `ios.team_id`（以及可选的 `signing_style`、`signing_certificate`、`provisioning_profiles`、`export_options`）后，fpack 会在 `build/fpack/` 中生成 ExportOptions.plist——不会修改你的项目。

## Flavor

iOS 的 flavor 就是 Xcode scheme。fpack 会检测共享、用户级以及 workspace 中的 scheme；项目没有自定义 scheme 时不会把 flavor 传给 `flutter build ipa`（并给出提示），因此只在 Android 上使用的 flavor 不会导致 iOS 构建失败。

## 上传

`dist/` 中的 IPA 可以用 Transporter 或 `xcrun altool --upload-app` 上传，例如在 `post_package.ipa` 钩子中执行。
