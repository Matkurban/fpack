---
title: "macOS"
description: "构建压缩的 app、DMG 或 pkg 安装包，用 Developer ID 签名并交给 Apple 公证。"
---

本教程为**在 Mac App Store 之外分发**的 macOS 应用构建：压缩的 `.app`（`macos`）、磁盘映像（`dmg`）和安装包（`pkg`），用 Developer ID 证书签名并由 Apple 公证，这样 Gatekeeper 打开时不会有警告。三者都只能在 macOS 上构建，并共用一次 `flutter build macos`。

## 你需要什么

- 装有 **Xcode** 的 Mac（工程有 Podfile 时还需要 CocoaPods）。
- 签名与公证：付费的 Apple Developer Program 会员资格、钥匙串里的 **Developer ID Application** 证书（`pkg` 还需要 **Developer ID Installer**）以及一种公证凭证。每一项怎么获取见[证书与凭证](/platforms/macos-credentials)。

前面几步不需要这些就能做。

## 1. 检查机器

```bash
fpack doctor macos dmg pkg
```

doctor 检查 Xcode、CocoaPods、`hdiutil` / `create-dmg`、`pkgbuild` 和 `productbuild`，列出钥匙串里的签名证书，并校验描述文件和 entitlements。

## 2. 先构建一次（未签名）

```bash
fpack build macos dmg
```

什么都没配置时，`.app` 保持 Xcode 给它的签名，DMG 不签名，汇总里的提示会说明如何开启签名。你会得到：

```text
dist/1.0.0+1/
  my_app-1.0.0+1-macos-universal.zip
  my_app-1.0.0+1-macos-universal.dmg
```

在你自己的 Mac 上测试没问题。在别的 Mac 上，Gatekeeper 会拦截未签名、未公证的下载文件：继续第 3 步。

::: tip
配置好签名之后，想快速打本地包，可以加 `--no-notarize`（跳过上传 Apple）或 `--no-sign`（跳过签名和公证）。
:::

## 3. 配置签名与公证

只有通过 fpack 自己的设置才会启用签名：`fpack.yaml` 的 `macos.sign`、`FPACK_MACOS_*` 变量或命令行参数。优先级从低到高：文件 → 变量 → 参数。

```yaml
macos:
  sign:
    identity: "Developer ID Application: Your Name (ABCDE12345)"   # 设置后即启用签名
    notary_profile: fpack-notary                                    # 设置后即启用公证
```

规则：

- 设置了 `identity` 即启用签名；设置了任一公证凭证即启用公证；关闭签名同时关闭公证；
- `--sign` 不带证书时，自动选用钥匙串中第一个 “Developer ID Application” 证书；找不到证书时会列出可用的；
- `--notarize`（或 `notarize: true`）没有任何凭证时，使用名为 `NotaryProfile` 的钥匙串配置；
- 签名使用 **Hardened Runtime**（`macos.sign.hardened_runtime`，默认开启，公证必需）和 `macos/Runner/Release.entitlements`（或 `macos.sign.entitlements`）里的 entitlements；
- fpack 不读取其他工具在 `pubspec.yaml` 里的设置，例如 `dmg` 包的 `dmg:` 段。

