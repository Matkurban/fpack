---
title: "macOS"
description: "Developer ID 签名、公证、DMG 与 pkg 安装包，以及如何获取证书和凭证。"
---

## 签名与公证

用于 App Store 之外的分发（Developer ID）。**只**通过 fpack 自己的配置开启（低 → 高）：

1. `fpack.yaml` 的 `macos.sign`（`fpack init` 会生成带占位符的注释段，并在注释中列出本机钥匙串里的 Developer ID 证书）
2. 环境变量 `FPACK_MACOS_SIGN`、`FPACK_MACOS_SIGN_IDENTITY`、`FPACK_MACOS_INSTALLER_IDENTITY`、`FPACK_MACOS_NOTARIZE`、`FPACK_MACOS_NOTARY_PROFILE`
3. 命令行 `--sign/--no-sign`、`--sign-identity`、`--installer-identity`、`--notarize/--no-notarize`、`--notary-profile`

```yaml
macos:
  sign:
    identity: "Developer ID Application: Your Name (TEAMID)"   # 设置后即启用签名
    notary_profile: XueHua                                     # 设置后即启用公证
```

公证凭证三选一（优先级从高到低）：钥匙串配置 `notary_profile`（本机推荐）→ App Store Connect API 密钥 `notary_api_key` + `notary_api_key_id`（+ `notary_api_issuer`，CI 推荐）→ Apple ID `notary_apple_id` + `notary_team_id` + `notary_password`（App 专用密码，日志中隐藏）。签名默认启用 Hardened Runtime（`hardened_runtime`，公证必需）。

