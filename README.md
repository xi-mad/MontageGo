# MontageGo  MontageGo

[![Go version](https://img.shields.io/github/go-mod/go-version/xi-mad/MontageGo?style=flat-square)](https://golang.org)
[![License](https://img.shields.io/github/license/xi-mad/MontageGo?style=flat-square)](LICENSE)
[![Release](https://img.shields.io/github/v/release/xi-mad/MontageGo?style=flat-square)](https://github.com/xi-mad/MontageGo/releases)

---

`MontageGo` 是一个现代化、功能强大的命令行工具，它通过智能地包装 FFmpeg，为您的视频文件生成布局精美、信息丰富的缩略图预览图集 (Thumbnail Sheet)。

与直接编写复杂、难以维护的 FFmpeg 命令不同，`MontageGo` 使用 Go 语言动态计算所有参数，生成一条完美的 FFmpeg 命令来执行任务。这使得它能够实现复杂的布局、美学定制和灵活的逻辑控制。

## ✨ 功能特性

- **动态美学计算**: 自动计算缩略图尺寸、间距和整体布局，确保视觉效果和谐。
- **丰富的信息展示**: 在图集顶部展示视频的文件名、分辨率、帧率、码率、时长、文件大小和编码格式。
- **智能选帧策略**: 自动跳过视频的开头和结尾部分，在中间 90% 的核心内容区域均匀选取帧，避免选中片头或黑屏。
- **自适应标题**: 无论视频文件名多长，标题字体大小都会自动缩放以完整地显示在图片内。
- **高度可定制**:
    - 自定义网格布局（行数、列数）。
    - 自定义缩略图尺寸、内外边距、边框和头部高度。
    - 自定义所有颜色（背景、字体、阴影、边框）。
    - 自定义输出 JPEG 图像的质量。
- **灵活的日志系统**:
    - **默认模式**: 显示程序日志和 FFmpeg 进度。
    - **静默模式 (`-q`)**: 只在成功或失败时输出关键信息。
    - **详细模式 (`-v`)**: 打印出最终执行的完整 FFmpeg 命令，便于调试。
    - 支持独立控制程序日志和 FFmpeg 日志的显示。
- **流式输出**: 支持将生成的图像数据直接输出到标准输出 (stdout)，方便与其他工具通过管道 (`|`) 组合使用。

## 🛠️ 安装与依赖

### 依赖项

在运行 `MontageGo` 之前，您必须确保您的系统中已经安装了 **`ffmpeg`** 和 **`ffprobe`**，并且它们所在的路径已经被添加到了系统的 `PATH` 环境变量中。

您也可以通过 `--ffmpeg-path` 和 `--ffprobe-path` 标志来指定它们的可执行文件路径。

### 从源码构建

1.  克隆本仓库:
    ```bash
    git clone https://github.com/xi-mad/MontageGo.git
    cd MontageGo
    ```

2.  构建可执行文件:
    ```bash
    go build -o MontageGo ./cmd/montagego
    ```
    您也可以使用项目提供的跨平台构建脚本：
    ```bash
    ./scripts/build.sh
    ```

## 🚀 使用方法

```bash
./MontageGo [视频文件路径] [选项...]
```

### 基本示例

为视频创建一个 4x5 的默认图集：
```bash
./MontageGo "my awesome movie.mp4"
```

### 高级示例

创建一个 5x6 的网格，每个缩略图宽度为 400px，背景为浅灰色，并输出到指定路径：
```bash
./MontageGo "my awesome movie.mp4" -c 5 -r 6 --thumb-width 400 --bg-color "#eeeeee" --font-color "#333333" -o "~/Desktop/my_montage.jpg"
```

### 流式输出示例

生成图集并直接通过管道在 macOS 的预览中打开它：
```bash
./MontageGo "my awesome movie.mp4" -q -o - | open -a Preview.app -f
```

## 命令行选项

| 短标志 | 长标志                | 描述                                                       | 默认值          |
|--------|-----------------------|------------------------------------------------------------|-----------------|
| `-o`   | `--output`            | 输出路径。使用 `-` 可将图像流输出到 stdout。               | `[输入文件名]_montage.jpg` |
| `-c`   | `--columns`           | 网格的列数。                                               | `4`               |
| `-r`   | `--rows`              | 网格的行数。                                               | `5`               |
|        | `--thumb-width`       | 每个缩略图的宽度。                                         | `640`             |
|        | `--thumb-height`      | 每个缩略图的高度。`-1` 表示根据宽度和宽高比自动缩放。     | `-1`              |
|        | `--padding`           | 缩略图之间的内边距（像素）。                               | `5`               |
|        | `--margin`            | 网格距离图片边缘的外边距（像素）。                         | `20`              |
|        | `--header`            | 顶部标题区域的高度（像素）。                               | `150`             |
|        | `--font-file`         | 用于渲染文本的 `.ttf` 字体文件路径。                       | (无)            |
|        | `--font-color`        | 主字体颜色。                                               | `white`           |
|        | `--shadow-color`      | 文本阴影颜色。                                             | `black`           |
|        | `--bg-color`          | 图集背景色。                                               | `#222222`        |
|        | `--jpeg-quality`      | 输出 JPEG 图像的质量 (1-31, 数值越低质量越高)。        | `2`               |
|        | `--border-thickness`  | 缩略图边框的厚度（像素）。`0` 表示无边框。                | `1`               |
|        | `--border-color`      | 缩略图边框的颜色。                                         | `#111111`        |
|        | `--ffmpeg-path`       | `ffmpeg` 可执行文件的路径。                                | `ffmpeg`          |
|        | `--ffprobe-path`      | `ffprobe` 可执行文件的路径。                               | `ffprobe`         |
| `-q`   | `--quiet`             | 静默模式，隐藏所有程序和 FFmpeg 的日志。                   | `false`           |
| `-v`   | `--verbose`           | 详细模式，打印出将要执行的完整 FFmpeg 命令。               | `false`           |
|        | `--show-app-log`      | 显示程序自身的日志。                                       | `true`            |
|        | `--show-ffmpeg-log`   | 显示 FFmpeg 进程的实时输出。                               | `true`            |


## 🤝 贡献

欢迎提交 Pull Request 或创建 Issue 来为 `MontageGo` 做出贡献！

## 📄 许可证

本项目基于 [MIT License](LICENSE) 授权。
