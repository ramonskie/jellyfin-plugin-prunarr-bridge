using Jellyfin.Plugin.PrunarrBridge.Services;
using MediaBrowser.Controller;
using MediaBrowser.Controller.Library;
using Microsoft.Extensions.DependencyInjection;

namespace Jellyfin.Plugin.PrunarrBridge;

/// <summary>
/// Plugin service registrator.
/// </summary>
public class PluginServiceRegistrator : IPluginServiceRegistrator
{
    /// <inheritdoc />
    public void RegisterServices(IServiceCollection serviceCollection, IServerApplicationHost applicationHost)
    {
        serviceCollection.AddSingleton<SymlinkManager>();
        serviceCollection.AddSingleton<VirtualFolderManager>();
    }
}