规则：设置了证书即启用签名；设置了任一公证凭证即启用公证；关闭签名同时关闭公证；`--sign` 不带证书时自动选用钥匙串中第一个 “Developer ID Application” 证书；指定的证书不存在时会列出可用证书。**什么都不配置时**，.app 保留 Xcode 工程自己的签名、DMG 不签名，产物说明中会提示如何配置。fpack 不读取 `pubspec.yaml` 中其他插件（如 [`dmg`](https://pub.dev/packages/dmg) 包的 `dmg:` 段）的配置。

DMG 流程：复制 .app → `codesign --deep` 签名内嵌代码 → 用 Hardened Runtime + `macos/Runner/Release.entitlements` 重新签名 App → 校验 → `hdiutil`（或 `create-dmg`，支持背景图、窗口大小/位置、图标大小与位置、卷图标、许可协议；`format`/`filesystem` 两种工具都支持）制作 DMG → 签名 DMG → `xcrun notarytool submit` → 等待结果 → `stapler staple` → `spctl` 评估。

**公证等待**：Apple 公证通常需要几分钟。上传完成、拿到提交 ID 后，fpack 立即在输出目录写入 `NOTARIZATION.md`（中英文随 `--lang`）和 `notarization.json`：产物路径与 SHA-256、提交 ID、提交时间、使用的凭证（钥匙串配置名 / API 密钥 ID / Apple ID，绝不写密码），以及可直接复制的命令（`xcrun notarytool info/wait/log …`、`xcrun stapler staple/validate …`、`spctl -a -vv …`、`fpack notarize status/finish`），并在终端打印该文件路径。等待期间显示已等待时间，并提示：*公证通常需要几分钟。可以按 Ctrl-C 停止等待 —— Apple 端会继续处理；之后可用 NOTARIZATION.md 中的命令查询。* 结果出来后文件会更新为最终状态（Accepted / Invalid）；被拒绝时 fpack 自动下载 Apple 的日志（`notary-log-<文件>.json`），总结问题并给出修复建议。

- **不等待**：`macos.notarize.wait: false`（或 `FPACK_NOTARIZE_WAIT=false`、`--notarize-no-wait`）只提交、写入记录后就结束，汇总和 `--json` 中该产物的公证状态为 `submitted`（文件尚未装订）。
- **之后继续**：`fpack notarize status [目录|ID]` 向 Apple 查询所有未完成的提交；`fpack notarize finish [目录]` 等待结果，对通过的 DMG/pkg 装订票据（App zip 会解压、装订 .app 后重新压缩），重写 `SHA256SUMS`，并更新记录；被拒绝的会下载并总结日志。Ctrl-C 中断等待后同样可用。

证书与公证凭证的获取方法见下一节[获取 macOS 签名证书与公证凭证](/platforms/macos#credentials)。

`fpack build macos` 产出签名的 zip；开启公证时 zip 也会公证（流程：提交 zip → 装订到 .app → 重新压缩）。

<a id="credentials"></a>

## 获取 macOS 签名证书与公证凭证

在 App Store 之外分发 macOS 应用需要：**Developer ID Application** 证书（签名 .app / zip / DMG）、可选的 **Developer ID Installer** 证书（签名 pkg），以及一种**公证凭证**。都需要付费的 [Apple Developer Program](https://developer.apple.com/programs/) 账号；Developer ID 证书只能由团队的 **Account Holder** 创建。

**1. 创建 Developer ID 证书（在 Mac 上）**

- Xcode → Settings… → Accounts → 选择团队 → Manage Certificates… → 左下角 **+** → *Developer ID Application*（需要 pkg 时再建 *Developer ID Installer*）。证书和私钥会直接进入「登录」钥匙串。
- 或者在网页上创建：钥匙串访问 → 证书助理 → 从证书颁发机构请求证书…（保存 CSR 到磁盘）→ [developer.apple.com/account/resources/certificates](https://developer.apple.com/account/resources/certificates/list) → **+** → Developer ID Application / Developer ID Installer → 上传 CSR → 下载 `.cer` 并双击导入（必须在生成 CSR 的那台 Mac 上导入，私钥在那里）。
- 检查：

  ```bash
  security find-identity -v -p codesigning   # "Developer ID Application: Your Name (ABCDE12345)"
  security find-identity -v -p basic         # 也会列出 "Developer ID Installer: …"
  ```

- 填入 fpack（引号内的名称与上面输出完全一致；括号中的 10 位字符就是 **Team ID**）：

  | 用途 | fpack.yaml | 环境变量 | 参数 |
  | --- | --- | --- | --- |
  | App 签名证书 | `macos.sign.identity` | `FPACK_MACOS_SIGN_IDENTITY` | `--sign-identity` |
  | pkg 签名证书 | `macos.sign.installer_identity` | `FPACK_MACOS_INSTALLER_IDENTITY` | `--installer-identity` |

**2. 公证凭证（三选一）**

*a) 钥匙串配置（本机推荐）*：Apple ID + App 专用密码 + Team ID，只需保存一次。

1. 在 [account.apple.com](https://account.apple.com) → 登录与安全 → **App 专用密码** → 生成一个密码（形如 `abcd-efgh-ijkl-mnop`）。
2. Team ID：[developer.apple.com/account](https://developer.apple.com/account) → 会员资格详细信息（Membership details），或证书名称括号中的 10 位字符。
3. 保存到钥匙串（不写 `--password` 时会提示输入）：

   ```bash
   xcrun notarytool store-credentials fpack-notary \
     --apple-id you@example.com --team-id ABCDE12345 --password abcd-efgh-ijkl-mnop
   xcrun notarytool history --keychain-profile fpack-notary   # 验证
   ```

4. fpack：`macos.sign.notary_profile: fpack-notary`（或 `FPACK_MACOS_NOTARY_PROFILE`、`--notary-profile`）。

也可以不保存配置，直接给出 Apple ID：`notary_apple_id` / `notary_team_id` / `notary_password`（`FPACK_NOTARY_APPLE_ID`、`FPACK_NOTARY_TEAM_ID`、`FPACK_NOTARY_PASSWORD`）。密码只放在环境变量或 CI secret 中，fpack 在日志、`--dry-run` 和 NOTARIZATION.md 中都会隐藏它。

*b) App Store Connect API 密钥（CI 推荐）*：不依赖个人 Apple ID，可随时吊销。

1. [App Store Connect](https://appstoreconnect.apple.com) → 用户和访问 → 集成 → App Store Connect API → 团队密钥 → **+**，选择 Developer（或更高）权限。
2. 下载 `AuthKey_<KEY_ID>.p8`（**只能下载一次**，请妥善保存）；记下页面上的 **Key ID** 和 **Issuer ID**（个人密钥没有 Issuer ID，可省略）。
3. fpack：

   | 值 | fpack.yaml | 环境变量 |
   | --- | --- | --- |
   | `.p8` 文件路径 | `macos.sign.notary_api_key` | `FPACK_NOTARY_API_KEY` |
   | Key ID | `macos.sign.notary_api_key_id` | `FPACK_NOTARY_API_KEY_ID` |
   | Issuer ID | `macos.sign.notary_api_issuer` | `FPACK_NOTARY_API_ISSUER` |

   验证：`xcrun notarytool history --key AuthKey_ABC123DEF4.p8 --key-id ABC123DEF4 --issuer <Issuer ID>`。

优先级：`notary_profile` → API 密钥 → Apple ID；设置任意一种即开启公证。

**3. 导出证书（.p12）用于 CI**

1. 钥匙串访问 → 「登录」钥匙串 → **我的证书** → 选中 “Developer ID Application: …”（展开能看到私钥，说明私钥在本机）→ 文件 → 导出项目… → 格式选「个人信息交换 (.p12)」→ 设置导出密码。需要 pkg 时对 “Developer ID Installer: …” 重复一次（也可以两个一起选中导出到同一个 .p12）。
2. 转成 base64 存为 CI secret（例如 `MACOS_CERTS_P12_BASE64`、`MACOS_CERTS_P12_PASSWORD`），API 密钥同理（`NOTARY_API_KEY_P8_BASE64`）：

   ```bash
   base64 -i DeveloperID.p12 | pbcopy
   base64 -i AuthKey_ABC123DEF4.p8 | pbcopy
   ```

3. 在 CI 机器上导入到临时钥匙串（GitHub Actions 示例）：

   ```yaml
   - name: import Developer ID certificates
     env:
       P12_BASE64: ${{ secrets.MACOS_CERTS_P12_BASE64 }}
       P12_PASSWORD: ${{ secrets.MACOS_CERTS_P12_PASSWORD }}
       KEYCHAIN_PASSWORD: ${{ secrets.KEYCHAIN_PASSWORD }}   # 任意随机字符串
       API_KEY_BASE64: ${{ secrets.NOTARY_API_KEY_P8_BASE64 }}
     run: |
       KEYCHAIN="$RUNNER_TEMP/signing.keychain-db"
       echo "$P12_BASE64" | base64 --decode > "$RUNNER_TEMP/certs.p12"
       security create-keychain -p "$KEYCHAIN_PASSWORD" "$KEYCHAIN"
       security set-keychain-settings -lut 21600 "$KEYCHAIN"
       security unlock-keychain -p "$KEYCHAIN_PASSWORD" "$KEYCHAIN"
       security import "$RUNNER_TEMP/certs.p12" -k "$KEYCHAIN" -P "$P12_PASSWORD" \
         -T /usr/bin/codesign -T /usr/bin/productbuild -T /usr/bin/pkgbuild
       security set-key-partition-list -S apple-tool:,apple: -s -k "$KEYCHAIN_PASSWORD" "$KEYCHAIN"
       security list-keychains -d user -s "$KEYCHAIN" $(security list-keychains -d user | tr -d '"')
       security find-identity -v -p codesigning "$KEYCHAIN"
       echo "$API_KEY_BASE64" | base64 --decode > "$RUNNER_TEMP/AuthKey.p8"
       rm "$RUNNER_TEMP/certs.p12"
   - name: fpack build macos dmg pkg
     env:
       FPACK_MACOS_SIGN_IDENTITY: "Developer ID Application: Your Name (ABCDE12345)"
       FPACK_MACOS_INSTALLER_IDENTITY: "Developer ID Installer: Your Name (ABCDE12345)"
       FPACK_NOTARY_API_KEY: ${{ runner.temp }}/AuthKey.p8
       FPACK_NOTARY_API_KEY_ID: ${{ secrets.NOTARY_API_KEY_ID }}
       FPACK_NOTARY_API_ISSUER: ${{ secrets.NOTARY_API_ISSUER }}
     run: fpack build macos dmg pkg
   - name: clean up keychain
     if: always()
     run: security delete-keychain "$RUNNER_TEMP/signing.keychain-db" || true
   ```

   `set-key-partition-list` 让 codesign / productbuild 无需弹窗即可使用私钥；`-lut 21600` 让钥匙串 6 小时内不自动锁定。证书、.p12、.p8 与密码都不要提交到仓库。

## macOS 安装包（pkg）

`fpack build pkg` 用 `pkgbuild` 生成把 App 安装到 `/Applications` 的组件包（不可重定位，升级时总是覆盖 /Applications 中的版本），再用 `productbuild` 生成分发包（支持 Apple silicon 与 Intel，不会提示安装 Rosetta）。可在 `macos.pkg` 中设置 `identifier`、`install_location`、`title` 以及安装界面的 `welcome` / `readme` / `license` / `conclusion`（.html/.rtf/.txt）和 `background`（图片）。

- **签名**：pkg 需要单独的 **“Developer ID Installer”** 证书（与签名 App 的 “Developer ID Application” 不同），通过 `macos.sign.installer_identity` / `FPACK_MACOS_INSTALLER_IDENTITY` / `--installer-identity` 配置；签名后自动用 `pkgutil --check-signature` 校验。未配置时生成未签名 pkg 并给出说明；`--no-sign` 同样关闭 pkg 签名。
- **App 签名**：配置了 `macos.sign.identity` 时，pkg 中的 App 与 zip/DMG 一样先用 Developer ID 重新签名。
- **公证**：与 DMG 共用公证配置；只有已签名的 pkg 才会提交公证并 `stapler staple`，最后用 `spctl --assess --type install` 评估。
- **更多选项**：`min_os`（低于此版本拒绝安装，默认取工程的 `MACOSX_DEPLOYMENT_TARGET`）、`require_restart`、`relocatable`、`preinstall` / `postinstall` 脚本（自动设为可执行）、`version`。
- `doctor` 会检查 pkgbuild/productbuild，并列出钥匙串中的安装包证书。