公证凭证有三种，按优先级：钥匙串配置（`notary_profile`，本机最推荐）、App Store Connect API 密钥（`notary_api_key`、`notary_api_key_id`、`notary_api_issuer`，CI 最推荐）、Apple ID 加 App 专用密码（`notary_apple_id`、`notary_team_id`、`notary_password`）。每一种怎么创建见[证书与凭证](/platforms/macos-credentials#notary-zh)。

| 设置 | fpack.yaml | 变量 | 参数 |
| --- | --- | --- | --- |
| 开启 / 关闭签名 | `macos.sign.enabled` | `FPACK_MACOS_SIGN` | `--sign`、`--no-sign` |
| App 证书 | `macos.sign.identity` | `FPACK_MACOS_SIGN_IDENTITY` | `--sign-identity` |
| pkg 证书 | `macos.sign.installer_identity` | `FPACK_MACOS_INSTALLER_IDENTITY` | `--installer-identity` |
| 开启 / 关闭公证 | `macos.sign.notarize` | `FPACK_MACOS_NOTARIZE` | `--notarize`、`--no-notarize` |
| 钥匙串配置 | `macos.sign.notary_profile` | `FPACK_MACOS_NOTARY_PROFILE` | `--notary-profile` |
| 描述文件 | `macos.sign.provisioning_profile` | `FPACK_MACOS_PROVISIONING_PROFILE` | `--provisioning-profile` |
| 冒烟测试 | `macos.smoke_test` | `FPACK_MACOS_SMOKE_TEST` | `--smoke-test` |

## 4. 构建、签名、公证

```bash
fpack build macos dmg
```

对 DMG，fpack 会复制 `.app`、用 `codesign --deep` 给内嵌代码签名、用 Hardened Runtime 和 entitlements 给 App 重新签名并验证，然后制作磁盘映像、给 DMG 签名、用 `xcrun notarytool submit` 上传、等待 Apple 的结果、用 `stapler staple` 装订票据，最后用 `spctl` 检查。对 zip（`macos`），提交的是 zip，装订 `.app` 后再重新压缩。

## 5. 等待公证

Apple 通常需要几分钟。fpack 会显示已等待的时间，并在上传拿到提交 ID 之后立刻向输出目录写入两个文件：`NOTARIZATION.md`（产物、SHA-256、提交 ID、使用的凭证（但绝不含密码）以及可直接复制的命令）和 `notarization.json`。

::: note
Ctrl-C 只是停止*等待*，Apple 仍在处理。你可以稍后在这台 Mac 的任何 shell 里继续。
:::

也可以完全不等待：

```bash
fpack build dmg --notarize-no-wait      # 提交、写入记录、结束
fpack notarize status                   # 向 Apple 查询所有未完成的提交
fpack notarize finish                   # 等待、给通过的文件装订、更新 SHA256SUMS
```

- `fpack notarize status [目录|ID]` 报告 `notarization.json` 中每个提交的状态（或某一个提交 ID）。
- `fpack notarize finish [目录]` 等待结果，给通过的 DMG 和 pkg 装订票据（app zip 会先解压，装订 app 后重新压缩），重写 `SHA256SUMS`；对被拒绝的提交下载 Apple 的日志（`notary-log-<文件>.json`）并总结问题。
- 用 `macos.notarize.wait: false` 或 `FPACK_NOTARIZE_WAIT=false` 也能得到与 `--notarize-no-wait` 相同的行为。此时汇总和 `--json` 报告 `notarization.state: submitted`。
- 用 Apple ID 提交时，运行 `notarize` 需要设置 `FPACK_NOTARY_PASSWORD`。

提交被拒绝（`Invalid`）通常是内嵌代码没有签名或缺少 Hardened Runtime；总结后的日志会指出是哪个文件。

## 6. 验证结果

```bash
spctl --assess --type open --context context:primary-signature -vv dist/1.0.0+1/my_app-1.0.0+1-macos-universal.dmg
xcrun stapler validate dist/1.0.0+1/my_app-1.0.0+1-macos-universal.dmg
codesign --verify --deep --strict --verbose=2 /Applications/MyApp.app
```

## 7. 定制 DMG

fpack 默认用 `hdiutil` 制作 DMG；`create-dmg` 已安装或设置了布局键时改用它（`macos.dmg.tool: auto`）。用 `--dmg-tool hdiutil|create-dmg` 强制指定其一。

```yaml
macos:
  dmg:
    volume_name: My App
    background: macos/dmg/background.png
    window_size: [660, 400]
    window_position: [200, 120]
    icon_size: 128
    app_position: [180, 190]
    applications_position: [480, 190]
    format: ULFO
```

还可以用 `volume_icon` 和 `license`（挂载时显示的许可协议）；`format` 和 `filesystem` 对两种工具都有效。用 `brew install create-dmg` 安装 `create-dmg`。

## 8. pkg 安装包

```bash
fpack build pkg
```

fpack 用 `pkgbuild` 创建把应用安装到 `/Applications` 的组件包（不可重定位，所以升级总是替换 /Applications 里的那份），再用 `productbuild` 创建分发包，同时支持 Apple 芯片和 Intel，不会弹出 Rosetta 提示。

```yaml
macos:
  pkg:
    identifier: com.example.app.pkg
    title: My App
    welcome: macos/installer/welcome.html
    license: macos/installer/license.rtf
    min_os: "11.0"
    postinstall: macos/installer/postinstall.sh
```

- **签名**：pkg 需要单独的 **Developer ID Installer** 证书（`macos.sign.installer_identity`、`--installer-identity`），结果会用 `pkgutil --check-signature` 检查。没有它会生成未签名的 pkg 并给出提示；`--no-sign` 会关闭 pkg 签名。
- **公证**：与 DMG 共用设置；只有已签名的 pkg 才会提交和装订，然后用 `spctl --assess --type install` 评估。
- 其他选项：`readme`、`conclusion`、`background`、`install_location`、`version`、`preinstall`、`require_restart`、`relocatable`（脚本会自动设为可执行）。`min_os` 默认是工程的 `MACOSX_DEPLOYMENT_TARGET`。

## 9. 使用推送、iCloud 等受限 entitlements 的应用

如果应用使用了受限 entitlements（推送通知、iCloud、关联域名、钥匙串访问组），就必须有 Developer ID 描述文件，否则签名后的应用无法启动。阅读[描述文件](/platforms/macos-profiles)；加上 `--smoke-test` 可以在公证之前发现问题。

## Flavor 与 CI

Flavor 是 Xcode scheme（[Flavor](/guides/build-variants#flavor)）。在 CI 里做签名构建需要把证书导入临时钥匙串：[为 CI 导出证书](/platforms/macos-credentials#ci-zh)有完整的工作流。

## 出问题时

| 提示 | 处理 |
| --- | --- |
| `signing identity … not found in the keychain` | 使用列出的证书，名称与打印的完全一致，或用 `--no-sign` |
| 公证 `Invalid` | 阅读总结后的 Apple 日志；通常是内嵌代码未签名或缺少 Hardened Runtime |
| 公证上传超时 | fpack 会重试上传两次；检查网络后重新构建（没有提交任何内容） |
| 签名后的应用无法启动（`RBSRequestErrorDomain Code=5`） | 使用了受限 entitlements 却没有 Developer ID 描述文件：[描述文件](/platforms/macos-profiles) |
| pkg 未签名 | 设置 `macos.sign.installer_identity`（与 App 证书不同的证书） |

更多见[故障排查页](/troubleshooting#macos-zh)。
