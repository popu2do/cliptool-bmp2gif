# ClipTool BMP to GIF

ClipTool is a Windows desktop utility for building GIFs from images copied in File Explorer. It listens to the Windows clipboard, collects supported image files into a small Wails GUI, lets the user reorder frames, and writes the generated GIF back to the clipboard for direct paste.

The primary workflow is:

```text
File Explorer Ctrl+C -> ClipTool collects frames -> reorder/adjust interval -> Generate GIF -> target app Ctrl+V
```

## Features

- Collects image files from the Windows clipboard.
- Supports repeated copy operations with append-only collection.
- Deduplicates frames by normalized file path.
- Supports drag-and-drop frame ordering.
- Supports single-frame removal and clearing the current batch.
- Generates GIFs directly to the clipboard.
- Keeps the original comparison layout: all thumbnails on the left, current frame on the right.
- Provides a compact Wails + Svelte desktop UI.

## Supported Inputs

- BMP
- PNG
- JPG / JPEG
- RAW fingerprint image: `1560` bytes, `13x60`, `uint16`, 12-bit source
- RAW fingerprint image: `4800` bytes, `24x100`, `uint16`, 12-bit source
- RAW fingerprint image: `6240` bytes, `26x120` or `20x156`, `uint16`, 12-bit source
- RAW fingerprint image: `8320` bytes, `26x160`, `uint16`, 12-bit source
- RAW fingerprint image: `10240` bytes, `32x160`, `uint16`, 12-bit source
- RAW fingerprint image: `39192` bytes, `138x142`, `uint16`, 10-bit source
- RAW fingerprint image: `43808` bytes, `148x148`, `uint16`, 12-bit source
- RAW fingerprint image: `51200` bytes, `160x160`, `uint16`, 10-bit or 12-bit source
- RAW fingerprint image: `88200` bytes, `210x210`, `uint16`, 12-bit source
- BIN fingerprint image: `9600` bytes, `24x100`, `uint32`, 16-bit source
- BIN fingerprint image: `12480` bytes, `26x120` or `20x156`, `uint32`, 16-bit source
- BIN fingerprint image: `16640` bytes, `26x160`, `uint32`, 16-bit source
- BIN fingerprint image: `20480` bytes, `32x160`, `uint32`, 16-bit source
- BIN fingerprint image: `102400` bytes, `160x160`, `uint32`, 16-bit source
- BIN fingerprint image: `176400` bytes, `210x210`, `uint32`, 16-bit source

Fingerprint RAW/BIN support is implemented as an extension loader under `internal/extensions/fingerprint`.
The core image pipeline remains responsible for normal image loading, frame management, thumbnails, and GIF encoding.
Encrypted business dumps are intentionally not built into this tool.

## Clipboard Handling

ClipTool reads File Explorer copies through the standard Windows file clipboard formats first:

- `CF_HDROP`
- `FileNameW` / `FileName`
- OLE `IDataObject`

For long local file paths where Explorer exposes file items through shell clipboard data but `CF_HDROP` cannot be materialized, ClipTool falls back to `Shell IDList Array` and resolves PIDLs with `SHGetNameFromIDList(SIGDN_FILESYSPATH)`.
This keeps local Explorer copies working for long RAW/BIN sample paths without adding PowerShell polling or business-specific file copying behavior.

## Requirements

Runtime:

- Windows
- Microsoft Edge WebView2 Runtime
- PowerShell, used to write the generated GIF path back to the clipboard

Development:

- Go 1.24+
- Node.js and npm
- Wails v2.12.0, optional as a global command because `build.bat` can run the pinned version through `go run`

## Quick Start

Run `cliptool.exe`, then:

1. Select one or more supported images in Windows File Explorer.
2. Press `Ctrl+C`.
3. Review the thumbnails in ClipTool.
4. Drag thumbnails to adjust frame order.
5. Set `Gif图片间隔时间` if needed.
6. Click `生成 GIF / Enter`, or press Enter.
7. Paste the generated GIF into the target app with `Ctrl+V`.

After successful generation, the current frame list is cleared. Clicking `清空` also clears the current batch and ignores the current clipboard contents until the user copies again.

## Development

Install frontend dependencies:

```bash
cd frontend
npm install
```

Run Go tests:

```bash
go test ./...
```

Run frontend tests:

```bash
cd frontend
npm test
```

Build the frontend:

```bash
cd frontend
npm run build
```

Build the Windows executable:

```bash
build.bat
```

The build output is copied to `cliptool.exe`.

## Project Structure

```text
internal/core       Image loading, thumbnails, GIF layout and encoding
internal/clipboard  Windows clipboard read/write integration
internal/session    Frame collection, deduplication, deletion and ordering
frontend            Wails + Svelte UI
docs                User-facing wiki docs and design specs
```

## Documentation

- [Docs index](docs/index.md)
- [User guide](docs/cliptool-bmp2gif-user-guide.md)
- [GUI requirements spec](docs/spec/cliptool-bmp2gif-gui-spec.md)

## Notes

- This is not a general-purpose GIF editor.
- It does not crop, annotate, draw on images, or save project history.
- The first release is Windows-only.
- The generated GIF is written through a temporary file under `temp/`.

## License

License file is not included yet. Add a repository-level `LICENSE` before public distribution.
