using MediaBrowser.Model.Plugins;

namespace Jellyfin.Plugin.OxiCleanarr.Configuration;

/// <summary>
/// Plugin configuration.
/// Note: v2.0+ has no configuration - all paths are provided via API requests.
/// </summary>
public class PluginConfiguration : BasePluginConfiguration
{
    // Empty configuration - plugin is now stateless and configuration-free
}
