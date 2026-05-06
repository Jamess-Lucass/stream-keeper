# Stream Keeper

A CLI tool for keeping YouTube live streams alive by streaming a static image using FFmpeg.

## Features

- Stream a single image to one or multiple YouTube live streams
- Support for batch streaming via CSV file with multiple stream keys
- Single FFmpeg process with tee muxer for efficient multi-stream output
- Automatic restart on unexpected FFmpeg exit
- Configurable output resolution with auto-calculated bitrate

## Installation

### Prerequisites

**FFmpeg** is required. Install it for your platform:

#### macOS

```bash
brew install ffmpeg
```

#### Ubuntu/Debian

```bash
sudo apt-get install ffmpeg
```

#### Windows

Download from [ffmpeg.org](https://ffmpeg.org/download.html) or use:

```bash
choco install ffmpeg
```

#### Verify Installation

```bash
ffmpeg -version
```

### Option 1: Download Pre-built Binary

Download the latest executable for your platform from the [Releases](https://github.com/Jamess-Lucass/stream-keeper/releases) page.

### Option 2: Build from Source

Ensure you have Go 1.24+ installed, then:

```bash
git clone https://github.com/Jamess-Lucass/stream-keeper.git
cd stream-keeper
go build -o stream-keeper
```

## Usage

### Single Stream

Stream an image to a single YouTube live stream:

```bash
./stream-keeper stream -i image.png -k YOUR_STREAM_KEY
```

### Multiple Streams (CSV)

Stream an image to multiple YouTube live streams using a CSV file:

```bash
./stream-keeper stream -i image.png -c keys.csv
```

The CSV file should contain one stream key per line:

```
key1
key2
key3
```

### Custom Resolution

Specify the output resolution to match your YouTube stream configuration:

```bash
./stream-keeper stream -i image.png -k YOUR_STREAM_KEY -r 1280x720
```

## Options

- `-i, --image` (required): Path to the image file to stream
- `-k, --key`: Single YouTube stream key
- `-c, --csv`: Path to CSV file with multiple stream keys
- `-r, --resolution`: Output resolution in WxH format (default: `1920x1080`)

Note: You must specify either `--key` or `--csv`, but not both.

## FFmpeg Configuration

The tool uses the following FFmpeg settings:

- **Input**: Static image with looping enabled
- **Frame rate**: 10 FPS
- **Video codec**: H.264 (libx264) with `tune=stillimage`
- **Bitrate**: Auto-calculated based on resolution (e.g. ~1555 kbps for 1080p)
- **Output**: Tee muxer with `onfail=ignore` for fault-tolerant multi-stream output
- **Audio**: Silent AAC audio source (128 kbps)
- **Restart**: Automatic restart with 5-second delay on unexpected exit

## Requirements

- **FFmpeg**: Must be installed and available in your PATH (see Installation section above)
- **Valid YouTube Live Stream Keys**: Obtain from your YouTube Studio dashboard
- **Go 1.24+**: Only required if building from source

## License

MIT
