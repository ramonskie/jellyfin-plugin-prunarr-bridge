# Plugin Icons

Jellyfin plugins can have icons displayed in the plugin catalog. This directory contains icon assets for the Prunarr Bridge plugin.

## Icon Requirements

- **Format**: PNG recommended (SVG also supported)
- **Sizes**: 
  - `thumb.png` - 300x300px (catalog thumbnail)
  - `logo.png` - 512x512px (full logo)
- **Background**: Transparent recommended
- **Style**: Simple, recognizable, matches Jellyfin's aesthetic

## Current Icon

A basic placeholder icon is provided. Replace with a custom design that represents:
- Prunarr branding
- Connection/bridge concept
- Media management theme

## Using Custom Icons

1. Create your icon images (300x300 and 512x512)
2. Save as `thumb.png` and `logo.png` in this directory
3. Reference in `manifest.json`:

```json
{
  "imageUrl": "https://raw.githubusercontent.com/ramonskie/jellyfin-plugin-prunarr-bridge/main/images/thumb.png"
}
```

## Design Ideas

- Bridge connecting two systems
- Media files with a timer/clock
- Prunarr logo adapted for Jellyfin style
- Folder with symlink arrows

## Resources

- [Jellyfin Design Guidelines](https://jellyfin.org/contribute/branding/)
- Free icon resources: Feather Icons, Heroicons, FontAwesome
- Design tools: Inkscape (free), Figma (free tier), Adobe Illustrator
