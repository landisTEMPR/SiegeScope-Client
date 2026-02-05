# Assets Directory

Place your application icons here:

- `icon.ico` - Windows icon (256x256 recommended)
- `icon.icns` - macOS icon  
- `icon.png` - Linux icon (256x256 or 512x512)

You can create these from a single high-resolution PNG using tools like:
- [IconConverter](https://iconverticons.com/online/)
- [GIMP](https://www.gimp.org/)
- macOS `iconutil` command

## Creating icons on macOS:

```bash
# From a 1024x1024 PNG:
mkdir icon.iconset
sips -z 16 16     icon.png --out icon.iconset/icon_16x16.png
sips -z 32 32     icon.png --out icon.iconset/icon_16x16@2x.png
sips -z 32 32     icon.png --out icon.iconset/icon_32x32.png
sips -z 64 64     icon.png --out icon.iconset/icon_32x32@2x.png
sips -z 128 128   icon.png --out icon.iconset/icon_128x128.png
sips -z 256 256   icon.png --out icon.iconset/icon_128x128@2x.png
sips -z 256 256   icon.png --out icon.iconset/icon_256x256.png
sips -z 512 512   icon.png --out icon.iconset/icon_256x256@2x.png
sips -z 512 512   icon.png --out icon.iconset/icon_512x512.png
sips -z 1024 1024 icon.png --out icon.iconset/icon_512x512@2x.png
iconutil -c icns icon.iconset
```
