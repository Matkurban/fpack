---
title: "macOS：描述文件"
description: "让使用推送、iCloud 等受限 entitlements 的应用在 Developer ID 签名后能够启动，并在公证前发现问题。"
---

大多数应用用不到本页。只有当应用的 entitlements 里含有**受限**键时才需要，例如推送通知（`com.apple.developer.aps-environment`）、iCloud、关联域名或钥匙串访问组。

## 问题所在

`flutter build macos` 会嵌入 Xcode 的*开发*描述文件（“Mac Team Provisioning Profile”）。fpack 用 Developer ID 重新签名后，这个描述文件不再授权受限的 entitlements，macOS 会拒绝启动应用（`RBSRequestErrorDomain Code=5`、POSIX 163、“Launch failed”），尽管签名和公证都成功了。解决办法是为你的应用 ID 创建一个 **Developer ID 描述文件**。

## 1. 创建描述文件

你需要是账号持有者或管理员。

1. [developer.apple.com](https://developer.apple.com/account) → Certificates, IDs & Profiles → **Identifiers**：打开应用 ID（例如 `com.example.app`，即 `macos/Runner/Configs/AppInfo.xcconfig` 里的 `PRODUCT_BUNDLE_IDENTIFIER`），启用你的 entitlements 用到的能力（Push Notifications、iCloud、Associated Domains…），保存。
2. **Profiles** → **+** → 在 *Distribution* 下选择 **Developer ID** → 平台 macOS → 选应用 ID → 选你用来签名的 **Developer ID Application** 证书。必须是那张确切的证书：用 `security find-identity -v -p codesigning` 查看它的 SHA-1；为别的证书生成的描述文件会在启动时被拒绝。命名、**Generate**、**Download**。
3. 把 `.provisionprofile` 放进项目（例如 `macos/MyApp.provisionprofile`）。它不含机密，但你也可以不提交到 git，在 CI 里用 `FPACK_MACOS_PROVISIONING_PROFILE` 指向它。
4. 让 release entitlements 与描述文件一致。Developer ID 描述文件授权 `aps-environment` 为 **production**，所以 `macos/Runner/Release.entitlements` 里需要有 `<key>com.apple.developer.aps-environment</key><string>production</string>`。

## 2. 告诉 fpack

```yaml
macos:
  sign:
    identity: "Developer ID Application: Your Name (ABCDE12345)"
    provisioning_profile: macos/MyApp.provisionprofile      # 或 FPACK_MACOS_PROVISIONING_PROFILE / --provisioning-profile
    provisioning_profiles:                                  # 扩展和登录项：bundle id → 描述文件
      com.example.app.share: macos/MyAppShare.provisionprofile
  smoke_test: true                                          # 或 FPACK_MACOS_SMOKE_TEST / --smoke-test
```

签名之前，fpack 会把描述文件复制到 `Contents/embedded.provisionprofile`（并把映射的描述文件复制到 App 扩展和登录项里），然后由内向外签名：内嵌代码、带各自 entitlements 的内嵌 bundle，最后是应用本身。每个有自己 bundle id 的扩展或登录项都要重复创建描述文件。

## 3. fpack 检查什么

签名前会用 `security cms -D -i` 检查每个描述文件；有问题就带着提示停止构建，而不是产出一个无法启动的应用。`fpack doctor` 会报告同样的结果。

- 描述文件是 **Developer ID** 类型（`ProvisionsAllDevices`）；
- 它与签名证书属于同一个团队，并且是为*这张*证书生成的；
- 应用 ID 匹配（`TEAM.bundle.id`，允许通配符）；
- 没有过期；
- 签名使用的每个受限 entitlement 都被授权，且值匹配（例如 `aps-environment` 必须是 `production`）。

没有描述文件、使用了受限 entitlements 或嵌入了开发描述文件，会在签名前停止构建。想强行签名，设置 `macos.sign.profile_check: false`（`FPACK_MACOS_PROFILE_CHECK=false`）；描述文件仍会被嵌入。

| 提示 | 含义与处理 |
| --- | --- |
| `… uses restricted entitlements (…) that a Developer ID app may only use with a provisioning profile` | 没有配置描述文件：设置 `macos.sign.provisioning_profile` |
| `the app embeds Xcode's development profile "…"` | Xcode 嵌入了它的开发描述文件：配置 Developer ID 描述文件，fpack 会替换它 |
| `… is a development/ad hoc profile (N registered devices)` | 你下载的是 Development 描述文件：创建 **Developer ID** 的 |
| `… is not a Developer ID profile (ProvisionsAllDevices is missing)` | 是 App Store 描述文件：创建 **Developer ID** 的 |
| `… was generated for another Developer ID Application certificate (…), not the signing certificate …` | 描述文件里的证书不是 fpack 签名用的那张（例如证书被重新创建过）；macOS 会记录 `Unsatisfied entitlements` 并拒绝启动：编辑描述文件，选中签名证书，重新下载 |
| `… belongs to team X, but the signing certificate is from team Y` | 描述文件和证书属于不同团队 |
| `… is for app id TEAM.a, not TEAM.b` | 描述文件是为另一个 bundle id 创建的 |
| `… expired on YYYY-MM-DD` | 重新生成并下载描述文件 |
| `entitlement … is not authorized by …` | 为应用 ID 启用对应能力，重新生成描述文件 |
| `aps-environment is "development" in the entitlements but the profile allows "production"` | 在 release entitlements 里设置 `production` |

## 4. 冒烟测试：签名后的应用能启动吗？

用 `macos.smoke_test: true`、`FPACK_MACOS_SMOKE_TEST=true` 或 `--smoke-test` 开启：

```bash
fpack build macos dmg --smoke-test
```

fpack 对签名后的应用运行 `open -n`，等待 `macos.smoke_test_seconds`（默认 5，范围 1 到 300），用 `pgrep` 检查应用是否仍在运行，再用 `pkill` 退出它。如果没有在运行，构建会失败并提示 *the app was not running … after launch (it crashed or macOS refused to start it)*，同时打印一条能看到原因的 `log show` 命令（通常是 `amfid` 或 RunningBoard 的 `Unsatisfied entitlements`、`No matching profile found` 之类消息）。它在**公证之前**运行，所以有问题的应用不会被上传。

## 自己检查描述文件和应用

```bash
security cms -D -i macos/MyApp.provisionprofile                          # 描述文件的内容
codesign -d --entitlements - --xml MyApp.app                              # 已构建应用的 entitlements
security cms -D -i MyApp.app/Contents/embedded.provisionprofile           # 嵌入的描述文件
```

## 下一步

回到 [macOS 教程](/platforms/macos)，或看[故障排查](/troubleshooting#macos-zh)。
