---
title: "CI 示例"
description: "各平台的 GitHub Actions 工作流、缓存、密钥与机器可读结果。"
---

fpack 为 CI 而生：不提问、退出码稳定、有 JSON 输出、日志已脱敏。下面的示例用 GitHub Actions，命令在任何 CI 中都一样。

## 原则

- **固定版本**，保证 CI 可复现：`dart pub global activate fpack 1.1.17`。
- **每个平台在自己的系统上构建。** iOS 和 macOS 用 `macos-*`，Windows 用 `windows-*`，Linux 用 `ubuntu-*`；Android 和 Web 在哪里都行（[为什么](/concepts#目标与产物)）。
- **密钥通过环境变量传入。** 永远不要写进 `fpack.yaml`（[密钥与环境变量](/guides/secrets)）。
- **上传 `dist/`。** 里面有产物和 `SHA256SUMS`。
- **用 `--json` 和退出码**决定下一步（[退出码与 JSON](/exit-codes)）。

没有 TTY 时，进度动画和颜色会关闭，每 60 秒输出一行心跳，避免 CI 因长时间无输出而超时。

## Android

```yaml
jobs:
  android:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
      - uses: subosito/flutter-action@v2
      - run: dart pub global activate fpack 1.1.17 && echo "$HOME/.pub-cache/bin" >> $GITHUB_PATH
      - run: fpack build apk aab --split-per-abi=both
        env:
          FPACK_ANDROID_KEYSTORE_BASE64: ${{ secrets.KEYSTORE_BASE64 }}
          FPACK_ANDROID_KEYSTORE_PASSWORD: ${{ secrets.KEYSTORE_PASSWORD }}
          FPACK_ANDROID_KEY_ALIAS: upload
      - uses: actions/upload-artifact@v7
        with: { name: android, path: dist/ }
```

用 `base64 -i upload.jks | pbcopy`（Linux：`base64 -w0 upload.jks`）生成密钥内容。如果密钥密码与密钥库密码不同，再设置 `FPACK_ANDROID_KEY_PASSWORD`。fpack 会把密钥库以 0600 权限写到 `build/fpack/secrets/`，构建后删除。详情：[Android](/platforms/android#在-ci-中)。

## iOS 与 macOS

```yaml
  apple:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v7
      - uses: subosito/flutter-action@v2
      - run: dart pub global activate fpack 1.1.17 && echo "$HOME/.pub-cache/bin" >> $GITHUB_PATH
      - run: fpack build ipa dmg --json > result.json
```

已签名并公证的 macOS 构建需要在临时钥匙串里准备证书，以及公证凭证：[macOS：为 CI 导出证书](/platforms/macos-credentials#ci-zh)里有完整的导入步骤（用 App Store Connect API 密钥）。如果不想在 Apple 处理期间占着 runner，可以用 `--notarize-no-wait`，之后再运行 `fpack notarize finish`。

## Windows

```yaml
  windows:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v7
      - uses: subosito/flutter-action@v2
      - run: dart pub global activate fpack 1.1.17
      - run: fpack build windows exe
```

GitHub 的 Windows runner 预装了 Inno Setup；代码签名时，把 `.pfx` 做成 base64 存入密钥，在 runner 上解码（[Windows](/platforms/windows#5-代码签名)）。

## Linux 与 Web

```yaml
  linux:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
      - uses: subosito/flutter-action@v2
      - run: sudo apt-get install -y ninja-build libgtk-3-dev rpm
      - run: dart pub global activate fpack 1.1.17 && echo "$HOME/.pub-cache/bin" >> $GITHUB_PATH
      - run: fpack build linux deb rpm web
```

在 `ubuntu-24.04` 上构建 snap 时，设置 `linux.snap.build_mode: destructive`；arm64 的 snap 用 `ubuntu-24.04-arm`（[Linux](/platforms/linux#7-snap-zh)）。

## 小技巧

- **缓存核心**：原生核心（约 4 MB）每个版本只下载一次，存放在 `~/.cache/fpack`（Linux）、`~/Library/Caches/fpack`（macOS）或 `%LOCALAPPDATA%\fpack`（Windows）。缓存这个目录就能跳过下载。
- **机器可读的结果**：`fpack build … --json > result.json` 会写出每个产物的路径、大小、SHA-256 和公证状态；人类可读的输出走 stderr。
- **尽早失败**：退出码 3 表示缺少前置条件，什么都没构建。
- **每次运行一个版本**：传 `--build-number "$GITHUB_RUN_NUMBER"`，重跑就不会与已有产物冲突（[版本](/guides/project-and-output#版本)）。
- **先预览**：在 pull request 里运行 `fpack build --all --dry-run`，可以看到发布会做什么。
