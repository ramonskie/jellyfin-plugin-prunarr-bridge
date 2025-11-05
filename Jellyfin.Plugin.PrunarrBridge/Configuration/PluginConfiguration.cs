using MediaBrowser.Model.Plugins;

namespace Jellyfin.Plugin.PrunarrBridge.Configuration;

/// <summary>
/// Plugin configuration.
/// </summary>
public class PluginConfiguration : BasePluginConfiguration
{
    /// <summary>
    /// Gets or sets the base path for symlinks (where "Leaving Soon" items will be symlinked).
    /// </summary>
    public string SymlinkBasePath { get; set; } = "/var/lib/jellyfin/leaving-soon";

    /// <summary>
    /// Gets or sets the name of the Virtual Folder for "Leaving Soon" items.
    /// </summary>
    public string VirtualFolderName { get; set; } = "Leaving Soon";

    /// <summary>
    /// Gets or sets a value indicating whether to auto-create the Virtual Folder on startup.
    /// </summary>
    public bool AutoCreateVirtualFolder { get; set; } = true;

    /// <summary>
    /// Gets or sets the API key for authenticating Prunarr requests.
    /// </summary>
    public string ApiKey { get; set; } = string.Empty;
}
