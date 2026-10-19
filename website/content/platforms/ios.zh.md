---
title: "iOS"
description: "构建用于 App Store、TestFlight 或 Ad Hoc 分发的 IPA：团队、导出方式、描述文件与上传。"
---

本教程产出用于 App Store Connect（TestFlight 和 App Store）的 `.ipa`、面向已登记设备的 Ad Hoc IPA，或之后再签名的未签名 IPA。目标：`ipa`。只能在 macOS 上构建。

## 你需要什么

- 装有 **Xcode** 的 Mac（工程有 Podfile 时还需要 **CocoaPods**）。
- 付费的 Apple Developer Program 会员资格、你的 **Team ID**（十位字符，在 Apple Developer 网站的“会员资格”里）和应用 ID（`PRODUCT_BUNDLE_IDENTIFIER`）。
- Xcode 已登录你的账号（Xcode → Settings → Accounts），这样自动签名才能创建证书和描述文件。

## 1. 检查机器

```bash
fpack doctor ipa
```

doctor 会检查 Xcode、CocoaPods、团队（`ios.team_id`，或工程里的 `DEVELOPMENT_TEAM`）以及钥匙串中的签名证书。每个问题都附带修复命令。

## 2. 告诉 fpack 你的团队

如果 Xcode 工程里已经设置了开发团队（Runner → Signing & Capabilities），就不需要做任何事。否则请设置它。这同时会让 fpack 为你生成 `ExportOptions.plist`，你的工程不会被修改。

```yaml
ios:
  team_id: ABCDE12345
```

## 3. 构建 App Store 的 IPA

```bash
fpack build ipa
```

这会用 Flutter 默认的导出方式（App Store Connect）运行 `flutter build ipa`，并把结果复制到 `dist/1.0.0+1/my_app-1.0.0+1-ios-arm64.ipa`。构建失败时，关键的 Xcode 错误（证书、描述文件、团队、pods…）会被翻译成具体的修复办法。

## 4. 上传

用 **Transporter**（Mac App Store）、`xcrun altool --upload-app` 上传，或者让 Xcode 来做：把导出目标设为 `upload`，fpack 生成的导出选项就会直接把构建发送到 App Store Connect。

```yaml
ios:
  team_id: ABCDE12345
  destination: upload       # export（默认）只写出 IPA；upload 会发送到 App Store Connect
```

也可以在[钩子](/guides/hooks)里上传：`post_package.ipa: [xcrun altool --upload-app …]`。

::: tip
App Store Connect 要求每次上传都使用新的构建号。提高 `pubspec.yaml` 中 `+` 后面的数字，或传 `--build-number 43`。`ios.manage_app_version_and_build_number: true`（默认）时，App Store Connect 也可能代为管理构建号。
:::

## 其他分发方式

### 导出方式

| 方式 | 用途 |
| --- | --- |
| `app-store-connect`（默认）、`app-store` | TestFlight 和 App Store |
| `release-testing`、`ad-hoc` | 指定的已登记设备 |
| `development`、`debugging` | 开发设备 |
| `enterprise` | 企业内部分发（企业计划） |

```bash
fpack build ipa --export-method ad-hoc
```

同样的设置在 `fpack.yaml` 里是 `ios.export_method`，变量是 `FPACK_IOS_EXPORT_METHOD`。

### 用描述文件手动签名

默认由 Xcode 管理签名。手动签名（CI 上常见）时，要说明每个 bundle id（包括 App 扩展）由哪个证书和哪个描述文件签名：

```yaml
ios:
  export_method: ad-hoc
  team_id: ABCDE12345
  signing_style: manual
  signing_certificate: Apple Distribution
  provisioning_profiles:
    com.example.app: My App AdHoc
    com.example.app.share: My App Share AdHoc
```

fpack 会把这些写进生成的 `ExportOptions.plist`（在 `build/fpack/`）。其他任何 ExportOptions 键都可以放在 `ios.export_options` 下（例如 `iCloudContainerEnvironment: Production`），`ios.thinning`、`ios.upload_symbols` 和 `ios.strip_swift_symbols` 对应各自的 plist 键。

### 使用你自己的 ExportOptions.plist

```bash
fpack build ipa --export-options-plist ios/ExportOptions.plist
```

你的文件优先于所有生成的选项。设置：`ios.export_options_plist`、`FPACK_IOS_EXPORT_OPTIONS_PLIST`。

### 未签名的 IPA

```bash
fpack build ipa --no-codesign
```

使用 `--no-codesign` 构建并把 `Payload/` 目录打成 zip，例如之后在别的系统里重新签名。设置：`ios.codesign: false`、`FPACK_IOS_CODESIGN`。

## Flavor

iOS 的 flavor 是 Xcode scheme。fpack 会检测共享、用户级和 workspace 的 scheme。工程没有自定义 scheme 时，不会把 flavor 传给 `flutter build ipa`（并给出提示），所以只存在于 Android 的 flavor 不会弄坏 iOS 构建。见 [Flavor](/guides/build-variants#flavor)。

## 在 CI 中

在 `macos-latest` 上运行，并把证书导入临时钥匙串；macOS 凭证页展示了钥匙串的步骤（[为 CI 导出证书](/platforms/macos-credentials#ci-zh)）。然后：

```yaml
- run: fpack build ipa --export-method app-store-connect --json > result.json
```

## 出问题时

| 提示 | 处理 |
| --- | --- |
| 没有签名证书 / `No Accounts` | 在 Xcode 里登录账号，或导入证书和描述文件 |
| `No profiles for 'com.example.app' were found` | 手动签名时，把 bundle id（和扩展）写进 `ios.provisioning_profiles` |
| 提示里的团队名不对 | 设置 `ios.team_id`（它优先于工程里的 `DEVELOPMENT_TEAM`） |
| `pod install` 失败 | 运行 `pod repo update`，或 `cd ios && pod install` 查看错误 |
| 构建号已被使用 | 提高 `pubspec.yaml` 中的 `+N` 或传 `--build-number` |

更多见[故障排查页](/troubleshooting#ios-zh)。
