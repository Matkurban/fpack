---
title: "macOS：证书与凭证"
description: "创建 Developer ID 证书和公证凭证，并导出给 CI 使用。"
---

在 App Store 之外分发 macOS 应用需要：**Developer ID Application** 证书（给 `.app`、zip 和 DMG 签名）、可选的 **Developer ID Installer** 证书（给 pkg 签名），以及一种**公证凭证**。这些都需要付费的 [Apple Developer Program](https://developer.apple.com/programs/) 账号，而且 Developer ID 证书只能由团队的**账号持有者（Account Holder）**创建。

怎么使用它们见 [macOS 教程](/platforms/macos)。

## 1. 创建 Developer ID 证书

在 Mac 上操作，二选一。

**在 Xcode 里**（最简单）：Xcode → Settings… → Accounts → 选择团队 → Manage Certificates… → **+** → *Developer ID Application*（要构建 pkg 就再加 *Developer ID Installer*）。证书和私钥会直接进入登录钥匙串。

**在网页上**：钥匙串访问 → 证书助理 → 从证书颁发机构请求证书…（把请求文件存到磁盘）→ [developer.apple.com/account/resources/certificates](https://developer.apple.com/account/resources/certificates/list) → **+** → Developer ID Application / Developer ID Installer → 上传请求文件 → 下载 `.cer` 并双击，必须在创建请求的那台 Mac 上操作，因为私钥在那里。

检查是否已安装：

```bash
security find-identity -v -p codesigning   # "Developer ID Application: Your Name (ABCDE12345)"
security find-identity -v -p basic         # 同时列出 "Developer ID Installer: …"
```

把名称按打印的原样告诉 fpack；括号里的十位字符就是你的 **Team ID**：

| 用途 | fpack.yaml | 变量 | 参数 |
| --- | --- | --- | --- |
| App 签名证书 | `macos.sign.identity` | `FPACK_MACOS_SIGN_IDENTITY` | `--sign-identity` |
| pkg 签名证书 | `macos.sign.installer_identity` | `FPACK_MACOS_INSTALLER_IDENTITY` | `--installer-identity` |

<a id="notary-zh"></a>

## 2. 创建公证凭证

**三选一**。fpack 的优先顺序是：钥匙串配置、API 密钥、Apple ID。任何一种都会开启公证。

### a) 钥匙串配置（自己的 Mac 上最推荐）

Apple ID + App 专用密码 + Team ID，一次性存进钥匙串。

1. 打开 [account.apple.com](https://account.apple.com) → 登录与安全 → **App 专用密码**，生成一个（形如 `abcd-efgh-ijkl-mnop`）。
2. 查 Team ID：[developer.apple.com/account](https://developer.apple.com/account) → 会员资格详情，或证书名称里的十位字符。
3. 存储（省略 `--password` 会改为提示输入）：

   ```bash
   xcrun notarytool store-credentials fpack-notary \
     --apple-id you@example.com --team-id ABCDE12345 --password abcd-efgh-ijkl-mnop
   xcrun notarytool history --keychain-profile fpack-notary   # 检查
   ```

4. 使用：`macos.sign.notary_profile: fpack-notary`（`FPACK_MACOS_NOTARY_PROFILE`、`--notary-profile`）。

没有存储配置时，也可以直接传 Apple ID：`notary_apple_id`、`notary_team_id` 和 `notary_password`（`FPACK_NOTARY_APPLE_ID`、`FPACK_NOTARY_TEAM_ID`、`FPACK_NOTARY_PASSWORD`）。密码请放在环境变量或 CI 密钥中；fpack 会在日志、`--dry-run` 输出和 `NOTARIZATION.md` 里隐藏它。

### b) App Store Connect API 密钥（CI 最推荐）

不绑定某个人的 Apple ID，可随时吊销。

1. [App Store Connect](https://appstoreconnect.apple.com) → 用户和访问 → 集成 → App Store Connect API → 团队密钥 → **+**，访问权限选 *Developer* 或更高。
2. 下载 `AuthKey_<KEY_ID>.p8`。**它只能下载一次**，请妥善保管。记下页面上显示的 **Key ID** 和 **Issuer ID**（个人密钥没有 issuer，不要填）。
3. 告诉 fpack：

   | 值 | fpack.yaml | 变量 |
   | --- | --- | --- |
   | `.p8` 文件路径 | `macos.sign.notary_api_key` | `FPACK_NOTARY_API_KEY` |
   | Key ID | `macos.sign.notary_api_key_id` | `FPACK_NOTARY_API_KEY_ID` |
   | Issuer ID | `macos.sign.notary_api_issuer` | `FPACK_NOTARY_API_ISSUER` |

   检查：`xcrun notarytool history --key AuthKey_ABC123DEF4.p8 --key-id ABC123DEF4 --issuer <Issuer ID>`。

<a id="ci-zh"></a>

## 3. 为 CI 导出证书（.p12）

CI 机器的钥匙串里没有你的证书，需要从密钥（secret）里导入。

1. 钥匙串访问 → 登录钥匙串 → **我的证书** → 选中 “Developer ID Application: …”（展开它：下面必须有私钥）→ 文件 → 导出项目… → 格式选 *个人信息交换（.p12）* → 设置导出密码。要构建 pkg 的话，对 “Developer ID Installer: …” 重复一次（或两个一起选中导出成一个文件）。
2. 把它 base64 后放进 CI 密钥（例如 `MACOS_CERTS_P12_BASE64` 和 `MACOS_CERTS_P12_PASSWORD`）；API 密钥同理（`NOTARY_API_KEY_P8_BASE64`）：

   ```bash
   base64 -i DeveloperID.p12 | pbcopy
   base64 -i AuthKey_ABC123DEF4.p8 | pbcopy
   ```

3. 在运行机器上导入到临时钥匙串并构建（GitHub Actions）：

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

   `set-key-partition-list` 让 `codesign` 和 `productbuild` 无需弹窗就能使用私钥；`-lut 21600` 让钥匙串保持解锁六小时。

::: danger
绝不要提交证书、`.p12` / `.p8` 文件或密码，把它们放在 CI 密钥里。fpack 会在所有输出里隐藏密码，但无法保护你提交进仓库的文件。
:::

## 下一步

回到 [macOS 教程](/platforms/macos)；如果你的应用使用推送、iCloud 或其他受限 entitlements，继续看[描述文件](/platforms/macos-profiles)。
