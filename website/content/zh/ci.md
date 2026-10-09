---
title: "CI 示例"
description: "各平台的 GitHub Actions 工作流。"
lang: zh-CN
---

```yaml
jobs:
  android:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: subosito/flutter-action@v2
      - run: dart pub global activate fpack && echo "$HOME/.pub-cache/bin" >> $GITHUB_PATH
      - run: fpack build apk aab --split-per-abi=both
        env:
          FPACK_ANDROID_KEYSTORE_BASE64: ${{ secrets.KEYSTORE_BASE64 }}
          FPACK_ANDROID_KEYSTORE_PASSWORD: ${{ secrets.KEYSTORE_PASSWORD }}
          FPACK_ANDROID_KEY_ALIAS: upload
      - uses: actions/upload-artifact@v4
        with: { name: android, path: dist/ }
  apple:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v4
      - uses: subosito/flutter-action@v2
      - run: dart pub global activate fpack && echo "$HOME/.pub-cache/bin" >> $GITHUB_PATH
      - run: fpack build ipa dmg --json > result.json
  windows:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v4
      - uses: subosito/flutter-action@v2
      - run: dart pub global activate fpack
      - run: fpack build windows exe
  linux:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: subosito/flutter-action@v2
      - run: sudo apt-get install -y ninja-build libgtk-3-dev rpm
      - run: dart pub global activate fpack && echo "$HOME/.pub-cache/bin" >> $GITHUB_PATH
      - run: fpack build linux deb rpm web
```


非 TTY 环境下自动关闭动画与颜色，每 60 秒输出一次心跳，避免 CI 因长时间无输出而超时。

## 提示

- **固定版本**：`dart pub global activate fpack 1.1.4` 让 CI 可复现。
- **缓存核心**：原生核心每个版本只下载一次，位于 `~/.cache/fpack`（Linux）、`~/Library/Caches/fpack`（macOS）或 `%LOCALAPPDATA%\fpack`（Windows）；缓存该目录可省去约 4 MB 的下载。
- **机器可读结果**：`fpack build … --json > result.json` 输出每个产物的路径、大小、SHA-256 和公证状态；人类可读的输出写到 stderr。
- **尽早失败**：退出码 3 表示缺少前置条件——什么都没有构建。
- **上传全部产物**：`dist/<版本>+<构建号>/` 中包含产物和 `SHA256SUMS`。

## 在 CI 中构建已签名的 macOS 包

完整的钥匙串导入步骤（使用 App Store Connect API 密钥）见 [macOS → 为 CI 导出证书](/zh/platforms/macos#credentials)。

## Android 签名密钥

```bash
base64 -i upload.jks | pbcopy   # → 密钥 KEYSTORE_BASE64
```

设置 `FPACK_ANDROID_KEYSTORE_BASE64`、`FPACK_ANDROID_KEYSTORE_PASSWORD`、`FPACK_ANDROID_KEY_ALIAS`（密钥密码不同时还需 `FPACK_ANDROID_KEY_PASSWORD`）。fpack 把密钥库写到 `build/fpack/secrets/`（权限 0600），构建后删除；日志中的密码会被隐藏。
