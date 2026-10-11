---
title: "Windows"
description: "便携 zip、Inno Setup 安装包、MSIX 与代码签名。"
---

代码签名是可选的。在 `windows.sign` 中选择一种证书来源：

- `.pfx` 文件：`certificate` + `password`（`FPACK_WINDOWS_CERTIFICATE` / `FPACK_WINDOWS_CERTIFICATE_PASSWORD`；密码不会被打印），
- 证书存储：`thumbprint`（`/sha1`）或 `subject`（`/n`），可配合 `store`（`/s`，默认 `My`）和 `machine_store: true`（`/sm`），
- 云签名的 signtool 插件，例如 Azure Trusted Signing：`dlib` + `dlib_metadata`（`/dlib` / `/dmdf`）。

`flutter build windows` 完成后，fpack 立即签名 Release 文件夹中的 `files`（默认 `*.exe` 和 `*.dll`），因此绿色版 zip、安装程序和 MSIX 中都是已签名的文件。Inno Setup 安装程序和卸载程序通过 `SignTool=` 签名；MSIX 由 `msix:create` 用同一证书签名（使用 `dlib` 时由 fpack 对生成的 MSIX 签名）。所有签名都使用 SHA-256 和 RFC 3161 时间戳（`timestamp_url`，默认 `http://timestamp.digicert.com`），并带有 `description`（`/d`）和 `url`（`/du`）；签名后用 `signtool verify /pa` 校验（`verify: false` 可关闭；失败时——例如自签名测试证书——只给出警告）。signtool 会在 PATH 或最新的 Windows SDK 中自动查找（也可设置 `signtool`）；`fpack doctor` 会报告它以及安装 SDK 的方法（`winget install Microsoft.WindowsSDK.10.0.26100`，需要管理员权限；没有管理员权限时，可从 NuGet 包 `Microsoft.Windows.SDK.BuildTools` 中解压出 signtool 并设置 `signtool`）。

```yaml
windows:
  sign:
    certificate: ${WINDOWS_PFX}            # 或 thumbprint: / subject: / dlib:
    password: ${WINDOWS_PFX_PASSWORD}
    url: https://example.com
```

Inno Setup 安装包可配置发布者/网址、版权、安装目录、开始菜单、桌面快捷方式、许可/信息页、图标与向导图片、最低 Windows 版本、`privileges`（`user` 免管理员 / `admin` / `ask`）以及多语言（`languages: [zh-CN, en]`，第一个为默认；旧版 Inno Setup 缺少的中文语言文件由 fpack 自带）。MSIX 的显示名、发布者、能力、文件关联、协议、版本等可直接在 `windows.msix` 中设置，优先于 pubspec 的 `msix_config`。

## 目标

| 目标 | 产物 | 需要 |
| --- | --- | --- |
| `windows` | `-windows-x64-portable.zip` | 装有「使用 C++ 的桌面开发」工作负载的 Visual Studio |
| `exe` | `-windows-x64-setup.exe` | [Inno Setup 6](https://jrsoftware.org/isinfo.php)（`ISCC.exe`；默认安装位置会自动查找，也可用 `windows.inno_setup.iscc` 指定） |
| `msix` | `-windows-x64.msix` | `dart pub add --dev msix` |

三者共用一次 `flutter build windows`。

## 注意事项

- **项目路径**：项目请放在只含 ASCII 字符的路径下。路径含中文等字符时 `flutter build windows` 会失败（`Unable to read file … app.dill`），`fpack doctor` 会给出警告；也可以在 Windows 区域设置中启用“Beta 版：使用 Unicode UTF-8 提供全球语言支持”。
- **MSIX 描述**：`msix` 包只从 `pubspec.yaml` 读取描述（`msix_config.description`，否则为 `description`）；当 `windows.msix.description` / `app.description` 与之不同时，fpack 会提示需要添加的条目。
- **钩子**通过 `cmd /S /C` 运行，钩子变量请写 `%VAR%`（如 `%FPACK_OUTPUT_DIR%`）；带引号的路径会按原样生效。

## 安装包示例

```yaml
windows:
  inno_setup:
    languages: [zh-CN, en]      # 第一个为默认语言
    privileges: user            # user | admin | ask
    desktop_icon: checked       # checked | unchecked | none
    license_file: LICENSE.txt
  msix:
    publisher: "CN=Your Company"
    capabilities: [internetClient]
  sign:
    certificate: C:/certs/codesign.pfx
    password: ${WINDOWS_CERT_PASSWORD}
```

`windows.inno_setup.app_id` 由 `fpack init` 生成一次——请保持不变，升级安装时才能替换已安装的版本。要使用自己的 `.iss` 脚本，请设置 `windows.inno_setup.script`。
